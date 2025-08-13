package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"
	"time"

	"stormlink/server/cmd/modules"
)

func main() {
    modules.InitEnv()

    client := modules.ConnectDB()
    defer client.Close()

    resetDB := flag.Bool("reset-db", false, "drop and recreate all tables and columns")
    seed := flag.Bool("seed", false, "seed roles, default host etc.")
    flag.Parse()

    modules.MigrateDB(client, *resetDB, *seed)

    // запуск сервера
    modules.StartGraphQLServer(client)

    // ожидание сигналов и корректное завершение HTTP-сервера в модуле
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()
    <-ctx.Done()
    log.Println("👋 graphql server stopping...")
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    _ = modules.ShutdownGraphQLServer(shutdownCtx)
}
