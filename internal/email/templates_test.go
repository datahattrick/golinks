package email

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/models"
)

func TestNewTemplates(t *testing.T) {
	cfg := &config.Config{
		SiteTitle: "TestGoLinks",
		BaseURL:   "https://go.example.com",
	}

	tmpl := NewTemplates(cfg)
	if tmpl == nil {
		t.Fatal("NewTemplates returned nil")
	}
	if tmpl.cfg != cfg {
		t.Error("Templates config not set correctly")
	}
}

func TestTemplates_BaseHTML(t *testing.T) {
	cfg := &config.Config{
		SiteTitle: "TestGoLinks",
		BaseURL:   "https://go.example.com",
	}
	tmpl := NewTemplates(cfg)

	html := tmpl.baseHTML("Test Title", "<p>Test content</p>")

	checks := []string{
		"<!DOCTYPE html>",
		"<title>Test Title</title>",
		"TestGoLinks",
		"https://go.example.com",
		"<p>Test content</p>",
	}

	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("baseHTML missing %q", check)
		}
	}
}

func TestTemplates_BaseHTML_EscapesHTML(t *testing.T) {
	cfg := &config.Config{
		SiteTitle: "<script>alert('xss')</script>",
		BaseURL:   "https://go.example.com",
	}
	tmpl := NewTemplates(cfg)

	html := tmpl.baseHTML("Test", "Content")

	// Should escape the script tag in site title
	if strings.Contains(html, "<script>") {
		t.Error("baseHTML should escape HTML in site title")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("baseHTML should contain escaped script tag")
	}
}

func TestTemplates_LinkSubmittedForReview(t *testing.T) {
	cfg := &config.Config{
		SiteTitle: "GoLinks",
		BaseURL:   "https://go.example.com",
	}
	tmpl := NewTemplates(cfg)

	link := &models.Link{
		Keyword:     "test-link",
		URL:         "https://example.com/destination",
		Description: "Test description",
		Scope:       models.ScopeGlobal,
	}
	submitter := &models.User{
		Name:  "John Doe",
		Email: "john@example.com",
	}

	subject, htmlBody, textBody := tmpl.LinkSubmittedForReview(link, submitter)

	// Check subject
	if !strings.Contains(subject, "test-link") {
		t.Errorf("Subject should contain keyword, got: %s", subject)
	}
	if !strings.Contains(subject, "GoLinks") {
		t.Errorf("Subject should contain site title, got: %s", subject)
	}

	// Check HTML body
	htmlChecks := []string{
		"test-link",
		"https://example.com/destination",
		"Test description",
		"Global",
		"John Doe",
		"john@example.com",
		"/moderation",
	}
	for _, check := range htmlChecks {
		if !strings.Contains(htmlBody, check) {
			t.Errorf("HTML body missing %q", check)
		}
	}

	// Check text body
	textChecks := []string{
		"test-link",
		"https://example.com/destination",
		"Test description",
		"Global",
		"John Doe",
		"john@example.com",
	}
	for _, check := range textChecks {
		if !strings.Contains(textBody, check) {
			t.Errorf("Text body missing %q", check)
		}
	}
}

func TestTemplates_LinkSubmittedForReview_OrgScope(t *testing.T) {
	cfg := &config.Config{
		SiteTitle: "GoLinks",
		BaseURL:   "https://go.example.com",
	}
	tmpl := NewTemplates(cfg)

	orgID := uuid.New()
	link := &models.Link{
		Keyword:        "org-link",
		URL:            "https://example.com",
		Scope:          models.ScopeOrg,
		OrganizationID: &orgID,
	}
	submitter := &models.User{Name: "Jane", Email: "jane@example.com"}

	_, htmlBody, textBody := tmpl.LinkSubmittedForReview(link, submitter)

	if !strings.Contains(htmlBody, "Organization") {
		t.Error("HTML body should show Organization scope")
	}
	if !strings.Contains(textBody, "Organization") {
		t.Error("Text body should show Organization scope")
	}
}

func TestTemplates_LinkApproved(t *testing.T) {
	cfg := &config.Config{
		SiteTitle: "GoLinks",
		BaseURL:   "https://go.example.com",
	}
	tmpl := NewTemplates(cfg)

	link := &models.Link{
		Keyword: "approved-link",
		URL:     "https://example.com/approved",
	}
	approver := &models.User{Name: "Mod User"}

	subject, htmlBody, textBody := tmpl.LinkApproved(link, approver)

	// Check subject
	if !strings.Contains(subject, "approved-link") {
		t.Errorf("Subject should contain keyword, got: %s", subject)
	}
	if !strings.Contains(subject, "approved") {
		t.Errorf("Subject should mention approval, got: %s", subject)
	}

	// Check HTML body contains short link
	if !strings.Contains(htmlBody, "go.example.com/go/approved-link") {
		t.Error("HTML body should contain short link URL")
	}

	// Check text body
	if !strings.Contains(textBody, "approved") {
		t.Error("Text body should mention approval")
	}
}

func TestTemplates_LinkRejected(t *testing.T) {
	cfg := &config.Config{
		SiteTitle: "GoLinks",
		BaseURL:   "https://go.example.com",
	}
	tmpl := NewTemplates(cfg)

	link := &models.Link{
		Keyword: "rejected-link",
		URL:     "https://example.com/rejected",
	}

	tests := []struct {
		name   string
		reason string
	}{
		{name: "with reason", reason: "Link violates policy"},
		{name: "without reason", reason: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subject, htmlBody, textBody := tmpl.LinkRejected(link, tt.reason)

			if !strings.Contains(subject, "rejected-link") {
				t.Errorf("Subject should contain keyword, got: %s", subject)
			}

			if !strings.Contains(htmlBody, "rejected-link") {
				t.Error("HTML body should contain keyword")
			}

			if tt.reason != "" {
				if !strings.Contains(htmlBody, tt.reason) {
					t.Error("HTML body should contain reason when provided")
				}
				if !strings.Contains(textBody, tt.reason) {
					t.Error("Text body should contain reason when provided")
				}
			}

			// Should have link to create new
			if !strings.Contains(htmlBody, "/new") {
				t.Error("HTML body should contain link to create new")
			}
		})
	}
}

func TestTemplates_LinkDeleted(t *testing.T) {
	cfg := &config.Config{
		SiteTitle: "GoLinks",
		BaseURL:   "https://go.example.com",
	}
	tmpl := NewTemplates(cfg)

	link := &models.Link{
		Keyword: "deleted-link",
		URL:     "https://example.com/deleted",
	}

	tests := []struct {
		name   string
		reason string
	}{
		{name: "with reason", reason: "Link no longer valid"},
		{name: "without reason", reason: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subject, htmlBody, textBody := tmpl.LinkDeleted(link, tt.reason)

			if !strings.Contains(subject, "deleted-link") {
				t.Errorf("Subject should contain keyword, got: %s", subject)
			}
			if !strings.Contains(subject, "removed") {
				t.Errorf("Subject should mention removal, got: %s", subject)
			}

			if !strings.Contains(htmlBody, "deleted-link") {
				t.Error("HTML body should contain keyword")
			}

			if tt.reason != "" {
				if !strings.Contains(htmlBody, tt.reason) {
					t.Error("HTML body should contain reason when provided")
				}
				if !strings.Contains(textBody, tt.reason) {
					t.Error("Text body should contain reason when provided")
				}
			}
		})
	}
}

func TestTemplates_HealthCheckFailed(t *testing.T) {
	cfg := &config.Config{
		SiteTitle: "GoLinks",
		BaseURL:   "https://go.example.com",
	}
	tmpl := NewTemplates(cfg)

	errorMsg := "Connection timeout"
	links := []models.Link{
		{
			Keyword:     "broken1",
			URL:         "https://broken1.example.com",
			HealthError: &errorMsg,
		},
		{
			Keyword:     "broken2",
			URL:         "https://broken2.example.com",
			HealthError: nil,
		},
	}

	subject, htmlBody, textBody := tmpl.HealthCheckFailed(links)

	// Check subject mentions count
	if !strings.Contains(subject, "2") {
		t.Errorf("Subject should mention link count, got: %s", subject)
	}

	// Check HTML body
	htmlChecks := []string{
		"broken1",
		"broken2",
		"https://broken1.example.com",
		"https://broken2.example.com",
		"Connection timeout",
		"Unknown error",
		"/manage?filter=unhealthy",
	}
	for _, check := range htmlChecks {
		if !strings.Contains(htmlBody, check) {
			t.Errorf("HTML body missing %q", check)
		}
	}

	// Check text body
	textChecks := []string{
		"broken1",
		"broken2",
		"Connection timeout",
	}
	for _, check := range textChecks {
		if !strings.Contains(textBody, check) {
			t.Errorf("Text body missing %q", check)
		}
	}
}

func TestTemplates_WelcomeUser(t *testing.T) {
	cfg := &config.Config{
		SiteTitle: "GoLinks",
		BaseURL:   "https://go.example.com",
	}
	tmpl := NewTemplates(cfg)

	user := &models.User{
		Name:  "New User",
		Email: "newuser@example.com",
	}

	subject, htmlBody, textBody := tmpl.WelcomeUser(user)

	// Check subject
	if !strings.Contains(subject, "Welcome") {
		t.Errorf("Subject should contain Welcome, got: %s", subject)
	}
	if !strings.Contains(subject, "GoLinks") {
		t.Errorf("Subject should contain site title, got: %s", subject)
	}

	// Check HTML body
	if !strings.Contains(htmlBody, "New User") {
		t.Error("HTML body should contain user name")
	}
	if !strings.Contains(htmlBody, "go.example.com") {
		t.Error("HTML body should contain base URL")
	}

	// Check text body
	if !strings.Contains(textBody, "New User") {
		t.Error("Text body should contain user name")
	}
}

func TestTemplates_HTMLEscaping(t *testing.T) {
	cfg := &config.Config{
		SiteTitle: "GoLinks",
		BaseURL:   "https://go.example.com",
	}
	tmpl := NewTemplates(cfg)

	// Test XSS prevention in keyword and description
	link := &models.Link{
		Keyword:     "<script>alert('xss')</script>",
		URL:         "https://example.com/safe",
		Description: "<img src=x onerror=alert('xss')>",
	}
	submitter := &models.User{
		Name:  "<script>evil</script>",
		Email: "test@example.com",
	}

	_, htmlBody, _ := tmpl.LinkSubmittedForReview(link, submitter)

	// Should not contain unescaped script tags in keyword or name
	if strings.Contains(htmlBody, "<script>alert") {
		t.Error("HTML body should escape script tags in keyword")
	}
	if strings.Contains(htmlBody, "<script>evil") {
		t.Error("HTML body should escape script tags in name")
	}

	// Should contain escaped versions for keyword
	if !strings.Contains(htmlBody, "&lt;script&gt;alert") {
		t.Error("HTML body should contain escaped script tags for keyword")
	}

	// Description should also be escaped
	if strings.Contains(htmlBody, "<img src=x") {
		t.Error("HTML body should escape img tags in description")
	}
}
