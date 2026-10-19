package mail

import (
	"fmt"

	"gopkg.in/gomail.v2"
)

// Sender sends transactional email via Gmail SMTP using an app password.
type Sender struct {
	email       string
	appPassword string
}

func NewSender(email, appPassword string) *Sender {
	return &Sender{email: email, appPassword: appPassword}
}

func (s *Sender) SendPasswordResetCode(to, code string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.email)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Your open-location password reset code")
	m.SetBody("text/html", fmt.Sprintf(`
        <p>A password reset was requested for this account.</p>
        <p>If this wasn't you, you can ignore this email.</p>
        <p>Your reset code is:</p>
        <h2 style="letter-spacing: 4px;">%s</h2>
        <p>The code is valid for 15 minutes.</p>
    `, code))

	d := gomail.NewDialer("smtp.gmail.com", 587, s.email, s.appPassword)
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}
