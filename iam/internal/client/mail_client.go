package main

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

type MailClient struct {
	smtpHost     string
	smtpPort     string
	smtpUser     string
	smtpPassword string
}

func NewMailClient() *MailClient {
	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "127.0.0.1"
	}

	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "587"
	}

	smtpUser := os.Getenv("SMTP_USER")
	if smtpUser == "" {
		smtpUser = "user"
	}

	smtpPassword := os.Getenv("SMTP_PASSWORD")
	if smtpPassword == "" {
		smtpPassword = "password"
	}

	return &MailClient{
		smtpHost:     smtpHost,
		smtpPort:     smtpPort,
		smtpUser:     smtpUser,
		smtpPassword: smtpPassword,
	}
}

func (this *MailClient) SendHTML(from string, to string, subject string, message string) error {
	headers := []string{
		fmt.Sprintf("Subject: %s\r\n", subject),
		"MIME-version: 1.0;\r\n",
		"Content-Type: text/html; charset=\"UTF-8\";\r\n\r\n",
	}

	return smtp.SendMail(
		fmt.Sprintf("%s:%s", this.smtpHost, this.smtpPort),
		smtp.PlainAuth(
			"",
			this.smtpUser,
			this.smtpPassword,
			this.smtpHost,
		),
		from,
		[]string{
			to,
		},
		[]byte(
			strings.Join(headers, "")+message,
		),
	)
}
