package utils

import (
	"crypto/rand"
	"encoding/base64"
)

func GenerateSalt() string {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(bytes)
}
