package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user authenticated via OIDC.
type User struct {
	ID        uuid.UUID `json:"id"`
	Sub       string    `json:"sub"`     // OIDC subject identifier
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Picture   string    `json:"picture"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
