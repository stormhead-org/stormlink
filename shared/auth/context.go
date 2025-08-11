package auth

import (
	"context"
	"errors"
)

// typedKey исключает коллизии ключей в контексте
type typedKey struct{ name string }

var (
    userIDKey = typedKey{name: "userID"}
)

// WithUserID добавляет userID в контекст типобезопасно
func WithUserID(ctx context.Context, id int) context.Context {
    ctx = context.WithValue(ctx, userIDKey, id)
    // Совместимость со старым кодом (строковый ключ)
    ctx = context.WithValue(ctx, "userID", id)
    return ctx
}

// UserIDFromContext достает userID из контекста (поддерживает typed и строковый ключ)
func UserIDFromContext(ctx context.Context) (int, error) {
    if val := ctx.Value(userIDKey); val != nil {
        if id, ok := val.(int); ok {
            return id, nil
        }
    }
    if val := ctx.Value("userID"); val != nil {
        if id, ok := val.(int); ok {
            return id, nil
        }
    }
    return 0, errors.New("user ID not found in context")
}


