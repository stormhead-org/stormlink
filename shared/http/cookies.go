package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
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
    if w == nil { return }
    secure := false
    domain := os.Getenv("APP_COOKIE_DOMAIN")
    if domain == "" { domain = "localhost" }
    if os.Getenv("ENV") == "production" { secure = true }

    http.SetCookie(w, &http.Cookie{
        Name:     "auth_token",
        Value:    accessToken,
        HttpOnly: true,
        Secure:   secure,
        Path:     "/",
        Domain:   domain,
        SameSite: http.SameSiteLaxMode,
        MaxAge:   15 * 60,
    })

    http.SetCookie(w, &http.Cookie{
        Name:     "refresh_token",
        Value:    refreshToken,
        HttpOnly: true,
        Secure:   secure,
        Path:     "/",
        Domain:   domain,
        SameSite: http.SameSiteStrictMode,
        MaxAge:   7 * 24 * 3600,
    })

    // Опциональный double-submit CSRF токен (включается ENV CSRF_ENABLE=true)
    if os.Getenv("CSRF_ENABLE") == "true" {
        // генерируем случайный токен и ставим не HttpOnly куку
        buf := make([]byte, 32)
        if _, err := rand.Read(buf); err == nil {
            csrf := hex.EncodeToString(buf)
            http.SetCookie(w, &http.Cookie{
                Name:     "csrf_token",
                Value:    csrf,
                HttpOnly: false, // читается фронтом
                Secure:   secure,
                Path:     "/",
                Domain:   domain,
                SameSite: http.SameSiteLaxMode,
                MaxAge:   15 * 60,
            })
        }
    }
}

func ClearAuthCookies(w http.ResponseWriter) {
    if w == nil { return }
    secure := false
    domain := os.Getenv("APP_COOKIE_DOMAIN")
    if domain == "" { domain = "localhost" }
    if os.Getenv("ENV") == "production" { secure = true }

    http.SetCookie(w, &http.Cookie{
        Name:     "auth_token",
        Value:    "",
        HttpOnly: false,
        Secure:   secure,
        Path:     "/",
        Domain:   domain,
        SameSite: http.SameSiteLaxMode,
        Expires:  time.Unix(0, 0),
        MaxAge:   -1,
    })
    http.SetCookie(w, &http.Cookie{
        Name:     "refresh_token",
        Value:    "",
        HttpOnly: true,
        Secure:   secure,
        Path:     "/",
        Domain:   domain,
        SameSite: http.SameSiteLaxMode,
        Expires:  time.Unix(0, 0),
        MaxAge:   -1,
    })

    http.SetCookie(w, &http.Cookie{
        Name:     "csrf_token",
        Value:    "",
        HttpOnly: false,
        Secure:   secure,
        Path:     "/",
        Domain:   domain,
        SameSite: http.SameSiteLaxMode,
        Expires:  time.Unix(0, 0),
        MaxAge:   -1,
    })
}


