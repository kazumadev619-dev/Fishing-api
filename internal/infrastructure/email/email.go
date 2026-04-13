package email

import (
	"fmt"

	"github.com/resend/resend-go/v2"
)

type EmailClient struct {
	client      *resend.Client
	fromAddress string
}

func NewEmailClient(apiKey, fromAddress string) *EmailClient {
	return &EmailClient{
		client:      resend.NewClient(apiKey),
		fromAddress: fromAddress,
	}
}

func (e *EmailClient) SendVerificationEmail(toEmail, token, appBaseURL string) error {
	verifyURL := fmt.Sprintf("%s/api/auth/verify-email?token=%s", appBaseURL, token)

	params := &resend.SendEmailRequest{
		From:    e.fromAddress,
		To:      []string{toEmail},
		Subject: "【釣りコンディションApp】メールアドレスの確認",
		Html: fmt.Sprintf(`
			<h2>メールアドレスの確認</h2>
			<p>以下のリンクをクリックしてメールアドレスを確認してください。</p>
			<p>このリンクは1時間有効です。</p>
			<a href="%s" style="background:#0066cc;color:white;padding:12px 24px;text-decoration:none;border-radius:4px;">
				メールアドレスを確認する
			</a>
			<p>リンクが機能しない場合は以下のURLをブラウザに貼り付けてください：</p>
			<p>%s</p>
		`, verifyURL, verifyURL),
	}

	_, err := e.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("sending verification email to %s: %w", toEmail, err)
	}
	return nil
}
