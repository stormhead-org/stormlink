package utils

import (
	"context"
	"net/http"
	"os"
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
		return
	}

	secure := false
	domain := "localhost"
	if os.Getenv("ENV") == "production" {
			secure = true
			domain = os.Getenv("APP_DOMAIN")
	}

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
			SameSite: http.SameSiteLaxMode,
			MaxAge:   7 * 24 * 3600,
	})
}
