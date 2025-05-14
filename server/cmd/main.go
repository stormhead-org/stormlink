package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/time/rate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"net"
	"net/http"
	"os"
	"stormlink/server/grpc/auth"
	"stormlink/server/grpc/user"
	"stormlink/server/middleware"
	"stormlink/server/usecase"
	"stormlink/server/utils"
	"time"

	"entgo.io/ent/dialect/sql/schema"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"google.golang.org/grpc"
	"stormlink/server/ent"
	authpb "stormlink/server/grpc/auth/protobuf"
	userpb "stormlink/server/grpc/user/protobuf"

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
		// Если нет interceptors, просто вызываем handler
		if len(interceptors) == 0 {
			return handler(ctx, req)
		}

		// Создаем цепочку, начиная с последнего interceptor
		var chainHandler grpc.UnaryHandler = handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			current := interceptors[i]
			// Формируем новый handler, который вызывает текущий interceptor
			chainHandler = func(currentCtx context.Context, currentReq interface{}, currentInfo *grpc.UnaryServerInfo, next grpc.UnaryHandler) grpc.UnaryHandler {
				return func(ctx context.Context, req interface{}) (interface{}, error) {
					return current(ctx, req, currentInfo, next)
				}
			}(ctx, req, info, chainHandler)
		}

		// Вызываем первый handler в цепочке
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
				http.Error(w, `{"error": "rate limit exceeded, try again later"}`, http.StatusTooManyRequests)
				return
			}
			gwruntime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
		}),
	)

	// Подключаем grpc-gateway хендлеры (кроме login и refresh-token)
	err = userpb.RegisterUserServiceHandlerFromEndpoint(ctx, gwmux, "localhost:4000", []grpc.DialOption{grpc.WithInsecure()})
	if err != nil {
		log.Fatalf("не удалось зарегистрировать grpc-gateway хендлер UserService: %v", err)
	}

	// Кастомный мультиплексор для маршрутизации
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/users/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req authpb.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
			return
		}

		authClient := authpb.NewAuthServiceClient(grpcConn)
		resp, err := authClient.Login(r.Context(), &req)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%v"}`, err), http.StatusUnauthorized)
			return
		}

		// Устанавливаем куки
		utils.SetAuthCookies(w, resp.AccessToken, resp.RefreshToken)

		// Отправляем JSON-ответ
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/v1/users/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Извлекаем заголовок Authorization
		authHeader := r.Header.Get("Authorization")

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
			http.Error(w, fmt.Sprintf(`{"error": "%v"}`, err), http.StatusInternalServerError)
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

		// Отправляем JSON-ответ
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
			return
		}

	})

	mux.HandleFunc("/v1/users/refresh-token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req authpb.RefreshTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Проверяем куки, если тело пустое
			cookie, err := r.Cookie("refresh_token")
			if err == nil && cookie != nil {
				req.RefreshToken = cookie.Value
			} else {
				http.Error(w, `{"error": "refresh token required"}`, http.StatusBadRequest)
				return
			}
		}

		authClient := authpb.NewAuthServiceClient(grpcConn)
		resp, err := authClient.RefreshToken(r.Context(), &req)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%v"}`, err), http.StatusUnauthorized)
			return
		}

		// Устанавливаем куки
		utils.SetAuthCookies(w, resp.AccessToken, resp.RefreshToken)

		// Отправляем JSON-ответ
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
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
