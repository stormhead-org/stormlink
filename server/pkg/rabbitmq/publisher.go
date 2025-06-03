package rabbitmq

import (
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

var rabbitURL string

func init() {
	// Если RABBITMQ_URL не задана, пробуем загрузить .env из папки server
	if os.Getenv("RABBITMQ_URL") == "" {
		if err := godotenv.Load("server/.env"); err != nil {
			log.Println("🔍 .env для RabbitMQ не найден или не загружен, смотрим в реальное окружение")
		}
	}
	rabbitURL = os.Getenv("RABBITMQ_URL")
}

// EmailJob — структура задачи для подтверждения по почте.
type EmailJob struct {
    To    string `json:"to"`
    Token string `json:"token"`
}

// PublishEmailJob публикует EmailJob в очередь "email_verification".
func PublishEmailJob(job EmailJob) error {
	// Проверяем, что RABBITMQ_URL задана
	if rabbitURL == "" {
		err := errors.New("переменная окружения RABBITMQ_URL не задана")
		log.Printf("❌ RabbitMQ: %v", err)
		return err
	}

    // Простое подключение. Можно заменить на DialConfig, если нужен кастомный таймаут.
    conn, err := amqp.Dial(rabbitURL)
    if err != nil {
        log.Printf("❌ RabbitMQ: не удалось подключиться: %v", err)
        return err
    }
    defer conn.Close()

    ch, err := conn.Channel()
    if err != nil {
        log.Printf("❌ RabbitMQ: не удалось создать канал: %v", err)
        return err
    }
    defer ch.Close()

    // Объявляем очередь (если её ещё нет)
    q, err := ch.QueueDeclare(
        "email_verification", // имя очереди
        true,                 // durable
        false,                // delete when unused
        false,                // exclusive
        false,                // no-wait
        nil,                  // arguments
    )
    if err != nil {
        log.Printf("❌ RabbitMQ: не удалось объявить очередь: %v", err)
        return err
    }

    body, err := json.Marshal(job)
    if err != nil {
        return err
    }

    // Публикуем сообщение в очередь
    err = ch.Publish(
        "",     // exchange (пустая — default exchange)
        q.Name, // routing key = имя очереди
        false,  // mandatory
        false,  // immediate
        amqp.Publishing{
            DeliveryMode: amqp.Persistent,
            ContentType:  "application/json",
            Body:         body,
        },
    )
    if err != nil {
        log.Printf("❌ RabbitMQ: не удалось опубликовать сообщение: %v", err)
        return err
    }

    log.Printf("✅ RabbitMQ: задача email job опубликована: %+v", job)
    return nil
}
