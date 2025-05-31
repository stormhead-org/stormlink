package modules

import (
	"log"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"stormlink/server/ent"
	"stormlink/server/graphql"
	authpb "stormlink/server/grpc/auth/protobuf"
	mailpb "stormlink/server/grpc/mail/protobuf"
	mediapb "stormlink/server/grpc/media/protobuf"
	userpb "stormlink/server/grpc/user/protobuf"
	"stormlink/server/middleware"
	httpWithCookies "stormlink/server/pkg/http"
	"stormlink/server/usecase/user"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/rs/cors"
)

func StartGraphQLServer(client *ent.Client) {
    // Подключаемся к gRPC-серверу
    conn, err := grpc.Dial("localhost:4000", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("❌ Не удалось подключиться к gRPC-серверу: %v", err)
    }

    // Создаем gRPC-клиенты
    authClient := authpb.NewAuthServiceClient(conn)
    userClient := userpb.NewUserServiceClient(conn)
    mailClient := mailpb.NewMailServiceClient(conn)
    mediaClient := mediapb.NewMediaServiceClient(conn)

    // Инициализируем HTTPAuthMiddleware с gRPC-клиентом
    middleware.InitHTTPAuthMiddleware(authClient)

    uc := user.NewUserUsecase(client)
    resolver := &graphql.Resolver{
        Client:     client,
        UC:         uc,
        AuthClient: authClient,
        UserClient: userClient,
        MailClient: mailClient,
        MediaClient: mediaClient,
    }

    srv := handler.NewDefaultServer(graphql.NewExecutableSchema(graphql.Config{
        Resolvers: resolver,
    }))

    mux := http.NewServeMux()
    mux.Handle("/", playground.Handler("GraphQL", "/query"))
    mux.Handle("/query", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := httpWithCookies.WithHTTPContext(r.Context(), w, r)
        r = r.WithContext(ctx)
        middleware.HTTPAuthMiddleware(srv).ServeHTTP(w, r)
    }))

    // Настройка CORS
    corsHandler := cors.New(cors.Options{
        AllowedOrigins:   []string{"http://localhost:3000"},
        AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
        AllowedHeaders:   []string{"Authorization", "Content-Type"},
        AllowCredentials: true,
    }).Handler(mux)

    log.Println("🚀 GraphQL-сервер запущен на :8080")
    if err := http.ListenAndServe(":8080", corsHandler); err != nil {
        log.Fatalf("❌ Ошибка при запуске GraphQL-сервера: %v", err)
    }
}