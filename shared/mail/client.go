package mail

import (
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


