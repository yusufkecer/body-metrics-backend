package service

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
)

type EmailService struct {
	host string
	port string
	user string
	pass string
	from string
}

func NewEmailService(host, port, user, pass, from string) *EmailService {
	return &EmailService{host: host, port: port, user: user, pass: pass, from: from}
}

func (s *EmailService) SendPasswordReset(to, token string) error {
	subject := "BodyMetrics - Şifre Sıfırlama Kodu"
	body := buildResetEmail(to, token)

	msg := []byte(
		"From: " + s.from + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"\r\n" +
			body,
	)

	addr := s.host + ":" + s.port
	auth := smtp.PlainAuth("", s.user, s.pass, s.host)

	tlsConfig := &tls.Config{
		ServerName: s.host,
	}

	conn, err := tls.Dial("tcp", s.host+":465", tlsConfig)
	if err != nil {
		// Fallback to STARTTLS on port 587
		return s.sendSTARTTLS(addr, auth, msg, to)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("failed to create smtp client: %w", err)
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth failed: %w", err)
	}
	if err := client.Mail(s.user); err != nil {
		return fmt.Errorf("smtp mail from failed: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt failed: %w", err)
	}
	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data failed: %w", err)
	}
	_, err = wc.Write(msg)
	if err != nil {
		return fmt.Errorf("smtp write failed: %w", err)
	}
	return wc.Close()
}

func (s *EmailService) sendSTARTTLS(addr string, auth smtp.Auth, msg []byte, to string) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to dial smtp: %w", err)
	}
	defer client.Close()

	tlsConfig := &tls.Config{ServerName: s.host}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("starttls failed: %w", err)
	}
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth failed: %w", err)
	}
	if err := client.Mail(s.user); err != nil {
		return fmt.Errorf("smtp mail from failed: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt failed: %w", err)
	}
	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data failed: %w", err)
	}
	_, err = wc.Write(msg)
	if err != nil {
		return fmt.Errorf("smtp write failed: %w", err)
	}
	return wc.Close()
}

func buildResetEmail(to, token string) string {
	return `<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family:Arial,sans-serif;background:#f4f4f4;padding:20px;">
  <div style="max-width:480px;margin:0 auto;background:#fff;border-radius:8px;padding:32px;">
    <h2 style="color:#333;">BodyMetrics Şifre Sıfırlama</h2>
    <p>Merhaba,</p>
    <p>Şifrenizi sıfırlamak için aşağıdaki 6 haneli doğrulama kodunu kullanın:</p>
    <div style="text-align:center;margin:24px 0;">
      <span style="font-size:36px;font-weight:bold;letter-spacing:8px;color:#6200EE;">` + token + `</span>
    </div>
    <p>Bu kod <strong>15 dakika</strong> geçerlidir.</p>
    <p>Eğer bu işlemi siz yapmadıysanız, bu e-postayı görmezden gelebilirsiniz.</p>
    <hr style="border:none;border-top:1px solid #eee;margin:24px 0;">
    <p style="color:#999;font-size:12px;">BodyMetrics Ekibi</p>
  </div>
</body>
</html>`
}
