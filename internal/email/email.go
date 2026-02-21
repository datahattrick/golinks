package email

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"

	"github.com/google/uuid"

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
		slog.Info("email notifications enabled", "smtp_host", cfg.SMTPHost, "smtp_port", cfg.SMTPPort)
	} else {
		slog.Info("email notifications disabled (SMTP not configured)")
	}

	return s
}

// IsEnabled returns true if email sending is enabled.
func (s *Service) IsEnabled() bool {
	return s.enabled
}

// Send sends an email with the given subject and body to the recipients.
func (s *Service) Send(to []string, subject, htmlBody, textBody string) error {
	if !s.enabled {
		return nil
	}

	if len(to) == 0 {
		return nil
	}

	// Build the email message
	from := s.cfg.SMTPFrom
	if s.cfg.SMTPFromName != "" {
		from = fmt.Sprintf("%s <%s>", s.cfg.SMTPFromName, s.cfg.SMTPFrom)
	}

	// Build MIME message
	msg := strings.Builder{}
	msg.WriteString(fmt.Sprintf("From: %s\r\n", from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")

	if htmlBody != "" && textBody != "" {
		// Multipart message — use a random UUID as boundary to avoid collisions with message content.
		boundary := "----=_Part_" + uuid.New().String()
		msg.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		msg.WriteString("\r\n")
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(textBody)
		msg.WriteString("\r\n")
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(htmlBody)
		msg.WriteString("\r\n")
		msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if htmlBody != "" {
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(htmlBody)
	} else {
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(textBody)
	}

	// Send based on TLS mode
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)

	var auth smtp.Auth
	if s.cfg.SMTPUsername != "" {
		auth = smtp.PlainAuth("", s.cfg.SMTPUsername, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	}

	switch s.cfg.SMTPTLS {
	case "tls":
		return s.sendTLS(addr, auth, s.cfg.SMTPFrom, to, []byte(msg.String()))
	case "starttls":
		return s.sendStartTLS(addr, auth, s.cfg.SMTPFrom, to, []byte(msg.String()))
	default:
		return smtp.SendMail(addr, auth, s.cfg.SMTPFrom, to, []byte(msg.String()))
	}
}

// sendTLS sends email over implicit TLS (port 465).
func (s *Service) sendTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: s.cfg.SMTPHost,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.cfg.SMTPHost)
	if err != nil {
		return fmt.Errorf("SMTP client creation failed: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL failed: %w", err)
	}

	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("SMTP RCPT failed: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP close failed: %w", err)
	}

	return client.Quit()
}

// sendStartTLS sends email using STARTTLS (port 587).
func (s *Service) sendStartTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("SMTP dial failed: %w", err)
	}
	defer client.Close()

	tlsConfig := &tls.Config{
		ServerName: s.cfg.SMTPHost,
	}

	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS failed: %w", err)
	}

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL failed: %w", err)
	}

	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("SMTP RCPT failed: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP close failed: %w", err)
	}

	return client.Quit()
}

// SendAsync sends an email asynchronously (non-blocking).
func (s *Service) SendAsync(to []string, subject, htmlBody, textBody string) {
	if !s.enabled {
		return
	}

	go func() {
		if err := s.Send(to, subject, htmlBody, textBody); err != nil {
			slog.Warn("failed to send email", "to", to, "subject", subject, "error", err)
		}
	}()
}
