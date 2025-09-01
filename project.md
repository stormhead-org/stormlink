# 📝 Анализ backend кодовой базы: Stormlink⚡

## 📁 Структура проекта

```
stormlink/
├── server/                    # Основной GraphQL сервер
│   ├── cmd/                   # Entry points и модули инициализации
│   │   ├── main.go           # Точка входа приложения
│   │   └── modules/          # Модули инициализации (DB, GraphQL, S3)
│   ├── ent/                   # Ent ORM модели и схемы (автогенерированные)
│   │   ├── schema/           # Определения схем сущностей
│   │   └── [generated]/      # Автосгенерированные CRUD операции
│   ├── graphql/               # GraphQL резолверы и схемы
│   │   ├── ent.graphql       # Автогенерированная GraphQL схема из Ent
│   │   ├── handlers.graphql  # Пользовательские GraphQL операции
│   │   └── *.resolvers.go    # Реализация резолверов
│   ├── grpc/                  # gRPC клиенты и protobuf
│   ├── middleware/            # HTTP/gRPC middleware
│   ├── model/                 # Domain модели
│   ├── proto/                 # Protocol Buffers определения
│   ├── server/                # Вспомогательная серверная логика
│   └── usecase/               # Бизнес-логика (Clean Architecture)
│       ├── user/             # Управление пользователями
│       ├── community/        # Логика сообществ
│       ├── post/             # Управление постами
│       └── [other domains]/  # Другие бизнес-домены
├── services/                  # Микросервисы
│   ├── auth/                  # Сервис аутентификации
│   │   ├── cmd/              # Entry point микросервиса
│   │   └── internal/         # Внутренняя логика сервиса
│   ├── mail/                  # Email уведомления
│   ├── media/                 # Управление медиафайлами
│   ├── user/                  # Пользовательские данные
│   └── workers/               # Фоновые задачи
├── shared/                    # Общий код между сервисами
│   ├── auth/                  # Утилиты аутентификации
│   ├── errors/                # Унифицированная обработка ошибок
│   ├── jwt/                   # JWT токены
│   ├── redis/, s3/, rabbitmq/ # Интеграции с внешними сервисами
│   └── http/                  # HTTP утилиты
└── proto/                     # Общие protobuf файлы
    └── googleapis/            # Google APIs protobuf определения
```

**Принципы организации:**
- **Clean Architecture** с четким разделением слоев: usecase → repository → database
- **Domain-driven design** с выделением бизнес-сущностей в отдельные модули
- **Микросервисная архитектура** с основным GraphQL gateway и специализированными gRPC сервисами
- **Shared kernel** паттерн для общего кода между сервисами
- **Protocol-first** подход с использованием protobuf для межсервисного взаимодействия

## 🛠 Технологический стек

| Категория | Технология | Версия | Назначение |
|-----------|------------|--------|------------|
| **Язык** | Go | 1.24.2 | Основной язык разработки |
| **API** | GraphQL (gqlgen) | 0.17.78 | Основной API для клиентов |
| **RPC** | gRPC | 1.70.0 | Межсервисное взаимодействие |
| **ORM** | Ent | 0.14.4 | Работа с базой данных |
| **База данных** | PostgreSQL | - | Основное хранилище |
| **Кэширование** | Redis | 9.12.0 | Сессии, кэширование |
| **Очереди** | RabbitMQ | 1.10.0 | Асинхронные задачи |
| **Аутентификация** | JWT | 5.2.2 | Токены доступа |
| **Хранилище** | AWS S3 | 1.55.7 | Медиафайлы |
| **Валидация** | validator | 10.26.0 | Валидация входных данных |
| **WebSockets** | gorilla/websocket | 1.5.0 | Real-time подписки |
| **CORS** | rs/cors | 1.11.1 | Cross-origin запросы |
| **Cryptography** | golang.org/x/crypto | 0.40.0 | Хеширование паролей |
| **Rate Limiting** | golang.org/x/time | 0.11.0 | Ограничение запросов |

### Дополнительные инструменты
- **protoc-gen-validate** для валидации protobuf сообщений
- **grpc-gateway** для REST API совместимости
- **buf.work.yaml** для управления protobuf зависимостями
- **multierror** для агрегации ошибок
- **godotenv** для управления переменными окружения

## 🏗 Архитектура

### Архитектурные слои

**1. Presentation Layer (GraphQL + gRPC)**
```go
// server/cmd/modules/graphql_server.go
resolver := &graphql.Resolver{
    Client:          client,
    UserUC:          uUC,
    CommunityUC:     cUC,
    PostUC:          pUC,
    CommentUC:       commentUC,
    HostRoleUC:      hostRoleUC,
    CommunityRoleUC: communityRoleUC,
    AuthClient:      authClient,
    UserClient:      userClient,
}
```

**2. Application Layer (UseCase)**
```go
// server/usecase/user/user.go
type UserUsecase interface {
	GetUserByID(ctx context.Context, id int) (*ent.User, error)
	GetPermissionsByCommunities(ctx context.Context, userID int, communityIDs []int) (map[int]*model.CommunityPermissions, error)
	GetUserStatus(ctx context.Context, currentUserID int, userID int) (*models.UserStatus, error)
}

type userUsecase struct {
	client *ent.Client
}
```

**3. Domain Layer (Ent Models)**
```go
// server/ent/schema/user.go
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),
		field.String("name").NotEmpty(),
		field.String("slug").Unique().NotEmpty(),
		field.String("email").Unique().NotEmpty(),
		field.String("password_hash").NotEmpty().Annotations(
			entgql.Skip(entgql.SkipAll),
		),
		field.Bool("is_verified").Default(false),
		field.Time("created_at").Default(time.Now),
	}
}
```

**4. Infrastructure Layer (Database, External Services)**
```go
// server/cmd/modules/database.go
func ConnectDB() *ent.Client {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"), os.Getenv("SSL_MODE"),
	)
	
	maxOpenConns := getEnvInt("DB_MAX_OPEN_CONNS", 15)
	maxIdleConns := getEnvInt("DB_MAX_IDLE_CONNS", 5)
	connMaxLifetime := getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 5)
	
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Minute)
}
```

### Межсервисное взаимодействие

**gRPC Services с Middleware Chain:**
```go
// services/auth/cmd/main.go
s := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        middleware.GRPCAuthRateLimitMiddleware,
        middleware.GRPCAuthMiddleware,
    ),
)
authpb.RegisterAuthServiceServer(s, svc)

// gRPC health-check
hs := health.NewServer()
healthpb.RegisterHealthServer(s, hs)
hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
```

**Protocol Buffers для типобезопасности:**
```protobuf
// server/proto/auth.proto
message User {
  string id = 1;
  string name = 2;
  string slug = 3;
  Avatar avatar = 4;
  repeated UserInfo user_info = 5;
  repeated HostRole host_roles = 6;
  repeated CommunityRole community_roles = 7;
}
```

### Middleware и безопасность

**Security Middleware Chain:**
```go
// server/cmd/modules/graphql_server.go
graphqlHandler := middleware.SecurityAuditMiddleware(
    middleware.AuditMiddleware(
        middleware.RateLimitMiddleware(
            http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                // CSRF Origin check для POST
                if r.Method == http.MethodPost {
                    if !allowOrigin(r.Header.Get("Origin")) {
                        http.Error(w, "invalid origin", http.StatusForbidden)
                        return
                    }
                }
                // Лимит размера тела запроса
                maxBody := int64(1 * 1024 * 1024)
                if r.Method == http.MethodPost {
                    r.Body = http.MaxBytesReader(w, r.Body, maxBody)
                }
            }),
        ),
    ),
)
```

**Authorization Interceptor:**
```go
// server/cmd/modules/graphql_server.go
func authClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// Извлекаем Authorization из контекста
	if authHeader, ok := ctx.Value("authorization").(string); ok && authHeader != "" {
		md := metadata.New(map[string]string{
			"authorization": authHeader,
		})
		ctx = metadata.NewOutgoingContext(ctx, md)
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}
```

## 💾 Работа с данными

### Database Schema и миграции

**Ent Schema с GraphQL интеграцией:**
```go
// server/ent/schema/user.go
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("avatar", Media.Type).
			Field("avatar_id").
			Unique(),
		edge.To("banner", Media.Type).
			Field("banner_id").
			Unique(),
		edge.From("user_info", ProfileTableInfoItem.Type).
			Ref("user"),
		// Роли хоста (HostRole)
		edge.To("host_roles", HostRole.Type),
		// Роли в сообществах (Role)
		edge.To("communities_roles", Role.Type),
		// Баны и муты в сообществах
		edge.To("communities_bans", CommunityUserBan.Type),
		edge.To("communities_mutes", CommunityUserMute.Type),
	}
}
```

**Автоматические миграции:**
```go
// server/cmd/modules/database.go
func MigrateDB(client *ent.Client, reset bool, seed bool) {
	if reset {
		log.Println("⚠️  Полный сброс базы данных с удалением колонок и индексов...")
		if err := client.Schema.Create(
			context.Background(),
			schema.WithDropIndex(true),
			schema.WithDropColumn(true),
		); err != nil {
			log.Fatalf("ошибка сброса схемы: %v", err)
		}
	}
	if seed {
		if err := Seed(client); err != nil {
			log.Fatalf("❌ Ошибка сидинга: %v", err)
		}
	}
}
```

### Сложная модель данных

**Основные сущности:**
- **User** - пользователи с аватарами, профильной информацией
- **Community** - сообщества с модерацией и правилами
- **Post/Comment** - контент с системой лайков и закладок
- **Role** - система ролей на уровне хоста и сообществ
- **Media** - медиафайлы с интеграцией S3
- **Ban/Mute** - система модерации с временными ограничениями

**Связи между сущностями:**
- Многоуровневая система разрешений (Host → Community → User)
- Подписки пользователей на сообщества и друг на друга
- Система модерации с различными уровнями (хост, сообщество)
- Email верификация и восстановление паролей

### Кэширование и производительность

**Redis интеграция для сессий:**
```go
// services/auth/internal/service/service.go
func (s *AuthService) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	// ... аутентификация ...
	
	// Опционально: записать сессию/refresh в Redis с TTL
	if s.redis != nil {
		ttl := 7 * 24 * time.Hour
		_ = s.redis.Set(ctx, "refresh:"+refreshToken, u.ID, ttl).Err()
	}
	
	return &authpb.LoginResponse{
		AccessToken: accessToken, 
		RefreshToken: refreshToken, 
		User: sharedmapper.UserToProto(u)
	}, nil
}
```

**Connection Pooling оптимизация:**
- Настраиваемые параметры пула соединений PostgreSQL
- Максимальное время жизни соединений для предотвращения "too many clients"
- Мониторинг состояния пула через health checks

## ✅ Качество кода

### Унифицированная обработка ошибок

**gRPC to GraphQL Error Mapping:**
```go
// shared/errors/errors.go
func ToGraphQL(err error) *GraphQLError {
    if err == nil { return nil }
    st, ok := status.FromError(err)
    if ok {
        return &GraphQLError{Message: st.Message(), Code: st.Code().String()}
    }
    if isEntNotFound(err) {
        return &GraphQLError{Message: "not found", Code: codes.NotFound.String()}
    }
    return &GraphQLError{Message: fmt.Sprintf("internal error: %v", err), Code: codes.Internal.String()}
}
```

**Error Normalization в GraphQL:**
```go
// server/cmd/modules/graphql_server.go
srv.SetErrorPresenter(func(ctx context.Context, err error) *gqlerror.Error {
    // Специальный маппинг ent.NotFound → GraphQL code=NotFound
    if ent.IsNotFound(err) {
        e := gqlerror.Errorf("not found")
        if e.Extensions == nil {
            e.Extensions = map[string]any{}
        }
        e.Extensions["code"] = codes.NotFound.String()
        return e
    }
    // Если это gRPC status — нормализуем через shared/errors
    ge := errorsx.ToGraphQL(err)
    if ge == nil {
        return gqlerror.Errorf("unknown error")
    }
    e := gqlerror.Errorf("%s", ge.Message)
    e.Extensions["code"] = ge.Code
    return e
})
```

### JWT и аутентификация

**Type-safe JWT Operations:**
```go
// shared/jwt/jwtutil.go
func ParseAccessToken(tokenString string) (*AccessTokenClaims, error) {
    claims, err := ParseToken(tokenString)
    if err != nil { return nil, err }
    t, ok := claims["type"].(string)
    if !ok || t != "access" { 
        return nil, errors.New("invalid token type") 
    }
    uid, ok := claims["user_id"].(string)
    if !ok { 
        return nil, errors.New("user_id not found or invalid") 
    }
    id, err := strconv.Atoi(uid)
    if err != nil { 
        return nil, errors.New("invalid user_id format") 
    }
    return &AccessTokenClaims{UserID: id}, nil
}
```

**Password Security:**
```go
// shared/jwt/hashutil.go (предполагаемая реализация)
func ComparePassword(hash, password, salt string) error {
    // Безопасное сравнение паролей с использованием salt
}
```

### Context Propagation

**HTTP Authentication Middleware:**
```go
// server/middleware/http_auth.go
func HTTPAuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Базовый контекст HTTP (для работы с куками в резолверах)
        ctx := httpCookies.WithHTTPContext(r.Context(), w, r)
        ctx = sharedauth.WithUserID(ctx, 0)

        // Источник access токена: Authorization или cookie auth_token
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            if c, err := r.Cookie("auth_token"); err == nil && c.Value != "" {
                authHeader = "Bearer " + c.Value
            }
        }
        ctx = context.WithValue(ctx, "authorization", authHeader)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Валидация данных

**protobuf Validation:**
```protobuf
// server/proto/auth.proto
import "validate/validate.proto";

message LoginRequest {
  string email = 1 [(validate.rules).string.email = true];
  string password = 2 [(validate.rules).string.min_len = 8];
}
```

**Go Validation с validator:**
```go
// Используется github.com/go-playground/validator/v10 для структурной валидации
```

## 🔧 Ключевые модули

### 1. Authentication Service (services/auth)

**Назначение:** Центральный микросервис аутентификации и авторизации  
**Роль:** JWT токены, сессии, валидация пользователей

**Основные интерфейсы:**
```go
// server/grpc/auth/protobuf (автогенерированный)
type AuthServiceClient interface {
    Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error)
    Logout(ctx context.Context, req *emptypb.Empty) (*LogoutResponse, error)
    RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*RefreshTokenResponse, error)
    ValidateToken(ctx context.Context, req *ValidateTokenRequest) (*ValidateTokenResponse, error)
}
```

**Пример реализации:**
```go
// services/auth/internal/service/service.go
func (s *AuthService) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
    if err := req.Validate(); err != nil {
        return nil, errorsx.FromGRPCCode(codes.InvalidArgument, "validation error", err)
    }
    
    u, err := s.client.User.
        Query().
        Where(entuser.EmailEQ(req.GetEmail())).
        WithAvatar().
        WithHostRoles().
        WithCommunitiesRoles().
        Only(ctx)
    
    if err := jwt.ComparePassword(u.PasswordHash, req.GetPassword(), u.Salt); err != nil {
        return nil, errorsx.FromGRPCCode(codes.Unauthenticated, "invalid credentials", nil)
    }
    
    accessToken, _ := jwt.GenerateAccessToken(u.ID)
    refreshToken, _ := jwt.GenerateRefreshToken(u.ID)
    
    return &authpb.LoginResponse{
        AccessToken: accessToken, 
        RefreshToken: refreshToken, 
        User: sharedmapper.UserToProto(u)
    }, nil
}
```

**Взаимодействие с другими слоями:**
- Получает запросы от GraphQL резолверов через gRPC
- Обращается к базе данных через Ent ORM
- Использует Redis для хранения refresh токенов
- Устанавливает HTTP cookies через shared/http утилиты

### 2. GraphQL Gateway (server/graphql)

**Назначение:** Основной API gateway для клиентских приложений  
**Роль:** Агрегация данных, real-time subscriptions, query optimization

**Архитектурное решение:**
```go
// server/graphql/resolver.go
type Resolver struct {
	Client *ent.Client
	UserUC user.UserUsecase
	CommunityUC community.CommunityUsecase
	PostUC post.PostUsecase
	CommentUC comment.CommentUsecase
	HostRoleUC hostrole.HostRoleUsecase
	CommunityRoleUC communityrole.CommunityRoleUsecase
	AuthClient authpb.AuthServiceClient
	UserClient userpb.UserServiceClient
	MailClient mailpb.MailServiceClient
	MediaClient mediapb.MediaServiceClient
}
```

**GraphQL Schema Generation:**
- Автоматическая генерация схемы из Ent моделей (`ent.graphql`)
- Пользовательские операции в `handlers.graphql`
- Automatic Persisted Queries (APQ) для производительности
- WebSocket транспорт для real-time подписок

**Взаимодействие:**
- Делегирует бизнес-логику UseCase слою
- Вызывает микросервисы через gRPC клиенты
- Использует middleware chain для безопасности

### 3. UseCase Layer

**Назначение:** Изолированная бизнес-логика без внешних зависимостей  
**Роль:** Orchestration, валидация, композиция данных

**Пример User UseCase:**
```go
// server/usecase/user/user.go
func (uc *userUsecase) GetUserByID(ctx context.Context, id int) (*ent.User, error) {
	return uc.client.User.
		Query().
		Where(user.IDEQ(id)).
		WithAvatar().
		WithUserInfo().
		WithHostRoles().
		WithCommunitiesRoles().
		Only(ctx)
}

func (uc *userUsecase) GetPermissionsByCommunities(ctx context.Context, userID int, communityIDs []int) (map[int]*model.CommunityPermissions, error) {
	// Сложная бизнес-логика для вычисления разрешений пользователя в сообществах
	// Учитывает роли хоста, роли в сообществах, баны и муты
}
```

**Основные UseCase модули:**
- `user/` - управление пользователями, профили, подписки
- `community/` - сообщества, модерация, правила
- `post/` - создание, редактирование, публикация контента
- `comment/` - комментарии, вложенные треды
- `hostrole/`, `communityrole/` - система ролей и разрешений
- `ban/`, `hostmute/` - модерация и ограничения

### 4. Ent ORM Layer

**Назначение:** Type-safe доступ к данным с автоматической генерацией  
**Роль:** Схема БД, миграции, GraphQL интеграция

**Schema Definition Example:**
```go
// server/ent/schema/community.go (предполагаемая структура)
func (Community) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.String("slug").Unique(),
		field.String("description").Optional(),
		field.Bool("is_private").Default(false),
		field.Time("created_at").Default(time.Now),
	}
}

func (Community) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("owner", User.Type).
			Ref("communities_owner").
			Unique().
			Required(),
		edge.To("posts", Post.Type),
		edge.To("rules", CommunityRule.Type),
		edge.To("moderators", CommunityModerator.Type),
		edge.To("followers", CommunityFollow.Type),
	}
}
```

**Автоматическая генерация:**
- CRUD операции для всех сущностей
- GraphQL схема с Relay-compatible пагинацией
- Type-safe query builder
- Автоматические foreign key relationships

### 5. Shared Libraries

**Назначение:** Переиспользуемый код между всеми сервисами  
**Роль:** Унификация, DRY принцип, консистентность

**Модули shared пакета:**

```go
// shared/auth - контекст и извлечение пользователя
func UserIDFromContext(ctx context.Context) (int, error)
func WithUserID(ctx context.Context, userID int) context.Context

// shared/errors - унифицированная обработка ошибок
func ToGRPC(err error, msg string) error
func ToGraphQL(err error) *GraphQLError

// shared/jwt - токены и криптография
func GenerateAccessToken(userID int) (string, error)
func ParseAccessToken(tokenString string) (*AccessTokenClaims, error)

// shared/http - HTTP утилиты и cookies
func WithHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context
func SetAuthCookies(w http.ResponseWriter, accessToken, refreshToken string)

// shared/redis - Redis клиент и утилиты
func NewClient() (*redis.Client, error)

// shared/s3 - AWS S3 интеграция
func NewS3Client() S3Client
```

**Взаимодействие:**
- Используется всеми микросервисами для консистентности
- Предоставляет общие интерфейсы для внешних зависимостей
- Обеспечивает type safety между сервисами

## 📋 Паттерны и best practices

### Context-Driven Architecture

**Request Context Propagation:**
```go
// Контекст передается через все слои приложения
ctx := context.Background()
ctx = httpCookies.WithHTTPContext(ctx, w, r)
ctx = sharedauth.WithUserID(ctx, userID)
ctx = context.WithValue(ctx, "authorization", authHeader)

// UseCase получает enriched контекст
user, err := uc.GetUserByID(ctx, userID)
```

**Graceful Shutdown:**
```go
// server/cmd/main.go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()
<-ctx.Done()

shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
_ = modules.ShutdownGraphQLServer(shutdownCtx)
```

### Error Handling Patterns

**Layered Error Mapping:**
```go
// 1. Domain layer - ent.NotFound
user, err := client.User.Get(ctx, id)

// 2. UseCase layer - business logic errors
if user.IsBanned {
    return nil, fmt.Errorf("user is banned")
}

// 3. Service layer - gRPC status errors
return nil, status.Error(codes.PermissionDenied, "user banned")

// 4. Presentation layer - GraphQL errors
return errorsx.ToGraphQL(err)
```

**Error Wrapping и Unwrapping:**
```go
// shared/errors/errors.go
func isEntNotFound(err error) bool {
    var nf *entruntime.NotFoundError
    if errors.As(err, &nf) { return true }
    if status.Code(err) == codes.NotFound { return true }
    return false
}
```

### Security Patterns

**Defense in Depth:**
```go
// 1. Network layer - CORS, TLS
corsHandler := cors.New(cors.Options{
    AllowedOrigins:   []string{frontend},
    AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
    AllowCredentials: true,
})

// 2. Application layer - CSRF, Origin validation
if !allowOrigin(r.Header.Get("Origin")) {
    http.Error(w, "invalid origin", http.StatusForbidden)
    return
}

// 3. Rate limiting - per-IP limiters
func rateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ip := getClientIP(r)
        limiter := getIPLimiter(ip)
        if !limiter.Allow() {
            http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}

// 4. Business layer - permissions, role-based access
func (uc *userUsecase) CanModerateCommnity(ctx context.Context, userID, communityID int) bool {
    // Проверка ролей и разрешений
}
```

**Token Security:**
```go
// Access/Refresh token rotation
accessToken := jwt.GenerateAccessToken(userID) // 15 minutes
refreshToken := jwt.GenerateRefreshToken(userID) // 7 days

// Secure cookie flags
cookie := &http.Cookie{
    Name:     "auth_token",
    Value:    accessToken,
    HttpOnly: true,
    Secure:   true,
    SameSite: http.SameSiteStrictMode,
}
```

### Performance Optimization Patterns

**Connection Pooling:**
```go
// Настройка пула соединений PostgreSQL
db.SetMaxOpenConns(15)  // Максимум открытых соединений
db.SetMaxIdleConns(5)   // Максимум idle соединений  
db.SetConnMaxLifetime(5 * time.Minute) // Время жизни соединения
```

**GraphQL Query Optimization:**
```go
// APQ (Automatic Persisted Queries) для кэширования
srv.Use(extension.AutomaticPersistedQuery{Cache: lru.New[string](1000)})

// Complexity limiting для защиты от DoS
srv.Use(extension.FixedComplexityLimit(300))

// Лимит размера запроса
r.Body = http.MaxBytesReader(w, r.Body, 1*1024*1024) // 1MB
```

**Batch Loading и N+1 Problem:**
```go
// Ent автоматически оптимизирует запросы с WithXXX()
user := client.User.
    Query().
    WithAvatar().
    WithHostRoles().
    WithCommunitiesRoles().
    Only(ctx) // Один JOIN запрос вместо N+1
```

### Concurrency Patterns

**Worker Pool для фоновых задач:**
```go
// services/workers - асинхронная обработка
type WorkerPool struct {
    workers   int
    taskQueue chan Task
    quit      chan bool
}

func (wp *WorkerPool) Start() {
    for i := 0; i < wp.workers; i++ {
        go wp.worker()
    }
}
```

**Channel-based Communication:**
```go
// WebSocket subscriptions через channels
type Subscription struct {
    ID       string
    UserID   int
    EventCh  chan Event
    QuitCh   chan bool
}
```

## 🚀 Инфраструктура разработки

### Конфигурация и Environment

**Environment-based Configuration:**
```go
// server/cmd/modules/env.go (предполагаемая структура)
func InitEnv() {
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found, using system environment")
    }
}

// Типичные переменные окружения:
// DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME
// REDIS_URL, RABBITMQ_URL, AWS_S3_BUCKET
// JWT_SECRET, FRONTEND_ORIGIN
// GRAPHQL_MAX_COMPLEXITY, RATE_LIMIT_PER_MINUTE
```

**Feature Flags и конфигурация:**
```go
// Условная функциональность через ENV
if os.Getenv("CSRF_ENABLE") == "true" {
    // CSRF validation logic
}

if os.Getenv("ENV") != "production" {
    srv.Use(extension.Introspection{})
    mux.Handle("/", playground.Handler("GraphQL playground", "/query"))
}
```

### Protocol Buffers Management

**buf.work.yaml конфигурация:**
```yaml
version: v1
directories:
  - proto/googleapis      # Google APIs definitions
  - server/proto         # Service-specific protobuf
```

**Code Generation Pipeline:**
```bash
# Автоматическая генерация из .proto файлов:
# 1. Go structs и gRPC клиенты/серверы
# 2. Валидация через protoc-gen-validate  
# 3. gRPC Gateway для REST совместимости
# 4. OpenAPI/Swagger документация
```

### GraphQL Development Workflow

**gqlgen.yml конфигурация:**
```yaml
schema:
  - server/graphql/ent.graphql      # Автогенерированная из Ent
  - server/graphql/handlers.graphql # Пользовательские операции

autobind:
  # Автоматический binding к существующим Go типам

models:
  Community:
    model: stormlink/server/ent.Community
  User:
    model: stormlink/server/ent.User
  # Прямое использование Ent моделей в GraphQL
```

### Health Checks и Monitoring

**Comprehensive Readiness Probes:**
```go
// server/cmd/modules/graphql_server.go
mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 800*time.Millisecond)
    defer cancel()
    
    // Database health check
    if _, err := client.User.Query().Limit(1).All(ctx); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("db not ready"))
        return
    }
    
    // S3 health check
    if err := s3client.HealthCheck(); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("s3 not ready"))
        return
    }
    
    // Upstream gRPC services health checks
    for _, service := range []string{"auth", "user", "mail", "media"} {
        if !checkGRPCHealth(ctx, service) {
            w.WriteHeader(http.StatusServiceUnavailable)
            w.Write([]byte(service + " grpc not ready"))
            return
        }
    }
    
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("ready"))
})
```

**Service Discovery через Environment:**
```go
// Адреса микросервисов из переменных окружения
authConn, err := grpc.DialContext(ctx, 
    os.Getenv("AUTH_GRPC_ADDR") ?: "localhost:4001",
    creds)
userConn, err := grpc.DialContext(ctx,
    os.Getenv("USER_GRPC_ADDR") ?: "localhost:4002", 
    creds)
```

### Security Infrastructure

**TLS и Certificates:**
```go
// Поддержка как insecure, так и TLS соединений
useInsecure := os.Getenv("GRPC_INSECURE") == "true"
var creds grpc.DialOption
if useInsecure {
    creds = grpc.WithTransportCredentials(insecure.NewCredentials())
} else {
    tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
    creds = grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg))
}
```

**Request Size Limiting:**
```go
// Защита от больших payload
maxBody := int64(1 * 1024 * 1024) // 1MB default
if v := os.Getenv("GRAPHQL_MAX_BODY_BYTES"); v != "" {
    if n, err := strconv.Atoi(v); err == nil && n > 0 { 
        maxBody = int64(n) 
    }
}
r.Body = http.MaxBytesReader(w, r.Body, maxBody)
```

## 📊 Выводы и рекомендации

### Сильные стороны

✅ **Архитектурная зрелость:** Превосходная реализация Clean Architecture с четким разделением ответственности между слоями  
✅ **Type Safety:** Строгая типизация на всех уровнях - от протобуф до GraphQL схемы  
✅ **Безопасность:** Многоуровневая защита включая CSRF, rate limiting, JWT rotation, Origin validation  
✅ **Производительность:** Connection pooling, APQ, complexity limiting, оптимизированные запросы Ent ORM  
✅ **Observability:** Comprehensive health checks для всех зависимостей, audit middleware  
✅ **Microservices Design:** Хорошо спроектированная микросервисная архитектура с правильными границами  
✅ **Error Handling:** Унифицированная система обработки ошибок через все слои  
✅ **Protocol-first:** Использование protobuf для contract-first разработки  

### Зоны для улучшения

⚠️ **Тестирование:** 
- Отсутствуют unit тесты для UseCase слоя
- Нет integration тестов для gRPC сервисов
- Отсутствуют end-to-end тесты для GraphQL API

⚠️ **Документация:** 
- Нет OpenAPI документации для REST endpoints
- Отсутствует архитектурная документация (ADR)
- Нет примеров использования API

⚠️ **Observability:**
- Отсутствует интеграция с Prometheus/Grafana
- Нет distributed tracing (OpenTelemetry)
- Минимальное структурированное логирование

⚠️ **DevOps:**
- Отсутствуют Dockerfile и containerization
- Нет CI/CD pipeline конфигураций
- Отсутствуют Kubernetes deployments

⚠️ **Data Validation:**
- Ограниченная валидация на GraphQL input level
- Нет бизнес-правил валидации в UseCase слое

### Архитектурные риски

🔴 **Single Point of Failure:** GraphQL gateway является центральной точкой отказа  
🟡 **Database Coupling:** Все микросервисы используют одну PostgreSQL базу  
🟡 **Session State:** Redis dependency для refresh токенов без fallback  
🟡 **Configuration Management:** Зависимость от переменных окружения без validation  

### Уровень сложности

**Senior/Expert-level проект** требующий глубокого понимания:
- Go advanced patterns (interfaces, embedding, generics)
- GraphQL ecosystem (resolvers, subscriptions, federation)
- gRPC и Protocol Buffers
- Microservices architecture patterns
- Database design и ORM advanced features
- Security в distributed systems
- Performance optimization в Go

### Приоритетные рекомендации

#### Краткосрочные улучшения (1-2 спринта)

1. **Добавить базовое тестирование:**
```bash
# Структура тестов
server/usecase/user/user_test.go
server/middleware/auth_test.go  
services/auth/internal/service/service_test.go
```

2. **Внедрить структурированное логирование:**
```go
import "go.uber.org/zap"

logger, _ := zap.NewProduction()
logger.Info("user login", 
    zap.Int("user_id", userID),
    zap.String("ip", clientIP),
    zap.Duration("duration", time.Since(start)),
)
```

3. **Создать Dockerfile для сервисов:**
```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main server/cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/main .
CMD ["./main"]
```

#### Среднесрочные улучшения (1-2 месяца)

4. **Добавить OpenTelemetry tracing:**
```go
import "go.opentelemetry.io/otel"

tracer := otel.Tracer("stormlink")
ctx, span := tracer.Start(ctx, "user.GetUserByID")
defer span.End()
```

5. **Внедрить метрики Prometheus:**
```go
import "github.com/prometheus/client_golang/prometheus"

var (
    requestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "graphql_requests_total"},
        []string{"operation", "status"},
    )
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{Name: "graphql_request_duration_seconds"},
        []string{"operation"},
    )
)
```

6. **Создать CI/CD pipeline (GitHub Actions):**
```yaml
name: CI/CD
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with: 
          go-version: '1.24'
      - run: go test ./...
      - run: go build ./...
```

#### Долгосрочные улучшения (3-6 месяцев)

7. **Database per Service pattern:**
   - Выделить отдельные БД для каждого микросервиса
   - Реализовать event sourcing для синхронизации данных

8. **GraphQL Federation:**
   - Разделить GraphQL схему между микросервисами
   - Внедрить Apollo Federation или GraphQL Mesh

9. **Advanced Security:**
   - OAuth2/OIDC интеграция
   - API rate limiting с Redis
   - WAF и DDoS protection

### Заключение

**Stormlink** представляет собой высококачественный enterprise-уровень проект с современной архитектурой и strong engineering practices. Код демонстрирует глубокое понимание Go ecosystem, microservices patterns и security best practices.

Основные достоинства проекта:
- Отличная архитектурная организация
- Типобезопасность на всех уровнях  
- Комплексный подход к безопасности
- Производительные решения

Проект готов к production deployment с минимальными доработками в области тестирования и observability. Рекомендуется как reference implementation для современных Go backend приложений.

**Итоговая оценка:** ⭐⭐⭐⭐⭐ (Excellent) - Professional-grade codebase с потенциалом для scale-up в enterprise окружении.