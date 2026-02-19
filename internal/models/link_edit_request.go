package models

import (
	"time"

	"github.com/google/uuid"
)

// LinkEditRequest represents a pending edit request for an approved link.
type LinkEditRequest struct {
	ID          uuid.UUID  `json:"id"`
	LinkID      uuid.UUID  `json:"link_id"`
	UserID      uuid.UUID  `json:"user_id"`
	URL         string     `json:"url"`
	Description string     `json:"description"`
	Reason      string     `json:"reason"`
	Status      string     `json:"status"` // pending, approved, rejected
	ReviewedBy  *uuid.UUID `json:"reviewed_by"`
	ReviewedAt  *time.Time `json:"reviewed_at"`
	CreatedAt   time.Time  `json:"created_at"`

	// Non-DB fields, populated via JOIN for display
	Keyword     string `json:"keyword,omitempty"`
	AuthorName  string `json:"author_name,omitempty"`
	AuthorEmail string `json:"author_email,omitempty"`
}
