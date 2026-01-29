package handlers

import (
	"testing"

	"github.com/google/uuid"

	"golinks/internal/models"
)

func TestCanModerate(t *testing.T) {
	orgID := uuid.New()
	otherOrgID := uuid.New()

	tests := []struct {
		name     string
		user     *models.User
		link     *models.Link
		expected bool
	}{
		{
			name:     "admin can moderate global link",
			user:     &models.User{Role: models.RoleAdmin},
			link:     &models.Link{Scope: models.ScopeGlobal},
			expected: true,
		},
		{
			name:     "admin can moderate org link",
			user:     &models.User{Role: models.RoleAdmin},
			link:     &models.Link{Scope: models.ScopeOrg, OrganizationID: &orgID},
			expected: true,
		},
		{
			name:     "global mod can moderate global link",
			user:     &models.User{Role: models.RoleGlobalMod},
			link:     &models.Link{Scope: models.ScopeGlobal},
			expected: true,
		},
		{
			name:     "global mod can moderate org link",
			user:     &models.User{Role: models.RoleGlobalMod},
			link:     &models.Link{Scope: models.ScopeOrg, OrganizationID: &orgID},
			expected: true,
		},
		{
			name:     "org mod can moderate own org link",
			user:     &models.User{Role: models.RoleOrgMod, OrganizationID: &orgID},
			link:     &models.Link{Scope: models.ScopeOrg, OrganizationID: &orgID},
			expected: true,
		},
		{
			name:     "org mod cannot moderate other org link",
			user:     &models.User{Role: models.RoleOrgMod, OrganizationID: &otherOrgID},
			link:     &models.Link{Scope: models.ScopeOrg, OrganizationID: &orgID},
			expected: false,
		},
		{
			name:     "org mod cannot moderate global link",
			user:     &models.User{Role: models.RoleOrgMod, OrganizationID: &orgID},
			link:     &models.Link{Scope: models.ScopeGlobal},
			expected: false,
		},
		{
			name:     "regular user cannot moderate anything",
			user:     &models.User{Role: models.RoleUser},
			link:     &models.Link{Scope: models.ScopeGlobal},
			expected: false,
		},
		{
			name:     "regular user in org cannot moderate org link",
			user:     &models.User{Role: models.RoleUser, OrganizationID: &orgID},
			link:     &models.Link{Scope: models.ScopeOrg, OrganizationID: &orgID},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canModerate(tt.user, tt.link); got != tt.expected {
				t.Errorf("canModerate() = %v, want %v", got, tt.expected)
			}
		})
	}
}
