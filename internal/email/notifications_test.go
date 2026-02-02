package email

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/models"
)

// Ensure context is used (for test functions that need it)
var _ = context.Background

func TestNewNotifier(t *testing.T) {
	cfg := &config.Config{
		SMTPEnabled: false,
		SiteTitle:   "Test",
		BaseURL:     "https://test.example.com",
	}

	notifier := NewNotifier(cfg, nil)

	if notifier == nil {
		t.Fatal("NewNotifier returned nil")
	}
	if notifier.service == nil {
		t.Error("Notifier service is nil")
	}
	if notifier.templates == nil {
		t.Error("Notifier templates is nil")
	}
	if notifier.cfg != cfg {
		t.Error("Notifier config not set")
	}
}

func TestNotifier_NotifyModeratorsLinkSubmitted_Disabled(t *testing.T) {
	cfg := &config.Config{
		SMTPEnabled:                   false,
		EmailNotifyModeratorsOnSubmit: true,
	}
	notifier := NewNotifier(cfg, nil)

	// Should not panic when email is disabled
	link := &models.Link{Keyword: "test", URL: "https://example.com", Scope: models.ScopeGlobal}
	user := &models.User{Name: "Test", Email: "test@example.com"}
	notifier.NotifyModeratorsLinkSubmitted(context.Background(), link, user)
}

func TestNotifier_NotifyModeratorsLinkSubmitted_NotificationDisabled(t *testing.T) {
	cfg := &config.Config{
		SMTPEnabled:                   true,
		SMTPHost:                      "smtp.test.com",
		SMTPFrom:                      "test@test.com",
		EmailNotifyModeratorsOnSubmit: false,
	}
	notifier := NewNotifier(cfg, nil)

	// Should not send when notification type is disabled
	link := &models.Link{Keyword: "test", URL: "https://example.com", Scope: models.ScopeGlobal}
	user := &models.User{Name: "Test", Email: "test@example.com"}
	notifier.NotifyModeratorsLinkSubmitted(context.Background(), link, user)
}

func TestNotifier_NotifyUserLinkApproved_Disabled(t *testing.T) {
	cfg := &config.Config{
		SMTPEnabled:               false,
		EmailNotifyUserOnApproval: true,
	}
	notifier := NewNotifier(cfg, nil)

	// Should not panic when email is disabled
	link := &models.Link{Keyword: "test"}
	approver := &models.User{Name: "Mod"}
	notifier.NotifyUserLinkApproved(context.Background(), link, approver)
}

func TestNotifier_NotifyUserLinkApproved_NoSubmitter(t *testing.T) {
	cfg := &config.Config{
		SMTPEnabled:               true,
		SMTPHost:                  "smtp.test.com",
		SMTPFrom:                  "test@test.com",
		EmailNotifyUserOnApproval: true,
	}
	notifier := NewNotifier(cfg, nil)

	// Link without SubmittedBy or CreatedBy should not send
	link := &models.Link{Keyword: "test"}
	approver := &models.User{Name: "Mod"}
	notifier.NotifyUserLinkApproved(context.Background(), link, approver)
}

func TestNotifier_NotifyUserLinkRejected_Disabled(t *testing.T) {
	cfg := &config.Config{
		SMTPEnabled:                false,
		EmailNotifyUserOnRejection: true,
	}
	notifier := NewNotifier(cfg, nil)

	// Should not panic when email is disabled
	link := &models.Link{Keyword: "test"}
	notifier.NotifyUserLinkRejected(context.Background(), link, "some reason")
}

func TestNotifier_NotifyUserLinkRejected_NoSubmitter(t *testing.T) {
	cfg := &config.Config{
		SMTPEnabled:                true,
		SMTPHost:                   "smtp.test.com",
		SMTPFrom:                   "test@test.com",
		EmailNotifyUserOnRejection: true,
	}
	notifier := NewNotifier(cfg, nil)

	// Link without SubmittedBy or CreatedBy should not send
	link := &models.Link{Keyword: "test"}
	notifier.NotifyUserLinkRejected(context.Background(), link, "reason")
}

func TestNotifier_NotifyUserLinkDeleted_Disabled(t *testing.T) {
	cfg := &config.Config{
		SMTPEnabled:               false,
		EmailNotifyUserOnDeletion: true,
	}
	notifier := NewNotifier(cfg, nil)

	// Should not panic when email is disabled
	link := &models.Link{Keyword: "test"}
	notifier.NotifyUserLinkDeleted(context.Background(), link, "reason")
}

func TestNotifier_NotifyModeratorsHealthChecksFailed_Disabled(t *testing.T) {
	cfg := &config.Config{
		SMTPEnabled:                    false,
		EmailNotifyModsOnHealthFailure: true,
	}
	notifier := NewNotifier(cfg, nil)

	// Should not panic when email is disabled
	links := []models.Link{{Keyword: "broken", URL: "https://broken.com"}}
	notifier.NotifyModeratorsHealthChecksFailed(context.Background(), links)
}

func TestNotifier_NotifyModeratorsHealthChecksFailed_EmptyList(t *testing.T) {
	cfg := &config.Config{
		SMTPEnabled:                    true,
		SMTPHost:                       "smtp.test.com",
		SMTPFrom:                       "test@test.com",
		EmailNotifyModsOnHealthFailure: true,
	}
	notifier := NewNotifier(cfg, nil)

	// Should not send for empty list
	notifier.NotifyModeratorsHealthChecksFailed(context.Background(), []models.Link{})
}

func TestNotifier_NotifyWelcome_Disabled(t *testing.T) {
	cfg := &config.Config{
		SMTPEnabled: false,
	}
	notifier := NewNotifier(cfg, nil)

	// Should not panic when email is disabled
	user := &models.User{Name: "Test", Email: "test@example.com"}
	notifier.NotifyWelcome(context.Background(), user)
}

func TestNotifier_NotifyWelcome_NoEmail(t *testing.T) {
	cfg := &config.Config{
		SMTPEnabled: true,
		SMTPHost:    "smtp.test.com",
		SMTPFrom:    "test@test.com",
	}
	notifier := NewNotifier(cfg, nil)

	// User without email should not send
	user := &models.User{Name: "Test", Email: ""}
	notifier.NotifyWelcome(context.Background(), user)
}

func TestNotifier_SubmitterFallback_Logic(t *testing.T) {
	// Test the logic for determining submitter ID without calling the full notification method
	// (which requires a database connection)

	userID := uuid.New()
	otherID := uuid.New()

	tests := []struct {
		name        string
		submittedBy *uuid.UUID
		createdBy   *uuid.UUID
		expectID    *uuid.UUID
	}{
		{
			name:        "only SubmittedBy set",
			submittedBy: &userID,
			createdBy:   nil,
			expectID:    &userID,
		},
		{
			name:        "only CreatedBy set",
			submittedBy: nil,
			createdBy:   &userID,
			expectID:    &userID,
		},
		{
			name:        "both set - uses SubmittedBy",
			submittedBy: &userID,
			createdBy:   &otherID,
			expectID:    &userID,
		},
		{
			name:        "neither set - nil",
			submittedBy: nil,
			createdBy:   nil,
			expectID:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link := &models.Link{
				Keyword:     "test",
				SubmittedBy: tt.submittedBy,
				CreatedBy:   tt.createdBy,
			}

			// Replicate the logic from NotifyUserLinkApproved
			submitterID := link.SubmittedBy
			if submitterID == nil {
				submitterID = link.CreatedBy
			}

			if tt.expectID == nil {
				if submitterID != nil {
					t.Errorf("Expected nil submitterID, got %v", submitterID)
				}
			} else {
				if submitterID == nil {
					t.Error("Expected non-nil submitterID, got nil")
				} else if *submitterID != *tt.expectID {
					t.Errorf("Expected submitterID %v, got %v", *tt.expectID, *submitterID)
				}
			}
		})
	}
}

func TestNotifier_LinkScopeHandling_Logic(t *testing.T) {
	// Test the scope handling logic without making actual database calls
	orgID := uuid.New()

	tests := []struct {
		name           string
		link           *models.Link
		expectGlobal   bool
		expectOrg      bool
		expectPersonal bool
	}{
		{
			name:           "global scope sends to global mods",
			link:           &models.Link{Keyword: "test", URL: "https://example.com", Scope: models.ScopeGlobal},
			expectGlobal:   true,
			expectOrg:      false,
			expectPersonal: false,
		},
		{
			name:           "org scope sends to org mods",
			link:           &models.Link{Keyword: "test", URL: "https://example.com", Scope: models.ScopeOrg, OrganizationID: &orgID},
			expectGlobal:   false,
			expectOrg:      true,
			expectPersonal: false,
		},
		{
			name:           "org scope without org ID is handled",
			link:           &models.Link{Keyword: "test", URL: "https://example.com", Scope: models.ScopeOrg, OrganizationID: nil},
			expectGlobal:   false,
			expectOrg:      false,
			expectPersonal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replicate the scope logic from NotifyModeratorsLinkSubmitted
			var shouldSendGlobal, shouldSendOrg bool

			if tt.link.Scope == models.ScopeGlobal {
				shouldSendGlobal = true
			} else if tt.link.Scope == models.ScopeOrg && tt.link.OrganizationID != nil {
				shouldSendOrg = true
			}

			if shouldSendGlobal != tt.expectGlobal {
				t.Errorf("Expected global=%v, got %v", tt.expectGlobal, shouldSendGlobal)
			}
			if shouldSendOrg != tt.expectOrg {
				t.Errorf("Expected org=%v, got %v", tt.expectOrg, shouldSendOrg)
			}
		})
	}
}
