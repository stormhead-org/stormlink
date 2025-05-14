package utils

import (
	"context"
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
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    accessToken,
		HttpOnly: false,
		Secure:   false, // Для localhost
		Path:     "/",
		Domain:   "localhost", // Доступно для всех портов localhost
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(15 * time.Minute),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Secure:   false,
		Path:     "/",
		Domain:   "localhost",
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})
}
