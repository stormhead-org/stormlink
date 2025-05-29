package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"stormlink/server/grpc/auth/protobuf"
	httpCookies "stormlink/server/pkg/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var authClient protobuf.AuthServiceClient

func InitHTTPAuthMiddleware(client protobuf.AuthServiceClient) {
    authClient = client
}

func HTTPAuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Извлекаем заголовок Authorization
        authHeader := r.Header.Get("Authorization")

        // Добавляем http.ResponseWriter и http.Request в контекст
        ctx := httpCookies.WithHTTPContext(r.Context(), w, r)
        ctx = context.WithValue(ctx, "userID", 0)
        ctx = context.WithValue(ctx, "authorization", authHeader)

        if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }

        token := strings.TrimPrefix(authHeader, "Bearer ")

        // Подключаемся к gRPC-серверу
        conn, err := grpc.Dial("localhost:4000", grpc.WithTransportCredentials(insecure.NewCredentials()))
        if err != nil {
            log.Printf("❌ GraphQL: Ошибка подключения к gRPC: %v", err)
            http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
            return
        }
        defer conn.Close()

        // Создаём gRPC-клиент
        authClient := protobuf.NewAuthServiceClient(conn)

        // Валидируем токен
        resp, err := authClient.ValidateToken(ctx, &protobuf.ValidateTokenRequest{Token: token})
        if err != nil || !resp.Valid {
            log.Printf("❌ GraphQL: Ошибка валидации токена: %v", err)
            ctx = context.WithValue(ctx, "userID", 0)
            ctx = context.WithValue(ctx, "authorization", authHeader)
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }

        // Добавляем userID в контекст
        ctx = context.WithValue(ctx, "userID", int(resp.UserId))
        ctx = context.WithValue(ctx, "authorization", authHeader)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}