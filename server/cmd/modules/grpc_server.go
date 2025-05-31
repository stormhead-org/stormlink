package modules

import (
	"context"
	"log"
	"net"

	"stormlink/server/ent"
	"stormlink/server/grpc/auth"
	authpb "stormlink/server/grpc/auth/protobuf"
	"stormlink/server/grpc/mail"
	mailpb "stormlink/server/grpc/mail/protobuf"
	"stormlink/server/grpc/media"
	mediapb "stormlink/server/grpc/media/protobuf"
	"stormlink/server/grpc/user"
	userpb "stormlink/server/grpc/user/protobuf"
	"stormlink/server/middleware"
	"stormlink/server/pkg/s3"
	usecase "stormlink/server/usecase/user"

	"google.golang.org/grpc"

	"golang.org/x/time/rate"
)

func chainInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if len(interceptors) == 0 {
			return handler(ctx, req)
		}
		var chainHandler grpc.UnaryHandler = handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			current := interceptors[i]
			chainHandler = func(currentCtx context.Context, currentReq interface{}, currentInfo *grpc.UnaryServerInfo, next grpc.UnaryHandler) grpc.UnaryHandler {
				return func(ctx context.Context, req interface{}) (interface{}, error) {
					return current(ctx, req, currentInfo, next)
				}
			}(ctx, req, info, chainHandler)
		}
		return chainHandler(ctx, req)
	}
}

func SetupGRPCServer(client *ent.Client) *grpc.Server {
	rl := middleware.NewRateLimiter(rate.Limit(1), 3)
	chain := []grpc.UnaryServerInterceptor{
		middleware.RateLimitMiddleware(rl),
		middleware.GRPCAuthMiddleware,
	}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(chainInterceptors(chain...)),
	)

	userUsecase := usecase.NewUserUsecase(client)
	userService := user.NewUserService(client, userUsecase)
	userpb.RegisterUserServiceServer(grpcServer, userService)

	authService := auth.NewAuthService(client, userUsecase)
	authpb.RegisterAuthServiceServer(grpcServer, authService)

	mailService := mail.NewMailService(client)
	mailpb.RegisterMailServiceServer(grpcServer, mailService)

	s3Client, err := s3.NewS3Client()
	if err != nil {
		log.Fatalf("failed to init S3 client: %v", err)
	}
	mediaService := media.NewMediaServiceWithClient(s3Client)
	mediapb.RegisterMediaServiceServer(grpcServer, mediaService)

	return grpcServer
}

func StartGRPCServer(grpcServer *grpc.Server) {
	listener, err := net.Listen("tcp", ":4000")
	if err != nil {
		log.Fatalf("не удалось слушать порт 4000: %v", err)
	}
	log.Println("📡 gRPC-сервер запущен на :4000")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("ошибка при запуске gRPC-сервера: %v", err)
	}
}
