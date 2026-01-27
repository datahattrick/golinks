package models

import (
	"time"

	"github.com/google/uuid"
)

// UserLink represents a user-specific keyword override.
// These take priority over organizational links for the owning user.
type UserLink struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"user_id"`
	Keyword         string     `json:"keyword"`
	URL             string     `json:"url"`
	Description     string     `json:"description"`
	ClickCount      int64      `json:"click_count"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	HealthStatus    string     `json:"health_status"`
	HealthCheckedAt *time.Time `json:"health_checked_at"`
	HealthError     *string    `json:"health_error"`
}

// IsHealthy returns true if the link has a healthy status.
func (l *UserLink) IsHealthy() bool {
	return l.HealthStatus == HealthHealthy
}

// IsUnhealthy returns true if the link has an unhealthy status.
func (l *UserLink) IsUnhealthy() bool {
	return l.HealthStatus == HealthUnhealthy
}

// NeedsHealthCheck returns true if the link needs a health check.
func (l *UserLink) NeedsHealthCheck(maxAge time.Duration) bool {
	if l.HealthCheckedAt == nil {
		return true
	}
	return time.Since(*l.HealthCheckedAt) > maxAge
}
