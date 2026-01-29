package models

import (
	"time"

	"github.com/google/uuid"
)

// Tier constants for link resolution priority.
// Higher tier = higher priority.
const (
	TierGlobal   = 0   // Global links (stored in links table with scope='global')
	TierPersonal = 100 // Personal links (stored in user_links table)
	// Group tiers are 1-99 (stored in groups table, linked via group_links)
)

// Group membership role constants.
const (
	GroupRoleMember    = "member"
	GroupRoleModerator = "moderator"
	GroupRoleAdmin     = "admin"
)

// Group represents a group in the tier-based hierarchy.
// Groups have tiers between 1-99 (0=global, 100=personal are implicit).
type Group struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	Tier      int        `json:"tier"` // 1-99, higher = higher priority
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// UserGroupMembership represents a user's membership in a group.
type UserGroupMembership struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	GroupID   uuid.UUID `json:"group_id"`
	IsPrimary bool      `json:"is_primary"` // Primary group for tie-breaking
	Role      string    `json:"role"`       // member, moderator, admin
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Populated by joins
	Group *Group `json:"group,omitempty"`
}

// GroupLink represents a link scoped to a group.
type GroupLink struct {
	ID              uuid.UUID  `json:"id"`
	GroupID         uuid.UUID  `json:"group_id"`
	Keyword         string     `json:"keyword"`
	URL             string     `json:"url"`
	Description     string     `json:"description,omitempty"`
	Status          string     `json:"status"` // pending, approved, rejected
	ClickCount      int        `json:"click_count"`
	CreatedBy       *uuid.UUID `json:"created_by,omitempty"`
	SubmittedBy     *uuid.UUID `json:"submitted_by,omitempty"`
	ReviewedBy      *uuid.UUID `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	HealthStatus    string     `json:"health_status"`
	HealthCheckedAt *time.Time `json:"health_checked_at,omitempty"`
	HealthError     *string    `json:"health_error,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	// Populated by joins
	Group *Group `json:"group,omitempty"`
}

// ResolvedLink represents the result of keyword resolution across all tiers.
// Used to return the winning link from the resolution query.
type ResolvedLink struct {
	ID        uuid.UUID `json:"id"`
	URL       string    `json:"url"`
	Tier      int       `json:"tier"`      // 0=global, 1-99=group, 100=personal
	IsPrimary bool      `json:"is_primary"` // For tie-breaking at same tier
	Source    string    `json:"source"`    // "global", "group", "personal"
}

// IsModerator returns true if the membership has moderator or admin role.
func (m *UserGroupMembership) IsModerator() bool {
	return m.Role == GroupRoleModerator || m.Role == GroupRoleAdmin
}

// IsAdmin returns true if the membership has admin role.
func (m *UserGroupMembership) IsAdmin() bool {
	return m.Role == GroupRoleAdmin
}
