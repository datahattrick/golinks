package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestUser_IsAdmin(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		expected bool
	}{
		{"admin user", RoleAdmin, true},
		{"global mod", RoleGlobalMod, false},
		{"org mod", RoleOrgMod, false},
		{"regular user", RoleUser, false},
		{"empty role", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Role: tt.role}
			if got := user.IsAdmin(); got != tt.expected {
				t.Errorf("IsAdmin() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUser_IsGlobalMod(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		expected bool
	}{
		{"admin user", RoleAdmin, true},
		{"global mod", RoleGlobalMod, true},
		{"org mod", RoleOrgMod, false},
		{"regular user", RoleUser, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Role: tt.role}
			if got := user.IsGlobalMod(); got != tt.expected {
				t.Errorf("IsGlobalMod() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUser_IsOrgMod(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		expected bool
	}{
		{"admin user", RoleAdmin, true},
		{"global mod", RoleGlobalMod, true},
		{"org mod", RoleOrgMod, true},
		{"regular user", RoleUser, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Role: tt.role}
			if got := user.IsOrgMod(); got != tt.expected {
				t.Errorf("IsOrgMod() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUser_CanModerateOrg(t *testing.T) {
	orgID := uuid.New()
	otherOrgID := uuid.New()

	tests := []struct {
		name           string
		role           string
		userOrgID      *uuid.UUID
		targetOrgID    uuid.UUID
		expected       bool
	}{
		{"admin can moderate any org", RoleAdmin, nil, orgID, true},
		{"global mod can moderate any org", RoleGlobalMod, nil, orgID, true},
		{"org mod can moderate own org", RoleOrgMod, &orgID, orgID, true},
		{"org mod cannot moderate other org", RoleOrgMod, &otherOrgID, orgID, false},
		{"org mod with no org cannot moderate", RoleOrgMod, nil, orgID, false},
		{"regular user cannot moderate", RoleUser, &orgID, orgID, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				Role:           tt.role,
				OrganizationID: tt.userOrgID,
			}
			if got := user.CanModerateOrg(tt.targetOrgID); got != tt.expected {
				t.Errorf("CanModerateOrg() = %v, want %v", got, tt.expected)
			}
		})
	}
}
