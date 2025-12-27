package validation

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// ValidateChatwootURL validates a Chatwoot instance URL to prevent SSRF attacks
func ValidateChatwootURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Only allow http and https schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: only http and https are allowed, got %q", parsedURL.Scheme)
	}

	// Extract hostname
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL must contain a hostname")
	}

	// Check for localhost variants
	if isLocalhost(hostname) {
		return fmt.Errorf("localhost URLs are not allowed")
	}

	// Check for cloud metadata endpoints
	if isCloudMetadata(hostname) {
		return fmt.Errorf("cloud metadata endpoints are not allowed")
	}

	// If it's an IP address, validate it
	if ip := net.ParseIP(hostname); ip != nil {
		if err := validateIPAddress(ip); err != nil {
			return err
		}
	} else {
		// For domain names, resolve and check all IPs
		if err := validateDomainName(hostname); err != nil {
			return err
		}
	}

	return nil
}

// isLocalhost checks for localhost variants
func isLocalhost(hostname string) bool {
	lowercase := strings.ToLower(hostname)
	localhostVariants := []string{
		"localhost",
		"127.0.0.1",
		"::1",
		"0.0.0.0",
		"::",
	}

	for _, variant := range localhostVariants {
		if lowercase == variant {
			return true
		}
	}

	// Check for localhost subdomains
	if strings.HasSuffix(lowercase, ".localhost") {
		return true
	}

	return false
}

// isCloudMetadata checks for cloud metadata endpoints
func isCloudMetadata(hostname string) bool {
	lowercase := strings.ToLower(hostname)
	cloudMetadataEndpoints := []string{
		"169.254.169.254",          // AWS, Azure, GCP, DigitalOcean
		"metadata.google.internal", // GCP
		"metadata",                 // Generic
		"instance-data",            // AWS
		"fd00:ec2::254",            // AWS IPv6
	}

	for _, endpoint := range cloudMetadataEndpoints {
		if lowercase == endpoint {
			return true
		}
	}

	// Check for metadata subdomains
	if strings.HasSuffix(lowercase, ".metadata.google.internal") {
		return true
	}

	return false
}

// validateIPAddress validates that an IP address is not private or reserved
func validateIPAddress(ip net.IP) error {
	// Check for cloud metadata IP first (most specific)
	if ip.String() == "169.254.169.254" {
		return fmt.Errorf("cloud metadata IP address is not allowed")
	}

	// Check for unspecified (0.0.0.0 or ::)
	if ip.IsUnspecified() {
		return fmt.Errorf("unspecified IP addresses are not allowed")
	}

	// Check for loopback
	if ip.IsLoopback() {
		return fmt.Errorf("loopback IP addresses are not allowed")
	}

	// Check for link-local addresses
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("link-local IP addresses are not allowed")
	}

	// Check for private networks
	if isPrivateIP(ip) {
		return fmt.Errorf("private IP addresses are not allowed")
	}

	return nil
}

// isPrivateIP checks if an IP is in a private range
func isPrivateIP(ip net.IP) bool {
	// Private IPv4 ranges
	privateIPv4Ranges := []string{
		"10.0.0.0/8",      // RFC1918
		"172.16.0.0/12",   // RFC1918
		"192.168.0.0/16",  // RFC1918
		"100.64.0.0/10",   // RFC6598 - Shared Address Space
		"169.254.0.0/16",  // RFC3927 - Link Local
		"192.0.0.0/24",    // RFC6890
		"192.0.2.0/24",    // RFC5737 - Documentation
		"198.18.0.0/15",   // RFC2544 - Benchmarking
		"198.51.100.0/24", // RFC5737 - Documentation
		"203.0.113.0/24",  // RFC5737 - Documentation
		"240.0.0.0/4",     // RFC1112 - Reserved
	}

	// Private IPv6 ranges
	privateIPv6Ranges := []string{
		"fc00::/7",      // RFC4193 - Unique Local Addresses
		"fe80::/10",     // RFC4291 - Link Local
		"ff00::/8",      // RFC4291 - Multicast
		"::1/128",       // RFC4291 - Loopback
		"::/128",        // RFC4291 - Unspecified
		"100::/64",      // RFC6666 - Discard Prefix
		"2001::/32",     // RFC4380 - Teredo
		"2001:10::/28",  // RFC4843 - ORCHID
		"2001:db8::/32", // RFC3849 - Documentation
	}

	allRanges := append(privateIPv4Ranges, privateIPv6Ranges...)

	for _, cidr := range allRanges {
		_, subnet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if subnet.Contains(ip) {
			return true
		}
	}

	return false
}

// validateDomainName validates a domain name by resolving it and checking all IPs
func validateDomainName(hostname string) error {
	// Create a resolver with timeout to prevent DNS rebinding attacks
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resolver := &net.Resolver{}
	ips, err := resolver.LookupIP(ctx, "ip", hostname)
	if err != nil {
		// DNS resolution failure is allowed - we don't block domains that don't resolve yet
		// This allows for testing against domains that might not be live yet
		return nil
	}

	// Check all resolved IPs
	for _, ip := range ips {
		if err := validateIPAddress(ip); err != nil {
			return fmt.Errorf("domain %q resolves to forbidden IP %s: %w", hostname, ip.String(), err)
		}
	}

	return nil
}

// ValidateWebhookURL validates a webhook URL to prevent SSRF attacks.
// Unlike ValidateChatwootURL, this allows localhost for development purposes.
func ValidateWebhookURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Only allow http and https schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: only http and https are allowed, got %q", parsedURL.Scheme)
	}

	// Extract hostname
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL must contain a hostname")
	}

	// Check for cloud metadata endpoints (still blocked even for webhooks)
	if isCloudMetadata(hostname) {
		return fmt.Errorf("cloud metadata endpoints are not allowed")
	}

	// If it's an IP address, validate it (but allow localhost)
	if ip := net.ParseIP(hostname); ip != nil {
		if err := validateWebhookIPAddress(ip); err != nil {
			return err
		}
	} else {
		// For domain names, check if it's localhost-related (allowed)
		if !isLocalhost(hostname) {
			// For non-localhost domains, resolve and check all IPs
			if err := validateWebhookDomainName(hostname); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateWebhookIPAddress validates webhook IP addresses.
// Unlike validateIPAddress, this allows loopback addresses for development.
func validateWebhookIPAddress(ip net.IP) error {
	// Check for cloud metadata IP first (most specific) - always blocked
	if ip.String() == "169.254.169.254" {
		return fmt.Errorf("cloud metadata IP address is not allowed")
	}

	// Allow loopback addresses (localhost) for development
	if ip.IsLoopback() {
		return nil
	}

	// Allow unspecified addresses (0.0.0.0 or ::) for development
	if ip.IsUnspecified() {
		return nil
	}

	// Check for link-local addresses (excluding the cloud metadata IP already checked)
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("link-local IP addresses are not allowed")
	}

	// Check for private networks
	if isPrivateIP(ip) {
		return fmt.Errorf("private IP addresses are not allowed")
	}

	return nil
}

// validateWebhookDomainName validates a webhook domain name by resolving it and checking all IPs.
// Unlike validateDomainName, this allows loopback IPs for development.
func validateWebhookDomainName(hostname string) error {
	// Create a resolver with timeout to prevent DNS rebinding attacks
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resolver := &net.Resolver{}
	ips, err := resolver.LookupIP(ctx, "ip", hostname)
	if err != nil {
		// DNS resolution failure is allowed - we don't block domains that don't resolve yet
		// This allows for testing against domains that might not be live yet
		return nil
	}

	// Check all resolved IPs using webhook-specific validation
	for _, ip := range ips {
		if err := validateWebhookIPAddress(ip); err != nil {
			return fmt.Errorf("domain %q resolves to forbidden IP %s: %w", hostname, ip.String(), err)
		}
	}

	return nil
}
