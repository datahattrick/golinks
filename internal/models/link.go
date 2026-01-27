package models

import (
	"time"

	"github.com/google/uuid"
)

// Link scope constants
const (
	ScopeGlobal = "global"
	ScopeOrg    = "org"
)

// Link status constants
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

// Health status constants
const (
	HealthUnknown   = "unknown"
	HealthHealthy   = "healthy"
	HealthUnhealthy = "unhealthy"
)

// Link represents a keyword-to-URL mapping.
type Link struct {
	ID             uuid.UUID  `json:"id"`
	Keyword        string     `json:"keyword"`
	URL            string     `json:"url"`
	Description    string     `json:"description"`
	Scope          string     `json:"scope"`           // global, org
	OrganizationID *uuid.UUID `json:"organization_id"` // Set for org-scoped links
	Status         string     `json:"status"`          // pending, approved, rejected
	CreatedBy      *uuid.UUID `json:"created_by"`      // Original creator (for approved links)
	SubmittedBy    *uuid.UUID `json:"submitted_by"`    // User who submitted for approval
	ReviewedBy     *uuid.UUID `json:"reviewed_by"`     // Moderator who approved/rejected
	ReviewedAt     *time.Time `json:"reviewed_at"`
	ClickCount      int64      `json:"click_count"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	HealthStatus    string     `json:"health_status"`
	HealthCheckedAt *time.Time `json:"health_checked_at"`
	HealthError     *string    `json:"health_error"`
}

// IsPending returns true if the link is awaiting moderation.
func (l *Link) IsPending() bool {
	return l.Status == StatusPending
}

// IsApproved returns true if the link is approved and active.
func (l *Link) IsApproved() bool {
	return l.Status == StatusApproved
}

// IsHealthy returns true if the link has a healthy status.
func (l *Link) IsHealthy() bool {
	return l.HealthStatus == HealthHealthy
}

// IsUnhealthy returns true if the link has an unhealthy status.
func (l *Link) IsUnhealthy() bool {
	return l.HealthStatus == HealthUnhealthy
}

// NeedsHealthCheck returns true if the link needs a health check.
// A link needs checking if it has never been checked or if the last check
// was older than maxAge.
func (l *Link) NeedsHealthCheck(maxAge time.Duration) bool {
	if l.HealthCheckedAt == nil {
		return true
	}
	return time.Since(*l.HealthCheckedAt) > maxAge
}
