package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveCommand_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	err := Execute(context.Background(), []string{"resolve", "--help"})

	// Help should not return an error
	assert.NoError(t, err)
	// Note: output goes to stdout which we can't easily capture here
	// The main test is that the command exists and doesn't error
	_ = buf
}

func TestResolveCommand_RequiresConversationID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"resolve"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires at least 1 arg")
}

func TestResolveCommand_ValidatesConversationID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_NO_KEYCHAIN", "1")

	err := Execute(context.Background(), []string{"resolve", "abc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid conversation ID")
}

func TestResolveCommand_AcceptsURLAndPrefixedIDs(t *testing.T) {
	t.Cleanup(func() { validation.SetAllowPrivate(false) })
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch r.URL.Path {
		case "/api/v1/accounts/1/conversations/123/toggle_status",
			"/api/v1/accounts/1/conversations/456/toggle_status":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"payload": map[string]any{
					"success":        true,
					"current_status": "resolved",
				},
			})
		default:
			t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Setenv("CHATWOOT_BASE_URL", server.URL)
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_NO_KEYCHAIN", "1")

	err := Execute(context.Background(), []string{
		"resolve",
		"https://app.chatwoot.com/app/accounts/1/conversations/123",
		"conv:456",
		"--allow-private",
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, callCount, "should resolve both conversations")
}

func TestResolveCommand_SingleConversation(t *testing.T) {
	t.Cleanup(func() { validation.SetAllowPrivate(false) })
	toggleCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/api/v1/accounts/1/conversations/123/toggle_status" {
			toggleCalled = true
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			assert.Equal(t, "resolved", payload["status"])
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"payload": map[string]any{
					"success":         true,
					"conversation_id": 123,
					"current_status":  "resolved",
				},
			})
		} else {
			t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Setenv("CHATWOOT_BASE_URL", server.URL)
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_NO_KEYCHAIN", "1")

	err := Execute(context.Background(), []string{"resolve", "123", "--allow-private"})
	assert.NoError(t, err)
	assert.True(t, toggleCalled, "toggle_status endpoint should be called")
}

func TestResolveCommand_MultipleConversations(t *testing.T) {
	t.Cleanup(func() { validation.SetAllowPrivate(false) })
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/accounts/1/conversations/123/toggle_status",
			"/api/v1/accounts/1/conversations/456/toggle_status",
			"/api/v1/accounts/1/conversations/789/toggle_status":
			callCount++
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			assert.Equal(t, "resolved", payload["status"])
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"payload": map[string]any{
					"success":        true,
					"current_status": "resolved",
				},
			})
		default:
			t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Setenv("CHATWOOT_BASE_URL", server.URL)
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_NO_KEYCHAIN", "1")

	err := Execute(context.Background(), []string{"resolve", "123", "456", "789", "--allow-private"})
	assert.NoError(t, err)
	assert.Equal(t, 3, callCount, "should call toggle_status for each conversation")
}

func TestResolveCommand_PartialFailure(t *testing.T) {
	t.Cleanup(func() { validation.SetAllowPrivate(false) })
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch r.URL.Path {
		case "/api/v1/accounts/1/conversations/123/toggle_status":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"payload": map[string]any{
					"success":        true,
					"current_status": "resolved",
				},
			})
		case "/api/v1/accounts/1/conversations/456/toggle_status":
			// Simulate failure
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": "not found",
			})
		case "/api/v1/accounts/1/conversations/789/toggle_status":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"payload": map[string]any{
					"success":        true,
					"current_status": "resolved",
				},
			})
		default:
			t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Setenv("CHATWOOT_BASE_URL", server.URL)
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_NO_KEYCHAIN", "1")

	err := Execute(context.Background(), []string{"resolve", "123", "456", "789", "--allow-private"})
	// Should return error due to partial failure
	require.Error(t, err)
	// But should have attempted all conversations
	assert.Equal(t, 3, callCount, "should attempt all conversations despite failures")
}

func TestResolveCommand_JSONOutput(t *testing.T) {
	t.Cleanup(func() { validation.SetAllowPrivate(false) })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/accounts/1/conversations/123/toggle_status",
			"/api/v1/accounts/1/conversations/456/toggle_status":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"payload": map[string]any{
					"success":        true,
					"current_status": "resolved",
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Setenv("CHATWOOT_BASE_URL", server.URL)
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_NO_KEYCHAIN", "1")

	err := Execute(context.Background(), []string{"resolve", "123", "456", "--output", "json", "--allow-private"})
	assert.NoError(t, err)
}

func TestResolveCommand_JSONOutputWithPartialFailure(t *testing.T) {
	t.Cleanup(func() { validation.SetAllowPrivate(false) })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/accounts/1/conversations/123/toggle_status":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"payload": map[string]any{
					"success":        true,
					"current_status": "resolved",
				},
			})
		case "/api/v1/accounts/1/conversations/456/toggle_status":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": "not found",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Setenv("CHATWOOT_BASE_URL", server.URL)
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_NO_KEYCHAIN", "1")

	err := Execute(context.Background(), []string{"resolve", "123", "456", "--output", "json", "--allow-private"})
	// In JSON mode the close command (which resolve aliases) returns the
	// summary without propagating the error — the caller inspects the
	// closed vs total counts instead.
	assert.NoError(t, err)
}
