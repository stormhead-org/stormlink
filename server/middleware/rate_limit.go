package middleware

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"log"
	"strings"
	"sync"
)

// Метрики Prometheus
var (
	rateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
		[]string{"client_id", "method"},
	)
	rateLimitPasses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_rate_limit_passes_total",
			Help: "Total number of rate limit passes",
		},
		[]string{"client_id", "method"},
	)
)

// RateLimiter хранит лимитеры для каждого клиента
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.Mutex
	clientMu map[string]*sync.Mutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter создает новый RateLimiter
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		clientMu: make(map[string]*sync.Mutex),
		rate:     r,
		burst:    b,
	}
}

// getLimiter возвращает лимитер и мьютекс для клиента или создает новые
func (rl *RateLimiter) getLimiter(clientID string) (*rate.Limiter, *sync.Mutex) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[clientID]
	clientMu, muExists := rl.clientMu[clientID]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[clientID] = limiter
	}
	if !muExists {
		clientMu = &sync.Mutex{}
		rl.clientMu[clientID] = clientMu
	}
	return limiter, clientMu
}

// RateLimitInterceptor создает gRPC middleware для ограничения запросов
func RateLimitInterceptor(rl *RateLimiter) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		publicMethods := map[string]bool{
			"/auth.AuthService/Login":   true,
			"/UserService/RegisterUser": true,
		}

		if !publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		clientID := getClientID(ctx)
		if clientID == "" {
			log.Println("⚠️ Не удалось определить client ID, пропускаем rate limiting")
			return handler(ctx, req)
		}

		limiter, clientMu := rl.getLimiter(clientID)

		clientMu.Lock()
		defer clientMu.Unlock()

		reservation := limiter.Reserve()
		if !reservation.OK() {
			rateLimitHits.WithLabelValues(clientID, info.FullMethod).Inc()
			log.Printf("🚫 Rate limit exceeded for client %s on method %s (no tokens available)", clientID, info.FullMethod)
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
		}

		delay := reservation.Delay()
		if delay > 0 {
			rateLimitHits.WithLabelValues(clientID, info.FullMethod).Inc()
			log.Printf("🚫 Rate limit exceeded for client %s on method %s (delay required: %v)", clientID, info.FullMethod, delay)
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
		}

		rateLimitPasses.WithLabelValues(clientID, info.FullMethod).Inc()
		log.Printf("✅ Rate limit check passed for client %s on method %s", clientID, info.FullMethod)
		return handler(ctx, req)
	}
}

// getClientID извлекает идентификатор клиента (например, IP) из контекста
func getClientID(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get("x-forwarded-for"); len(values) > 0 {
			return values[0]
		}
	}
	if p, ok := peer.FromContext(ctx); ok {
		addr := p.Addr.String()
		if addr == "::1" || strings.HasPrefix(addr, "[::1]:") {
			return "127.0.0.1"
		}
		if idx := strings.LastIndex(addr, ":"); idx != -1 {
			return addr[:idx]
		}
		return addr
	}
	return ""
}
