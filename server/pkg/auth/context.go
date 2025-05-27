// package auth

// import (
// 	"context"
// 	"errors"
// )

// type ctxKey string

// const (
//     UserIDKey ctxKey = "userID"
// )

// func UserIDFromContext(ctx context.Context) (int, error) {
//     val := ctx.Value(UserIDKey)
//     if val == nil {
//         return 0, errors.New("user ID not found in context")
//     }
//     userID, ok := val.(int)
//     if !ok {
//         return 0, errors.New("invalid user ID in context, expected int")
//     }
//     return userID, nil
// }

package auth

import (
	"context"
	"errors"
)

func UserIDFromContext(ctx context.Context) (int, error) {
	val := ctx.Value("userID")
	if val == nil {
		return 0, errors.New("user ID not found in context")
	}
	userID, ok := val.(int)
	if !ok {
		return 0, errors.New("invalid user ID in context, expected int")
	}
	return userID, nil
}
