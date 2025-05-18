package middleware

import (
	"context"
	"log"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"stormlink/server/utils"
)

func GRPCAuthInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	log.Println("🔎 Full method:", info.FullMethod)

	// Публичные методы
	publicMethods := map[string]bool{
		"/auth.AuthService/Login":                   true,
		"/UserService/RegisterUser":                 true,
		"/auth.AuthService/VerifyEmail":             true,
		"/auth.AuthService/ResendVerificationEmail": true,
	}

	if publicMethods[info.FullMethod] {
		log.Println("✅ Публичный метод, не требуется авторизация")
		return handler(ctx, req)
	}

	// Извлекаем метаданные из контекста
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Println("❌ [AuthInterceptor] Missing metadata")
		return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
	}

	// Проверяем заголовок Authorization
	authHeader, ok := md["authorization"]
	if !ok || len(authHeader) == 0 {
		log.Println("❌ [AuthInterceptor] Missing Authorization header")
		return nil, status.Errorf(codes.Unauthenticated, "missing or invalid token")
	}
	if !strings.HasPrefix(authHeader[0], "Bearer ") {
		log.Println("❌ [AuthInterceptor] Invalid Authorization format:", authHeader[0])
		return nil, status.Errorf(codes.Unauthenticated, "missing or invalid token")
	}

	tokenStr := strings.TrimPrefix(authHeader[0], "Bearer ")

	// Валидируем токен
	claims, err := utils.ParseAccessToken(tokenStr)
	if err != nil {
		log.Println("❌ [AuthInterceptor] Invalid token:", err)
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	newCtx := context.WithValue(ctx, "userID", claims.UserID)
	return handler(newCtx, req)
}
