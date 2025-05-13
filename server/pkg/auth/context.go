package auth

import (
	"context"
	"errors"
)

func UserIDFromContext(ctx context.Context) (string, error) {
	val := ctx.Value("userID")
	if val == nil {
		return "", errors.New("user ID not found in context")
	}

	userID, ok := val.(string)
	if !ok {
		return "", errors.New("invalid user ID in context")
	}

	return userID, nil
}
