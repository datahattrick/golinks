package validation

import (
	"net"
	"testing"
)

func TestValidateKeyword(t *testing.T) {
	tests := []struct {
		name    string
		keyword string
		want    bool
	}{
		{"valid alphanumeric", "abc123", true},
		{"valid with hyphen", "my-link", true},
		{"valid with underscore", "my_link", true},
		{"valid mixed", "My-Link_123", true},
		{"empty string", "", false},
		{"too long", string(make([]byte, 101)), false},
		{"max length", string(make([]byte, 100)), false}, // all zeros, not alphanumeric
		{"contains space", "my link", false},
		{"contains dot", "my.link", false},
		{"contains slash", "my/link", false},
		{"contains backslash", "my\\link", false},
		{"path traversal attempt", "../etc/passwd", false},
		{"url encoded", "my%20link", false},
		{"special chars", "link@#$", false},
		{"unicode", "日本語", false},
		{"single char", "a", true},
		{"numbers only", "12345", true},
		{"hyphens only", "---", true},
		{"underscores only", "___", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateKeyword(tt.keyword)
			if got != tt.want {
				t.Errorf("ValidateKeyword(%q) = %v, want %v", tt.keyword, got, tt.want)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		valid   bool
		wantMsg string
	}{
		{"valid https", "https://example.com", true, ""},
		{"valid http", "http://example.com", true, ""},
		{"valid with path", "https://example.com/path/to/page", true, ""},
		{"valid with query", "https://example.com?foo=bar", true, ""},
		{"valid with port", "https://example.com:8080", true, ""},
		{"empty string", "", false, "URL is required"},
		{"javascript scheme", "javascript:alert(1)", false, "URL must use http:// or https:// scheme"},
		{"data scheme", "data:text/html,<script>alert(1)</script>", false, "URL must use http:// or https:// scheme"},
		{"vbscript scheme", "vbscript:msgbox", false, "URL must use http:// or https:// scheme"},
		{"file scheme", "file:///etc/passwd", false, "URL must use http:// or https:// scheme"},
		{"ftp scheme", "ftp://example.com", false, "URL must use http:// or https:// scheme"},
		{"no scheme", "example.com", false, "URL must use http:// or https:// scheme"},
		{"relative url", "/path/to/page", false, "URL must use http:// or https:// scheme"},
		{"uppercase scheme", "HTTPS://example.com", true, ""},
		{"mixed case scheme", "HtTpS://example.com", true, ""},
		{"scheme only", "https://", false, "URL must have a valid host"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, msg := ValidateURL(tt.url)
			if valid != tt.valid {
				t.Errorf("ValidateURL(%q) valid = %v, want %v", tt.url, valid, tt.valid)
			}
			if !valid && msg != tt.wantMsg {
				t.Errorf("ValidateURL(%q) msg = %q, want %q", tt.url, msg, tt.wantMsg)
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		// Loopback addresses
		{"localhost IPv4", "127.0.0.1", true},
		{"localhost IPv4 other", "127.0.0.2", true},
		{"localhost IPv6", "::1", true},

		// Private ranges
		{"10.x.x.x range", "10.0.0.1", true},
		{"10.x.x.x range max", "10.255.255.255", true},
		{"172.16.x.x range", "172.16.0.1", true},
		{"172.31.x.x range", "172.31.255.255", true},
		{"192.168.x.x range", "192.168.0.1", true},
		{"192.168.x.x range max", "192.168.255.255", true},

		// Link-local
		{"link-local IPv4", "169.254.1.1", true},
		{"link-local IPv6", "fe80::1", true},

		// Cloud metadata endpoints
		{"AWS/GCP metadata", "169.254.169.254", true},
		{"Azure metadata", "168.63.129.16", true},

		// Unspecified
		{"unspecified IPv4", "0.0.0.0", true},
		{"unspecified IPv6", "::", true},

		// Public IPs (should not be blocked)
		{"Google DNS", "8.8.8.8", false},
		{"Cloudflare DNS", "1.1.1.1", false},
		{"random public IP", "203.0.113.1", false},
		{"public IPv6", "2001:4860:4860::8888", false},

		// Edge cases
		{"nil IP", "", false}, // Will be handled specially
		{"172.15.x.x not private", "172.15.255.255", false},
		{"172.32.x.x not private", "172.32.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ip net.IP
			if tt.ip != "" {
				ip = net.ParseIP(tt.ip)
			}
			got := IsPrivateIP(ip)
			if got != tt.want {
				t.Errorf("IsPrivateIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestValidateURLForHealthCheck(t *testing.T) {
	// Note: These tests may fail if DNS resolution behaves unexpectedly
	// In a real test environment, you might mock the DNS resolver

	tests := []struct {
		name    string
		url     string
		valid   bool
		wantMsg string
	}{
		// Invalid URL schemes (should fail basic validation)
		{"javascript scheme", "javascript:alert(1)", false, "URL must use http:// or https:// scheme"},
		{"empty url", "", false, "URL is required"},

		// URLs that would resolve to private IPs (these depend on DNS)
		{"localhost", "http://localhost", false, "URL points to a private or reserved IP address"},
		{"127.0.0.1", "http://127.0.0.1", false, "URL points to a private or reserved IP address"},
		{"loopback with port", "http://127.0.0.1:8080", false, "URL points to a private or reserved IP address"},

		// Private IP ranges in URL
		{"10.x range", "http://10.0.0.1", false, "URL points to a private or reserved IP address"},
		{"192.168.x range", "http://192.168.1.1", false, "URL points to a private or reserved IP address"},
		{"172.16.x range", "http://172.16.0.1", false, "URL points to a private or reserved IP address"},

		// Cloud metadata endpoints
		{"AWS metadata", "http://169.254.169.254/latest/meta-data/", false, "URL points to a private or reserved IP address"},

		// Valid public URLs (these should pass if DNS works)
		// Commented out because they require network access
		// {"google", "https://www.google.com", true, ""},
		// {"example.com", "https://example.com", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, msg := ValidateURLForHealthCheck(tt.url)
			if valid != tt.valid {
				t.Errorf("ValidateURLForHealthCheck(%q) valid = %v, want %v (msg: %s)", tt.url, valid, tt.valid, msg)
			}
			if !valid && tt.wantMsg != "" && msg != tt.wantMsg {
				t.Errorf("ValidateURLForHealthCheck(%q) msg = %q, want %q", tt.url, msg, tt.wantMsg)
			}
		})
	}
}
