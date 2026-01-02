// Package urlparse provides URL parsing utilities for Chatwoot URLs.
package urlparse

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// ParsedURL represents a parsed Chatwoot URL with extracted resource information.
type ParsedURL struct {
	BaseURL      string
	AccountID    int
	ResourceType string // singular form: conversation, contact, inbox, etc.
	ResourceID   int    // optional, 0 if not present
}

// Supported resource types (plural form in URL, mapped to singular)
var resourceTypes = map[string]string{
	"conversations": "conversation",
	"contacts":      "contact",
	"inboxes":       "inbox",
	"teams":         "team",
	"agents":        "agent",
	"campaigns":     "campaign",
}

// urlPattern matches Chatwoot URLs of the form:
// /app/accounts/{account_id}/{resource_type}/{resource_id}?
var urlPattern = regexp.MustCompile(`^/app/accounts/(\d+)/([a-z]+)(?:/(\d+))?(?:/.*)?$`)

// Parse extracts resource information from a Chatwoot URL.
// It accepts full URLs like https://app.chatwoot.com/app/accounts/1/conversations/123
// and returns the parsed components.
func Parse(rawURL string) (*ParsedURL, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	// Parse the URL
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Ensure scheme is present
	if parsed.Scheme == "" {
		return nil, fmt.Errorf("invalid URL: missing scheme (expected https://...)")
	}

	// Ensure it's HTTP/HTTPS
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("invalid URL scheme %q: expected http or https", parsed.Scheme)
	}

	// Extract base URL (scheme + host)
	baseURL := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)

	// Match the path pattern
	matches := urlPattern.FindStringSubmatch(parsed.Path)
	if matches == nil {
		return nil, fmt.Errorf("invalid Chatwoot URL format: expected /app/accounts/{account_id}/{resource_type}[/{resource_id}]")
	}

	// Extract account ID
	accountID, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid account ID: %w", err)
	}

	// Extract and validate resource type
	resourceTypePlural := matches[2]
	resourceTypeSingular, ok := resourceTypes[resourceTypePlural]
	if !ok {
		validTypes := make([]string, 0, len(resourceTypes))
		for k := range resourceTypes {
			validTypes = append(validTypes, k)
		}
		return nil, fmt.Errorf("unsupported resource type %q: expected one of %s", resourceTypePlural, strings.Join(validTypes, ", "))
	}

	// Extract resource ID if present
	var resourceID int
	if matches[3] != "" {
		resourceID, err = strconv.Atoi(matches[3])
		if err != nil {
			return nil, fmt.Errorf("invalid resource ID: %w", err)
		}
	}

	return &ParsedURL{
		BaseURL:      baseURL,
		AccountID:    accountID,
		ResourceType: resourceTypeSingular,
		ResourceID:   resourceID,
	}, nil
}

// HasResourceID returns true if the parsed URL includes a resource ID.
func (p *ParsedURL) HasResourceID() bool {
	return p.ResourceID > 0
}
