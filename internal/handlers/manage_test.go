package handlers

import (
	"testing"

	"github.com/google/uuid"

	"golinks/internal/models"
)

func TestCanManageLink(t *testing.T) {
	orgID := uuid.New()
	otherOrgID := uuid.New()
	userID := uuid.New()
	otherUserID := uuid.New()

	tests := []struct {
		name     string
		user     *models.User
		link     *models.Link
		expected bool
	}{
		{
			name:     "admin can manage global link",
			user:     &models.User{Role: models.RoleAdmin},
			link:     &models.Link{Scope: models.ScopeGlobal},
			expected: true,
		},
		{
			name:     "admin can manage org link",
			user:     &models.User{Role: models.RoleAdmin},
			link:     &models.Link{Scope: models.ScopeOrg, OrganizationID: &orgID},
			expected: true,
		},
		{
			name:     "global mod can manage global link",
			user:     &models.User{Role: models.RoleGlobalMod},
			link:     &models.Link{Scope: models.ScopeGlobal},
			expected: true,
		},
		{
			name:     "global mod can manage org link",
			user:     &models.User{Role: models.RoleGlobalMod},
			link:     &models.Link{Scope: models.ScopeOrg, OrganizationID: &orgID},
			expected: true,
		},
		{
			name:     "org mod can manage own org link",
			user:     &models.User{Role: models.RoleOrgMod, OrganizationID: &orgID},
			link:     &models.Link{Scope: models.ScopeOrg, OrganizationID: &orgID},
			expected: true,
		},
		{
			name:     "org mod cannot manage other org link",
			user:     &models.User{Role: models.RoleOrgMod, OrganizationID: &otherOrgID},
			link:     &models.Link{Scope: models.ScopeOrg, OrganizationID: &orgID},
			expected: false,
		},
		{
			name:     "org mod cannot manage global link",
			user:     &models.User{Role: models.RoleOrgMod, OrganizationID: &orgID},
			link:     &models.Link{Scope: models.ScopeGlobal},
			expected: false,
		},
		{
			name:     "author can manage own link",
			user:     &models.User{ID: userID, Role: models.RoleUser},
			link:     &models.Link{Scope: models.ScopeGlobal, CreatedBy: &userID},
			expected: true,
		},
		{
			name:     "user cannot manage link by another author",
			user:     &models.User{ID: userID, Role: models.RoleUser},
			link:     &models.Link{Scope: models.ScopeGlobal, CreatedBy: &otherUserID},
			expected: false,
		},
		{
			name:     "regular user cannot manage unowned global link",
			user:     &models.User{Role: models.RoleUser},
			link:     &models.Link{Scope: models.ScopeGlobal},
			expected: false,
		},
		{
			name:     "regular user in org cannot manage org link they did not create",
			user:     &models.User{Role: models.RoleUser, OrganizationID: &orgID},
			link:     &models.Link{Scope: models.ScopeOrg, OrganizationID: &orgID},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canManageLink(tt.user, tt.link); got != tt.expected {
				t.Errorf("canManageLink() = %v, want %v", got, tt.expected)
			}
		})
	}
}
