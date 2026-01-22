package models

import (
	"time"

	"github.com/google/uuid"
)

// Link represents a keyword-to-URL mapping.
type Link struct {
	ID          uuid.UUID  `json:"id"`
	Keyword     string     `json:"keyword"`
	URL         string     `json:"url"`
	Description string     `json:"description"`
	CreatedBy   *uuid.UUID `json:"created_by"`
	ClickCount  int64      `json:"click_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
