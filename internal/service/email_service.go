package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type EmailService struct {
	apiKey string
	from   string
}

func NewEmailService(apiKey, from string) *EmailService {
	return &EmailService{apiKey: apiKey, from: from}
}

func (s *EmailService) SendPasswordReset(to, token string) error {
	payload := map[string]interface{}{
		"from":    s.from,
		"to":      []string{to},
		"subject": "BodyMetrics - Şifre Sıfırlama Kodu",
		"html":    buildResetEmail(to, token),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("resend http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend api error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
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
