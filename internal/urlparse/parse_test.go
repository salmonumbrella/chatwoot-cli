package urlparse

import (
	"strings"
	"testing"
)

func TestParse_ValidConversationURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantBase  string
		wantAcct  int
		wantType  string
		wantID    int
		wantHasID bool
	}{
		{
			name:      "basic conversation URL",
			url:       "https://app.chatwoot.com/app/accounts/1/conversations/123",
			wantBase:  "https://app.chatwoot.com",
			wantAcct:  1,
			wantType:  "conversation",
			wantID:    123,
			wantHasID: true,
		},
		{
			name:      "conversation URL with trailing path",
			url:       "https://chatwoot.example.com/app/accounts/5/conversations/42/messages",
			wantBase:  "https://chatwoot.example.com",
			wantAcct:  5,
			wantType:  "conversation",
			wantID:    42,
			wantHasID: true,
		},
		{
			name:      "conversations list URL (no ID)",
			url:       "https://app.chatwoot.com/app/accounts/1/conversations",
			wantBase:  "https://app.chatwoot.com",
			wantAcct:  1,
			wantType:  "conversation",
			wantID:    0,
			wantHasID: false,
		},
		{
			name:      "http scheme",
			url:       "http://localhost:3000/app/accounts/1/conversations/999",
			wantBase:  "http://localhost:3000",
			wantAcct:  1,
			wantType:  "conversation",
			wantID:    999,
			wantHasID: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.url)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got.BaseURL != tt.wantBase {
				t.Errorf("BaseURL = %q, want %q", got.BaseURL, tt.wantBase)
			}
			if got.AccountID != tt.wantAcct {
				t.Errorf("AccountID = %d, want %d", got.AccountID, tt.wantAcct)
			}
			if got.ResourceType != tt.wantType {
				t.Errorf("ResourceType = %q, want %q", got.ResourceType, tt.wantType)
			}
			if got.ResourceID != tt.wantID {
				t.Errorf("ResourceID = %d, want %d", got.ResourceID, tt.wantID)
			}
			if got.HasResourceID() != tt.wantHasID {
				t.Errorf("HasResourceID() = %v, want %v", got.HasResourceID(), tt.wantHasID)
			}
		})
	}
}

func TestParse_ValidContactURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantBase string
		wantAcct int
		wantID   int
	}{
		{
			name:     "basic contact URL",
			url:      "https://app.chatwoot.com/app/accounts/1/contacts/456",
			wantBase: "https://app.chatwoot.com",
			wantAcct: 1,
			wantID:   456,
		},
		{
			name:     "contact URL with port",
			url:      "https://chatwoot.local:8080/app/accounts/99/contacts/789",
			wantBase: "https://chatwoot.local:8080",
			wantAcct: 99,
			wantID:   789,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.url)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got.BaseURL != tt.wantBase {
				t.Errorf("BaseURL = %q, want %q", got.BaseURL, tt.wantBase)
			}
			if got.AccountID != tt.wantAcct {
				t.Errorf("AccountID = %d, want %d", got.AccountID, tt.wantAcct)
			}
			if got.ResourceType != "contact" {
				t.Errorf("ResourceType = %q, want %q", got.ResourceType, "contact")
			}
			if got.ResourceID != tt.wantID {
				t.Errorf("ResourceID = %d, want %d", got.ResourceID, tt.wantID)
			}
		})
	}
}

func TestParse_AllResourceTypes(t *testing.T) {
	tests := []struct {
		pluralPath   string
		singularType string
	}{
		{"conversations", "conversation"},
		{"contacts", "contact"},
		{"inboxes", "inbox"},
		{"teams", "team"},
		{"agents", "agent"},
		{"campaigns", "campaign"},
	}

	for _, tt := range tests {
		t.Run(tt.pluralPath, func(t *testing.T) {
			url := "https://example.com/app/accounts/1/" + tt.pluralPath + "/123"
			got, err := Parse(url)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got.ResourceType != tt.singularType {
				t.Errorf("ResourceType = %q, want %q", got.ResourceType, tt.singularType)
			}
		})
	}
}

func TestParse_InvalidURLs(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr string
	}{
		{
			name:    "empty URL",
			url:     "",
			wantErr: "URL cannot be empty",
		},
		{
			name:    "missing scheme",
			url:     "app.chatwoot.com/app/accounts/1/conversations/123",
			wantErr: "missing scheme",
		},
		{
			name:    "invalid scheme",
			url:     "ftp://app.chatwoot.com/app/accounts/1/conversations/123",
			wantErr: "invalid URL scheme",
		},
		{
			name:    "missing /app prefix",
			url:     "https://app.chatwoot.com/accounts/1/conversations/123",
			wantErr: "invalid Chatwoot URL format",
		},
		{
			name:    "missing account path",
			url:     "https://app.chatwoot.com/app/conversations/123",
			wantErr: "invalid Chatwoot URL format",
		},
		{
			name:    "non-numeric account ID",
			url:     "https://app.chatwoot.com/app/accounts/abc/conversations/123",
			wantErr: "invalid Chatwoot URL format",
		},
		{
			name:    "unsupported resource type",
			url:     "https://app.chatwoot.com/app/accounts/1/widgets/123",
			wantErr: "unsupported resource type",
		},
		{
			name:    "random path",
			url:     "https://app.chatwoot.com/some/random/path",
			wantErr: "invalid Chatwoot URL format",
		},
		{
			name:    "root path only",
			url:     "https://app.chatwoot.com/",
			wantErr: "invalid Chatwoot URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.url)
			if err == nil {
				t.Fatalf("Parse() expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Parse() error = %q, want error containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestParse_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantType string
		wantID   int
	}{
		{
			name:     "URL with query string",
			url:      "https://app.chatwoot.com/app/accounts/1/conversations/123?page=1&status=open",
			wantType: "conversation",
			wantID:   123,
		},
		{
			name:     "URL with fragment",
			url:      "https://app.chatwoot.com/app/accounts/1/conversations/123#messages",
			wantType: "conversation",
			wantID:   123,
		},
		{
			name:     "URL with query and fragment",
			url:      "https://app.chatwoot.com/app/accounts/1/contacts/456?tab=info#details",
			wantType: "contact",
			wantID:   456,
		},
		{
			name:     "URL with nested path after ID",
			url:      "https://app.chatwoot.com/app/accounts/1/conversations/123/messages/new",
			wantType: "conversation",
			wantID:   123,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.url)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got.ResourceType != tt.wantType {
				t.Errorf("ResourceType = %q, want %q", got.ResourceType, tt.wantType)
			}
			if got.ResourceID != tt.wantID {
				t.Errorf("ResourceID = %d, want %d", got.ResourceID, tt.wantID)
			}
		})
	}
}
