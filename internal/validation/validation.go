package validation

import (
	"net"
	"net/url"
	"regexp"
	"strings"
)

// KeywordPattern defines the valid keyword format: alphanumeric, hyphens, underscores.
var KeywordPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidateKeyword checks if a keyword matches the allowed pattern.
func ValidateKeyword(keyword string) bool {
	if keyword == "" || len(keyword) > 100 {
		return false
	}
	return KeywordPattern.MatchString(keyword)
}

// NormalizeKeyword lowercases a keyword so lookups are case-insensitive.
func NormalizeKeyword(keyword string) string {
	return strings.ToLower(keyword)
}

// ValidateURL checks if a URL is valid and uses an allowed scheme (http/https only).
// This prevents javascript:, data:, vbscript:, and other dangerous URL schemes.
func ValidateURL(urlStr string) (bool, string) {
	if urlStr == "" {
		return false, "URL is required"
	}

	// Parse the URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return false, "Invalid URL format"
	}

	// Check scheme - only allow http and https
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return false, "URL must use http:// or https:// scheme"
	}

	// Ensure host is present
	if u.Host == "" {
		return false, "URL must have a valid host"
	}

	return true, ""
}

// IsPrivateIP checks if an IP address is in a private/reserved range.
// Used to prevent SSRF attacks against internal networks.
func IsPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// Check for loopback
	if ip.IsLoopback() {
		return true
	}

	// Check for link-local
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Check for private ranges
	if ip.IsPrivate() {
		return true
	}

	// Check for unspecified (0.0.0.0 or ::)
	if ip.IsUnspecified() {
		return true
	}

	// Cloud metadata IP (AWS, GCP, Azure)
	// 169.254.169.254 is the standard metadata endpoint
	metadataIP := net.ParseIP("169.254.169.254")
	if ip.Equal(metadataIP) {
		return true
	}

	// Additional cloud metadata endpoints
	// Azure also uses 168.63.129.16
	azureMetadata := net.ParseIP("168.63.129.16")
	if ip.Equal(azureMetadata) {
		return true
	}

	return false
}

// IsPrivateHost checks if a hostname resolves to a private IP address.
// Returns true if the host is private/blocked, false if it's safe to access.
func IsPrivateHost(host string) (bool, error) {
	// Remove port if present
	hostname := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		hostname = h
	}

	// Resolve the hostname
	ips, err := net.LookupIP(hostname)
	if err != nil {
		// If we can't resolve, be conservative and block
		return true, err
	}

	// Check all resolved IPs
	for _, ip := range ips {
		if IsPrivateIP(ip) {
			return true, nil
		}
	}

	return false, nil
}

// ValidateURLForHealthCheck validates a URL is safe for health checking.
// Blocks private IPs, localhost, and cloud metadata endpoints.
func ValidateURLForHealthCheck(urlStr string) (bool, string) {
	// First do basic URL validation
	valid, msg := ValidateURL(urlStr)
	if !valid {
		return false, msg
	}

	u, _ := url.Parse(urlStr)

	// Check if host resolves to private IP
	isPrivate, err := IsPrivateHost(u.Host)
	if err != nil {
		return false, "Cannot resolve hostname"
	}
	if isPrivate {
		return false, "URL points to a private or reserved IP address"
	}

	return true, ""
}
