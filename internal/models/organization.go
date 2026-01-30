package models

import (
	"time"

	"github.com/google/uuid"
)

// Organization represents a group/team that can have its own links.
type Organization struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	Slug                string    `json:"slug"`
	FallbackRedirectURL *string   `json:"fallback_redirect_url,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
