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
	userpb "stormlink/server/grpc/user/protobuf"
	useruc "stormlink/server/usecase/user"
	"stormlink/services/user/internal/service"

	"google.golang.org/grpc"
	health "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
    modules.InitEnv()
    client := modules.ConnectDB()
    defer client.Close()

    addr := os.Getenv("USER_GRPC_ADDR")
    if addr == "" { addr = ":4002" }

    uc := useruc.NewUserUsecase(client)
    svc := service.NewUserService(client, uc)

    s := grpc.NewServer()
    userpb.RegisterUserServiceServer(s, svc)

    hs := health.NewServer()
    healthpb.RegisterHealthServer(s, hs)
    hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

    lis, err := net.Listen("tcp", addr)
    if err != nil { log.Fatalf("listen %s: %v", addr, err) }

    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    go func() {
        log.Printf("📡 user gRPC on %s", addr)
        if err := s.Serve(lis); err != nil {
            log.Printf("grpc serve stopped: %v", err)
        }
    }()

    <-ctx.Done()
    log.Println("👋 user: shutting down...")
    hs.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
    done := make(chan struct{})
    go func() { s.GracefulStop(); close(done) }()
    select {
    case <-done:
    case <-time.After(5 * time.Second):
        log.Println("⚠️ user: force stop")
        s.Stop()
    }
}


