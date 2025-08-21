package middleware

import (
	"context"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"golang.org/x/time/rate"
)

// Глобальные переменные для rate limiting
var (
	grpcVisitors        = make(map[string]*GRPCRateLimiter)
	grpcAuthLoginVisitors = make(map[string]*GRPCRateLimiter) // Отдельный лимитер для Login
	grpcAuthGeneralVisitors = make(map[string]*GRPCRateLimiter) // Отдельный лимитер для других auth endpoints
	grpcMu              sync.RWMutex
	grpcAuthLoginMu     sync.RWMutex
	grpcAuthGeneralMu   sync.RWMutex
)

// init запускает очистку старых записей
func init() {
	// Очистка старых записей каждые 3 минуты для обычных gRPC запросов
	go func() {
		for {
			time.Sleep(time.Minute * 3)
			grpcMu.Lock()
			for ip, v := range grpcVisitors {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(grpcVisitors, ip)
				}
			}
			grpcMu.Unlock()
		}
	}()

	// Очистка старых записей каждые 5 минут для auth login gRPC запросов
	go func() {
		for {
			time.Sleep(time.Minute * 5)
			grpcAuthLoginMu.Lock()
			for ip, v := range grpcAuthLoginVisitors {
				if time.Since(v.lastSeen) > 5*time.Minute {
					delete(grpcAuthLoginVisitors, ip)
				}
			}
			grpcAuthLoginMu.Unlock()
		}
	}()

	// Очистка старых записей каждые 3 минуты для auth general gRPC запросов
	go func() {
		for {
			time.Sleep(time.Minute * 3)
			grpcAuthGeneralMu.Lock()
			for ip, v := range grpcAuthGeneralVisitors {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(grpcAuthGeneralVisitors, ip)
				}
			}
			grpcAuthGeneralMu.Unlock()
		}
	}()
}

// GRPCRateLimiter представляет rate limiter для каждого IP в gRPC
type GRPCRateLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// GRPCRateLimitMiddleware ограничивает количество gRPC запросов с одного IP
func GRPCRateLimitMiddleware(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Извлекаем IP из метаданных или контекста
	ip := getGRPCClientIP(ctx)
	
	grpcMu.Lock()
	if limiter, exists := grpcVisitors[ip]; exists {
		limiter.lastSeen = time.Now()
		log.Printf("🔍 gRPC Rate limiting: Existing limiter for IP %s", ip)
	} else {
		// Лимит: 10 запросов в секунду для gRPC
		grpcVisitors[ip] = &GRPCRateLimiter{
			limiter:  rate.NewLimiter(10, 20), // 10 запросов в секунду, burst 20
			lastSeen: time.Now(),
		}
		log.Printf("🔍 gRPC Rate limiting: Created new limiter for IP %s", ip)
	}
	
	if !grpcVisitors[ip].limiter.Allow() {
		grpcMu.Unlock()
		log.Printf("🚫 gRPC Rate limit exceeded for IP: %s, method: %s", ip, info.FullMethod)
		return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
	}
	grpcMu.Unlock()
	
	log.Printf("🔍 gRPC Rate limiting: Request allowed for IP %s", ip)
	return handler(ctx, req)
}

// getGRPCClientIP извлекает IP клиента из gRPC метаданных
func getGRPCClientIP(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "unknown"
	}
	
	// Проверяем заголовки прокси
	if ips := md.Get("x-forwarded-for"); len(ips) > 0 {
		return ips[0]
	}
	if ips := md.Get("x-real-ip"); len(ips) > 0 {
		return ips[0]
	}
	if ips := md.Get("cf-connecting-ip"); len(ips) > 0 {
		return ips[0]
	}
	
	// Если нет специальных заголовков, используем peer info
	if peer, ok := peer.FromContext(ctx); ok {
		if tcpAddr, ok := peer.Addr.(*net.TCPAddr); ok {
			return tcpAddr.IP.String()
		}
	}
	
	return "unknown"
}

// GRPCAuthRateLimitMiddleware специальный rate limiter для auth gRPC endpoints
func GRPCAuthRateLimitMiddleware(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Проверяем, является ли это auth endpoint
	if !strings.Contains(info.FullMethod, "/auth.AuthService/") {
		return handler(ctx, req)
	}

	ip := getGRPCClientIP(ctx)
	
	// Разные лимиты для разных типов auth endpoints
	if strings.Contains(info.FullMethod, "/auth.AuthService/Login") {
		// Строгий лимит только для Login (защита от brute force)
		log.Printf("🔍 gRPC Auth Login Rate limiting: IP %s, method %s", ip, info.FullMethod)
		
		grpcAuthLoginMu.Lock()
		if limiter, exists := grpcAuthLoginVisitors[ip]; exists {
			limiter.lastSeen = time.Now()
			log.Printf("🔍 gRPC Auth Login Rate limiting: Existing limiter for IP %s", ip)
		} else {
			// Строгий лимит для Login: 5 попыток в минуту, burst 5
			grpcAuthLoginVisitors[ip] = &GRPCRateLimiter{
				limiter: rate.NewLimiter(rate.Every(time.Minute/5), 5), // 5 попыток в минуту
				lastSeen: time.Now(),
			}
			log.Printf("🔍 gRPC Auth Login Rate limiting: Created new limiter for IP %s", ip)
		}
		
		if !grpcAuthLoginVisitors[ip].limiter.Allow() {
			grpcAuthLoginMu.Unlock()
			log.Printf("🚫 gRPC Auth Login Rate limit exceeded for IP: %s", ip)
			return nil, status.Errorf(codes.ResourceExhausted, "login rate limit exceeded")
		}
		grpcAuthLoginMu.Unlock()
		
		log.Printf("🔍 gRPC Auth Login Rate limiting: Request allowed for IP %s", ip)
	} else {
		// Более мягкий лимит для других auth endpoints (ValidateToken, GetMe)
		log.Printf("🔍 gRPC Auth General Rate limiting: IP %s, method %s", ip, info.FullMethod)
		
		grpcAuthGeneralMu.Lock()
		if limiter, exists := grpcAuthGeneralVisitors[ip]; exists {
			limiter.lastSeen = time.Now()
			log.Printf("🔍 gRPC Auth General Rate limiting: Existing limiter for IP %s", ip)
		} else {
			// Мягкий лимит для других auth endpoints: 60 запросов в минуту, burst 100
			grpcAuthGeneralVisitors[ip] = &GRPCRateLimiter{
				limiter: rate.NewLimiter(60, 100), // 60 запросов в минуту, burst 100
				lastSeen: time.Now(),
			}
			log.Printf("🔍 gRPC Auth General Rate limiting: Created new limiter for IP %s", ip)
		}
		
		if !grpcAuthGeneralVisitors[ip].limiter.Allow() {
			grpcAuthGeneralMu.Unlock()
			log.Printf("🚫 gRPC Auth General Rate limit exceeded for IP: %s", ip)
			return nil, status.Errorf(codes.ResourceExhausted, "auth rate limit exceeded")
		}
		grpcAuthGeneralMu.Unlock()
		
		log.Printf("🔍 gRPC Auth General Rate limiting: Request allowed for IP %s", ip)
	}
	
	return handler(ctx, req)
}
