package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"entgo.io/ent/dialect/sql/schema"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/joho/godotenv"
	"github.com/rs/cors"

	"stormlink/server/ent"
	"stormlink/server/grpc/auth"
	authpb "stormlink/server/grpc/auth/protobuf"
	"stormlink/server/grpc/media"
	mediapb "stormlink/server/grpc/media/protobuf"
	"stormlink/server/grpc/user"
	userpb "stormlink/server/grpc/user/protobuf"
	"stormlink/server/middleware"
	"stormlink/server/usecase"
	"stormlink/server/utils"

	_ "github.com/lib/pq"
)

func initEnv() {
	err := godotenv.Load("server/.env")
	if err != nil {
		log.Println("⚠️  .env файл не найден")
	}
}

// chainInterceptors объединяет несколько interceptors в один
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

func main() {
	// Путь к .env
	initEnv()

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

	// Инициализация RateLimiter: 1 запрос в секунду, burst 3
	rl := middleware.NewRateLimiter(rate.Limit(1), 3)

	// Комбинируем middleware
	chain := []grpc.UnaryServerInterceptor{
		middleware.RateLimitInterceptor(rl),
		middleware.GRPCAuthInterceptor,
	}

	// Инициализация gRPC сервера
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(chainInterceptors(chain...)),
	)

	// Инициализация UserUsecase
	userUsecase := usecase.NewUserUsecase(client)

	// Инициализация сервисов
	userService := user.NewUserService(client, userUsecase)
	userpb.RegisterUserServiceServer(grpcServer, userService)

	authService := auth.NewAuthService(client)
	authpb.RegisterAuthServiceServer(grpcServer, authService)

	s3Client, err := utils.NewS3Client()
	if err != nil {
		log.Fatalf("failed to init S3 client: %v", err)
	}
	mediaService := media.NewMediaServiceWithClient(s3Client)
	mediapb.RegisterMediaServiceServer(grpcServer, mediaService)

	// gRPC listener (на 4000)
	go func() {
		listener, err := net.Listen("tcp", ":4000")
		if err != nil {
			log.Fatalf("не удалось слушать порт 4000: %v", err)
		}
		log.Println("📡 gRPC-сервер запущен на :4000")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("ошибка при запуске gRPC-сервера: %v", err)
		}
	}()

	// Подключаемся к gRPC-серверу для кастомных хендлеров
	grpcConn, err := grpc.Dial("localhost:4000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("не удалось подключиться к gRPC-серверу: %v", err)
	}
	defer grpcConn.Close()

	// HTTP Gateway mux
	ctx := context.Background()
	gwmux := gwruntime.NewServeMux(
		gwruntime.WithErrorHandler(func(ctx context.Context, mux *gwruntime.ServeMux, marshaler gwruntime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
			statusCode := codes.Unknown
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code()
			}
			if statusCode == codes.ResourceExhausted {
				http.Error(w,
					`{"error": "rate limit exceeded, try again later"}`,
					http.StatusTooManyRequests,
				)
				return
			}
			gwruntime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
		}),
	)

	// Подключаем grpc-gateway хендлеры
	err = userpb.RegisterUserServiceHandlerFromEndpoint(ctx, gwmux, "localhost:4000", []grpc.DialOption{grpc.WithInsecure()})
	if err != nil {
		log.Fatalf("не удалось зарегистрировать grpc-gateway хендлер UserService: %v", err)
	}

	// Кастомный мультиплексор для маршрутизации
	mux := http.NewServeMux()

	// Хендлер для /v1/users/login
	mux.HandleFunc("/v1/users/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		log.Printf("📥 [HTTP] Request: POST /v1/users/login")

		var req authpb.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("❌ [Login] Invalid request body: %v", err)
			http.Error(w,
				`{"error": "invalid request body"}`,
				http.StatusBadRequest,
			)
			return
		}

		authClient := authpb.NewAuthServiceClient(grpcConn)
		resp, err := authClient.Login(r.Context(), &req)
		if err != nil {
			log.Printf("❌ [Login] gRPC error: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		// Устанавливаем куки
		utils.SetAuthCookies(w, resp.AccessToken, resp.RefreshToken)
		log.Printf("✅ [Login] Set auth cookies")

		// Отправляем JSON-ответ
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("❌ [Login] Failed to encode response: %v", err)
			http.Error(w,
				`{"error": "failed to encode response"}`,
				http.StatusInternalServerError,
			)
			return
		}

		log.Printf("📤 [HTTP] Response: Login successful")
	})

	// Хендлер для /v1/users/logout
	mux.HandleFunc("/v1/users/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		log.Printf("📥 [HTTP] Request: POST /v1/users/logout")

		// Извлекаем заголовок Authorization
		authHeader := r.Header.Get("Authorization")
		log.Printf("🔍 [Logout] HTTP Authorization header: %s", authHeader)

		// Создаем gRPC-метаданные
		md := metadata.New(map[string]string{})
		if authHeader != "" {
			md.Set("authorization", authHeader)
		}

		// Создаем контекст с метаданными
		ctx := metadata.NewOutgoingContext(r.Context(), md)

		authClient := authpb.NewAuthServiceClient(grpcConn)
		resp, err := authClient.Logout(ctx, &emptypb.Empty{})
		if err != nil {
			log.Printf("❌ [Logout] gRPC error: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}

		// Очищаем куки
		http.SetCookie(w, &http.Cookie{
			Name:     "auth_token",
			Value:    "",
			Path:     "/",
			Domain:   "localhost",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: false,
			SameSite: http.SameSiteLaxMode,
		})
		log.Printf("✅ [Logout] Cleared auth_token cookie")
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    "",
			Path:     "/",
			Domain:   "localhost",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		log.Printf("✅ [Logout] Cleared refresh_token cookie")

		// Отправляем JSON-ответ
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("❌ [Logout] Failed to encode response: %v", err)
			http.Error(w,
				`{"error": "failed to encode response"}`,
				http.StatusInternalServerError,
			)
			return
		}

		log.Printf("📤 [HTTP] Response: Successfully logged out")
	})

	// Хендлер для /v1/users/refresh-token
	mux.HandleFunc("/v1/users/refresh-token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		log.Printf("📥 [HTTP] Request: POST /v1/users/refresh-token")

		var req authpb.RefreshTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Проверяем куки, если тело пустое
			cookie, err := r.Cookie("refresh_token")
			if err == nil && cookie != nil {
				req.RefreshToken = cookie.Value
			} else {
				log.Printf("❌ [RefreshToken] Refresh token required: %v", err)
				http.Error(w,
					`{"error": "refresh token required"}`,
					http.StatusBadRequest,
				)
				return
			}
		}

		authClient := authpb.NewAuthServiceClient(grpcConn)
		resp, err := authClient.RefreshToken(r.Context(), &req)
		if err != nil {
			log.Printf("❌ [RefreshToken] gRPC error: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		// Устанавливаем куки
		utils.SetAuthCookies(w, resp.AccessToken, resp.RefreshToken)
		log.Printf("✅ [RefreshToken] Set auth cookies")

		// Отправляем JSON-ответ
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("❌ [RefreshToken] Failed to encode response: %v", err)
			http.Error(w,
				`{"error": "failed to encode response"}`,
				http.StatusInternalServerError,
			)
			return
		}

		log.Printf("📤 [HTTP] Response: Token refreshed")
	})

	// Хендлер для /v1/media/upload
	mux.HandleFunc("/v1/media/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		log.Printf("📥 [HTTP] Request: POST /v1/media/upload")

		// Парсим multipart/form-data (лимит 10MB)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			log.Printf("❌ [MediaUpload] Failed to parse form: %v", err)
			http.Error(w,
				`{"error": "failed to parse form"}`,
				http.StatusBadRequest,
			)
			return
		}

		file, handler, err := r.FormFile("file")
		if err != nil {
			log.Printf("❌ [MediaUpload] No file provided: %v", err)
			http.Error(w,
				`{"error": "file is required"}`,
				http.StatusBadRequest,
			)
			return
		}
		defer file.Close()

		fileContent, err := io.ReadAll(file)
		if err != nil {
			log.Printf("❌ [MediaUpload] Failed to read file: %v", err)
			http.Error(w,
				`{"error": "failed to read file"}`,
				http.StatusInternalServerError,
			)
			return
		}

		// Извлекаем Authorization
		authHeader := r.Header.Get("Authorization")
		log.Printf("🔍 [MediaUpload] HTTP Authorization header: %s", authHeader)

		// Создаем gRPC-метаданные
		md := metadata.New(map[string]string{})
		if authHeader != "" {
			md.Set("authorization", authHeader)
		}

		// Создаем контекст с метаданными
		ctx := metadata.NewOutgoingContext(r.Context(), md)

		mediaClient := mediapb.NewMediaServiceClient(grpcConn)
		// читаем из multipart-поля "dir"
		dir := r.FormValue("dir")
		if dir == "" {
			dir = "media"
		}
		resp, err := mediaClient.UploadMedia(ctx, &mediapb.UploadMediaRequest{
			Dir:         dir,
			Filename:    handler.Filename,
			FileContent: fileContent,
		})
		if err != nil {
			log.Printf("❌ [MediaUpload] gRPC error: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}

		log.Printf("✅ [MediaUpload] File uploaded: %s", resp.Url)

		// Отправляем JSON-ответ
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("❌ [MediaUpload] Failed to encode response: %v", err)
			http.Error(w,
				`{"error": "failed to encode response"}`,
				http.StatusInternalServerError,
			)
			return
		}

		log.Printf("📤 [HTTP] Response: File uploaded")
	})

	mux.HandleFunc("/storage/", func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/storage/")
		if key == "" {
			http.Error(w, "Bad storage path", http.StatusBadRequest)
			return
		}
		ctype, data, err := s3Client.GetFile(r.Context(), key)
		if err != nil {
			log.Printf("❌ StorageProxy GetFile(%q): %v", key, err)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", ctype)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})

	// Все остальные маршруты через gRPC-Gateway
	mux.Handle("/", gwmux)

	// Настройка CORS
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Set-Cookie"},
		AllowCredentials: true,
	}).Handler(mux)

	// HTTP сервер (на 4080)
	httpServer := &http.Server{
		Addr: ":4080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			corsHandler.ServeHTTP(w, r)
		}),
	}

	log.Println("🌐 HTTP-сервер запущен на :4080")
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("ошибка при запуске HTTP-сервера: %v", err)
	}
}
