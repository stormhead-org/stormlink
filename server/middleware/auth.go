package middleware

import (
	"context"
	"log"
	"stormlink/server/utils"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func GRPCAuthInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	// Показываем gRPC в консоль, чтобы легче было ориентироваться в запросах
	log.Println("🔎 Full method:", info.FullMethod)

	// Публичные методы
	publicMethods := map[string]bool{
		"/auth.AuthService/Login":                   true,
		"/UserService/RegisterUser":                 true,
		"/auth.AuthService/VerifyEmail":             true,
		"/auth.AuthService/ResendVerificationEmail": true,
	}

	if publicMethods[info.FullMethod] {
		// Не проверяем токен для публичных методов
		log.Println("✅ Публичный метод, не требуется авторизация")
		return handler(ctx, req)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
	}

	authHeader := md["authorization"]
	if len(authHeader) == 0 || !strings.HasPrefix(authHeader[0], "Bearer ") {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token")
	}

	tokenStr := strings.TrimPrefix(authHeader[0], "Bearer ")
	claims, err := utils.ParseAccessToken(tokenStr)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token")
	}

	// Устанавливаем userID как строку UUID
	newCtx := context.WithValue(ctx, "userID", claims.UserID.String())
	return handler(newCtx, req)
}
