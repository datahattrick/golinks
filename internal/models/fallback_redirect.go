package models

import (
	"time"

	"github.com/google/uuid"
)

// FallbackRedirect represents a named fallback redirect option for an organization.
// When a keyword is not found, users who have selected a fallback will be redirected
// to this URL with the keyword appended.
type FallbackRedirect struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	URL            string    `json:"url"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
