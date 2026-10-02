// Package mailer sends plain-text email over SMTP, or logs it when no SMTP
// server is configured (development).
package mailer

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// New returns an SMTP mailer, or a log mailer when cfg.Host is empty.
func New(cfg Config) Mailer {
	if cfg.Host == "" {
		slog.Warn("SMTP_HOST kosong: email ditulis ke log (hanya untuk pengembangan)")
		return logMailer{}
	}
	return smtpMailer{cfg: cfg}
}

type logMailer struct{}

func (logMailer) Send(_ context.Context, to, subject, body string) error {
	slog.Info("email (log mailer)", "to", to, "subject", subject, "body", body)
	return nil
}

type smtpMailer struct{ cfg Config }

// Send uses STARTTLS when the server offers it (smtp.SendMail does).
func (m smtpMailer) Send(ctx context.Context, to, subject, body string) error {
	from, err := mail.ParseAddress(m.cfg.From)
	if err != nil {
		return fmt.Errorf("MAIL_FROM: %w", err)
	}
	if strings.ContainsAny(to+subject, "\r\n") {
		return fmt.Errorf("header injection")
	}
	msg := strings.Join([]string{
		"From: " + from.String(),
		"To: " + to,
		"Subject: " + subject,
		"Date: " + time.Now().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(net.JoinHostPort(m.cfg.Host, strconv.Itoa(m.cfg.Port)), auth, from.Address, []string{to}, []byte(msg))
	}()
	select {
	case err = <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
