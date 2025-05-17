package modules

import (
	"context"
	"log"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	authpb "stormlink/server/grpc/auth/protobuf"
	userpb "stormlink/server/grpc/user/protobuf"
)

func SetupHTTPServer(grpcConn *grpc.ClientConn, mux *http.ServeMux) *http.Server {
	ctx := context.Background()
	gwmux := runtime.NewServeMux(
		runtime.WithErrorHandler(func(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
			statusCode := codes.Unknown
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code()
			}
			if statusCode == codes.ResourceExhausted {
				http.Error(w, `{"error": "rate limit exceeded, try again later"}`, http.StatusTooManyRequests)
				return
			}
			runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
		}),
	)

	if err := userpb.RegisterUserServiceHandlerFromEndpoint(
		ctx, gwmux, "localhost:4000",
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	); err != nil {
		log.Fatalf("не удалось зарегистрировать grpc-gateway хендлер UserService: %v", err)
	}

	if err := authpb.RegisterAuthServiceHandlerFromEndpoint(
		ctx, gwmux, "localhost:4000",
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	); err != nil {
		log.Fatalf("не удалось зарегистрировать grpc-gateway хендлер AuthService: %v", err)
	}

	mux.Handle("/", gwmux)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Set-Cookie"},
		AllowCredentials: true,
	}).Handler(mux)

	httpServer := &http.Server{
		Addr: ":4080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			corsHandler.ServeHTTP(w, r)
		}),
	}

	return httpServer
}

func StartHTTPServer(httpServer *http.Server) {
	log.Println("🌐 HTTP-сервер запущен на :4080")
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("ошибка при запуске HTTP-сервера: %v", err)
	}
}
