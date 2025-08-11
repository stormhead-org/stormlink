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
    if os.Getenv("RABBITMQ_URL") == "" {
        if err := godotenv.Load("server/.env"); err != nil {
            log.Println("🔍 .env для RabbitMQ не найден или не загружен, смотрим в реальное окружение")
        }
    }
    rabbitURL = os.Getenv("RABBITMQ_URL")
}

type EmailJob struct {
    To    string `json:"to"`
    Token string `json:"token"`
}

func PublishEmailJob(job EmailJob) error {
    if rabbitURL == "" {
        err := errors.New("переменная окружения RABBITMQ_URL не задана")
        log.Printf("❌ RabbitMQ: %v", err)
        return err
    }
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

    q, err := ch.QueueDeclare(
        "email_verification",
        true,
        false,
        false,
        false,
        nil,
    )
    if err != nil {
        log.Printf("❌ RabbitMQ: не удалось объявить очередь: %v", err)
        return err
    }

    body, err := json.Marshal(job)
    if err != nil {
        return err
    }

    err = ch.Publish(
        "",
        q.Name,
        false,
        false,
        amqp.Publishing{DeliveryMode: amqp.Persistent, ContentType: "application/json", Body: body},
    )
    if err != nil {
        log.Printf("❌ RabbitMQ: не удалось опубликовать сообщение: %v", err)
        return err
    }
    log.Printf("✅ RabbitMQ: задача email job опубликована: %+v", job)
    return nil
}


