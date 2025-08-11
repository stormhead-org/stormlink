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
	mediapb "stormlink/server/grpc/media/protobuf"
	"stormlink/services/media/internal/service"
	shareds3 "stormlink/shared/s3"

	"google.golang.org/grpc"
	health "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
    modules.InitEnv()
    client := modules.ConnectDB()
    defer client.Close()

    addr := os.Getenv("MEDIA_GRPC_ADDR")
    if addr == "" { addr = ":4004" }

    s3client, err := shareds3.NewS3Client()
    if err != nil { log.Fatalf("s3 client: %v", err) }

    svc := service.NewMediaServiceWithClient(s3client, client)

    s := grpc.NewServer()
    mediapb.RegisterMediaServiceServer(s, svc)

    hs := health.NewServer()
    healthpb.RegisterHealthServer(s, hs)
    hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

    lis, err := net.Listen("tcp", addr)
    if err != nil { log.Fatalf("listen %s: %v", addr, err) }

    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    go func() {
        log.Printf("📡 media gRPC on %s", addr)
        if err := s.Serve(lis); err != nil {
            log.Printf("grpc serve stopped: %v", err)
        }
    }()

    <-ctx.Done()
    log.Println("👋 media: shutting down...")
    hs.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
    done := make(chan struct{})
    go func() { s.GracefulStop(); close(done) }()
    select {
    case <-done:
    case <-time.After(5 * time.Second):
        log.Println("⚠️ media: force stop")
        s.Stop()
    }
}


