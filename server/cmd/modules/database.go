package modules

import (
	"context"
	"fmt"
	"log"
	"os"

	"entgo.io/ent/dialect/sql/schema"
	_ "github.com/lib/pq"
	"stormlink/server/ent"
)

func ConnectDB() *ent.Client {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"), os.Getenv("SSL_MODE"),
	)
	client, err := ent.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("не удалось подключиться к базе: %v", err)
	}
	return client
}

func MigrateDB(client *ent.Client, reset bool, seed bool) {
	if reset {
		log.Println("⚠️  Полный сброс базы данных с удалением колонок и индексов...")
		if err := client.Schema.Create(
			context.Background(),
			schema.WithDropIndex(true),
			schema.WithDropColumn(true),
		); err != nil {
			log.Fatalf("ошибка сброса схемы: %v", err)
		}
		log.Println("✅ Сброс базы завершён.")
	}
	if seed {
		log.Println("ℹ️  Обычная миграция схемы...")
		if err := client.Schema.Create(context.Background()); err != nil {
			log.Fatalf("ошибка миграции схемы: %v", err)
		}
		log.Println("✅ Миграция завершена.")
		log.Println("🌱 Выполняется сидинг...")
		if err := Seed(client); err != nil {
			log.Fatalf("❌ Ошибка сидинга: %v", err)
		}
		log.Println("✅ Сидинг завершён.")
	} else {
		log.Println("ℹ️  Обычная миграция схемы...")
		if err := client.Schema.Create(context.Background()); err != nil {
			log.Fatalf("ошибка миграции схемы: %v", err)
		}
		log.Println("✅ Миграция завершена.")
	}
}
