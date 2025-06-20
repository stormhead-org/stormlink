package jwt

import (
	"errors"
	"os"
	"time"

	"strconv"

	"github.com/golang-jwt/jwt/v5"
)

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		panic("JWT_SECRET not set in environment")
	}
	return []byte(secret)
}

func GenerateAccessToken(userID int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": strconv.Itoa(userID),
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
		"type":    "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

func GenerateRefreshToken(userID int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": strconv.Itoa(userID),
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"type":    "refresh",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

func ParseToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return getJWTSecret(), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, errors.New("exp not found or invalid")
	}
	if int64(exp) < time.Now().Unix() {
		return nil, errors.New("token has expired")
	}

	return claims, nil
}

func ParseAccessToken(tokenString string) (*AccessTokenClaims, error) {
	claims, err := ParseToken(tokenString)
	if err != nil {
		return nil, err
	}

	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "access" {
		return nil, errors.New("invalid token type")
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return nil, errors.New("user_id not found or invalid")
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return nil, errors.New("invalid user_id format")
	}

	return &AccessTokenClaims{
		UserID: userID,
	}, nil
}

type AccessTokenClaims struct {
	UserID int
}
