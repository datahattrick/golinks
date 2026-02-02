package email

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"golinks/internal/config"
)

// Service handles sending email notifications.
type Service struct {
	cfg     *config.Config
	enabled bool
}

// NewService creates a new email service.
func NewService(cfg *config.Config) *Service {
	s := &Service{
		cfg:     cfg,
		enabled: cfg.IsEmailEnabled(),
	}

	if s.enabled {
		log.Printf("Email notifications enabled (SMTP: %s:%d)", cfg.SMTPHost, cfg.SMTPPort)
	} else {
		log.Println("Email notifications disabled (SMTP not configured)")
	}

	return s
}

// IsEnabled returns true if email is enabled.
func (s *Service) IsEnabled() bool {
	return s.enabled
}

// SendEmail sends an email to the specified recipients.
func (s *Service) SendEmail(to []string, subject, htmlBody, textBody string) error {
	if !s.enabled {
		return nil
	}

	if len(to) == 0 {
		return nil
	}

	// Build email headers and body
	from := s.cfg.SMTPFrom
	if s.cfg.SMTPFromName != "" {
		from = fmt.Sprintf("%s <%s>", s.cfg.SMTPFromName, s.cfg.SMTPFrom)
	}

	// Build MIME message
	boundary := "GoLinksBoundary123456789"
	var msg strings.Builder

	msg.WriteString(fmt.Sprintf("From: %s\r\n", from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
	msg.WriteString("\r\n")

	// Plain text part
	if textBody != "" {
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(textBody)
		msg.WriteString("\r\n")
	}

	// HTML part
	if htmlBody != "" {
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(htmlBody)
		msg.WriteString("\r\n")
	}

	msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	// Send email based on TLS mode
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)

	var auth smtp.Auth
	if s.cfg.SMTPUsername != "" && s.cfg.SMTPPassword != "" {
		auth = smtp.PlainAuth("", s.cfg.SMTPUsername, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	}

	switch s.cfg.SMTPTLS {
	case "tls":
		return s.sendWithTLS(addr, auth, to, msg.String())
	case "starttls":
		return s.sendWithStartTLS(addr, auth, to, msg.String())
	default: // "none"
		return smtp.SendMail(addr, auth, s.cfg.SMTPFrom, to, []byte(msg.String()))
	}
}

// sendWithTLS sends email using implicit TLS (port 465).
func (s *Service) sendWithTLS(addr string, auth smtp.Auth, to []string, msg string) error {
	tlsConfig := &tls.Config{
		ServerName: s.cfg.SMTPHost,
		MinVersion: tls.VersionTLS12,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.cfg.SMTPHost)
	if err != nil {
		return fmt.Errorf("SMTP client failed: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	if err := client.Mail(s.cfg.SMTPFrom); err != nil {
		return fmt.Errorf("SMTP MAIL failed: %w", err)
	}

	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("SMTP RCPT failed: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}

	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP close failed: %w", err)
	}

	return client.Quit()
}

// sendWithStartTLS sends email using STARTTLS (port 587).
func (s *Service) sendWithStartTLS(addr string, auth smtp.Auth, to []string, msg string) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("SMTP dial failed: %w", err)
	}
	defer client.Close()

	// Send STARTTLS
	tlsConfig := &tls.Config{
		ServerName: s.cfg.SMTPHost,
		MinVersion: tls.VersionTLS12,
	}

	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS failed: %w", err)
	}

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	if err := client.Mail(s.cfg.SMTPFrom); err != nil {
		return fmt.Errorf("SMTP MAIL failed: %w", err)
	}

	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("SMTP RCPT failed: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}

	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP close failed: %w", err)
	}

	return client.Quit()
}

// SendAsync sends an email asynchronously (fire and forget with logging).
func (s *Service) SendAsync(to []string, subject, htmlBody, textBody string) {
	if !s.enabled || len(to) == 0 {
		return
	}

	go func() {
		if err := s.SendEmail(to, subject, htmlBody, textBody); err != nil {
			log.Printf("Failed to send email to %v: %v", to, err)
		} else {
			log.Printf("Email sent successfully to %v: %s", to, subject)
		}
	}()
}
