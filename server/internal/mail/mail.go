package mail

import (
	"fmt"
	"html"
	"log"

	"github.com/canavan-a/super-atlas/server/internal/auth"
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

// SendSignupAlert tells the server's runner that user just made an account,
// and how many of the cap's places are now taken. Without SMTP it only logs.
func (s *Sender) SendSignupAlert(to string, user *auth.User, count, cap int) error {
	if s.email == "" || s.appPassword == "" {
		log.Printf("new account: %s <%s>, %d of %d (SMTP not configured)", user.DisplayName, user.Email, count, cap)
		return nil
	}

	m := gomail.NewMessage()
	m.SetHeader("From", m.FormatAddress(s.email, "Super Atlas"))
	m.SetHeader("To", to)
	m.SetHeader("Subject", fmt.Sprintf("%s joined Super Atlas (%d/%d)", user.DisplayName, count, cap))
	m.SetBody("text/html", fmt.Sprintf(`
        <p><b>%s</b> (%s) just made a Super Atlas account.</p>
        <p>%d of %d places are taken. Raise the cap with <code>superatlas-admin cap &lt;n&gt;</code>.</p>
    `, html.EscapeString(user.DisplayName), html.EscapeString(user.Email), count, cap))

	d := gomail.NewDialer(s.host, s.port, s.email, s.appPassword)
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}
