// Package email sends transactional emails. Defaults to Mailpit's SMTP
// listener in dev (localhost:1025) — config via env in prod.
package email

import (
	"context"
	"fmt"
	"net/smtp"
	"os"
)

type Sender struct {
	host string
	port string
	from string
}

func NewSender() *Sender {
	return &Sender{
		host: envOr("SMTP_HOST", "localhost"),
		port: envOr("SMTP_PORT", "1025"),
		from: envOr("PACTLINE_FROM_EMAIL", "noreply@pactline.local"),
	}
}

func (s *Sender) Send(_ context.Context, to, subject, body string) error {
	addr := s.host + ":" + s.port
	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.from, to, subject, body,
	))
	return smtp.SendMail(addr, nil, s.from, []string{to}, msg)
}

func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
