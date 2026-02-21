package cmd

import (
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

func TestBuildLightContact(t *testing.T) {
	contact := &api.Contact{
		ID:          136014,
		Name:        "John Doe",
		Email:       "john@example.com",
		PhoneNumber: "+886912345678",
	}

	result := buildLightContact(contact)

	if result.ID != 136014 {
		t.Errorf("expected ID 136014, got %d", result.ID)
	}
	if result.Name == nil || *result.Name != "John Doe" {
		t.Errorf("expected name John Doe, got %v", result.Name)
	}
	if result.Email == nil || *result.Email != "john@example.com" {
		t.Errorf("expected email john@example.com, got %v", result.Email)
	}
	if result.Phone == nil || *result.Phone != "+886912345678" {
		t.Errorf("expected phone +886912345678, got %v", result.Phone)
	}
}

func TestBuildLightContact_WithCustomAttributes(t *testing.T) {
	contact := &api.Contact{
		ID:          136014,
		Name:        "Jane",
		Email:       "jane@example.com",
		PhoneNumber: "+886912345678",
		CustomAttributes: map[string]any{
			"membership_tier":          "🥈 Silver",
			"6167311875566d0016e3aea2": "https://admin.shoplineapp.com/admin/amyscanadashop/users/629c430dc7e798000957af45",
			"633e52851a4eb80025fcf3c9": "https://admin.shoplineapp.com/admin/panpanlive/users/abc123def456",
			"68468d58ef183a000827457d": "https://admin.shoplineapp.com/admin/themomentshop/users/deadbeef1234",
		},
	}

	result := buildLightContact(contact)

	if result.Tier == nil || *result.Tier != "🥈 Silver" {
		t.Errorf("expected tier '🥈 Silver', got %v", result.Tier)
	}
	if result.Amy == nil || *result.Amy != "629c430dc7e798000957af45" {
		t.Errorf("expected Amy Shop ID '629c430dc7e798000957af45', got %v", result.Amy)
	}
	if result.PP == nil || *result.PP != "abc123def456" {
		t.Errorf("expected Panpan ID 'abc123def456', got %v", result.PP)
	}
	if result.TM == nil || *result.TM != "deadbeef1234" {
		t.Errorf("expected The Moment ID 'deadbeef1234', got %v", result.TM)
	}
}

func TestBuildLightContact_NoCustomAttributes(t *testing.T) {
	contact := &api.Contact{
		ID:   42,
		Name: "Bob",
	}

	result := buildLightContact(contact)

	if result.Tier != nil {
		t.Errorf("expected nil tier, got %v", result.Tier)
	}
	if result.Amy != nil {
		t.Errorf("expected nil amy, got %v", result.Amy)
	}
	if result.PP != nil {
		t.Errorf("expected nil pp, got %v", result.PP)
	}
	if result.TM != nil {
		t.Errorf("expected nil tm, got %v", result.TM)
	}
}

func TestExtractShoplineID(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected *string
	}{
		{
			name:     "full admin URL",
			value:    "https://admin.shoplineapp.com/admin/amyscanadashop/users/629c430dc7e798000957af45",
			expected: strPtr("629c430dc7e798000957af45"),
		},
		{
			name:     "bare ID (no slash)",
			value:    "629c430dc7e798000957af45",
			expected: strPtr("629c430dc7e798000957af45"),
		},
		{
			name:     "empty string",
			value:    "",
			expected: nil,
		},
		{
			name:     "missing key",
			value:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ca := map[string]any{}
			if tt.value != nil {
				ca["key"] = tt.value
			}
			got := extractShoplineID(ca, "key")
			if tt.expected == nil {
				if got != nil {
					t.Errorf("expected nil, got %v", *got)
				}
			} else {
				if got == nil || *got != *tt.expected {
					t.Errorf("expected %v, got %v", *tt.expected, got)
				}
			}
		})
	}
}

func TestBuildLightContactEmptyFields(t *testing.T) {
	contact := &api.Contact{
		ID:   42,
		Name: "",
	}

	result := buildLightContact(contact)

	if result.ID != 42 {
		t.Errorf("expected ID 42, got %d", result.ID)
	}
	if result.Name != nil {
		t.Errorf("expected nil name, got %v", result.Name)
	}
	if result.Email != nil {
		t.Errorf("expected nil email, got %v", result.Email)
	}
	if result.Phone != nil {
		t.Errorf("expected nil phone, got %v", result.Phone)
	}
}

func TestBuildLightContactNil(t *testing.T) {
	result := buildLightContact(nil)
	if result.ID != 0 {
		t.Errorf("expected zero ID for nil contact, got %d", result.ID)
	}
}

func TestBuildLightContactConversation(t *testing.T) {
	conv := api.Conversation{
		ID:      8821,
		Status:  "open",
		InboxID: 3,
	}
	lastMsg := "When will my order arrive?"

	result := buildLightContactConversation(conv, lastMsg)

	if result.ID != 8821 {
		t.Errorf("expected ID 8821, got %d", result.ID)
	}
	if result.Status != "o" {
		t.Errorf("expected status o, got %s", result.Status)
	}
	if result.InboxID != 3 {
		t.Errorf("expected inbox 3, got %d", result.InboxID)
	}
	if result.Last == nil || *result.Last != lastMsg {
		t.Errorf("expected last message %q, got %v", lastMsg, result.Last)
	}
}

func TestBuildLightContactConversationEmptyMessage(t *testing.T) {
	conv := api.Conversation{ID: 100, Status: "pending", InboxID: 5}
	result := buildLightContactConversation(conv, "")
	if result.Last != nil {
		t.Errorf("expected nil last for empty message, got %v", result.Last)
	}
}

func TestExtractLastNonActivityMessage(t *testing.T) {
	content := "Hello there"
	conv := api.Conversation{
		LastNonActivityMessage: &api.LastNonActivityMessage{Content: content},
	}
	if got := extractLastNonActivityMessage(conv); got != content {
		t.Errorf("expected %q, got %q", content, got)
	}

	// Nil message
	conv2 := api.Conversation{}
	if got := extractLastNonActivityMessage(conv2); got != "" {
		t.Errorf("expected empty string for nil message, got %q", got)
	}
}
