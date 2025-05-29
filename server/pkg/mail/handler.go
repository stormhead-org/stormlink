package mail

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
)

func SendVerifyEmail(to, token string) error {
	config := NewEmailConfig()
	appDomain := os.Getenv("APP_DOMAIN")
	verificationLink := fmt.Sprintf("%s/verify-email?token=%s", appDomain, token)

	// Формируем тело письма
	subject := "Subject: Подтверждение почты\r\n"
	mime := "MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"
	body := fmt.Sprintf(`
        <h2>Подтверждение почты</h2>
        <p>Подтвердите свою почту, перейдя по ссылке:</p>
        <a href="%s">Подтвердить почту</a>
        <p>Эта ссылка будет доступна следующие 24 часа.</p>
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
