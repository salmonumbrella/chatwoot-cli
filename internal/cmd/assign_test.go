package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssignCommand_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	err := Execute(context.Background(), []string{"assign", "--help"})

	// Help should not return an error
	assert.NoError(t, err)
	// Note: output goes to stdout which we can't easily capture here
	// The main test is that the command exists and doesn't error
	_ = buf
}

func TestAssignCommand_RequiresConversationID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"assign", "--agent", "5"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg")
}

func TestAssignCommand_RequiresAgentOrTeam(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_NO_KEYCHAIN", "1")

	err := Execute(context.Background(), []string{"assign", "123", "--no-input"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--agent or --team")
}

func TestAssignCommand_AssignsToAgent(t *testing.T) {
	t.Cleanup(func() { validation.SetAllowPrivate(false) })
	assignCalled := false
	getCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/v1/accounts/1/conversations/123/assignments":
			assignCalled = true
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			assert.Equal(t, float64(5), payload["assignee_id"])
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":   5,
				"name": "Test Agent",
			})
		case r.Method == "GET" && r.URL.Path == "/api/v1/accounts/1/conversations/123":
			getCalled = true
			w.WriteHeader(http.StatusOK)
			displayID := 123
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          123,
				"display_id":  displayID,
				"status":      "open",
				"assignee_id": 5,
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

	err := Execute(context.Background(), []string{"assign", "123", "--agent", "5", "--allow-private"})
	assert.NoError(t, err)
	assert.True(t, assignCalled, "assign endpoint should be called")
	assert.True(t, getCalled, "get endpoint should be called to fetch updated conversation")
}

func TestAssignCommand_AssignsToAgentByName(t *testing.T) {
	t.Cleanup(func() { validation.SetAllowPrivate(false) })
	assignCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/accounts/1/agents":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 5, "name": "Test Agent", "email": "test@example.com", "role": "agent"},
			})
		case r.Method == "POST" && r.URL.Path == "/api/v1/accounts/1/conversations/123/assignments":
			assignCalled = true
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			assert.Equal(t, float64(5), payload["assignee_id"])
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 5})
		case r.Method == "GET" && r.URL.Path == "/api/v1/accounts/1/conversations/123":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          123,
				"status":      "open",
				"assignee_id": 5,
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

	err := Execute(context.Background(), []string{"assign", "123", "--agent", "Test Agent", "--allow-private"})
	assert.NoError(t, err)
	assert.True(t, assignCalled, "assign endpoint should be called")
}

func TestAssignCommand_AssignsToTeamByName(t *testing.T) {
	t.Cleanup(func() { validation.SetAllowPrivate(false) })
	assignCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/accounts/1/teams":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 2, "name": "Support Team"},
			})
		case r.Method == "POST" && r.URL.Path == "/api/v1/accounts/1/conversations/123/assignments":
			assignCalled = true
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			assert.Equal(t, float64(2), payload["team_id"])
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 2})
		case r.Method == "GET" && r.URL.Path == "/api/v1/accounts/1/conversations/123":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      123,
				"status":  "open",
				"team_id": 2,
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

	err := Execute(context.Background(), []string{"assign", "123", "--team", "Support Team", "--allow-private"})
	assert.NoError(t, err)
	assert.True(t, assignCalled, "assign endpoint should be called")
}

func TestAssignCommand_AssignsToTeam(t *testing.T) {
	t.Cleanup(func() { validation.SetAllowPrivate(false) })
	assignCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/v1/accounts/1/conversations/123/assignments":
			assignCalled = true
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			assert.Equal(t, float64(2), payload["team_id"])
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":   2,
				"name": "Test Team",
			})
		case r.Method == "GET" && r.URL.Path == "/api/v1/accounts/1/conversations/123":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      123,
				"status":  "open",
				"team_id": 2,
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

	err := Execute(context.Background(), []string{"assign", "123", "--team", "2", "--allow-private"})
	assert.NoError(t, err)
	assert.True(t, assignCalled, "assign endpoint should be called")
}

func TestAssignCommand_JSONOutput(t *testing.T) {
	t.Cleanup(func() { validation.SetAllowPrivate(false) })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/v1/accounts/1/conversations/123/assignments":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":   5,
				"name": "Test Agent",
			})
		case r.Method == "GET" && r.URL.Path == "/api/v1/accounts/1/conversations/123":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          123,
				"status":      "open",
				"assignee_id": 5,
				"team_id":     2,
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

	err := Execute(context.Background(), []string{"assign", "123", "--agent", "5", "--team", "2", "--output", "json", "--allow-private"})
	assert.NoError(t, err)
}

func TestAssignCommand_AgentOutput_CompactAliases(t *testing.T) {
	t.Cleanup(func() { validation.SetAllowPrivate(false) })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/v1/accounts/1/conversations/123/assignments":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 5})
		case r.Method == "GET" && r.URL.Path == "/api/v1/accounts/1/conversations/123":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          123,
				"status":      "open",
				"assignee_id": 5,
				"team_id":     2,
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

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"assign", "123", "--agent", "5", "--team", "2", "-o", "agent", "--allow-private"})
		assert.NoError(t, err)
	})

	if strings.Contains(output, `"kind"`) || strings.Contains(output, `"item"`) || strings.Contains(output, `"data"`) {
		t.Fatalf("agent output should be flat summary, got: %s", output)
	}
	var result struct {
		ID int    `json:"id"`
		St string `json:"st"`
		Ag int    `json:"ag"`
		Tm int    `json:"tm"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse agent output: %v\noutput: %s", err, output)
	}
	if result.ID != 123 || result.St != "o" || result.Ag != 5 || result.Tm != 2 {
		t.Fatalf("unexpected compact assign payload: %#v", result)
	}
}

func TestAssignCommand_LightOutput(t *testing.T) {
	t.Cleanup(func() { validation.SetAllowPrivate(false) })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/v1/accounts/1/conversations/123/assignments":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 5})
		case r.Method == "GET" && r.URL.Path == "/api/v1/accounts/1/conversations/123":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          123,
				"status":      "open",
				"assignee_id": 5,
				"team_id":     2,
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

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"assign", "123", "--agent", "5", "--team", "2", "--light", "-o", "agent", "--allow-private"})
		assert.NoError(t, err)
	})

	if strings.Contains(output, `"kind"`) || strings.Contains(output, `"item"`) {
		t.Fatalf("light output should bypass agent envelope, got: %s", output)
	}
	var result struct {
		ID      int  `json:"id"`
		AgentID *int `json:"ag"`
		TeamID  *int `json:"tm"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse light output: %v\noutput: %s", err, output)
	}
	if result.ID != 123 {
		t.Fatalf("expected id 123, got %d", result.ID)
	}
	if result.AgentID == nil || *result.AgentID != 5 {
		t.Fatalf("expected ag=5, got %#v", result.AgentID)
	}
	if result.TeamID == nil || *result.TeamID != 2 {
		t.Fatalf("expected tm=2, got %#v", result.TeamID)
	}
}
