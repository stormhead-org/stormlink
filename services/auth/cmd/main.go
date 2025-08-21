package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stormlink/server/cmd/modules"
	authpb "stormlink/server/grpc/auth/protobuf"
	"stormlink/server/middleware"
	usersuc "stormlink/server/usecase/user"
	"stormlink/services/auth/internal/service"

	"google.golang.org/grpc"
	health "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
    modules.InitEnv()

    client := modules.ConnectDB()
    defer client.Close()

    uUC := usersuc.NewUserUsecase(client)
    svc := service.NewAuthService(client, uUC)

    addr := os.Getenv("AUTH_GRPC_ADDR")
    if addr == "" { addr = ":4001" }

    // gRPC сервер с middleware для авторизации и rate limiting
    s := grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            middleware.GRPCAuthRateLimitMiddleware,
            middleware.GRPCAuthMiddleware,
        ),
    )
    authpb.RegisterAuthServiceServer(s, svc)

    // gRPC health-check
    hs := health.NewServer()
    healthpb.RegisterHealthServer(s, hs)
    hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

    lis, err := net.Listen("tcp", addr)
    if err != nil { log.Fatalf("listen %s: %v", addr, err) }

    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    go func() {
        log.Printf("📡 auth gRPC on %s", addr)
        if err := s.Serve(lis); err != nil {
            log.Printf("grpc serve stopped: %v", err)
        }
    }()

    <-ctx.Done()
    log.Println("👋 auth: shutting down...")
    hs.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)

    done := make(chan struct{})
    go func() {
        s.GracefulStop()
        close(done)
    }()

    select {
    case <-done:
    case <-time.After(5 * time.Second):
        log.Println("⚠️ auth: force stop")
        s.Stop()
    }
}


