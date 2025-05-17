package modules

import (
	"google.golang.org/grpc"
	"stormlink/server/middleware"
)

// RegisterMiddleware регистрирует gRPC middleware
func RegisterMiddleware(server *grpc.Server, rl *middleware.RateLimiter) {
	// Регистрируем RateLimitInterceptor
	rateLimit := middleware.RateLimitInterceptor(rl)
	// Регистрируем GRPCAuthInterceptor
	authInterceptor := middleware.GRPCAuthInterceptor

	// Применяем оба интерцептора в цепочке
	server.RegisterService(nil, grpc.ChainUnaryInterceptor(
		rateLimit,
		authInterceptor,
	))
}
