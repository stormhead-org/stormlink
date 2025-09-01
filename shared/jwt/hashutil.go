package jwt

import (
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

func ComparePassword(hashBase64 string, password, salt string) error {
	hash, err := base64.StdEncoding.DecodeString(hashBase64)
	if err != nil {
		return err
	}
	return bcrypt.CompareHashAndPassword(hash, []byte(password+salt))
}

func HashPassword(password, salt string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password+salt), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(hash), nil
}
