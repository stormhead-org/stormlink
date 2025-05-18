package utils

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strconv"
)

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
}

func NewEmailConfig() *EmailConfig {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	return &EmailConfig{
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     port,
		SMTPUsername: os.Getenv("SMTP_USERNAME"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		FromEmail:    os.Getenv("SMTP_USERNAME"),
	}
}

func SendVerificationEmail(to, token string) error {
	config := NewEmailConfig()
	appDomain := os.Getenv("APP_DOMAIN")
	verificationLink := fmt.Sprintf("%s/verify-email?token=%s", appDomain, token)

	// Формируем тело письма
	subject := "Subject: Verify Your Email Address\r\n"
	mime := "MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"
	body := fmt.Sprintf(`
        <h2>Email Verification</h2>
        <p>Please verify your email by clicking the link below:</p>
        <a href="%s">Verify Email</a>
        <p>This link will expire in 24 hours.</p>
    `, verificationLink)
	message := []byte(subject + mime + body)

	// Настройка SMTP
	addr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)
	auth := smtp.PlainAuth("", config.SMTPUsername, config.SMTPPassword, config.SMTPHost)

	// Отправка письма
	err := smtp.SendMail(addr, auth, config.FromEmail, []string{to}, message)
	if err != nil {
		log.Printf("Ошибка отправки письма на %s: %v", to, err)
		return err
	}
	return nil
}
