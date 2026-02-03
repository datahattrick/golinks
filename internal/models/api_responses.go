package models

import (
	"time"

	"github.com/google/uuid"
)

// ResolveResponse contains the result of keyword resolution.
type ResolveResponse struct {
	Keyword string `json:"keyword"`
	URL     string `json:"url"`
	Tier    int    `json:"tier"`
	Source  string `json:"source"`
}

// KeywordCheckResponse indicates whether a keyword is available.
type KeywordCheckResponse struct {
	Available    bool   `json:"available"`
	ConflictType string `json:"conflict_type,omitempty"`
}

// HealthCheckAPIResponse contains health check results for the API.
type HealthCheckAPIResponse struct {
	LinkID    uuid.UUID  `json:"link_id"`
	Status    string     `json:"status"`
	CheckedAt *time.Time `json:"checked_at"`
	Error     string     `json:"error,omitempty"`
}
