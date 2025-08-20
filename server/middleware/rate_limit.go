package middleware

import (
	"log"
	"net/http"
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
	// Хранилище лимитеров для каждого IP
	visitors := make(map[string]*RateLimiter)
	var mu sync.RWMutex

	// Очистка старых записей каждые 3 минуты
	go func() {
		for {
			time.Sleep(time.Minute * 3)
			mu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > 3*time.Minute {
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
			// Более строгий лимит: 5 запросов в секунду, burst до 10
			visitors[ip] = &RateLimiter{
				limiter: rate.NewLimiter(rate.Every(time.Second/5), 10),
				lastSeen: time.Now(),
			}
		}
		
		if !visitors[ip].limiter.Allow() {
			mu.Unlock()
			log.Printf("🚫 Rate limit exceeded for IP: %s", ip)
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		mu.Unlock()
		
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
				limiter: rate.NewLimiter(rate.Every(time.Minute/5), 10),
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
