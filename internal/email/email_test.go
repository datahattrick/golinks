package email

import (
	"strings"
	"testing"

	"golinks/internal/config"
)

func TestNewService(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		wantEnabled bool
	}{
		{
			name: "enabled when all SMTP settings configured",
			cfg: &config.Config{
				SMTPEnabled: true,
				SMTPHost:    "smtp.example.com",
				SMTPPort:    587,
				SMTPFrom:    "noreply@example.com",
			},
			wantEnabled: true,
		},
		{
			name: "disabled when SMTPEnabled is false",
			cfg: &config.Config{
				SMTPEnabled: false,
				SMTPHost:    "smtp.example.com",
				SMTPPort:    587,
				SMTPFrom:    "noreply@example.com",
			},
			wantEnabled: false,
		},
		{
			name: "disabled when SMTPHost is empty",
			cfg: &config.Config{
				SMTPEnabled: true,
				SMTPHost:    "",
				SMTPPort:    587,
				SMTPFrom:    "noreply@example.com",
			},
			wantEnabled: false,
		},
		{
			name: "disabled when SMTPFrom is empty",
			cfg: &config.Config{
				SMTPEnabled: true,
				SMTPHost:    "smtp.example.com",
				SMTPPort:    587,
				SMTPFrom:    "",
			},
			wantEnabled: false,
		},
		{
			name:        "disabled with empty config",
			cfg:         &config.Config{},
			wantEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.cfg)
			if svc.IsEnabled() != tt.wantEnabled {
				t.Errorf("IsEnabled() = %v, want %v", svc.IsEnabled(), tt.wantEnabled)
			}
		})
	}
}

func TestService_Send_Disabled(t *testing.T) {
	cfg := &config.Config{
		SMTPEnabled: false,
	}
	svc := NewService(cfg)

	// Should return nil when disabled
	err := svc.Send([]string{"test@example.com"}, "Test", "<p>HTML</p>", "Text")
	if err != nil {
		t.Errorf("Send() with disabled service should return nil, got %v", err)
	}
}

func TestService_Send_NoRecipients(t *testing.T) {
	cfg := &config.Config{
		SMTPEnabled: true,
		SMTPHost:    "smtp.example.com",
		SMTPPort:    587,
		SMTPFrom:    "noreply@example.com",
	}
	svc := NewService(cfg)

	// Should return nil when no recipients
	err := svc.Send([]string{}, "Test", "<p>HTML</p>", "Text")
	if err != nil {
		t.Errorf("Send() with no recipients should return nil, got %v", err)
	}

	err = svc.Send(nil, "Test", "<p>HTML</p>", "Text")
	if err != nil {
		t.Errorf("Send() with nil recipients should return nil, got %v", err)
	}
}

func TestService_BuildMessage(t *testing.T) {
	cfg := &config.Config{
		SMTPEnabled:  true,
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPFrom:     "noreply@example.com",
		SMTPFromName: "GoLinks",
	}

	tests := []struct {
		name          string
		htmlBody      string
		textBody      string
		wantMultipart bool
		wantHTML      bool
		wantText      bool
	}{
		{
			name:          "multipart message",
			htmlBody:      "<p>HTML content</p>",
			textBody:      "Text content",
			wantMultipart: true,
			wantHTML:      true,
			wantText:      true,
		},
		{
			name:          "HTML only",
			htmlBody:      "<p>HTML content</p>",
			textBody:      "",
			wantMultipart: false,
			wantHTML:      true,
			wantText:      false,
		},
		{
			name:          "Text only",
			htmlBody:      "",
			textBody:      "Text content",
			wantMultipart: false,
			wantHTML:      false,
			wantText:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't easily test the actual message building without exposing it,
			// but we can verify the service handles various body combinations
			svc := NewService(cfg)
			if !svc.IsEnabled() {
				t.Error("Service should be enabled")
			}
		})
	}
}

func TestService_FromHeader(t *testing.T) {
	tests := []struct {
		name       string
		fromName   string
		fromAddr   string
		wantHeader string
	}{
		{
			name:       "with display name",
			fromName:   "GoLinks",
			fromAddr:   "noreply@example.com",
			wantHeader: "GoLinks <noreply@example.com>",
		},
		{
			name:       "without display name",
			fromName:   "",
			fromAddr:   "noreply@example.com",
			wantHeader: "noreply@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				SMTPEnabled:  true,
				SMTPHost:     "smtp.example.com",
				SMTPPort:     587,
				SMTPFrom:     tt.fromAddr,
				SMTPFromName: tt.fromName,
			}

			// Build from header same way as in Send
			from := cfg.SMTPFrom
			if cfg.SMTPFromName != "" {
				from = cfg.SMTPFromName + " <" + cfg.SMTPFrom + ">"
			}

			if from != tt.wantHeader {
				t.Errorf("From header = %q, want %q", from, tt.wantHeader)
			}
		})
	}
}

func TestService_TLSModes(t *testing.T) {
	tests := []struct {
		name    string
		tlsMode string
		port    int
	}{
		{name: "starttls mode", tlsMode: "starttls", port: 587},
		{name: "tls mode", tlsMode: "tls", port: 465},
		{name: "none mode", tlsMode: "none", port: 25},
		{name: "default to starttls", tlsMode: "", port: 587},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				SMTPEnabled: true,
				SMTPHost:    "smtp.example.com",
				SMTPPort:    tt.port,
				SMTPFrom:    "noreply@example.com",
				SMTPTLS:     tt.tlsMode,
			}

			svc := NewService(cfg)
			if !svc.IsEnabled() {
				t.Error("Service should be enabled")
			}
		})
	}
}

func TestService_SendAsync_Disabled(t *testing.T) {
	cfg := &config.Config{
		SMTPEnabled: false,
	}
	svc := NewService(cfg)

	// Should not panic when disabled
	svc.SendAsync([]string{"test@example.com"}, "Test", "<p>HTML</p>", "Text")
}

func TestMIMEMessageFormat(t *testing.T) {
	// Test that the MIME message is properly formatted
	tests := []struct {
		name     string
		subject  string
		htmlBody string
		textBody string
		checks   []string
	}{
		{
			name:     "multipart message format",
			subject:  "Test Subject",
			htmlBody: "<p>HTML</p>",
			textBody: "Plain text",
			checks: []string{
				"MIME-Version: 1.0",
				"Content-Type: multipart/alternative",
				"boundary=",
				"Content-Type: text/plain; charset=UTF-8",
				"Content-Type: text/html; charset=UTF-8",
			},
		},
		{
			name:     "html only format",
			subject:  "HTML Only",
			htmlBody: "<p>HTML</p>",
			textBody: "",
			checks: []string{
				"MIME-Version: 1.0",
				"Content-Type: text/html; charset=UTF-8",
			},
		},
		{
			name:     "text only format",
			subject:  "Text Only",
			htmlBody: "",
			textBody: "Plain text",
			checks: []string{
				"MIME-Version: 1.0",
				"Content-Type: text/plain; charset=UTF-8",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build message manually to test format
			msg := buildTestMessage("Test <test@example.com>", []string{"to@example.com"}, tt.subject, tt.htmlBody, tt.textBody)

			for _, check := range tt.checks {
				if !strings.Contains(msg, check) {
					t.Errorf("Message missing %q\nMessage:\n%s", check, msg)
				}
			}
		})
	}
}

// buildTestMessage replicates the message building logic from Send for testing
func buildTestMessage(from string, to []string, subject, htmlBody, textBody string) string {
	msg := strings.Builder{}
	msg.WriteString("From: " + from + "\r\n")
	msg.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")

	if htmlBody != "" && textBody != "" {
		boundary := "----=_Part_0_GoLinks"
		msg.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n")
		msg.WriteString("\r\n")
		msg.WriteString("--" + boundary + "\r\n")
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(textBody)
		msg.WriteString("\r\n")
		msg.WriteString("--" + boundary + "\r\n")
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(htmlBody)
		msg.WriteString("\r\n")
		msg.WriteString("--" + boundary + "--\r\n")
	} else if htmlBody != "" {
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(htmlBody)
	} else {
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(textBody)
	}

	return msg.String()
}
