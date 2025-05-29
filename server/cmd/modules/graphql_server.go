package modules

import (
	"log"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"stormlink/server/ent"
	"stormlink/server/graphql"
	authpb "stormlink/server/grpc/auth/protobuf"
	userpb "stormlink/server/grpc/user/protobuf"
	"stormlink/server/middleware"
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

    // Инициализируем HTTPAuthMiddleware с gRPC-клиентом
    middleware.InitHTTPAuthMiddleware(authClient)

    uc := user.NewUserUsecase(client)
    resolver := &graphql.Resolver{
        Client:     client,
        UC:         uc,
        AuthClient: authClient,
        UserClient: userClient,
    }

    srv := handler.NewDefaultServer(graphql.NewExecutableSchema(graphql.Config{
        Resolvers: resolver,
    }))

    mux := http.NewServeMux()
    mux.Handle("/", playground.Handler("GraphQL", "/query"))
    mux.Handle("/query", middleware.HTTPAuthMiddleware(srv))

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