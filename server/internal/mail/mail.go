package mail

import (
	"fmt"
	"log"

	"gopkg.in/gomail.v2"
)

// Sender sends transactional email over SMTP (Gmail by default) using an
// app password.
type Sender struct {
	host        string
	port        int
	email       string
	appPassword string
	// logCodes prints codes to the server log instead of failing when SMTP
	// isn't configured, so local dev works without a mail account.
	logCodes bool
}

func NewSender(host string, port int, email, appPassword string, logCodes bool) *Sender {
	return &Sender{host: host, port: port, email: email, appPassword: appPassword, logCodes: logCodes}
}

func (s *Sender) SendLoginCode(to, code string) error {
	if s.email == "" || s.appPassword == "" {
		if s.logCodes {
			log.Printf("login code for %s: %s (SMTP not configured)", to, code)
			return nil
		}
		return fmt.Errorf("smtp not configured")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", m.FormatAddress(s.email, "Super Atlas"))
	m.SetHeader("To", to)
	m.SetHeader("Subject", fmt.Sprintf("%s is your Super Atlas code", code))
	m.SetBody("text/html", fmt.Sprintf(`
        <p>Your Super Atlas sign-in code is:</p>
        <h2 style="letter-spacing: 4px;">%s</h2>
        <p>It's valid for 10 minutes. If you didn't ask for it, you can ignore this email.</p>
    `, code))

	d := gomail.NewDialer(s.host, s.port, s.email, s.appPassword)
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}
