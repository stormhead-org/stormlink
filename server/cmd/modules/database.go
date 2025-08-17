package modules

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"stormlink/server/ent"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/schema"
	_ "github.com/lib/pq"
)

func ConnectDB() *ent.Client {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"), os.Getenv("SSL_MODE"),
	)

	// Создаем database/sql DB для настройки пула соединений
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("не удалось открыть соединение с базой: %v", err)
	}

	// Настройка пула соединений PostgreSQL для предотвращения "too many clients"
	maxOpenConns := getEnvInt("DB_MAX_OPEN_CONNS", 15)  // По умолчанию 15 соединений
	maxIdleConns := getEnvInt("DB_MAX_IDLE_CONNS", 5)   // По умолчанию 5 idle соединений
	connMaxLifetime := getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 5) // По умолчанию 5 минут

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Minute)

	log.Printf("📊 Настройки пула БД: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%dм", 
		maxOpenConns, maxIdleConns, connMaxLifetime)

	// Создаем ent.Client с настроенным sql.DB
	drv := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(drv))

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		log.Fatalf("не удалось подключиться к базе: %v", err)
	}

	return client
}

// Вспомогательная функция для получения int из ENV с дефолтом
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
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
