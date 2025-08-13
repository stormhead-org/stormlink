## Stormlink GraphQL Gateway

Назначение: единая HTTP/WS-точка для клиентов. Проксирует бизнес-операции в gRPC‑микросервисы: `auth`, `user`, `mail`, `media`. Хранит и отдает данные через Ent/Postgres, медиа кладет в S3-совместимое хранилище.

### Запуск

- ENV-файл: `server/.env` (опционально). Переменные окружения ниже.
- Команда: `go run ./server/cmd -reset-db=false -seed=false`
- HTTP адрес: `GRAPHQL_HTTP_ADDR` (по умолчанию `:8080`)
- Точки:
  - POST/GET `/query` — GraphQL (HTTP)
  - WS `/query` — GraphQL Subscriptions (WebSocket)
  - GET `/healthz`, `/readyz`
  - GET `/storage/{key}` — прокси к S3
  - `/` — Playground (только если `ENV!=production`)

### Переменные окружения (ключевые)

- DB: `DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, SSL_MODE`
- GraphQL: `GRAPHQL_HTTP_ADDR`, `FRONTEND_ORIGIN`, `ENV`, `GRAPHQL_MAX_COMPLEXITY`, `GRAPHQL_MAX_BODY_BYTES`
- JWT: `JWT_SECRET`
- Cookies: `APP_COOKIE_DOMAIN`, `ENV` (влияет на Secure)
- gRPC адреса: `AUTH_GRPC_ADDR, USER_GRPC_ADDR, MAIL_GRPC_ADDR, MEDIA_GRPC_ADDR`, `GRPC_INSECURE=true|false`
- Uploads: `UPLOAD_MAX_BYTES` (байт, по умолчанию 20MB; поддержка: image/jpeg|png|gif)
- S3: `S3_BUCKET, S3_REGION, S3_ENDPOINT, S3_ACCESS_KEY_ID, S3_SECRET_ACCESS_KEY, S3_USE_PATH_STYLE, S3_ALIAS_HOST`

### Аутентификация и куки

- После `LoginUser` и `UserRefreshToken` сервер выставляет `HttpOnly` куки:
  - `auth_token` (15 мин, SameSite=Lax, Secure в проде)
  - `refresh_token` (7 дней)
- Токены больше не возвращаются в теле GraphQL‑ответов.
- Для вызовов, требующих авторизации, клиент должен:
  - либо передавать заголовок `Authorization: Bearer <access>` (если вы храните токен отдельно),
  - либо полагаться на `HttpOnly` куки и вызывать резолверы без спец‑заголовков (сервер добавит контекст автоматически).

### CORS и CSRF

- CORS ограничен до `FRONTEND_ORIGIN`, `AllowCredentials=true`.
- Для POST `/query` дополнительно проверяется заголовок `Origin` на совпадение с `FRONTEND_ORIGIN`.
- WS `CheckOrigin` также сверяет `Origin` с `FRONTEND_ORIGIN`.

### Безопасность GraphQL

- Complexity limit: `GRAPHQL_MAX_COMPLEXITY` (по умолчанию 300)
- APQ включен (LRU 1000 ключей)
- HTTP таймауты сервера, лимит размеров тела (`GRAPHQL_MAX_BODY_BYTES`, дефолт 1MB)
- Простой rate limit на `/query` по IP (10 rps, burst 30)
- В проде отключены Introspection и Playground

### Готовность и здоровье

- `/healthz` — всегда `200 ok` (проверка живости HTTP‑процесса).
- `/readyz` — проверяет готовность зависимостей:
  - Postgres через Ent (быстрый запрос);
  - S3 (`GetBucketLocation` через общий `shared/s3` клиент);
  - gRPC upstream (auth, user, mail, media) через стандартный `grpc.health.v1.Health`.

Таймаут проверки по умолчанию ~800ms.

### Единая модель ошибок

- gRPC‑сервисы возвращают ошибки через `shared/errors` (`FromGRPCCode`, `ToGRPC`).
- GraphQL имеет глобальный ErrorPresenter, конвертирующий ошибки в нормализованный вид:
  - gRPC‑status → `extensions.code` (например, `NotFound`, `Unauthenticated`, `InvalidArgument`);
  - `ent.NotFound` → `code=NotFound`;
  - остальные → `code=Internal` с безопасным сообщением.

Резолверам рекомендуется использовать ошибки из gRPC слоёв или возвращать коды через `shared/errors`, чтобы клиент получал корректные `extensions.code`.

### API: основные операции

GraphQL схемы: `server/graphql/handlers.graphql` и `server/graphql/ent.graphql`. Ниже — частые сценарии клиента.

1. Регистрация

```
mutation Register($name:String!, $email:String!, $password:String!) {
  registerUser(input:{name:$name, email:$email, password:$password}) {
    message
  }
}
```

2. Логин (устанавливает куки, тело без токенов)

```
mutation Login($email:String!, $password:String!) {
  loginUser(input:{email:$email, password:$password}) {
    user { id slug email name }
  }
}
```

3. Обновление токена (читает refresh из HttpOnly cookie; ориентируйтесь на Set-Cookie)

```
mutation { userRefreshToken { accessToken refreshToken } }
```

Ответные поля могут быть пусты по договорённости; ориентируйтесь на Set-Cookie.

4. Текущий пользователь (требует авторизации)

```
query { getMe { id slug email name avatar { url } }
}
```

5. Загрузка медиа (до 20MB, только изображения)

```
mutation($file: Upload!) { uploadMedia(file:$file) { id url filename } }
```

6. Сообщества/посты/комментарии: см. типы и резолверы в `handlers.graphql` и автосгенерированные модели `ent.graphql`.

### Пример клиента (fetch)

```
fetch("/query", {
  method: "POST",
  credentials: "include", // важно для куки
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ query, variables })
})
```

WS (graphql-ws): подключайтесь к `ws(s)://<host>/query` с Origin=`FRONTEND_ORIGIN`. Токен можно отправлять в `connection_init` как `Authorization: Bearer <token>`, либо полагаться на куки.

### S3 и /storage

- Медиа кладутся в S3 с UUID‑именами; публичный доступ через GET `/storage/<dir>/<filename>`.
- Ответы кэшируются клиентами (сервер выставляет Cache-Control). Добавьте ETag/Last-Modified, если требуется сильнее кэширование.

### Границы и лимиты

- Body: по умолчанию 1MB на POST `/query` (настраивается)
- Upload: 20MB и только JPEG/PNG/GIF (настраивается)
- RPS: 10 rps/burst 30 на IP (базовый, настраивайте под ingress)

### Архитектура сервера

- `cmd/main.go` — bootstrap: env, DB, миграции, запуск HTTP/WS (S3 клиент инициализируется в модуле сервера)
- `cmd/modules/graphql_server.go` — маршруты, CORS, WS, таймауты, лимиты, `/healthz`, `/readyz`, `/storage`
- `graphql/*` — схема, резолверы, модели
- `middleware/*` — HTTP/gRPC аутентификация, rate limiting
- `usecase/*` — доменная логика
- `shared/*` — общие утилиты (jwt, http/куки, s3, mail, mapper, errors)

### Примечания по продакшену

- Установите `ENV=production`, задайте `FRONTEND_ORIGIN`, включите TLS на внешнем прокси/ingress.
- Для межсервисного gRPC используйте TLS/mTLS (`GRPC_INSECURE=false`).
- Запретите флаг `-reset-db` в проде.
