package email

import (
	"context"
	"log"

	"github.com/google/uuid"

	"golinks/internal/config"
	"golinks/internal/models"
)

// ModeratorEmailGetter is an interface for getting moderator emails.
type ModeratorEmailGetter interface {
	GetModeratorEmails(ctx context.Context, scope string, orgID *string) ([]string, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error)
}

// Notifier sends email notifications for various events.
type Notifier struct {
	service   *Service
	templates *Templates
	cfg       *config.Config
	db        ModeratorEmailGetter
}

// NewNotifier creates a new email notifier.
func NewNotifier(cfg *config.Config, db ModeratorEmailGetter) *Notifier {
	return &Notifier{
		service:   NewService(cfg),
		templates: NewTemplates(cfg),
		cfg:       cfg,
		db:        db,
	}
}

// NotifyLinkSubmitted notifies moderators that a new link needs review.
func (n *Notifier) NotifyLinkSubmitted(ctx context.Context, link *models.Link, submitter *models.User) {
	if !n.service.IsEnabled() || !n.cfg.EmailNotifyModeratorsOnSubmit {
		return
	}

	// Get moderator emails based on link scope
	var orgID *string
	if link.OrganizationID != nil {
		id := link.OrganizationID.String()
		orgID = &id
	}

	emails, err := n.db.GetModeratorEmails(ctx, link.Scope, orgID)
	if err != nil {
		log.Printf("Failed to get moderator emails: %v", err)
		return
	}

	if len(emails) == 0 {
		log.Println("No moderator emails found for notification")
		return
	}

	subject, htmlBody, textBody := n.templates.LinkSubmittedForReview(link, submitter)
	n.service.SendAsync(emails, subject, htmlBody, textBody)
}

// NotifyLinkApproved notifies the link creator that their link was approved.
func (n *Notifier) NotifyLinkApproved(ctx context.Context, link *models.Link, approver *models.User) {
	if !n.service.IsEnabled() || !n.cfg.EmailNotifyUserOnApproval {
		return
	}

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

// NotifyLinkRejected notifies the link creator that their link was rejected.
func (n *Notifier) NotifyLinkRejected(ctx context.Context, link *models.Link, rejector *models.User, reason string) {
	if !n.service.IsEnabled() || !n.cfg.EmailNotifyUserOnRejection {
		return
	}

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

	subject, htmlBody, textBody := n.templates.LinkRejected(link, rejector, reason)
	n.service.SendAsync([]string{creator.Email}, subject, htmlBody, textBody)
}

// NotifyLinkDeleted notifies the link creator that their link was deleted.
func (n *Notifier) NotifyLinkDeleted(ctx context.Context, link *models.Link, deletedBy *models.User, reason string) {
	if !n.service.IsEnabled() || !n.cfg.EmailNotifyUserOnDeletion {
		return
	}

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

	subject, htmlBody, textBody := n.templates.LinkDeleted(link, deletedBy, reason)
	n.service.SendAsync([]string{creator.Email}, subject, htmlBody, textBody)
}

// NotifyHealthCheckFailures notifies moderators about failed health checks.
func (n *Notifier) NotifyHealthCheckFailures(ctx context.Context, links []models.Link) {
	if !n.service.IsEnabled() || !n.cfg.EmailNotifyModsOnHealthFailure {
		return
	}

	if len(links) == 0 {
		return
	}

	// Get all global moderators and admins for health notifications
	emails, err := n.db.GetModeratorEmails(ctx, models.ScopeGlobal, nil)
	if err != nil {
		log.Printf("Failed to get moderator emails for health notification: %v", err)
		return
	}

	if len(emails) == 0 {
		return
	}

	subject, htmlBody, textBody := n.templates.HealthCheckFailed(links)
	n.service.SendAsync(emails, subject, htmlBody, textBody)
}
