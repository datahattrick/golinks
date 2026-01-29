package models

import "testing"

func TestLink_IsPending(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"pending status", StatusPending, true},
		{"approved status", StatusApproved, false},
		{"rejected status", StatusRejected, false},
		{"empty status", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link := &Link{Status: tt.status}
			if got := link.IsPending(); got != tt.expected {
				t.Errorf("IsPending() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLink_IsApproved(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"pending status", StatusPending, false},
		{"approved status", StatusApproved, true},
		{"rejected status", StatusRejected, false},
		{"empty status", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link := &Link{Status: tt.status}
			if got := link.IsApproved(); got != tt.expected {
				t.Errorf("IsApproved() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLinkConstants(t *testing.T) {
	// Verify constants have expected values
	if ScopeGlobal != "global" {
		t.Errorf("ScopeGlobal = %q, want %q", ScopeGlobal, "global")
	}
	if ScopeOrg != "org" {
		t.Errorf("ScopeOrg = %q, want %q", ScopeOrg, "org")
	}
	if StatusPending != "pending" {
		t.Errorf("StatusPending = %q, want %q", StatusPending, "pending")
	}
	if StatusApproved != "approved" {
		t.Errorf("StatusApproved = %q, want %q", StatusApproved, "approved")
	}
	if StatusRejected != "rejected" {
		t.Errorf("StatusRejected = %q, want %q", StatusRejected, "rejected")
	}
}
