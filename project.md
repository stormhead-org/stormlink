# Анализ backend кодовой базы: Stormlink

## 📁 Структура проекта

```text
stormlink/
├── server/                    # GraphQL-gateway: HTTP/WS, маршруты, резолверы, usecase
│   ├── cmd/                   # Bootstrap: env, DB, миграции, запуск GraphQL, S3, маршруты
│   ├── ent/                   # Ent ORM (сгенерированные модели, билдеры запросов)
│   ├── graphql/               # Схемы GraphQL, резолверы, модели, связка с usecase/gRPC
│   ├── grpc/                  # gRPC клиенты и protobuf (auth, user, mail, media)
│   ├── middleware/            # HTTP/gRPC middleware: аутентификация, rate limit, аудит
│   ├── model/                 # Модели доменных прав/DTO над ent
│   └── usecase/               # Бизнес-логика (User, Community, Post, Comment, Role, ...)
├── services/                  # Микросервисы: auth, user, mail, media, workers
│   ├── auth/                  # JWT, логин/логаут, refresh, ValidateToken, GetMe
│   ├── user/                  # Регистрация, роли, первичная настройка Host
│   ├── mail/                  # Подтверждение почты и повторная отправка
│   ├── media/                 # Медиа-операции (интеграция с S3)
│   └── workers/               # Фоновые обработчики (через RabbitMQ)
├── shared/                    # Общий kernel: auth context, http cookies, jwt, s3, redis, mq
│   ├── auth/ | http/ | jwt/ | s3/ | redis/ | rabbitmq/ | mapper/ | errors/
├── tests/                     # Unit, integration (testcontainers), performance
└── proto/                     # proto-схемы и buf-конфиги
```

- Организация: mix из Clean Architecture и DDD.
  - Transport: GraphQL gateway (+ WebSocket), вход в gRPC-сервисы.
  - Usecase: доменная логика в `server/usecase/*`.
  - Data: доступ через Ent ORM (`server/ent`), без отдельного repository-слоя, но с чёткими usecase-границами.
  - Shared kernel: `shared/*` — переиспользуемая инфраструктура.
- Развёртывание: монорепозиторий с несколькими сервисами (gateway + микросервисы).

## 🛠 Технологический стек

| Категория | Технология              | Версия/прим.      | Назначение                      |
| --------- | ----------------------- | ----------------- | ------------------------------- |
| Язык      | Go                      | go.mod: 1.24.2    | Основной runtime                |
| ORM       | Ent                     | v0.14.4           | Типобезопасный доступ к БД      |
| GraphQL   | gqlgen                  | v0.17.81          | Сервер и кодоген                |
| gRPC      | google.golang.org/grpc  | v1.70.0           | Межсервисное API                |
| Gateway   | grpc-gateway/v2         | v2.26.3           | REST proxy (заявлен)            |
| БД        | PostgreSQL              | lib/pq            | Основная БД                     |
| Кэш       | Redis                   | go-redis/v9       | Кэш/сессии refresh              |
| Очереди   | RabbitMQ                | amqp091-go        | Рассылка писем и фоновые задачи |
| Хранилище | AWS S3                  | aws-sdk-go        | Медиа и /storage прокси         |
| JWT       | golang-jwt/jwt/v5       | v5.2.2            | Токены, валидация               |
| Валидация | protoc-gen-validate     | v1.1.0            | Валидация gRPC входа            |
| WebSocket | gorilla/websocket       | v1.5.0            | GraphQL Subscriptions           |
| Тесты     | testify, testcontainers | v1.11.1 / v0.34.0 | Unit/Integration                |
| Лимиты    | x/time/rate             | -                 | Rate limit HTTP/gRPC            |
| Сборка    | Makefile                | -                 | Тесты/линт/билд/CI              |

Примечание: Makefile объявляет GO_VERSION=1.21 для контейнера тестов; фактическая версия компилятора — 1.24.2 из go.mod.

## 🏗 Архитектура

- Bootstrap (`server/cmd/main.go`): env → DB → миграции → запуск GraphQL → graceful shutdown.
- GraphQL сервер (`server/cmd/modules/graphql_server.go`): маршруты `/query`, `/healthz`, `/readyz`, `/storage/*`, CORS, WebSocket, complexity limit, APQ, rate limit, аудит, CSRF.
- Подключение к микросервисам по gRPC (auth, user, mail, media) с клиентским интерсептором, автоматом прокидывающим Authorization из HTTP-контекста в gRPC metadata.
- Normalized error handling: единый маппинг gRPC/Ent ошибок в GraphQL extensions.code `shared/errors` + глобальный ErrorPresenter в gqlgen.

Кодовый пример — прокидывание авторизации в gRPC:

```84:96:/mnt/hp_nvme_1tb/Projects/GitHub/stormlink/server/cmd/modules/graphql_server.go
func authClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if authHeader, ok := ctx.Value("authorization").(string); ok && authHeader != "" {
		md := metadata.New(map[string]string{
			"authorization": authHeader,
		})
		ctx = metadata.NewOutgoingContext(ctx, md)
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}
```

Глобальный ErrorPresenter:

```165:189:/mnt/hp_nvme_1tb/Projects/GitHub/stormlink/server/cmd/modules/graphql_server.go
srv := handler.New(graphql.NewExecutableSchema(graphql.Config{Resolvers: resolver}))
srv.SetErrorPresenter(func(ctx context.Context, err error) *gqlerror.Error {
	if ent.IsNotFound(err) {
		// code=NotFound
		e := gqlerror.Errorf("not found")
		if e.Extensions == nil { e.Extensions = map[string]any{} }
		e.Extensions["code"] = codes.NotFound.String()
		return e
	}
	ge := errorsx.ToGraphQL(err)
	if ge == nil { return gqlerror.Errorf("unknown error") }
	e := gqlerror.Errorf("%s", ge.Message)
	if e.Extensions == nil { e.Extensions = map[string]any{} }
	e.Extensions["code"] = ge.Code
	return e
})
```

HTTP-цепочка безопасности:

```271:307:/mnt/hp_nvme_1tb/Projects/GitHub/stormlink/server/cmd/modules/graphql_server.go
graphqlHandler := middleware.SecurityAuditMiddleware(
	middleware.AuditMiddleware(
		middleware.RateLimitMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Origin/CSRF, ограничение тела, куки-контекст
				ctx := httpWithCookies.WithHTTPContext(r.Context(), w, r)
				r = r.WithContext(ctx)
				middleware.HTTPAuthMiddleware(srv).ServeHTTP(w, r)
			}),
		),
	),
)
```

## 💾 Работа с данными

Подключение к Postgres с управлением пулом соединений:

```20:56:/mnt/hp_nvme_1tb/Projects/GitHub/stormlink/server/cmd/modules/database.go
func ConnectDB() *ent.Client {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", ...)
	db, err := sql.Open("postgres", dsn)
	// pool tuning
	db.SetMaxOpenConns(getEnvInt("DB_MAX_OPEN_CONNS", 15))
	db.SetMaxIdleConns(getEnvInt("DB_MAX_IDLE_CONNS", 5))
	db.SetConnMaxLifetime(time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 5)) * time.Minute)
	drv := entsql.OpenDB(dialect.Postgres, db)
	return ent.NewClient(ent.Driver(drv))
}
```

Миграции Ent и сидинг:

```68:98:/mnt/hp_nvme_1tb/Projects/GitHub/stormlink/server/cmd/modules/database.go
func MigrateDB(client *ent.Client, reset bool, seed bool) {
	if reset { client.Schema.Create(ctx, schema.WithDropIndex(true), schema.WithDropColumn(true)) }
	if seed { client.Schema.Create(ctx); Seed(client) } else { client.Schema.Create(ctx) }
}
```

Usecase вместо repository: чтение пользователя с eager loading:

```25:34:/mnt/hp_nvme_1tb/Projects/GitHub/stormlink/server/usecase/user/user.go
func (uc *userUsecase) GetUserByID(ctx context.Context, id int) (*ent.User, error) {
	return uc.client.User.Query().Where(user.IDEQ(id)).WithAvatar().WithUserInfo().WithHostRoles().WithCommunitiesRoles().Only(ctx)
}
```

Кэш/сессии в Redis (refresh-токены, ротация):

```176:195:/mnt/hp_nvme_1tb/Projects/GitHub/stormlink/services/auth/internal/service/service.go
newAccess, _ := jwt.GenerateAccessToken(userID)
newRefresh, _ := jwt.GenerateRefreshToken(userID)
if s.redis != nil {
	_ = s.redis.Del(ctx, "refresh:"+refreshToken).Err()
	_ = s.redis.Set(ctx, "refresh:"+newRefresh, userID, 7*24*time.Hour).Err()
}
httpCookies.SetAuthCookies(w, newAccess, newRefresh)
```

Асинхронная почта: постановка задач в очередь и отправка SMTP:

```62:79:/mnt/hp_nvme_1tb/Projects/GitHub/stormlink/shared/rabbitmq/publisher.go
err = ch.Publish("", q.Name, false, false, amqp.Publishing{DeliveryMode: amqp.Persistent, ContentType: "application/json", Body: body})
```

```10:29:/mnt/hp_nvme_1tb/Projects/GitHub/stormlink/shared/mail/send.go
addr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)
auth := smtp.PlainAuth("", config.SMTPUsername, config.SMTPPassword, config.SMTPHost)
_ = smtp.SendMail(addr, auth, config.FromEmail, []string{to}, message)
```

## ✅ Качество кода

- Линтер: Makefile содержит цели `lint`/`ci-lint` с golangci-lint.
- Нейминг: описательные имена, интерфейсы в `usecase`, отсутствие 1-2 символных идентификаторов.
- Документация: README и `server/README.md` подробно описывают эндпоинты, безопасность и ENV.
- Тесты: unit, integration (testcontainers), performance присутствуют. Покрытие собирается через `make test-coverage`.
- Изоляция логики: usecase слой отделяет бизнес-правила от транспорта/инфраструктуры.
- Логирование/метрики: логирование детальное (аудит/безопасность), метрик/трейсинга нет — рекомендация добавить.

## 🔧 Ключевые модули

1. HTTP Auth middleware (контекст + куки + валидация через gRPC auth):

```28:91:/mnt/hp_nvme_1tb/Projects/GitHub/stormlink/server/middleware/http_auth.go
ctx := httpCookies.WithHTTPContext(r.Context(), w, r)
authHeader := r.Header.Get("Authorization") // fallback на cookie auth_token
resp, err := authClient.ValidateToken(ctx, &protobuf.ValidateTokenRequest{Token: token})
if err == nil && resp.GetValid() { ctx = sharedauth.WithUserID(ctx, int(resp.GetUserId())) }
next.ServeHTTP(w, r.WithContext(ctx))
```

2. Rate limiting (HTTP): разные лимиты для анонимных/авторизованных:

```46:111:/mnt/hp_nvme_1tb/Projects/GitHub/stormlink/server/middleware/rate_limit.go
if isAuthenticated { limiter = rate.NewLimiter(20, 50) } else { limiter = rate.NewLimiter(5, 10) }
if !limiter.limiter.Allow() { http.Error(w, "Too many requests", http.StatusTooManyRequests); return }
```

3. GraphQL security и health:

```221:343:/mnt/hp_nvme_1tb/Projects/GitHub/stormlink/server/cmd/modules/graphql_server.go
// /readyz: БД, S3, gRPC health checks; CORS; WS CheckOrigin; APQ; complexity limit
```

4. Auth service (gRPC) — login/refresh с куками и Redis:

```42:91:/mnt/hp_nvme_1tb/Projects/GitHub/stormlink/services/auth/internal/service/service.go
u := s.client.User.Query().Where(entuser.EmailEQ(email)).WithAvatar().Only(ctx)
accessToken, _ := jwt.GenerateAccessToken(u.ID)
refreshToken, _ := jwt.GenerateRefreshToken(u.ID)
if s.redis != nil { _ = s.redis.Set(ctx, "refresh:"+refreshToken, u.ID, 7*24*time.Hour).Err() }
httpCookies.SetAuthCookies(w, accessToken, refreshToken)
```

5. User service — регистрация, первичная инициализация Host, верификация e-mail:

```33:88:/mnt/hp_nvme_1tb/Projects/GitHub/stormlink/services/user/internal/service/service.go
exists := s.client.User.Query().Where(entu.EmailEQ(req.GetEmail())).Exist(ctx)
passwordHash := bcrypt.GenerateFromPassword([]byte(req.GetPassword()+salt), ...)
// Назначение ролей, генерация verification token и постановка EmailJob в RabbitMQ
```

## Паттерны и best practices

- Context propagation: `shared/http` хранит `http.Request`/`ResponseWriter` в контексте, `shared/auth` — `userID`. gRPC metadata заполняется из контекста.
- Ошибки: централизованный маппинг `shared/errors` для gRPC и GraphQL, спец‑обработка `ent.IsNotFound`.
- Асинхронность: goroutine-очистка карт лимитеров; RabbitMQ для задач; WebSocket подписки.
- Производительность: connection pooling в БД, APQ, complexity limit, eager loading Ent.
- Валидация: `protoc-gen-validate` на уровне gRPC; `go-playground/validator` в зависимостях — можно задействовать шире.

## 🏗 Инфраструктура разработки

- Makefile: полноценный набор целей (fmt, vet, lint, test, coverage, docker, ci, fuzz, bench).
- ENV: `server/README.md` перечисляет ключевые переменные; `.env` подгружается опционально.
- CI/CD: цели `ci-*`; явных GitHub Actions в репозитории не обнаружено — можно добавить.
- Контейнеризация: цели docker присутствуют, testcontainers — для интеграционных тестов.
- Наблюдаемость: есть healthz/readyz; отсутствуют метрики/трейсинг — рекомендовано добавить OTel + Prometheus.

## 📋 Выводы и рекомендации

Сильные стороны:

- Чёткие границы слоёв, хорошая модульность и повторное использование `shared/*`.
- Безопасность на уровне gateway: CSRF, CORS, APQ, complexity limit, JWT, rate limits, аудит.
- Микросервисы для auth/user/mail/media; согласованный error handling; единый cookie‑флоу.
- Богатый Makefile и тестовая инфраструктура (incl. testcontainers).

Зоны роста:

- Добавить OpenTelemetry (traces + metrics) и структурированное логирование (zap).
- Ввести миграции с версионированием (например, atlas migrate/goose) вместо «reset» mode для prod.
- Больше кеширования (Redis) на read‑трафике (профили, ленты) и кеш GraphQL.
- Политики rate limiting на операцию/пользователя (не только per IP).
- Синхронизировать версии Go между Makefile и go.mod.

Уровень сложности проекта: уверенный middle → senior-friendly (микросервисы, GraphQL, Ent, безопасность, очереди, testcontainers).
