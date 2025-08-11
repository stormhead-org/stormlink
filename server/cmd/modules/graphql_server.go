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
	"golang.org/x/time/rate"

	"stormlink/server/ent"
	"stormlink/server/graphql"
	authpb "stormlink/server/grpc/auth/protobuf"
	mailpb "stormlink/server/grpc/mail/protobuf"
	mediapb "stormlink/server/grpc/media/protobuf"
	userpb "stormlink/server/grpc/user/protobuf"
	"stormlink/server/middleware"
	communityuc "stormlink/server/usecase/community"
	useruc "stormlink/server/usecase/user"
	httpWithCookies "stormlink/shared/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
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

func StartGraphQLServer(client *ent.Client) {
    // Usecases
    uUC := useruc.NewUserUsecase(client)
    cUC := communityuc.NewCommunityUsecase(client)

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
    authConn, err := grpc.DialContext(context.Background(), get("AUTH_GRPC_ADDR", "localhost:4001"), creds)
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
        Client:      client,
        UserUC:      uUC,
        CommunityUC: cUC,
        AuthClient:  authClient,
        UserClient:  userClient,
        MailClient:  mailClient,
        MediaClient: mediaClient,
    }

    // 5) Конфигурируем gqlgen‑сервер вручную (не NewDefaultServer)
    srv := handler.New(graphql.NewExecutableSchema(graphql.Config{Resolvers: resolver}))

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
    mux := http.NewServeMux()
    // healthz/readyz
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK); _, _ = w.Write([]byte("ok")) })
    mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
        defer cancel()
        if _, err := client.User.Query().Limit(1).All(ctx); err != nil {
            w.WriteHeader(http.StatusServiceUnavailable)
            _, _ = w.Write([]byte("db not ready"))
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
	// GraphQL endpoint
    // CSRF: проверяем Origin для небезопасных методов; лимит размера тела; простой rate-limit по IP
    mux.Handle("/query", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
        // Простой rate limit по IP для публичной точки входа
        ip := getClientIP(r)
        ipLimiters.mu.Lock()
        lim, ok := ipLimiters.limiters[ip]
        if !ok {
            // 10 req/sec, burst 30 по умолчанию
            lim = rate.NewLimiter(10, 30)
            ipLimiters.limiters[ip] = lim
        }
        ipLimiters.mu.Unlock()
        if !lim.Allow() {
            http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        // Вставляем куки‑контекст и авторизацию
        ctx := httpWithCookies.WithHTTPContext(r.Context(), w, r)
        r = r.WithContext(ctx)
        middleware.HTTPAuthMiddleware(srv).ServeHTTP(w, r)
    }))

    // Static storage proxy to S3
    mux.HandleFunc("/storage/", StorageHandler)

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
