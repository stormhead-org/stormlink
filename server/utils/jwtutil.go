package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("your_secret_key_here")

func GenerateAccessToken(userID int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func GenerateRefreshToken(userID int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ParseToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}

func ParseAccessToken(tokenString string) (*AccessTokenClaims, error) {
	claims, err := ParseToken(tokenString)
	if err != nil {
		return nil, err
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return nil, errors.New("user_id not found or invalid")
	}

	return &AccessTokenClaims{
		UserID: int(userIDFloat),
	}, nil
}

type AccessTokenClaims struct {
	UserID int
}

