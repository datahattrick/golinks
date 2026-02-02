package email

import (
	"context"
	"log"

	"golinks/internal/config"
	"golinks/internal/db"
	"golinks/internal/models"
)

// Notifier handles sending email notifications for various events.
type Notifier struct {
	service   *Service
	templates *Templates
	db        *db.DB
	cfg       *config.Config
}

// NewNotifier creates a new email notifier.
func NewNotifier(cfg *config.Config, database *db.DB) *Notifier {
	return &Notifier{
		service:   NewService(cfg),
		templates: NewTemplates(cfg),
		db:        database,
		cfg:       cfg,
	}
}

// NotifyModeratorsLinkSubmitted notifies moderators when a new link is submitted for review.
func (n *Notifier) NotifyModeratorsLinkSubmitted(ctx context.Context, link *models.Link, submitter *models.User) {
	if !n.service.IsEnabled() || !n.cfg.EmailNotifyModeratorsOnSubmit {
		return
	}

	// Get moderator emails based on link scope
	var emails []string
	var err error

	if link.Scope == models.ScopeGlobal {
		// Global links: notify global mods and admins
		emails, err = n.db.GetGlobalModeratorEmails(ctx)
	} else if link.Scope == models.ScopeOrg && link.OrganizationID != nil {
		// Org links: notify org mods, global mods, and admins
		emails, err = n.db.GetOrgModeratorEmails(ctx, *link.OrganizationID)
	} else {
		// Personal links don't need moderation
		return
	}

	if err != nil {
		log.Printf("Failed to get moderator emails: %v", err)
		return
	}

	if len(emails) == 0 {
		return
	}

	subject, htmlBody, textBody := n.templates.LinkSubmittedForReview(link, submitter)
	n.service.SendAsync(emails, subject, htmlBody, textBody)
}

// NotifyUserLinkApproved notifies a user when their link is approved.
func (n *Notifier) NotifyUserLinkApproved(ctx context.Context, link *models.Link, approver *models.User) {
	if !n.service.IsEnabled() || !n.cfg.EmailNotifyUserOnApproval {
		return
	}

	// Get the link creator's email
	if link.CreatedBy == nil {
		return
	}

	creator, err := n.db.GetUserByID(ctx, *link.CreatedBy)
	if err != nil {
		log.Printf("Failed to get link creator: %v", err)
		return
	}

	if creator.Email == "" {
		return
	}

	subject, htmlBody, textBody := n.templates.LinkApproved(link, approver)
	n.service.SendAsync([]string{creator.Email}, subject, htmlBody, textBody)
}

// NotifyUserLinkRejected notifies a user when their link is rejected.
func (n *Notifier) NotifyUserLinkRejected(ctx context.Context, link *models.Link, reason string) {
	if !n.service.IsEnabled() || !n.cfg.EmailNotifyUserOnRejection {
		return
	}

	// Get the link creator's email
	if link.CreatedBy == nil {
		return
	}

	creator, err := n.db.GetUserByID(ctx, *link.CreatedBy)
	if err != nil {
		log.Printf("Failed to get link creator: %v", err)
		return
	}

	if creator.Email == "" {
		return
	}

	subject, htmlBody, textBody := n.templates.LinkRejected(link, reason)
	n.service.SendAsync([]string{creator.Email}, subject, htmlBody, textBody)
}

// NotifyUserLinkDeleted notifies a user when their link is deleted.
func (n *Notifier) NotifyUserLinkDeleted(ctx context.Context, link *models.Link, reason string) {
	if !n.service.IsEnabled() || !n.cfg.EmailNotifyUserOnDeletion {
		return
	}

	// Get the link creator's email
	if link.CreatedBy == nil {
		return
	}

	creator, err := n.db.GetUserByID(ctx, *link.CreatedBy)
	if err != nil {
		log.Printf("Failed to get link creator: %v", err)
		return
	}

	if creator.Email == "" {
		return
	}

	subject, htmlBody, textBody := n.templates.LinkDeleted(link, reason)
	n.service.SendAsync([]string{creator.Email}, subject, htmlBody, textBody)
}

// NotifyModeratorsHealthChecksFailed notifies moderators about failing health checks.
func (n *Notifier) NotifyModeratorsHealthChecksFailed(ctx context.Context, links []models.Link) {
	if !n.service.IsEnabled() || !n.cfg.EmailNotifyModsOnHealthFailure {
		return
	}

	if len(links) == 0 {
		return
	}

	// Get global moderator emails
	emails, err := n.db.GetGlobalModeratorEmails(ctx)
	if err != nil {
		log.Printf("Failed to get moderator emails: %v", err)
		return
	}

	if len(emails) == 0 {
		return
	}

	subject, htmlBody, textBody := n.templates.HealthCheckFailed(links)
	n.service.SendAsync(emails, subject, htmlBody, textBody)
}

// NotifyWelcome sends a welcome email to a new user.
func (n *Notifier) NotifyWelcome(ctx context.Context, user *models.User) {
	if !n.service.IsEnabled() {
		return
	}

	if user.Email == "" {
		return
	}

	subject, htmlBody, textBody := n.templates.WelcomeUser(user)
	n.service.SendAsync([]string{user.Email}, subject, htmlBody, textBody)
}
