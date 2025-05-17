package modules

import (
	"github.com/joho/godotenv"
	"log"
)

func InitEnv() {
	err := godotenv.Load("server/.env")
	if err != nil {
		log.Println("⚠️  .env файл не найден")
	}
}
