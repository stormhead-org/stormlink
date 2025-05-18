package main

import (
	"flag"
	"log"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"stormlink/server/cmd/modules"
)

func main() {
	// Инициализация окружения
	modules.InitEnv()
	modules.InitS3Client()

	// Подключение к базе данных
	client := modules.ConnectDB()
	defer client.Close()

	// Обработка флагов командной строки
	resetDB := flag.Bool("reset-db", false, "drop and recreate all tables and columns")
	seed := flag.Bool("seed", false, "seed roles, default host etc.")
	flag.Parse()

	// Миграция базы данных
	modules.MigrateDB(client, *resetDB, *seed)

	// Настройка gRPC-сервера
	grpcServer := modules.SetupGRPCServer(client)

	// Запуск gRPC-сервера в отдельной горутине
	go modules.StartGRPCServer(grpcServer)

	// 🚀 Запуск GraphQL-сервера
	go modules.StartGraphQLServer(client)

	// Подключение к gRPC-серверу для использования в HTTP-хендлерах
	grpcConn, err := grpc.Dial("localhost:4000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("не удалось подключиться к gRPC-серверу: %v", err)
	}
	defer grpcConn.Close()

	// Настройка HTTP-мультиплексора
	mux := http.NewServeMux()

	// Регистрация HTTP-хендлеров из modules/handlers.go
	mux.HandleFunc("/v1/users/login", func(w http.ResponseWriter, r *http.Request) {
		modules.LoginHandler(w, r, grpcConn)
	})
	mux.HandleFunc("/v1/users/logout", func(w http.ResponseWriter, r *http.Request) {
		modules.LogoutHandler(w, r, grpcConn)
	})
	mux.HandleFunc("/v1/users/refresh-token", func(w http.ResponseWriter, r *http.Request) {
		modules.RefreshTokenHandler(w, r, grpcConn)
	})
	mux.HandleFunc("/v1/media/upload", func(w http.ResponseWriter, r *http.Request) {
		modules.MediaUploadHandler(w, r, grpcConn, client)
	})
	mux.HandleFunc("/storage/", modules.StorageHandler)

	// Настройка и запуск HTTP-сервера
	httpServer := modules.SetupHTTPServer(grpcConn, mux)
	modules.StartHTTPServer(httpServer)
}
