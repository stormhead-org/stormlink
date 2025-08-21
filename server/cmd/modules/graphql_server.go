// server/cmd/modules/graphql_server.go
package modules

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"golang.org/x/time/rate"

	"stormlink/server/ent"
	"stormlink/server/graphql"
	authpb "stormlink/server/grpc/auth/protobuf"
	mailpb "stormlink/server/grpc/mail/protobuf"
	mediapb "stormlink/server/grpc/media/protobuf"
	userpb "stormlink/server/grpc/user/protobuf"
	"stormlink/server/middleware"
	banuc "stormlink/server/usecase/ban"
	commentuc "stormlink/server/usecase/comment"
	communityuc "stormlink/server/usecase/community"
	communityroleuc "stormlink/server/usecase/communityrole"
	hostroleuc "stormlink/server/usecase/hostrole"
	postuc "stormlink/server/usecase/post"
	useruc "stormlink/server/usecase/user"
	errorsx "stormlink/shared/errors"
	httpWithCookies "stormlink/shared/http"

	"stormlink/server/usecase/profiletableinfoitem"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

var httpSrv *http.Server
var upstreamClosers []io.Closer
var ipLimiters = struct {
    mu       sync.Mutex
    limiters map[string]*rate.Limiter
}{limiters: make(map[string]*rate.Limiter)}

func getClientIP(r *http.Request) string {
    // X-Forwarded-For first
    xf := r.Header.Get("X-Forwarded-For")
    if xf != "" {
        parts := strings.Split(xf, ",")
        return strings.TrimSpace(parts[0])
    }
    host, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        return r.RemoteAddr
    }
    return host
}

func allowOrigin(origin string) bool {
    allowed := os.Getenv("FRONTEND_ORIGIN")
    if allowed == "" {
        allowed = "http://localhost:3000"
    }
    return origin == "" || origin == allowed
}

// authClientInterceptor добавляет Authorization заголовок из контекста в gRPC метаданные
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

func StartGraphQLServer(client *ent.Client) {
    // Usecases
    uUC := useruc.NewUserUsecase(client)
    cUC := communityuc.NewCommunityUsecase(client)
    pUC := postuc.NewPostUsecase(client)
    commentUC := commentuc.NewCommentUsecase(client)
    hostRoleUC := hostroleuc.NewHostRoleUsecase(client)
    communityRoleUC := communityroleuc.NewCommunityRoleUsecase(client)
    banUC := banuc.NewBanUsecase(client)
    	profileTableInfoItemUC := profiletableinfoitem.NewProfileTableInfoItemUsecase(client)

    // gRPC-клиенты к микросервисам (адреса из ENV)
    get := func(key, def string) string { v := os.Getenv(key); if v == "" { return def }; return v }
    useInsecure := os.Getenv("GRPC_INSECURE") == "true"
    var creds grpc.DialOption
    if useInsecure {
        creds = grpc.WithTransportCredentials(insecure.NewCredentials())
    } else {
        tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
        creds = grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg))
    }
    
    // Добавляем authClientInterceptor для автоматического добавления Authorization заголовка
    authConn, err := grpc.DialContext(context.Background(), get("AUTH_GRPC_ADDR", "localhost:4001"), 
        creds, 
        grpc.WithUnaryInterceptor(authClientInterceptor))
    if err != nil { log.Fatalf("❌ AUTH gRPC dial: %v", err) }
    userConn, err := grpc.DialContext(context.Background(), get("USER_GRPC_ADDR", "localhost:4002"), creds)
    if err != nil { log.Fatalf("❌ USER gRPC dial: %v", err) }
    mailConn, err := grpc.DialContext(context.Background(), get("MAIL_GRPC_ADDR", "localhost:4003"), creds)
    if err != nil { log.Fatalf("❌ MAIL gRPC dial: %v", err) }
    mediaConn, err := grpc.DialContext(context.Background(), get("MEDIA_GRPC_ADDR", "localhost:4004"), creds)
    if err != nil { log.Fatalf("❌ MEDIA gRPC dial: %v", err) }

    upstreamClosers = []io.Closer{authConn, userConn, mailConn, mediaConn}

    authClient := authpb.NewAuthServiceClient(authConn)
    userClient := userpb.NewUserServiceClient(userConn)
    mailClient := mailpb.NewMailServiceClient(mailConn)
    mediaClient := mediapb.NewMediaServiceClient(mediaConn)

    // Инициализируем HTTPAuthMiddleware (валидация токена удалённо)
    middleware.InitHTTPAuthMiddleware(authClient)

    // Резолверы
    resolver := &graphql.Resolver{
        Client:          client,
        UserUC:          uUC,
        CommunityUC:     cUC,
        PostUC:          pUC,
        CommentUC:       commentUC,
        HostRoleUC:      hostRoleUC,
        CommunityRoleUC: communityRoleUC,
        BanUC:           banUC,
        AuthClient:      authClient,
        UserClient:      userClient,
        MailClient:      mailClient,
        MediaClient:     mediaClient,
        ProfileTableInfoItemUC: profileTableInfoItemUC,
    }

    // 5) Конфигурируем gqlgen‑сервер вручную (не NewDefaultServer)
    srv := handler.New(graphql.NewExecutableSchema(graphql.Config{Resolvers: resolver}))
    // Нормализованный presenter ошибок (глобально для GraphQL)
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
        if e.Extensions == nil {
            e.Extensions = map[string]any{}
        }
        e.Extensions["code"] = ge.Code
        return e
    })

    // Безопасность и производительность
    if os.Getenv("ENV") != "production" {
        srv.Use(extension.Introspection{})
    }
    // Complexity limit (из ENV или по умолчанию)
    maxComplexity := 300
    if v := os.Getenv("GRAPHQL_MAX_COMPLEXITY"); v != "" {
        if n, err := strconv.Atoi(v); err == nil && n > 0 { maxComplexity = n }
    }
    srv.Use(extension.FixedComplexityLimit(maxComplexity))
    // APQ
    srv.Use(extension.AutomaticPersistedQuery{Cache: lru.New[string](1000)})

	// 5a) HTTP POST и GET
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.GET{})

	// 5b) (Опционально) multipart form (upload)
	srv.AddTransport(transport.MultipartForm{})

    // 5c) WebSocket для подписок
    srv.AddTransport(&transport.Websocket{
        Upgrader: websocket.Upgrader{
            CheckOrigin: func(r *http.Request) bool {
                return allowOrigin(r.Header.Get("Origin"))
            },
        },
        KeepAlivePingInterval: 10 * time.Second,
    })

    // 6) HTTP маршруты
    // Инициализируем S3‑клиент один раз и переиспользуем в хэндлерах и проверках готовности
    s3client := InitS3()
    mux := http.NewServeMux()
    // healthz/readyz
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK); _, _ = w.Write([]byte("ok")) })
    mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 800*time.Millisecond)
        defer cancel()
        if _, err := client.User.Query().Limit(1).All(ctx); err != nil {
            w.WriteHeader(http.StatusServiceUnavailable)
            _, _ = w.Write([]byte("db not ready"))
            return
        }
        // S3 probe
        if err := s3client.HealthCheck(); err != nil {
            w.WriteHeader(http.StatusServiceUnavailable)
            _, _ = w.Write([]byte("s3 not ready"))
            return
        }
        // gRPC upstream health checks
        if resp, err := healthpb.NewHealthClient(authConn).Check(ctx, &healthpb.HealthCheckRequest{}); err != nil || resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
            w.WriteHeader(http.StatusServiceUnavailable)
            _, _ = w.Write([]byte("auth grpc not ready"))
            return
        }
        if resp, err := healthpb.NewHealthClient(userConn).Check(ctx, &healthpb.HealthCheckRequest{}); err != nil || resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
            w.WriteHeader(http.StatusServiceUnavailable)
            _, _ = w.Write([]byte("user grpc not ready"))
            return
        }
        if resp, err := healthpb.NewHealthClient(mailConn).Check(ctx, &healthpb.HealthCheckRequest{}); err != nil || resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
            w.WriteHeader(http.StatusServiceUnavailable)
            _, _ = w.Write([]byte("mail grpc not ready"))
            return
        }
        if resp, err := healthpb.NewHealthClient(mediaConn).Check(ctx, &healthpb.HealthCheckRequest{}); err != nil || resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
            w.WriteHeader(http.StatusServiceUnavailable)
            _, _ = w.Write([]byte("media grpc not ready"))
            return
        }
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ready"))
    })
    // Playground только вне production
    if os.Getenv("ENV") != "production" {
        mux.Handle("/", playground.Handler("GraphQL playground", "/query"))
    } else {
        mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })
    }
	// GraphQL endpoint с улучшенной безопасностью
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
						// Дополнительный double-submit CSRF (опционально)
						if os.Getenv("CSRF_ENABLE") == "true" {
							c, err := r.Cookie("csrf_token")
							tokenHeader := r.Header.Get("X-CSRF-Token")
							if err != nil || c == nil || c.Value == "" || tokenHeader == "" || tokenHeader != c.Value {
								http.Error(w, "invalid csrf token", http.StatusForbidden)
								return
							}
						}
					}
					// Лимит размера тела запроса (по умолчанию 1 МБ)
					maxBody := int64(1 * 1024 * 1024)
					if v := os.Getenv("GRAPHQL_MAX_BODY_BYTES"); v != "" {
						if n, err := strconv.Atoi(v); err == nil && n > 0 { maxBody = int64(n) }
					}
					if r.Method == http.MethodPost {
						r.Body = http.MaxBytesReader(w, r.Body, maxBody)
					}
					// Вставляем куки‑контекст и авторизацию
					ctx := httpWithCookies.WithHTTPContext(r.Context(), w, r)
					r = r.WithContext(ctx)
					middleware.HTTPAuthMiddleware(srv).ServeHTTP(w, r)
				}),
			),
		),
	)
	
	mux.Handle("/query", graphqlHandler)

    // Static storage proxy to S3 (инициализируем локальный клиент и передаем в handler)
    mux.HandleFunc("/storage/", NewStorageHandler(s3client))

    // 7) CORS
    frontend := os.Getenv("FRONTEND_ORIGIN")
    if frontend == "" { frontend = "http://localhost:3000" }
    corsHandler := cors.New(cors.Options{
        AllowedOrigins:   []string{frontend},
        AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
        AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Requested-With"},
        AllowCredentials: true,
        OptionsSuccessStatus: 204,
    }).Handler(mux)

    // 8) Запускаем сервер с graceful shutdown
    addr := os.Getenv("GRAPHQL_HTTP_ADDR")
    if addr == "" { addr = ":8080" }
    httpSrv = &http.Server{
        Addr:              addr,
        Handler:           corsHandler,
        ReadHeaderTimeout: 5 * time.Second,
        ReadTimeout:       15 * time.Second,
        WriteTimeout:      30 * time.Second,
        IdleTimeout:       60 * time.Second,
        MaxHeaderBytes:    1 << 20, // 1MB
    }
    log.Printf("🚀 GraphQL-сервер запущен на %s (HTTP и WS на /query, storage на /storage)", addr)
    go func() {
        if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("❌ Ошибка при запуске GraphQL-сервера: %v", err)
        }
    }()
}

// ShutdownGraphQLServer останавливает HTTP‑сервер и закрывает исходящие gRPC‑соединения
func ShutdownGraphQLServer(ctx context.Context) error {
    if httpSrv != nil {
        _ = httpSrv.Shutdown(ctx)
    }
    for _, c := range upstreamClosers {
        _ = c.Close()
    }
    return nil
}
