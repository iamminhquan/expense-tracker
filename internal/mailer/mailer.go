// Package mailer sends the transactional email the app needs (currently
// just the password-reset link) over plain SMTP, with no dependency beyond
// the standard library.
package mailer

import (
	"fmt"
	"net/smtp"
)

// Config holds the SMTP relay settings. Every field is optional, the same
// way the rest of internal/config treats non-critical settings: a deploy
// with no SMTP configured still starts, it just can't send mail (Send
// returns an error, which callers log rather than fail the request over).
type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// Mailer sends email through the relay it was constructed with.
type Mailer struct {
	cfg Config
}

func New(cfg Config) *Mailer {
	return &Mailer{cfg: cfg}
}

// Configured reports whether enough settings are present to attempt a send.
// Callers use this to log a clear "not configured" message instead of
// letting Send fail with an opaque dial error against an empty host.
func (m *Mailer) Configured() bool {
	return m.cfg.Host != "" && m.cfg.From != ""
}

// Send delivers a plain-text email. Auth is skipped when no username is
// configured; net/smtp.SendMail negotiates STARTTLS with the relay itself
// when the relay offers it, which covers the common port-587 providers
// (Gmail, Brevo, Mailgun, ...) without any TLS handling of our own.
func (m *Mailer) Send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%s", m.cfg.Host, m.cfg.Port)

	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		m.cfg.From, to, subject, body)

	return smtp.SendMail(addr, auth, m.cfg.From, []string{to}, []byte(msg))
}
