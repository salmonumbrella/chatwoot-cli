package validation

import (
	"net"
	"strings"
	"testing"
)

func TestValidateChatwootURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantError bool
		errorText string
	}{
		// Valid URLs
		{
			name:      "valid https URL",
			url:       "https://chatwoot.example.com",
			wantError: false,
		},
		{
			name:      "valid http URL",
			url:       "http://chatwoot.example.com",
			wantError: false,
		},
		{
			name:      "valid URL with port",
			url:       "https://chatwoot.example.com:8080",
			wantError: false,
		},
		{
			name:      "valid URL with path",
			url:       "https://chatwoot.example.com/api",
			wantError: false,
		},

		// Empty URL
		{
			name:      "empty URL",
			url:       "",
			wantError: true,
			errorText: "cannot be empty",
		},

		// Invalid schemes
		{
			name:      "file scheme",
			url:       "file:///etc/passwd",
			wantError: true,
			errorText: "only http and https are allowed",
		},
		{
			name:      "javascript scheme",
			url:       "javascript:alert(1)",
			wantError: true,
			errorText: "only http and https are allowed",
		},
		{
			name:      "ftp scheme",
			url:       "ftp://example.com",
			wantError: true,
			errorText: "only http and https are allowed",
		},
		{
			name:      "data scheme",
			url:       "data:text/html,<script>alert(1)</script>",
			wantError: true,
			errorText: "only http and https are allowed",
		},
		{
			name:      "gopher scheme",
			url:       "gopher://example.com",
			wantError: true,
			errorText: "only http and https are allowed",
		},

		// Localhost variants
		{
			name:      "localhost",
			url:       "http://localhost",
			wantError: true,
			errorText: "localhost URLs are not allowed",
		},
		{
			name:      "localhost with port",
			url:       "http://localhost:8080",
			wantError: true,
			errorText: "localhost URLs are not allowed",
		},
		{
			name:      "127.0.0.1",
			url:       "http://127.0.0.1",
			wantError: true,
			errorText: "localhost URLs are not allowed",
		},
		{
			name:      "127.0.0.1 with port",
			url:       "http://127.0.0.1:3000",
			wantError: true,
			errorText: "localhost URLs are not allowed",
		},
		{
			name:      "IPv6 localhost",
			url:       "http://[::1]",
			wantError: true,
			errorText: "localhost URLs are not allowed",
		},
		{
			name:      "0.0.0.0",
			url:       "http://0.0.0.0",
			wantError: true,
			errorText: "localhost URLs are not allowed",
		},
		{
			name:      "localhost subdomain",
			url:       "http://api.localhost",
			wantError: true,
			errorText: "localhost URLs are not allowed",
		},

		// Private IP ranges (RFC1918)
		{
			name:      "10.0.0.1",
			url:       "http://10.0.0.1",
			wantError: true,
			errorText: "private IP addresses are not allowed",
		},
		{
			name:      "172.16.0.1",
			url:       "http://172.16.0.1",
			wantError: true,
			errorText: "private IP addresses are not allowed",
		},
		{
			name:      "192.168.1.1",
			url:       "http://192.168.1.1",
			wantError: true,
			errorText: "private IP addresses are not allowed",
		},
		{
			name:      "192.168.0.100 with port",
			url:       "http://192.168.0.100:8080",
			wantError: true,
			errorText: "private IP addresses are not allowed",
		},

		// Cloud metadata endpoints
		{
			name:      "AWS metadata IP",
			url:       "http://169.254.169.254",
			wantError: true,
			errorText: "cloud metadata endpoints are not allowed",
		},
		{
			name:      "AWS metadata IP with path",
			url:       "http://169.254.169.254/latest/meta-data/",
			wantError: true,
			errorText: "cloud metadata endpoints are not allowed",
		},
		{
			name:      "GCP metadata hostname",
			url:       "http://metadata.google.internal",
			wantError: true,
			errorText: "cloud metadata endpoints are not allowed",
		},
		{
			name:      "metadata hostname",
			url:       "http://metadata",
			wantError: true,
			errorText: "cloud metadata endpoints are not allowed",
		},

		// Link-local addresses
		{
			name:      "link-local IPv4",
			url:       "http://169.254.1.1",
			wantError: true,
			errorText: "link-local IP addresses are not allowed",
		},
		{
			name:      "link-local IPv6",
			url:       "http://[fe80::1]",
			wantError: true,
			errorText: "link-local IP addresses are not allowed",
		},

		// Shared address space (RFC6598)
		{
			name:      "shared address space",
			url:       "http://100.64.0.1",
			wantError: true,
			errorText: "private IP addresses are not allowed",
		},

		// IPv6 private ranges
		{
			name:      "IPv6 unique local",
			url:       "http://[fc00::1]",
			wantError: true,
			errorText: "private IP addresses are not allowed",
		},
		{
			name:      "IPv6 unspecified",
			url:       "http://[::]",
			wantError: true,
			errorText: "localhost URLs are not allowed",
		},

		// Malformed URLs
		{
			name:      "no scheme",
			url:       "chatwoot.example.com",
			wantError: true,
			errorText: "only http and https are allowed",
		},
		{
			name:      "invalid URL format",
			url:       "ht!tp://invalid",
			wantError: true,
			errorText: "invalid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateChatwootURL(tt.url)
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateChatwootURL(%q) expected error containing %q, got nil", tt.url, tt.errorText)
					return
				}
				if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("ValidateChatwootURL(%q) error = %v, want error containing %q", tt.url, err, tt.errorText)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateChatwootURL(%q) unexpected error: %v", tt.url, err)
				}
			}
		})
	}
}

func TestValidateChatwootURL_AllowPrivate(t *testing.T) {
	SetAllowPrivate(true)
	defer SetAllowPrivate(false)

	tests := []struct {
		name      string
		url       string
		wantError bool
	}{
		{name: "localhost allowed", url: "http://localhost:3000", wantError: false},
		{name: "private IPv4 allowed", url: "http://192.168.0.10", wantError: false},
		{name: "loopback IPv6 allowed", url: "http://[::1]", wantError: false},
		{name: "metadata still blocked", url: "http://169.254.169.254", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateChatwootURL(tt.url)
			if tt.wantError && err == nil {
				t.Fatalf("expected error for %s, got nil", tt.url)
			}
			if !tt.wantError && err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.url, err)
			}
		})
	}
}

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		hostname string
		want     bool
	}{
		{"localhost", true},
		{"LOCALHOST", true},
		{"Localhost", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"0.0.0.0", true},
		{"::", true},
		{"api.localhost", true},
		{"example.localhost", true},
		{"example.com", false},
		{"192.168.1.1", false},
		{"chatwoot.local", false},
	}

	for _, tt := range tests {
		t.Run(tt.hostname, func(t *testing.T) {
			got := isLocalhost(tt.hostname)
			if got != tt.want {
				t.Errorf("isLocalhost(%q) = %v, want %v", tt.hostname, got, tt.want)
			}
		})
	}
}

func TestIsCloudMetadata(t *testing.T) {
	tests := []struct {
		hostname string
		want     bool
	}{
		{"169.254.169.254", true},
		{"metadata.google.internal", true},
		{"metadata", true},
		{"instance-data", true},
		{"fd00:ec2::254", true},
		{"api.metadata.google.internal", true},
		{"example.com", false},
		{"metadata.example.com", false},
		{"169.254.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.hostname, func(t *testing.T) {
			got := isCloudMetadata(tt.hostname)
			if got != tt.want {
				t.Errorf("isCloudMetadata(%q) = %v, want %v", tt.hostname, got, tt.want)
			}
		})
	}
}

func TestValidateIPAddress(t *testing.T) {
	tests := []struct {
		name      string
		ip        string
		wantError bool
		errorText string
	}{
		// Public IPs (should pass)
		{
			name:      "public IPv4",
			ip:        "8.8.8.8",
			wantError: false,
		},
		{
			name:      "public IPv4 2",
			ip:        "1.1.1.1",
			wantError: false,
		},

		// Loopback
		{
			name:      "IPv4 loopback",
			ip:        "127.0.0.1",
			wantError: true,
			errorText: "loopback",
		},
		{
			name:      "IPv6 loopback",
			ip:        "::1",
			wantError: true,
			errorText: "loopback",
		},

		// Private IPs
		{
			name:      "10.x.x.x",
			ip:        "10.0.0.1",
			wantError: true,
			errorText: "private",
		},
		{
			name:      "172.16.x.x",
			ip:        "172.16.0.1",
			wantError: true,
			errorText: "private",
		},
		{
			name:      "192.168.x.x",
			ip:        "192.168.1.1",
			wantError: true,
			errorText: "private",
		},

		// Link-local
		{
			name:      "link-local IPv4",
			ip:        "169.254.1.1",
			wantError: true,
			errorText: "link-local",
		},
		{
			name:      "link-local IPv6",
			ip:        "fe80::1",
			wantError: true,
			errorText: "link-local",
		},

		// Cloud metadata
		{
			name:      "AWS metadata",
			ip:        "169.254.169.254",
			wantError: true,
			errorText: "cloud metadata",
		},

		// Unspecified
		{
			name:      "IPv4 unspecified",
			ip:        "0.0.0.0",
			wantError: true,
			errorText: "unspecified IP addresses are not allowed",
		},
		{
			name:      "IPv6 unspecified",
			ip:        "::",
			wantError: true,
			errorText: "unspecified IP addresses are not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIP(t, tt.ip)
			err := validateIPAddress(ip)
			if tt.wantError {
				if err == nil {
					t.Errorf("validateIPAddress(%q) expected error containing %q, got nil", tt.ip, tt.errorText)
					return
				}
				if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("validateIPAddress(%q) error = %v, want error containing %q", tt.ip, err, tt.errorText)
				}
			} else {
				if err != nil {
					t.Errorf("validateIPAddress(%q) unexpected error: %v", tt.ip, err)
				}
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
		// Public IPs
		{"Google DNS", "8.8.8.8", false},
		{"Cloudflare DNS", "1.1.1.1", false},
		{"Public IP", "93.184.216.34", false},

		// RFC1918 Private ranges
		{"10.0.0.0/8 start", "10.0.0.0", true},
		{"10.0.0.0/8 mid", "10.128.0.1", true},
		{"10.0.0.0/8 end", "10.255.255.255", true},
		{"172.16.0.0/12 start", "172.16.0.0", true},
		{"172.16.0.0/12 mid", "172.20.0.1", true},
		{"172.16.0.0/12 end", "172.31.255.255", true},
		{"192.168.0.0/16 start", "192.168.0.0", true},
		{"192.168.0.0/16 mid", "192.168.1.1", true},
		{"192.168.0.0/16 end", "192.168.255.255", true},

		// Shared address space (RFC6598)
		{"100.64.0.0/10", "100.64.0.1", true},

		// Link-local (RFC3927)
		{"169.254.0.0/16", "169.254.1.1", true},
		{"169.254.169.254", "169.254.169.254", true},

		// IPv6 private ranges
		{"fc00::/7 unique local", "fc00::1", true},
		{"fe80::/10 link local", "fe80::1", true},
		{"::1 loopback", "::1", true},
		{":: unspecified", "::", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIP(t, tt.ip)
			got := isPrivateIP(ip)
			if got != tt.want {
				t.Errorf("isPrivateIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

// parseIP is a test helper that parses an IP and fails the test if invalid
func parseIP(t *testing.T, s string) net.IP {
	t.Helper()
	ip := net.ParseIP(s)
	if ip == nil {
		t.Fatalf("failed to parse IP: %s", s)
	}
	return ip
}

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantError bool
		errorText string
	}{
		// Valid URLs (including localhost for development)
		{
			name:      "valid https URL",
			url:       "https://webhook.example.com",
			wantError: false,
		},
		{
			name:      "valid http URL",
			url:       "http://webhook.example.com",
			wantError: false,
		},
		{
			name:      "valid URL with port",
			url:       "https://webhook.example.com:8080",
			wantError: false,
		},
		{
			name:      "valid URL with path",
			url:       "https://webhook.example.com/api/webhook",
			wantError: false,
		},
		{
			name:      "localhost allowed for development",
			url:       "http://localhost",
			wantError: false,
		},
		{
			name:      "localhost with port",
			url:       "http://localhost:8080",
			wantError: false,
		},
		{
			name:      "127.0.0.1 allowed for development",
			url:       "http://127.0.0.1",
			wantError: false,
		},
		{
			name:      "127.0.0.1 with port",
			url:       "http://127.0.0.1:3000",
			wantError: false,
		},
		{
			name:      "IPv6 localhost allowed",
			url:       "http://[::1]",
			wantError: false,
		},
		{
			name:      "0.0.0.0 allowed for development",
			url:       "http://0.0.0.0:8080",
			wantError: false,
		},
		{
			name:      "localhost subdomain allowed",
			url:       "http://api.localhost",
			wantError: false,
		},

		// Empty URL
		{
			name:      "empty URL",
			url:       "",
			wantError: true,
			errorText: "cannot be empty",
		},

		// Invalid schemes
		{
			name:      "file scheme",
			url:       "file:///etc/passwd",
			wantError: true,
			errorText: "only http and https are allowed",
		},
		{
			name:      "javascript scheme",
			url:       "javascript:alert(1)",
			wantError: true,
			errorText: "only http and https are allowed",
		},
		{
			name:      "ftp scheme",
			url:       "ftp://example.com",
			wantError: true,
			errorText: "only http and https are allowed",
		},
		{
			name:      "gopher scheme",
			url:       "gopher://example.com",
			wantError: true,
			errorText: "only http and https are allowed",
		},

		// Private IP ranges (still blocked for webhooks)
		{
			name:      "10.0.0.1",
			url:       "http://10.0.0.1",
			wantError: true,
			errorText: "private IP addresses are not allowed",
		},
		{
			name:      "172.16.0.1",
			url:       "http://172.16.0.1",
			wantError: true,
			errorText: "private IP addresses are not allowed",
		},
		{
			name:      "192.168.1.1",
			url:       "http://192.168.1.1",
			wantError: true,
			errorText: "private IP addresses are not allowed",
		},
		{
			name:      "192.168.0.100 with port",
			url:       "http://192.168.0.100:8080",
			wantError: true,
			errorText: "private IP addresses are not allowed",
		},

		// Cloud metadata endpoints (always blocked)
		{
			name:      "AWS metadata IP",
			url:       "http://169.254.169.254",
			wantError: true,
			errorText: "cloud metadata endpoints are not allowed",
		},
		{
			name:      "AWS metadata IP with path",
			url:       "http://169.254.169.254/latest/meta-data/",
			wantError: true,
			errorText: "cloud metadata endpoints are not allowed",
		},
		{
			name:      "GCP metadata hostname",
			url:       "http://metadata.google.internal",
			wantError: true,
			errorText: "cloud metadata endpoints are not allowed",
		},
		{
			name:      "metadata hostname",
			url:       "http://metadata",
			wantError: true,
			errorText: "cloud metadata endpoints are not allowed",
		},

		// Link-local addresses (excluding localhost range)
		{
			name:      "link-local IPv4 (not metadata)",
			url:       "http://169.254.1.1",
			wantError: true,
			errorText: "link-local IP addresses are not allowed",
		},
		{
			name:      "link-local IPv6",
			url:       "http://[fe80::1]",
			wantError: true,
			errorText: "link-local IP addresses are not allowed",
		},

		// Shared address space (RFC6598)
		{
			name:      "shared address space",
			url:       "http://100.64.0.1",
			wantError: true,
			errorText: "private IP addresses are not allowed",
		},

		// IPv6 private ranges
		{
			name:      "IPv6 unique local",
			url:       "http://[fc00::1]",
			wantError: true,
			errorText: "private IP addresses are not allowed",
		},

		// Malformed URLs
		{
			name:      "no scheme",
			url:       "webhook.example.com",
			wantError: true,
			errorText: "only http and https are allowed",
		},
		{
			name:      "invalid URL format",
			url:       "ht!tp://invalid",
			wantError: true,
			errorText: "invalid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateWebhookURL(%q) expected error containing %q, got nil", tt.url, tt.errorText)
					return
				}
				if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("ValidateWebhookURL(%q) error = %v, want error containing %q", tt.url, err, tt.errorText)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateWebhookURL(%q) unexpected error: %v", tt.url, err)
				}
			}
		})
	}
}

func TestValidateWebhookIPAddress(t *testing.T) {
	tests := []struct {
		name      string
		ip        string
		wantError bool
		errorText string
	}{
		// Public IPs (should pass)
		{
			name:      "public IPv4",
			ip:        "8.8.8.8",
			wantError: false,
		},
		{
			name:      "public IPv4 2",
			ip:        "1.1.1.1",
			wantError: false,
		},

		// Loopback (allowed for webhooks)
		{
			name:      "IPv4 loopback",
			ip:        "127.0.0.1",
			wantError: false,
		},
		{
			name:      "IPv6 loopback",
			ip:        "::1",
			wantError: false,
		},

		// Unspecified (allowed for webhooks)
		{
			name:      "IPv4 unspecified",
			ip:        "0.0.0.0",
			wantError: false,
		},
		{
			name:      "IPv6 unspecified",
			ip:        "::",
			wantError: false,
		},

		// Private IPs (still blocked)
		{
			name:      "10.x.x.x",
			ip:        "10.0.0.1",
			wantError: true,
			errorText: "private",
		},
		{
			name:      "172.16.x.x",
			ip:        "172.16.0.1",
			wantError: true,
			errorText: "private",
		},
		{
			name:      "192.168.x.x",
			ip:        "192.168.1.1",
			wantError: true,
			errorText: "private",
		},

		// Link-local (blocked except loopback which is handled separately)
		{
			name:      "link-local IPv4",
			ip:        "169.254.1.1",
			wantError: true,
			errorText: "link-local",
		},
		{
			name:      "link-local IPv6",
			ip:        "fe80::1",
			wantError: true,
			errorText: "link-local",
		},

		// Cloud metadata (always blocked)
		{
			name:      "AWS metadata",
			ip:        "169.254.169.254",
			wantError: true,
			errorText: "cloud metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIP(t, tt.ip)
			err := validateWebhookIPAddress(ip)
			if tt.wantError {
				if err == nil {
					t.Errorf("validateWebhookIPAddress(%q) expected error containing %q, got nil", tt.ip, tt.errorText)
					return
				}
				if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("validateWebhookIPAddress(%q) error = %v, want error containing %q", tt.ip, err, tt.errorText)
				}
			} else {
				if err != nil {
					t.Errorf("validateWebhookIPAddress(%q) unexpected error: %v", tt.ip, err)
				}
			}
		})
	}
}
