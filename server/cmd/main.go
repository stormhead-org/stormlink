package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"stormlink/server/grpc/auth"
	"stormlink/server/grpc/user"

	"entgo.io/ent/dialect/sql/schema"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"

	"stormlink/server/ent"
	authpb "stormlink/server/grpc/auth/protobuf"
	userpb "stormlink/server/grpc/user/protobuf"

	_ "github.com/lib/pq"
)

func main() {
	// Путь к .env
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(currentFile), "../..")

	_ = godotenv.Load(filepath.Join(projectRoot, "server/.env"))

	// Подключение к БД
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"), os.Getenv("SSL_MODE"),
	)
	client, err := ent.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("не удалось подключиться к базе: %v", err)
	}
	defer client.Close()

	// Миграции
	resetDB := flag.Bool("reset-db", false, "drop and recreate all tables and columns")
	flag.Parse()

	if *resetDB {
		log.Println("⚠️  Полный сброс базы данных с удалением колонок и индексов...")
		if err := client.Schema.Create(
			context.Background(),
			schema.WithDropIndex(true),
			schema.WithDropColumn(true),
		); err != nil {
			log.Fatalf("ошибка сброса схемы: %v", err)
		}
		log.Println("✅ Сброс базы завершён.")
	} else {
		log.Println("ℹ️  Обычная миграция схемы...")
		if err := client.Schema.Create(context.Background()); err != nil {
			log.Fatalf("ошибка миграции схемы: %v", err)
		}
		log.Println("✅ Миграция завершена.")
	}

	// Инициализация gRPC сервера
	grpcServer := grpc.NewServer()

	userService := user.NewUserService(client)
	userpb.RegisterUserServiceServer(grpcServer, userService)

	authService := auth.NewAuthService(client)
	authpb.RegisterAuthServiceServer(grpcServer, authService)


	// gRPC listener (на 9090)
	go func() {
		listener, err := net.Listen("tcp", ":9090")
		if err != nil {
			log.Fatalf("не удалось слушать порт 9090: %v", err)
		}
		log.Println("📡 gRPC-сервер запущен на :9090")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("ошибка при запуске gRPC-сервера: %v", err)
		}
	}()

	// HTTP Gateway mux
	ctx := context.Background()
	gwmux := gwruntime.NewServeMux()

	// Подключаем grpc-gateway хендлеры
	err = userpb.RegisterUserServiceHandlerFromEndpoint(ctx, gwmux, "localhost:9090", []grpc.DialOption{grpc.WithInsecure()})
	if err != nil {
		log.Fatalf("не удалось зарегистрировать grpc-gateway хендлер UserService: %v", err)
	}

	err = authpb.RegisterAuthServiceHandlerFromEndpoint(ctx, gwmux, "localhost:9090", []grpc.DialOption{grpc.WithInsecure()})
	if err != nil {
	log.Fatalf("не удалось зарегистрировать grpc-gateway хендлер AuthService: %v", err)
	}


	// HTTP сервер (на 8080)
	log.Println("🌐 HTTP-сервер (grpc-gateway) запущен на :8080")
	if err := http.ListenAndServe(":8080", gwmux); err != nil {
		log.Fatalf("ошибка при запуске HTTP-сервера: %v", err)
	}
}
