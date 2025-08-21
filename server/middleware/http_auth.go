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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

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
                if len(c.Value) > 20 {
                    log.Printf("🔍 HTTPAuthMiddleware: Found auth_token cookie: %s", c.Value[:20] + "...")
                } else {
                    log.Printf("🔍 HTTPAuthMiddleware: Found auth_token cookie: %s", c.Value)
                }
            } else {
                log.Printf("🔍 HTTPAuthMiddleware: No auth_token cookie found: %v", err)
            }
        } else {
            if len(authHeader) > 20 {
                log.Printf("🔍 HTTPAuthMiddleware: Found Authorization header: %s", authHeader[:20] + "...")
            } else {
                log.Printf("🔍 HTTPAuthMiddleware: Found Authorization header: %s", authHeader)
            }
        }
        ctx = context.WithValue(ctx, "authorization", authHeader)

        if authHeader == "" {
            log.Printf("🔍 HTTPAuthMiddleware: No auth header found")
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }

        if len(authHeader) > 30 {
            log.Printf("🔍 HTTPAuthMiddleware: Final authHeader: %s", authHeader[:30] + "...")
        } else {
            log.Printf("🔍 HTTPAuthMiddleware: Final authHeader: %s", authHeader)
        }
        
        if !strings.HasPrefix(authHeader, "Bearer ") {
            log.Printf("❌ HTTPAuthMiddleware: No Bearer prefix in authHeader")
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }

        token := strings.TrimPrefix(authHeader, "Bearer ")
        if len(token) > 20 {
            log.Printf("🔍 HTTPAuthMiddleware: Validating token: %s", token[:20] + "...")
        } else {
            log.Printf("🔍 HTTPAuthMiddleware: Validating token: %s", token)
        }
        // валидация удалённо через auth-сервис
        resp, err := authClient.ValidateToken(ctx, &protobuf.ValidateTokenRequest{Token: token})
        if err != nil || !resp.GetValid() {
            log.Printf("❌ HTTPAuthMiddleware: invalid access token: %v", err)
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }
        log.Printf("✅ HTTPAuthMiddleware: Token validated, userID: %d", resp.GetUserId())
        ctx = sharedauth.WithUserID(ctx, int(resp.GetUserId()))
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}