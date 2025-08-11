package mail

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
)

func SendVerifyEmail(to, token string) error {
    config := NewEmailConfig()
    publicURL := os.Getenv("APP_PUBLIC_URL")
    if publicURL == "" { publicURL = "http://localhost:3000" }
    verificationLink := fmt.Sprintf("%s/verify-email?token=%s", publicURL, token)

    subject := "Subject: Подтверждение почты\r\n"
    mime := "MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"
    body := fmt.Sprintf(`
        <h2>Подтверждение почты</h2>
        <p>Подтвердите свою почту, перейдя по ссылке:</p>
        <a href="%s">Подтвердить почту</a>
        <p>Эта ссылка будет доступна следующие 24 часа.</p>
    `, verificationLink)
    message := []byte(subject + mime + body)

    addr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)
    auth := smtp.PlainAuth("", config.SMTPUsername, config.SMTPPassword, config.SMTPHost)
    if err := smtp.SendMail(addr, auth, config.FromEmail, []string{to}, message); err != nil {
        log.Printf("Ошибка отправки письма на %s: %v", to, err)
        return err
    }
    return nil
}


