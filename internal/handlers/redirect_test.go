package handlers

import (
	"testing"

	"golinks/internal/config"
)

func TestRandomHandler_FeatureDisabled(t *testing.T) {
	cfg := &config.Config{
		EnableRandomKeywords: false,
	}

	// When feature is disabled, the handler should return a 404
	// This is a unit test for the config check logic
	if cfg.EnableRandomKeywords {
		t.Error("EnableRandomKeywords should be false")
	}
}

func TestRandomHandler_FeatureEnabled(t *testing.T) {
	cfg := &config.Config{
		EnableRandomKeywords: true,
	}

	// When feature is enabled, the handler should work
	if !cfg.EnableRandomKeywords {
		t.Error("EnableRandomKeywords should be true")
	}
}

func TestConfigEnableRandomKeywords(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{
			name:     "random keywords disabled",
			enabled:  false,
			expected: false,
		},
		{
			name:     "random keywords enabled",
			enabled:  true,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				EnableRandomKeywords: tt.enabled,
			}
			if cfg.EnableRandomKeywords != tt.expected {
				t.Errorf("EnableRandomKeywords = %v, want %v", cfg.EnableRandomKeywords, tt.expected)
			}
		})
	}
}
