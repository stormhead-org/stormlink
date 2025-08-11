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
	mailpb "stormlink/server/grpc/mail/protobuf"
	"stormlink/services/mail/internal/service"

	"google.golang.org/grpc"
	health "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
    modules.InitEnv()
    client := modules.ConnectDB()
    defer client.Close()

    addr := os.Getenv("MAIL_GRPC_ADDR")
    if addr == "" { addr = ":4003" }

    svc := service.NewMailService(client)

    s := grpc.NewServer()
    mailpb.RegisterMailServiceServer(s, svc)

    hs := health.NewServer()
    healthpb.RegisterHealthServer(s, hs)
    hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

    lis, err := net.Listen("tcp", addr)
    if err != nil { log.Fatalf("listen %s: %v", addr, err) }

    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    go func() {
        log.Printf("📡 mail gRPC on %s", addr)
        if err := s.Serve(lis); err != nil {
            log.Printf("grpc serve stopped: %v", err)
        }
    }()

    <-ctx.Done()
    log.Println("👋 mail: shutting down...")
    hs.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
    done := make(chan struct{})
    go func() { s.GracefulStop(); close(done) }()
    select {
    case <-done:
    case <-time.After(5 * time.Second):
        log.Println("⚠️ mail: force stop")
        s.Stop()
    }
}


