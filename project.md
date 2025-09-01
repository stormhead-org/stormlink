# Анализ backend кодовой базы: Stormlink⚡

## 📁 Структура проекта

```
stormlink/
├── server/                    # Основной GraphQL сервер
│   ├── cmd/                   # Точка входа и модули инициализации
│   ├── ent/                   # Ent ORM сгенерированный код и схемы
│   ├── graphql/               # GraphQL резолверы и модели
│   ├── grpc/                  # gRPC протобуф определения для клиентов
│   ├── middleware/            # HTTP/gRPC middleware (auth, rate limiting, audit)
│   ├── model/                 # Бизнес модели и DTO
│   └── usecase/               # Слой бизнес-логики (use cases)
├── services/                  # Микросервисы (auth, user, mail, media, workers)
│   ├── auth/                  # Сервис аутентификации (JWT, login/logout)
│   ├── mail/                  # Email сервис (SMTP, верификация)
│   ├── media/                 # Медиа сервис (S3, файлы)
│   ├── user/                  # Пользовательский сервис
│   └── workers/               # Фоновые задачи
├── shared/                    # Общие утилиты и библиотеки
│   ├── auth/                  # Контекст аутентификации
│   ├── errors/                # Нормализация ошибок gRPC/GraphQL
│   ├── http/                  # HTTP контекст и работа с cookies
│   ├── jwt/                   # JWT токены и хеширование
│   ├── mail/                  # SMTP клиент
│   ├── rabbitmq/              # Очереди сообщений
│   ├── redis/                 # Redis клиент
│   └── s3/                    # S3-совместимое хранилище
├── proto/                     # gRPC протобуф схемы
├── tests/                     # Тестовая инфраструктура
│   ├── unit/                  # Юнит тесты
│   ├── integration/           # Интеграционные тесты
│   ├── performance/           # Нагрузочные тесты
│   └── fixtures/              # Тестовые данные
└── tools/                     # Утилиты разработки
```

### Принципы организации кода

- **Микросервисная архитектура**: Основной GraphQL сервер + специализированные gRPC микросервисы
- **Clean Architecture**: Разделение на слои usecase, repository (через Ent), и transport (GraphQL/gRPC)  
- **Domain-Driven Design**: Организация по доменным сущностям (User, Community, Post, Comment)
- **Shared Kernel**: Общие утилиты вынесены в `shared/` для переиспользования

## 🛠 Технологический стек

| Категория | Технология | Версия | Назначение |
|-----------|------------|---------|------------|
| **Runtime** | Go | 1.24.2 | Основной язык |
| **ORM** | Ent | v0.14.4 | Type-safe ORM с кодогенерацией |
| **GraphQL** | gqlgen | v0.17.78 | GraphQL сервер с кодогенерацией |
| **gRPC** | google.golang.org/grpc | v1.70.0 | Межсервисное взаимодействие |
| **gRPC Gateway** | grpc-gateway | v2.26.3 | REST API проксирование |
| **Database** | PostgreSQL | - | Основная БД |
| **Cache** | Redis | v9.12.0 | Кеширование и сессии |
| **Message Queue** | RabbitMQ | v1.10.0 | Асинхронные задачи |
| **Storage** | AWS S3 | v1.55.7 | Файловое хранилище |
| **Authentication** | JWT | v5.2.2 | Токены доступа |
| **Validation** | go-playground/validator | v10.26.0 | Валидация входных данных |
| **WebSocket** | gorilla/websocket | v1.5.0 | Реальное время (GraphQL подписки) |
| **Testing** | testify + testcontainers | v1.10.0 | Юнит и интеграционные тесты |
| **Rate Limiting** | golang.org/x/time/rate | - | Защита от DDoS |
| **CORS** | rs/cors | v1.11.1 | Cross-origin запросы |

## 🏗 Архитектура

### Общая архитектура системы

```
[Frontend NextJS] 
    ↓ HTTP + WebSocket
[GraphQL Gateway Server :8080]
    ↓ gRPC calls
[Микросервисы]
    ├─ Auth Service :4001
    ├─ User Service :4002  
    ├─ Mail Service :4003
    └─ Media Service :4004
    ↓
[Shared Infrastructure]
    ├─ PostgreSQL (Ent ORM)
    ├─ Redis (кеш, сессии)
    ├─ RabbitMQ (очереди)
    └─ S3 (файлы)
```

### Архитектурные слои

#### 1. Transport Layer (GraphQL/gRPC)
```go
// server/graphql/*.resolvers.go
func (r *queryResolver) User(ctx context.Context, id int) (*ent.User, error) {
    return r.UserUC.GetUserByID(ctx, id)
}
```

#### 2. Use Case Layer 
```go
// server/usecase/user/user.go
type UserUsecase interface {
    GetUserByID(ctx context.Context, id int) (*ent.User, error)
    GetPermissionsByCommunities(ctx context.Context, userID int, communityIDs []int) (map[int]*model.CommunityPermissions, error)
}

func (uc *userUsecase) GetUserByID(ctx context.Context, id int) (*ent.User, error) {
    return uc.client.User.Query().
        Where(user.IDEQ(id)).
        WithAvatar().
        WithUserInfo().
        WithHostRoles().
        WithCommunitiesRoles().
        Only(ctx)
}
```

#### 3. Repository Layer (Ent ORM)
```go
// server/ent/schema/user.go
type User struct {
    ent.Schema
}

func (User) Fields() []ent.Field {
    return []ent.Field{
        field.Int("id").Unique(),
        field.String("name").NotEmpty(),
        field.String("slug").Unique().NotEmpty(),
        field.String("email").Unique().NotEmpty(),
        field.String("password_hash").NotEmpty().Annotations(entgql.Skip(entgql.SkipAll)),
        field.Bool("is_verified").Default(false),
        field.Time("created_at").Default(time.Now),
    }
}
```

### Межсервисное взаимодействие

#### gRPC с авто-авторизацией
```go
// server/cmd/modules/graphql_server.go
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

#### Нормализация ошибок
```go
// shared/errors/errors.go
func ToGraphQL(err error) *GraphQLError {
    if s, ok := status.FromError(err); ok {
        return &GraphQLError{
            Message: s.Message(),
            Code: s.Code().String(),
        }
    }
    return &GraphQLError{Message: err.Error(), Code: "INTERNAL"}
}
```

## 💾 Работа с данными

### База данных (PostgreSQL + Ent ORM)

#### Подключение с pool management
```go
// server/cmd/modules/database.go
func ConnectDB() *ent.Client {
    db, err := sql.Open("postgres", dsn)
    
    maxOpenConns := getEnvInt("DB_MAX_OPEN_CONNS", 15)
    maxIdleConns := getEnvInt("DB_MAX_IDLE_CONNS", 5)
    connMaxLifetime := getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 5)
    
    db.SetMaxOpenConns(maxOpenConns)
    db.SetMaxIdleConns(maxIdleConns)
    db.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Minute)
    
    drv := entsql.OpenDB(dialect.Postgres, db)
    return ent.NewClient(ent.Driver(drv))
}
```

#### Миграции
```go
func MigrateDB(client *ent.Client, reset bool, seed bool) {
    if reset {
        if err := client.Schema.Create(context.Background(), 
            schema.WithDropIndex(true),
            schema.WithDropColumn(true)); err != nil {
            log.Fatalf("ошибка сброса схемы: %v", err)
        }
    }
}
```

### Схема данных (основные сущности)

#### User - центральная сущность
```go
func (User) Edges() []ent.Edge {
    return []ent.Edge{
        edge.To("avatar", Media.Type).Field("avatar_id").Unique(),
        edge.To("posts", Post.Type),
        edge.To("comments", Comment.Type),
        edge.To("following", UserFollow.Type),
        edge.To("communities_owner", Community.Type),
        edge.To("communities_moderator", CommunityModerator.Type),
        edge.To("posts_likes", PostLike.Type),
        edge.To("bookmarks", Bookmark.Type),
    }
}
```

#### Community - сообщества
```go 
func (Community) Fields() []ent.Field {
    return []ent.Field{
        field.Int("id").Unique(),
        field.Int("owner_id"),
        field.String("title").NotEmpty(),
        field.String("slug").Unique().NotEmpty(),
        field.String("description").Optional().Nillable(),
        field.Bool("community_has_banned").Default(false),
    }
}
```

### Кеширование (Redis)
```go
// shared/redis/client.go - инициализация клиента
// services/auth/internal/service/service.go - сессии и refresh токены
if s.redis != nil {
    ttl := 7 * 24 * time.Hour
    _ = s.redis.Set(ctx, "refresh:"+refreshToken, userID, ttl).Err()
}
```

### Очереди (RabbitMQ)
```go
// shared/rabbitmq/ - публикация задач email верификации
// services/workers/ - обработчики фоновых задач
```

## ✅ Качество кода

### Стандарты и соглашения

#### Структурирование пакетов
- **Интерфейсы в usecase**: Определение контрактов бизнес-логики
- **Реализация в отдельных файлах**: `user.go`, `user_permissions.go`, `user_status.go`
- **Тесты рядом**: `user_test.go` в том же пакете

#### Нейминг
- **CamelCase** для публичных методов и типов
- **camelCase** для приватных
- **Описательные имена**: `GetPermissionsByCommunities`, `ValidateToken`
- **Контекстные префиксы**: `userUsecase`, `authClient`

### Обработка ошибок

#### Wrapping и типизация
```go
// shared/errors/errors.go
func FromGRPCCode(code codes.Code, message string, cause error) error {
    return status.Error(code, message)
}

// Использование в сервисах  
if err := jwt.ComparePassword(u.PasswordHash, password, u.Salt); err != nil {
    return nil, errorsx.FromGRPCCode(codes.Unauthenticated, "invalid credentials", nil)
}
```

#### GraphQL Error Presenter
```go
srv.SetErrorPresenter(func(ctx context.Context, err error) *gqlerror.Error {
    if ent.IsNotFound(err) {
        e := gqlerror.Errorf("not found")
        e.Extensions["code"] = codes.NotFound.String()
        return e
    }
    ge := errorsx.ToGraphQL(err)
    // нормализация через shared/errors
})
```

### Безопасность

#### Middleware Stack
```go
// Rate limiting per IP
middleware.RateLimitMiddleware(
    // Security audit logging  
    middleware.SecurityAuditMiddleware(
        // General request logging
        middleware.AuditMiddleware(
            // JWT validation
            middleware.HTTPAuthMiddleware(srv)
        )
    )
)
```

#### CSRF Protection
```go
if os.Getenv("CSRF_ENABLE") == "true" {
    c, err := r.Cookie("csrf_token")
    tokenHeader := r.Header.Get("X-CSRF-Token")
    if tokenHeader != c.Value {
        http.Error(w, "invalid csrf token", http.StatusForbidden)
    }
}
```

### Валидация

#### protobuf validation
```go
func (s *AuthService) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
    if err := req.Validate(); err != nil {
        return nil, errorsx.FromGRPCCode(codes.InvalidArgument, "validation error", err)
    }
}
```

### Тестирование

#### Комплексная test suite
```
tests/
├── unit/           # Быстрые изолированные тесты
├── integration/    # Тесты с реальной БД  
├── performance/    # Бенчмарки и нагрузочные тесты
└── fixtures/       # Тестовые данные
```

#### Test containers для изоляции
```go
// tests/testcontainers/setup.go
func Setup(ctx context.Context) (*TestContainers, error) {
    postgres, err := postgres.RunContainer(ctx, 
        testcontainers.WithImage("postgres:15"),
        postgres.WithDatabase("testdb"))
    
    redis, err := redis.RunContainer(ctx,
        testcontainers.WithImage("redis:7-alpine"))
}
```

## 🔧 Ключевые модули

### 1. Authentication Service (services/auth/)

**Назначение**: Централизованная аутентификация и авторизация с JWT токенами

**Ключевые интерфейсы**:
```go
type AuthService interface {
    Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error)
    ValidateToken(ctx context.Context, req *ValidateTokenRequest) (*ValidateTokenResponse, error)
    RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*RefreshTokenResponse, error)
}
```

**Пример использования**:
```go
// Валидация токена с ротацией refresh
claims, err := jwt.ParseRefreshToken(refreshToken)
if s.redis != nil {
    // Проверяем, что refresh не отозван
    if _, err := s.redis.Get(ctx, "refresh:"+refreshToken).Result(); err != nil {
        return nil, errorsx.FromGRPCCode(codes.Unauthenticated, "refresh token revoked", nil)
    }
    // Инвалидируем старый токен
    _ = s.redis.Del(ctx, "refresh:"+refreshToken).Err()
}
newAccess, _ := jwt.GenerateAccessToken(userID)
newRefresh, _ := jwt.GenerateRefreshToken(userID)
```

### 2. User Usecase (server/usecase/user/)

**Назначение**: Бизнес-логика работы с пользователями и правами доступа

**Основные методы**:
```go
func (uc *userUsecase) GetUserByID(ctx context.Context, id int) (*ent.User, error) {
    return uc.client.User.Query().
        Where(user.IDEQ(id)).
        WithAvatar().            // Eager loading аватара
        WithUserInfo().          // Профильная информация  
        WithHostRoles().         // Роли хоста
        WithCommunitiesRoles().  // Роли в сообществах
        Only(ctx)
}

func (uc *userUsecase) GetPermissionsByCommunities(ctx context.Context, userID int, communityIDs []int) (map[int]*model.CommunityPermissions, error) {
    // Сложная логика вычисления разрешений на основе ролей
}
```

### 3. GraphQL Resolver Layer (server/graphql/)

**Назначение**: Адаптация бизнес-логики для GraphQL API с поддержкой subscriptions

**Пример резолвера**:
```go
func (r *queryResolver) User(ctx context.Context, id int) (*ent.User, error) {
    // Проверка авторизации через shared/auth
    currentUserID, err := auth.UserIDFromContext(ctx)
    if err != nil {
        return nil, fmt.Errorf("unauthorized")
    }
    
    user, err := r.UserUC.GetUserByID(ctx, id) 
    if err != nil {
        return nil, err // Автоматическая нормализация через ErrorPresenter
    }
    
    return user, nil
}
```

### 4. HTTP/Auth Middleware (server/middleware/)

**Назначение**: Безопасность, аудит, rate limiting для HTTP запросов

**Цепочка middleware**:
```go
func HTTPAuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := httpCookies.WithHTTPContext(r.Context(), w, r)
        
        // Извлечение токена из Authorization header или cookie
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            if c, err := r.Cookie("auth_token"); err == nil && c.Value != "" {
                authHeader = "Bearer " + c.Value
            }
        }
        
        // Удаленная валидация через auth-service
        resp, err := authClient.ValidateToken(ctx, &protobuf.ValidateTokenRequest{
            Token: strings.TrimPrefix(authHeader, "Bearer "),
        })
        
        if err == nil && resp.GetValid() {
            ctx = sharedauth.WithUserID(ctx, int(resp.GetUserId()))
        }
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### 5. Shared Utilities (shared/)

**Назначение**: Переиспользуемые компоненты без бизнес-логики

**JWT утилиты**:
```go
// shared/jwt/jwtutil.go
func GenerateAccessToken(userID int) (string, error) {
    claims := jwt.MapClaims{
        "user_id": strconv.Itoa(userID),
        "exp":     time.Now().Add(15 * time.Minute).Unix(),
        "type":    "access",
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(getJWTSecret())
}
```

**S3 интеграция**:
```go
// shared/s3/client.go
func (c *S3Client) UploadFile(ctx context.Context, dir, filename string, content []byte) (string, string, error) {
    sanitized := sanitizeFilename(filename)
    key := fmt.Sprintf("%s/%s_%s", dir, uuid.New().String(), sanitized)
    
    _, err := c.client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(c.bucket),
        Key:    aws.String(key), 
        Body:   bytes.NewReader(content),
    })
    
    return c.constructURL(key), sanitized, err
}
```

## 📋 Паттерны и Best Practices

### 1. Context Propagation

**Передача контекста через все слои**:
```go
// HTTP Context обертка
ctx := httpCookies.WithHTTPContext(r.Context(), w, r)

// Авторизация в контексте
ctx = sharedauth.WithUserID(ctx, userID)

// gRPC metadata
if authHeader, ok := ctx.Value("authorization").(string); ok {
    md := metadata.New(map[string]string{"authorization": authHeader})
    ctx = metadata.NewOutgoingContext(ctx, md)
}
```

### 2. Error Handling

**Единообразная обработка ошибок**:
```go
// shared/errors - нормализация gRPC → GraphQL
func ToGraphQL(err error) *GraphQLError {
    if s, ok := status.FromError(err); ok {
        return &GraphQLError{Message: s.Message(), Code: s.Code().String()}
    }
    return &GraphQLError{Message: err.Error(), Code: "INTERNAL"}
}

// Ent NotFound специальная обработка
if ent.IsNotFound(err) {
    e := gqlerror.Errorf("not found")
    e.Extensions["code"] = codes.NotFound.String()
    return e
}
```

### 3. Connection Pooling & Performance

**Оптимизация подключений к БД**:
```go
db.SetMaxOpenConns(15)      // Лимит открытых соединений
db.SetMaxIdleConns(5)       // Idle соединения в пуле  
db.SetConnMaxLifetime(5 * time.Minute) // Переиспользование соединений
```

**Eager Loading в Ent**:
```go
return uc.client.User.Query().
    WithAvatar().WithUserInfo().WithHostRoles().  // Одним запросом
    Only(ctx)
```

### 4. Security & Rate Limiting

**IP-based rate limiting**:
```go
var ipLimiters = struct {
    mu       sync.Mutex
    limiters map[string]*rate.Limiter
}{limiters: make(map[string]*rate.Limiter)}

func getRateLimiter(ip string) *rate.Limiter {
    ipLimiters.mu.Lock()
    defer ipLimiters.mu.Unlock()
    
    if limiter, exists := ipLimiters.limiters[ip]; exists {
        return limiter
    }
    
    limiter := rate.NewLimiter(rate.Limit(100), 200) // 100 req/sec, burst 200
    ipLimiters.limiters[ip] = limiter
    return limiter
}
```

### 5. Graceful Shutdown

**Корректное завершение серверов**:
```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()
    
    modules.StartGraphQLServer(client)
    
    <-ctx.Done()
    log.Println("👋 graphql server stopping...")
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    _ = modules.ShutdownGraphQLServer(shutdownCtx)
}
```

## 🏗 Инфраструктура разработки

### Build System (Makefile)

**Комплексная система сборки с 50+ командами**:
```makefile
# Быстрая разработка
make dev-check      # format + vet + unit tests
make quick-test     # только unit тесты
make pre-commit     # полная проверка перед коммитом

# Тестирование
make test-unit      # юнит тесты
make test-integration # интеграционные тесты с Docker
make test-coverage  # отчет по покрытию
make test-performance # бенчмарки

# CI/CD
make ci             # полный CI pipeline
make docker-build   # Docker образ
```

### Environment Configuration

**Конфигурация через переменные окружения**:
```bash
# База данных
DB_HOST=localhost
DB_MAX_OPEN_CONNS=15
DB_MAX_IDLE_CONNS=5

# Микросервисы
AUTH_GRPC_ADDR=localhost:4001
USER_GRPC_ADDR=localhost:4002
GRPC_INSECURE=true

# Безопасность
JWT_SECRET=secret
CSRF_ENABLE=true
GRAPHQL_MAX_COMPLEXITY=300
```

### Docker & Orchestration

**Testcontainers для изоляции тестов**:
```go
containers, err := testcontainers.Setup(ctx)
defer containers.Cleanup()

postgres := testcontainers.GenericContainer{
    Image: "postgres:15",
    ExposedPorts: []string{"5432/tcp"},
    Env: map[string]string{
        "POSTGRES_DB": "testdb",
        "POSTGRES_USER": "test", 
        "POSTGRES_PASSWORD": "test",
    },
}
```

### Monitoring & Health Checks

**Комплексные health checks**:
```go
mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 800*time.Millisecond)
    defer cancel()
    
    // Database probe
    if _, err := client.User.Query().Limit(1).All(ctx); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        return
    }
    
    // S3 probe
    if err := s3client.HealthCheck(); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        return  
    }
    
    // gRPC upstream health checks
    for _, conn := range []*grpc.ClientConn{authConn, userConn, mailConn} {
        if resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{}); 
           err != nil || resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
            w.WriteHeader(http.StatusServiceUnavailable)
            return
        }
    }
    
    w.WriteHeader(http.StatusOK)
})
```

## 📋 Выводы и рекомендации

### ✅ Сильные стороны проекта

1. **Современная архитектура**: Правильное разделение на микросервисы с четкими границами
2. **Type Safety**: Использование Ent ORM и gqlgen обеспечивает compile-time безопасность
3. **Comprehensive Testing**: 4-уровневая система тестирования (unit/integration/performance/e2e)
4. **Security First**: Многослойная защита с JWT, CSRF, rate limiting, audit trail
5. **Developer Experience**: Отличный DX с подробным Makefile, автогенерацией кода
6. **Production Ready**: Graceful shutdown, health checks, connection pooling, monitoring

### 🔄 Области для улучшения

1. **Observability**: 
   - Добавить distributed tracing (OpenTelemetry)
   - Метрики Prometheus для мониторинга производительности
   - Структурированное логирование (zap/logrus)

2. **Caching Strategy**:
   - Implementовать более агрессивное кеширование на уровне GraphQL
   - Redis cache для часто запрашиваемых данных (пользователи, сообщества)

3. **Database Optimization**:
   - Добавить database migrations с версионированием
   - Индексы для performance-critical запросов
   - Read replicas для масштабирования чтения

4. **API Evolution**:
   - GraphQL schema versioning и deprecation strategy
   - API rate limiting per user/operation
   - Request/response compression

### 🚀 Рекомендации по развитию

1. **Микросервисы**: Добавить service mesh (Istio) для продакшена
2. **CI/CD**: Автоматизировать деплой через GitOps (ArgoCD)  
3. **Monitoring**: Интегрировать APM (Datadog/New Relic)
4. **Documentation**: Добавить OpenAPI specs для REST endpoints
5. **Performance**: Implements GraphQL query complexity analysis и caching

---

**Stormlink** представляет собой хорошо архитектурированное, современное backend-приложение на Go с правильным разделением ответственности, комплексным тестированием и готовностью к продакшен деплою. Проект демонстрирует best practices разработки на Go и может служить референсом для similar проектов.