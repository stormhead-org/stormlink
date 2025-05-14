package utils

import (
	"context"
	"log"
	"net/http"
	"time"
)

type contextKey string

const (
    httpRequestKey  contextKey = "httpRequest"
    httpResponseKey contextKey = "httpResponse"
)

func WithHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
    ctx = context.WithValue(ctx, httpRequestKey, r)
    ctx = context.WithValue(ctx, httpResponseKey, w)
    return ctx
}

func GetHTTPRequest(ctx context.Context) *http.Request {
    req, _ := ctx.Value(httpRequestKey).(*http.Request)
    return req
}

func GetHTTPResponseWriter(ctx context.Context) http.ResponseWriter {
    w, _ := ctx.Value(httpResponseKey).(http.ResponseWriter)
    return w
}

func SetAuthCookies(w http.ResponseWriter, accessToken, refreshToken string) {
    if w == nil {
        log.Println("⚠️ [SetAuthCookies] http.ResponseWriter is nil, cannot set cookies")
        return
    }

    http.SetCookie(w, &http.Cookie{
        Name:     "auth_token",
        Value:    accessToken,
        HttpOnly: false,
        Secure:   false, // Временно для localhost, включите true в продакшене
        Path:     "/",
        SameSite: http.SameSiteLaxMode, // Lax для локальной разработки
        Expires:  time.Now().Add(15 * time.Minute),
    })
    log.Printf("✅ [SetAuthCookies] Set auth_token: %s", accessToken)

    http.SetCookie(w, &http.Cookie{
        Name:     "refresh_token",
        Value:    refreshToken,
        HttpOnly: true,
        Secure:   false, // Временно для localhost
        Path:     "/",
        SameSite: http.SameSiteLaxMode,
        Expires:  time.Now().Add(7 * 24 * time.Hour),
    })
    log.Printf("✅ [SetAuthCookies] Set refresh_token: %s", refreshToken)
}