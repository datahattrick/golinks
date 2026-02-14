package models

import (
	"time"

	"github.com/google/uuid"
)

// SharedLink represents a pending link share from one user to another.
type SharedLink struct {
	ID          uuid.UUID `json:"id"`
	SenderID    uuid.UUID `json:"sender_id"`
	RecipientID uuid.UUID `json:"recipient_id"`
	Keyword     string    `json:"keyword"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// SharedLinkWithUser includes sender/recipient display info for template rendering.
type SharedLinkWithUser struct {
	SharedLink
	UserName  string // sender name (for incoming) or recipient name (for outgoing)
	UserEmail string
}
