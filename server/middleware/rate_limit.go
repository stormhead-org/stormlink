package middleware

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter представляет rate limiter для каждого IP
type RateLimiter struct {
	limiter *rate.Limiter
	lastSeen time.Time
}

// RateLimitMiddleware ограничивает количество запросов с одного IP
func RateLimitMiddleware(next http.Handler) http.Handler {
	// Хранилище лимитеров для каждого IP (неавторизованные пользователи)
	anonymousVisitors := make(map[string]*RateLimiter)
	// Хранилище лимитеров для каждого IP (авторизованные пользователи)
	authenticatedVisitors := make(map[string]*RateLimiter)
	var mu sync.RWMutex

	// Очистка старых записей каждые 3 минуты
	go func() {
		for {
			time.Sleep(time.Minute * 3)
			mu.Lock()
			for ip, v := range anonymousVisitors {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(anonymousVisitors, ip)
				}
			}
			for ip, v := range authenticatedVisitors {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(authenticatedVisitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		log.Printf("🔍 Rate limiting: IP %s, method %s", ip, r.Method)
		
		// Проверяем, есть ли авторизация
		isAuthenticated := false
		if authHeader := r.Header.Get("Authorization"); authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			isAuthenticated = true
		} else if _, err := r.Cookie("auth_token"); err == nil {
			isAuthenticated = true
		}
		
		mu.Lock()
		var limiter *RateLimiter
		var exists bool
		
		if isAuthenticated {
			// Для авторизованных пользователей: более мягкий лимит
			if limiter, exists = authenticatedVisitors[ip]; exists {
				limiter.lastSeen = time.Now()
				log.Printf("🔍 Rate limiting: Existing authenticated limiter for IP %s", ip)
			} else {
				// 20 запросов в секунду для авторизованных пользователей
				limiter = &RateLimiter{
					limiter: rate.NewLimiter(20, 50), // 20 запросов в секунду, burst 50
					lastSeen: time.Now(),
				}
				authenticatedVisitors[ip] = limiter
				log.Printf("🔍 Rate limiting: Created new authenticated limiter for IP %s", ip)
			}
		} else {
			// Для неавторизованных пользователей: строгий лимит
			if limiter, exists = anonymousVisitors[ip]; exists {
				limiter.lastSeen = time.Now()
				log.Printf("🔍 Rate limiting: Existing anonymous limiter for IP %s", ip)
			} else {
				// 5 запросов в секунду для неавторизованных пользователей
				limiter = &RateLimiter{
					limiter: rate.NewLimiter(5, 10), // 5 запросов в секунду, burst 10
					lastSeen: time.Now(),
				}
				anonymousVisitors[ip] = limiter
				log.Printf("🔍 Rate limiting: Created new anonymous limiter for IP %s", ip)
			}
		}
		
		if !limiter.limiter.Allow() {
			mu.Unlock()
			userType := "authenticated"
			if !isAuthenticated {
				userType = "anonymous"
			}
			log.Printf("🚫 Rate limit exceeded for %s IP: %s", userType, ip)
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		mu.Unlock()
		
		userType := "authenticated"
		if !isAuthenticated {
			userType = "anonymous"
		}
		log.Printf("🔍 Rate limiting: Request allowed for %s IP %s", userType, ip)
		next.ServeHTTP(w, r)
	})
}

// getClientIP извлекает реальный IP клиента
func getClientIP(r *http.Request) string {
	// Проверяем заголовки прокси
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	
	return r.RemoteAddr
}

// AuthRateLimitMiddleware специальный rate limiter для auth endpoints
func AuthRateLimitMiddleware(next http.Handler) http.Handler {
	// Более строгие лимиты для auth endpoints
	visitors := make(map[string]*RateLimiter)
	var mu sync.RWMutex

	go func() {
		for {
			time.Sleep(time.Minute * 5)
			mu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > 5*time.Minute {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		
		mu.Lock()
		if limiter, exists := visitors[ip]; exists {
			limiter.lastSeen = time.Now()
		} else {
			// 5 запросов в минуту для auth endpoints
			visitors[ip] = &RateLimiter{
				limiter: rate.NewLimiter(rate.Every(time.Minute/5), 5), // 5 запросов в минуту, burst 5
				lastSeen: time.Now(),
			}
		}
		
		if !visitors[ip].limiter.Allow() {
			mu.Unlock()
			http.Error(w, "Too many authentication attempts", http.StatusTooManyRequests)
			return
		}
		mu.Unlock()
		
		next.ServeHTTP(w, r)
	})
}
