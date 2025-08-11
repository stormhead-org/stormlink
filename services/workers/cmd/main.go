package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"stormlink/server/cmd/modules"
	mailworker "stormlink/services/workers/internal/mail"
)

func main() {
    modules.InitEnv()

    mail := flag.Bool("mail", false, "run mail worker only")
    healthAddr := flag.String("health-addr", ":8090", "http health endpoint addr")
    flag.Parse()

    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    // простой HTTP /healthz
    mux := http.NewServeMux()
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok"))
    })
    srv := &http.Server{Addr: *healthAddr, Handler: mux}
    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Printf("health http error: %v", err)
        }
    }()

    if *mail {
        log.Println("📬 starting mail worker...")
        if err := mailworker.Run(ctx); err != nil {
            log.Fatalf("mail worker error: %v", err)
        }
        _ = srv.Shutdown(context.Background())
        return
    }

    // Запуск всех доступных воркеров (пока почтовый)
    log.Println("🛠 starting all workers (mail)...")
    done := make(chan error, 1)
    go func() { done <- mailworker.Run(ctx) }()

    select {
    case <-ctx.Done():
        log.Println("👋 workers shutdown")
    case err := <-done:
        if err != nil {
            log.Fatalf("worker failed: %v", err)
        }
    }
    _ = srv.Shutdown(context.Background())
}


