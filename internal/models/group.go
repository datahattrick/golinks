package models

import "github.com/google/uuid"

// ResolvedLink represents the result of keyword resolution across all scopes.
// Used to return the winning link from the resolution query.
type ResolvedLink struct {
	ID     uuid.UUID `json:"id"`
	URL    string    `json:"url"`
	Source string    `json:"source"` // "personal", "org", "global"
}
