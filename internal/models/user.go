package models

import (
	"time"

	"github.com/google/uuid"
)

// Role constants
const (
	RoleUser      = "user"
	RoleOrgMod    = "org_mod"
	RoleGlobalMod = "global_mod"
	RoleAdmin     = "admin"
)

// User represents a user authenticated via OIDC.
type User struct {
	ID             uuid.UUID  `json:"id"`
	Sub            string     `json:"sub"`             // OIDC subject identifier
	Username       string     `json:"username"`        // Extracted from PKI CN e.g. "heatht" from "Heath Taylor (heatht)"
	Email          string     `json:"email"`
	Name           string     `json:"name"`
	Picture        string     `json:"picture"`
	Role           string     `json:"role"`            // user, org_mod, global_mod, admin
	OrganizationID     *uuid.UUID `json:"organization_id"`      // Optional org membership (legacy, use GroupMemberships)
	FallbackRedirectID *uuid.UUID `json:"fallback_redirect_id"` // User's chosen fallback redirect (nil = no fallback)
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`

	// Populated by auth middleware - group memberships for tier-based resolution
	GroupMemberships []UserGroupMembership `json:"group_memberships,omitempty"`
}

// IsAdmin returns true if the user is an admin.
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsGlobalMod returns true if the user can moderate global links.
func (u *User) IsGlobalMod() bool {
	return u.Role == RoleGlobalMod || u.Role == RoleAdmin
}

// IsOrgMod returns true if the user can moderate org links.
func (u *User) IsOrgMod() bool {
	return u.Role == RoleOrgMod || u.Role == RoleGlobalMod || u.Role == RoleAdmin
}

// CanModerateOrg returns true if the user can moderate links for a specific org.
func (u *User) CanModerateOrg(orgID uuid.UUID) bool {
	if u.IsGlobalMod() {
		return true
	}
	if u.Role == RoleOrgMod && u.OrganizationID != nil && *u.OrganizationID == orgID {
		return true
	}
	return false
}
