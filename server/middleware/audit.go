package middleware

import (
	"log"
	"net/http"
	"time"

	sharedauth "stormlink/shared/auth"
)

// AuditMiddleware логирует действия пользователей для аудита
func AuditMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Создаем response writer wrapper для захвата статуса
		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		// Выполняем запрос
		next.ServeHTTP(wrappedWriter, r)
		
		// Логируем действие
		duration := time.Since(start)
		userID, err := sharedauth.UserIDFromContext(r.Context())
		if err != nil {
			userID = 0 // Анонимный пользователь
		}
		
		// Определяем тип действия
		actionType := getActionType(r)
		
		// Логируем только важные действия
		if shouldLogAction(actionType, wrappedWriter.statusCode) {
			log.Printf("AUDIT: User %d | %s %s | Status: %d | Duration: %v | IP: %s | User-Agent: %s",
				userID,
				r.Method,
				r.URL.Path,
				wrappedWriter.statusCode,
				duration,
				getClientIP(r),
				r.UserAgent(),
			)
		}
	})
}

// responseWriter wrapper для захвата статуса ответа
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// getActionType определяет тип действия на основе пути и метода
func getActionType(r *http.Request) string {
	path := r.URL.Path
	method := r.Method
	
	switch {
	case path == "/query" && method == "POST":
		return "graphql_query"
	case path == "/query" && method == "POST" && containsMutation(r):
		return "graphql_mutation"
	case path == "/storage" && method == "POST":
		return "file_upload"
	default:
		return "http_request"
	}
}

// containsMutation проверяет, содержит ли запрос мутации
func containsMutation(r *http.Request) bool {
	// В реальной реализации нужно парсить GraphQL запрос
	// Здесь упрощенная проверка
	return true // Все POST запросы к /query считаем потенциальными мутациями
}

// shouldLogAction определяет, нужно ли логировать действие
func shouldLogAction(actionType string, statusCode int) bool {
	// Логируем все мутации
	if actionType == "graphql_mutation" {
		return true
	}
	
	// Логируем загрузки файлов
	if actionType == "file_upload" {
		return true
	}
	
	// Логируем ошибки
	if statusCode >= 400 {
		return true
	}
	
	// Логируем медленные запросы (>1 секунды)
	// Это будет проверяться в основном middleware
	
	return false
}

// SecurityAuditMiddleware специальный middleware для логирования событий безопасности
func SecurityAuditMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := sharedauth.UserIDFromContext(r.Context())
		if err != nil {
			userID = 0
		}
		
		// Логируем попытки доступа к чувствительным эндпоинтам
		if isSensitiveEndpoint(r.URL.Path) {
			log.Printf("SECURITY: User %d accessed sensitive endpoint %s | IP: %s | User-Agent: %s",
				userID,
				r.URL.Path,
				getClientIP(r),
				r.UserAgent(),
			)
		}
		
		next.ServeHTTP(w, r)
	})
}

// isSensitiveEndpoint проверяет, является ли эндпоинт чувствительным
func isSensitiveEndpoint(path string) bool {
	sensitivePaths := []string{
		"/storage",           // Загрузка файлов
		"/query",             // GraphQL запросы
	}
	
	for _, sensitivePath := range sensitivePaths {
		if path == sensitivePath {
			return true
		}
	}
	
	return false
}
