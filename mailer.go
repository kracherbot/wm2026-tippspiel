package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

type Mailer struct {
	config *AppConfig
}

func (m *Mailer) Send(to, subject, body string) error {
	if m.config.SMTPUser == "" || m.config.SMTPPass == "" {
		log.Printf("SMTP not configured: user=%s pass=%s", m.config.SMTPUser, "***")
		return fmt.Errorf("SMTP not configured")
	}

	from := m.config.SMTPUser
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", from, to, subject, body)

	auth := smtp.PlainAuth("", m.config.SMTPUser, m.config.SMTPPass, m.config.SMTPHost)
	addr := fmt.Sprintf("%s:%s", m.config.SMTPHost, m.config.SMTPPort)

	if m.config.SMTPPort == "587" {
		c, err := smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("SMTP dial error: %w", err)
		}
		defer c.Close()

		tlsConfig := &tls.Config{
			ServerName: m.config.SMTPHost,
		}
		if err := c.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("SMTP STARTTLS error: %w", err)
		}

		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth error: %w", err)
		}

		if err := c.Mail(from); err != nil {
			return fmt.Errorf("SMTP mail from error: %w", err)
		}

		if err := c.Rcpt(to); err != nil {
			return fmt.Errorf("SMTP rcpt to error: %w", err)
		}

		w, err := c.Data()
		if err != nil {
			return fmt.Errorf("SMTP data error: %w", err)
		}

		if _, err := w.Write([]byte(msg)); err != nil {
			return fmt.Errorf("SMTP write error: %w", err)
		}

		if err := w.Close(); err != nil {
			return fmt.Errorf("SMTP close data error: %w", err)
		}

		if err := c.Quit(); err != nil {
			return fmt.Errorf("SMTP quit error: %w", err)
		}

		log.Printf("SMTP: email sent to %s", to)
		return nil
	}

	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}

func (m *Mailer) SendVerification(to, token string) error {
	verifyURL := fmt.Sprintf("%s/verify?token=%s", strings.TrimRight(m.config.BaseURL, "/"), token)
	body := fmt.Sprintf(
		"Willkommen beim WM 2026 Tippspiel!\n\nBitte bestätige deine E-Mail-Adresse:\n%s\n\nViel Spass beim Tippen!",
		verifyURL,
	)
	return m.Send(to, "WM 2026 Tippspiel - E-Mail bestätigen", body)
}