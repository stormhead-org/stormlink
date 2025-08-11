package mail

import (
	"context"
	"encoding/json"
	"log"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"

	sharedmail "stormlink/shared/mail"

	"github.com/joho/godotenv"
)

type EmailJob struct {
    To    string `json:"to"`
    Token string `json:"token"`
}

// Run запускает потребителя очереди email_verification и обрабатывает сообщения до остановки контекста
func Run(ctx context.Context) error {
    // Подтягиваем .env из server/, если не подхватились переменные
    _ = godotenv.Load("server/.env")

    rabbitURL := os.Getenv("RABBITMQ_URL")
    if rabbitURL == "" {
        return Err("RABBITMQ_URL is empty")
    }

    conn, err := amqp.Dial(rabbitURL)
    if err != nil { return err }
    defer conn.Close()

    ch, err := conn.Channel()
    if err != nil { return err }
    defer ch.Close()

    q, err := ch.QueueDeclare(
        "email_verification",
        true, false, false, false, nil,
    )
    if err != nil { return err }

    msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
    if err != nil { return err }

    log.Println("📬 Mail worker: waiting messages...")

    for {
        select {
        case <-ctx.Done():
            return nil
        case d, ok := <-msgs:
            if !ok { return nil }
            var job EmailJob
            if err := json.Unmarshal(d.Body, &job); err != nil {
                log.Printf("❌ invalid job: %v", err)
                _ = d.Nack(false, false)
                continue
            }
            if err := sharedmail.SendVerifyEmail(job.To, job.Token); err != nil {
                log.Printf("❌ send email failed: %v", err)
                _ = d.Nack(false, true)
                continue
            }
            _ = d.Ack(false)
        }
    }
}

type stringError string
func (e stringError) Error() string { return string(e) }
func Err(msg string) error { return stringError(msg) }


