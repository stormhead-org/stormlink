// server/middleware/http_auth.go
package middleware

import (
	"context"
	"log"
	"net/http"
	"stormlink/server/utils"
	"strings"
)

func HTTPAuthMiddleware(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    hdr := r.Header.Get("Authorization")
    if strings.HasPrefix(hdr, "Bearer ") {
      token := strings.TrimPrefix(hdr, "Bearer ")
      log.Printf("🔑 GraphQL: Получен токен: %s", token)
      if claims, err := utils.ParseAccessToken(token); err == nil {
        ctx := context.WithValue(r.Context(), "userID", claims.UserID)
        r = r.WithContext(ctx)
        log.Printf("✅ GraphQL: Установлен userID: %d", claims.UserID)
      } else {
        log.Printf("❌ GraphQL: Ошибка токена: %v", err)
      }
    } else {
      log.Println("❌ GraphQL: Заголовок Authorization отсутствует")
    }
    next.ServeHTTP(w, r)
  })
}