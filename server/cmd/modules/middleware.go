package modules

import (
	"stormlink/server/middleware"

	"google.golang.org/grpc"
)

func RegisterMiddleware(server *grpc.Server, rl *middleware.RateLimiter) {
	// Регистрируем наши middleware
	rateLimitMiddleware := middleware.RateLimitMiddleware(rl)
	authMiddleware := middleware.GRPCAuthMiddleware

	server.RegisterService(nil, grpc.ChainUnaryInterceptor(
		rateLimitMiddleware,
		authMiddleware,
	))
}
