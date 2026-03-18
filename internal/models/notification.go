package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	NotifTypeLinkSubmitted = "link_submitted"
	NotifTypeLinkApproved  = "link_approved"
	NotifTypeLinkRejected  = "link_rejected"
)

// Notification represents an in-app notification for a user.
type Notification struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	ActionURL string     `json:"action_url"`
	LinkID    *uuid.UUID `json:"link_id,omitempty"`
	Read      bool       `json:"read"`
	CreatedAt time.Time  `json:"created_at"`
}
