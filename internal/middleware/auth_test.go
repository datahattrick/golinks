package middleware

import "testing"

func TestExtractUsernameFromCN(t *testing.T) {
	tests := []struct {
		name     string
		cn       string
		expected string
	}{
		{
			name:     "standard format with name and username",
			cn:       "Heath Taylor (heatht)",
			expected: "heatht",
		},
		{
			name:     "username only in parentheses",
			cn:       "(admin)",
			expected: "admin",
		},
		{
			name:     "name with middle initial",
			cn:       "John Q. Public (jpublic)",
			expected: "jpublic",
		},
		{
			name:     "extra spaces around username",
			cn:       "Test User ( testuser )",
			expected: "testuser",
		},
		{
			name:     "no parentheses",
			cn:       "Just A Name",
			expected: "",
		},
		{
			name:     "empty string",
			cn:       "",
			expected: "",
		},
		{
			name:     "parentheses in middle not at end",
			cn:       "Name (part) More",
			expected: "",
		},
		{
			name:     "multiple parentheses takes last",
			cn:       "Name (first) (second)",
			expected: "second",
		},
		{
			name:     "nested parentheses returns empty (invalid format)",
			cn:       "Name ((nested))",
			expected: "",
		},
		{
			name:     "special characters in username",
			cn:       "User Name (user-name_123)",
			expected: "user-name_123",
		},
		{
			name:     "unicode name",
			cn:       "José García (jgarcia)",
			expected: "jgarcia",
		},
		{
			name:     "trailing whitespace after parentheses",
			cn:       "Test User (testuser)   ",
			expected: "testuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractUsernameFromCN(tt.cn)
			if got != tt.expected {
				t.Errorf("extractUsernameFromCN(%q) = %q, want %q", tt.cn, got, tt.expected)
			}
		})
	}
}
