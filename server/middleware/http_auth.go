package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"stormlink/server/grpc/auth/protobuf"
	sharedauth "stormlink/shared/auth"
	httpCookies "stormlink/shared/http"
)

// HTTPAuthMiddleware валидирует Bearer access JWT локально и добавляет userID и Authorization в контекст
var authClient protobuf.AuthServiceClient

func InitHTTPAuthMiddleware(client protobuf.AuthServiceClient) {
    authClient = client
}

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

        if !strings.HasPrefix(authHeader, "Bearer ") {
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }

        token := strings.TrimPrefix(authHeader, "Bearer ")
        // валидация удалённо через auth-сервис
        resp, err := authClient.ValidateToken(ctx, &protobuf.ValidateTokenRequest{Token: token})
        if err != nil || !resp.GetValid() {
            log.Printf("❌ HTTPAuthMiddleware: invalid access token: %v", err)
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }
        ctx = sharedauth.WithUserID(ctx, int(resp.GetUserId()))
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}