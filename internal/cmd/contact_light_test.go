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
	if result.Status != "open" {
		t.Errorf("expected status open, got %s", result.Status)
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
