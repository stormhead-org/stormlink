package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"

	"stormlink/server/pkg/mail"
)

var rabbitURL string

type EmailJob struct {
    To    string `json:"to"`
    Token string `json:"token"`
}

func main() {
	if err := godotenv.Load("server/.env"); err != nil {
		log.Println("🔍 .env файл не найден в server/, смотрим в реальное окружение")
	}

    rabbitURL = os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Fatalf("❌ RabbitMQ: переменная окружения RABBITMQ_URL не задана")
	}

    // Подключаемся к RabbitMQ
    conn, err := amqp.Dial(rabbitURL)
    if err != nil {
        log.Fatalf("❌ RabbitMQ: не удалось подключиться: %v", err)
    }
    defer conn.Close()

    ch, err := conn.Channel()
    if err != nil {
        log.Fatalf("❌ RabbitMQ: не удалось создать канал: %v", err)
    }
    defer ch.Close()

    // Объявляем очередь (должна совпадать с тем, что у нас в publisher)
    q, err := ch.QueueDeclare(
        "email_verification",
        true,  // durable
        false, // delete when unused
        false, // exclusive
        false, // no-wait
        nil,   // args
    )
    if err != nil {
        log.Fatalf("❌ RabbitMQ: не удалось объявить очередь: %v", err)
    }

    msgs, err := ch.Consume(
        q.Name, // queue
        "",     // consumer
        false,  // auto-ack (мы сами подтверждаем после успешной обработки)
        false,  // exclusive
        false,  // no-local
        false,  // no-wait
        nil,    // args
    )
    if err != nil {
        log.Fatalf("❌ RabbitMQ: не удалось начать потребление: %v", err)
    }

    // Канал для graceful shutdown
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    log.Println("📬 Email worker запущен, ожидаем сообщений...")

    forever := make(chan struct{})

    go func() {
        for {
            select {
            case <-ctx.Done():
                log.Println("🛑 Получен сигнал завершения, останавливаем воркер")
                close(forever)
                return
            case d, ok := <-msgs:
                if !ok {
                    log.Println("🔒 Канал сообщений закрыт, выходим")
                    close(forever)
                    return
                }

                var job EmailJob
                if err := json.Unmarshal(d.Body, &job); err != nil {
                    log.Printf("❌ RabbitMQ: неверный формат задачи: %v", err)
                    d.Nack(false, false) // отбрасываем
                    continue
                }

                // Вызываем реальное отправление письма
                if err := mail.SendVerifyEmail(job.To, job.Token); err != nil {
                    log.Printf("❌ Ошибка отправки письма на %s: %v", job.To, err)
                    d.Nack(false, true) // можно попробовать позже
                    continue
                }

                log.Printf("✅ Письмо с токеном %s отправлено на %s", job.Token, job.To)
                d.Ack(false)
            }
        }
    }()

    <-forever
    log.Println("👋 Email worker завершён")
}
