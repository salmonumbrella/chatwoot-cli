package cmd

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRefCommand_TypedID_NoProbe(t *testing.T) {
	// No server needed: typed IDs should parse without probing.
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	out := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"ref", "label:123", "--emit", "json"}); err != nil {
			t.Fatalf("ref failed: %v", err)
		}
	})

	type action struct {
		ID   string   `json:"id"`
		Argv []string `json:"argv"`
	}
	type ref struct {
		Type    string   `json:"type"`
		ID      int      `json:"id"`
		TypedID string   `json:"typed_id"`
		Actions []action `json:"actions"`
	}

	var got ref
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("expected JSON output, got %q: %v", out, err)
	}
	if got.Type != "label" {
		t.Fatalf("expected type label, got %v", got.Type)
	}
	if got.ID != 123 {
		t.Fatalf("expected id 123, got %v", got.ID)
	}
	if got.TypedID != "label:123" {
		t.Fatalf("expected typed_id label:123, got %v", got.TypedID)
	}
	foundGet := false
	for _, a := range got.Actions {
		if a.ID == "get" {
			foundGet = true
			break
		}
	}
	if !foundGet {
		t.Fatalf("expected actions to include get, got %+v", got.Actions)
	}
}

func TestRefCommand_ActionsHaveInputs(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	out := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"ref", "conversation:1", "--emit", "json"}); err != nil {
			t.Fatalf("ref failed: %v", err)
		}
	})

	type input struct {
		Name string `json:"name"`
	}
	type action struct {
		ID     string   `json:"id"`
		Argv   []string `json:"argv"`
		Inputs []input  `json:"inputs"`
	}
	type ref struct {
		Type    string   `json:"type"`
		ID      int      `json:"id"`
		Actions []action `json:"actions"`
	}

	var got ref
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("expected JSON output, got %q: %v", out, err)
	}
	if got.Type != "conversation" || got.ID != 1 {
		t.Fatalf("expected conversation 1, got %s %d", got.Type, got.ID)
	}

	foundComment := false
	for _, a := range got.Actions {
		if a.ID != "comment" {
			continue
		}
		foundComment = true
		if len(a.Inputs) == 0 || a.Inputs[0].Name != "text" {
			t.Fatalf("expected comment action to have input 'text', got %+v", a.Inputs)
		}
		// Ensure placeholders are tokenized, not angle-bracketed.
		hasDollar := false
		for _, tok := range a.Argv {
			if tok == "$text" {
				hasDollar = true
				break
			}
		}
		if !hasDollar {
			t.Fatalf("expected comment argv to include $text placeholder, got %+v", a.Argv)
		}
	}
	if !foundComment {
		t.Fatalf("expected actions to include comment, got %+v", got.Actions)
	}
}

func TestRefCommand_URL_NoProbe(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	out := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"ref", "https://app.chatwoot.com/app/accounts/1/conversations/42", "--emit", "json"}); err != nil {
			t.Fatalf("ref failed: %v", err)
		}
	})

	type ref struct {
		Type string `json:"type"`
		ID   int    `json:"id"`
	}

	var got ref
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("expected JSON output, got %q: %v", out, err)
	}
	if got.Type != "conversation" {
		t.Fatalf("expected type conversation, got %v", got.Type)
	}
	if got.ID != 42 {
		t.Fatalf("expected id 42, got %v", got.ID)
	}
}

func TestRefCommand_UserTypedID_NoProbe(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	out := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"ref", "user:123", "--emit", "json"}); err != nil {
			t.Fatalf("ref failed: %v", err)
		}
	})

	type ref struct {
		Type    string `json:"type"`
		ID      int    `json:"id"`
		TypedID string `json:"typed_id"`
	}
	var got ref
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("expected JSON output, got %q: %v", out, err)
	}
	if got.Type != "user" {
		t.Fatalf("expected type user, got %v", got.Type)
	}
	if got.ID != 123 {
		t.Fatalf("expected id 123, got %v", got.ID)
	}
	if got.TypedID != "user:123" {
		t.Fatalf("expected typed_id user:123, got %v", got.TypedID)
	}
}

func TestRefCommand_BareID_ProbeConversation(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/1", jsonResponse(200, `{"id":1}`)).
		On("GET", "/api/v1/accounts/1/contacts/1", jsonResponse(404, `{"error":"not found"}`))

	setupTestEnvWithHandler(t, handler)

	out := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"ref", "1", "--emit", "json"}); err != nil {
			t.Fatalf("ref failed: %v", err)
		}
	})

	type action struct {
		ID string `json:"id"`
	}
	type ref struct {
		Type    string   `json:"type"`
		ID      int      `json:"id"`
		Actions []action `json:"actions"`
	}

	var got ref
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("expected JSON output, got %q: %v", out, err)
	}
	if got.Type != "conversation" {
		t.Fatalf("expected type conversation, got %v", got.Type)
	}
	if got.ID != 1 {
		t.Fatalf("expected id 1, got %v", got.ID)
	}
	foundOpen := false
	for _, a := range got.Actions {
		if a.ID == "open" {
			foundOpen = true
			break
		}
	}
	if !foundOpen {
		t.Fatalf("expected actions to include open, got %+v", got.Actions)
	}
}

func TestRefCommand_BareID_ProbeAmbiguous(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/1", jsonResponse(200, `{"id":1}`)).
		On("GET", "/api/v1/accounts/1/contacts/1", jsonResponse(200, `{"payload":{"id":1}}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"ref", "1", "--emit", "json"})
	if err == nil {
		t.Fatalf("expected error for ambiguous ID")
	}
	if got := err.Error(); got == "" {
		t.Fatalf("expected error message, got empty")
	}
}
