// server/cmd/modules/graphql_server.go
package modules

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"

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
	"stormlink/server/usecase/community"
	"stormlink/server/usecase/user"
)

func StartGraphQLServer(client *ent.Client) {
	// 1) Подключаемся к gRPC-серверу по контексту
	conn, err := grpc.DialContext(
		context.Background(),
		"localhost:4000",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("❌ Не удалось подключиться к gRPC-серверу: %v", err)
	}

	// 2) Создаём gRPC-клиенты
	authClient := authpb.NewAuthServiceClient(conn)
	userClient := userpb.NewUserServiceClient(conn)
	mailClient := mailpb.NewMailServiceClient(conn)
	mediaClient := mediapb.NewMediaServiceClient(conn)

	// 3) Инициализируем HTTPAuthMiddleware
	middleware.InitHTTPAuthMiddleware(authClient)

	// 4) Стек usecase и резолверы
	uUC := user.NewUserUsecase(client)
	cUC := community.NewCommunityUsecase(client)
	resolver := &graphql.Resolver{
		Client:       client,
		UserUC:       uUC,
		CommunityUC:  cUC,
		AuthClient:   authClient,
		UserClient:   userClient,
		MailClient:   mailClient,
		MediaClient:  mediaClient,
	}

	// 5) Конфигурируем gqlgen-сервер вручную (не NewDefaultServer)
	srv := handler.New(graphql.NewExecutableSchema(graphql.Config{
		Resolvers: resolver,
	}))

    // Включаем introspection (только в DEV!)
    srv.Use(extension.Introspection{})

	// 5a) HTTP POST и GET
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.GET{})

	// 5b) (Опционально) multipart form (upload)
	srv.AddTransport(transport.MultipartForm{})

	// 5c) WebSocket для подписок
	srv.AddTransport(&transport.Websocket{
		Upgrader: websocket.Upgrader{
			// Позволяем соединения с любых Origin (или замените на свою логику)
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		KeepAlivePingInterval: 10 * time.Second,
	})

	// 6) HTTP маршруты
	mux := http.NewServeMux()
	// Playground на корне
	mux.Handle("/", playground.Handler("GraphQL playground", "/query"))
	// GraphQL endpoint
	mux.Handle("/query", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Вставляем куки-контекст и авторизацию
		ctx := httpWithCookies.WithHTTPContext(r.Context(), w, r)
		r = r.WithContext(ctx)
		middleware.HTTPAuthMiddleware(srv).ServeHTTP(w, r)
	}))

	// 7) CORS
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}).Handler(mux)

	// 8) Запускаем сервер
	log.Println("🚀 GraphQL-сервер запущен на :8080 (HTTP и WS на /query)")
	if err := http.ListenAndServe(":8080", corsHandler); err != nil {
		log.Fatalf("❌ Ошибка при запуске GraphQL-сервера: %v", err)
	}
}
