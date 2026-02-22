package cmd

import (
	"context"
	"strings"
	"testing"
)

func TestURLFlag_Conversations(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "get", "461", "--url"})
		if err != nil {
			t.Fatalf("conversations get --url failed: %v", err)
		}
	})

	output = strings.TrimSpace(output)
	expected := "/app/accounts/1/conversations/461"
	if !strings.HasSuffix(output, expected) {
		t.Errorf("output = %q, want suffix %q", output, expected)
	}
}

func TestURLFlag_Contacts(t *testing.T) {
	// contacts get with numeric ID still calls getClient for resolveContactID,
	// but --url should skip the API fetch and print the URL
	handler := newRouteHandler()
	env := setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "get", "199519", "--url"})
		if err != nil {
			t.Fatalf("contacts get --url failed: %v", err)
		}
	})

	output = strings.TrimSpace(output)
	expected := env.server.URL + "/app/accounts/1/contacts/199519"
	if output != expected {
		t.Errorf("output = %q, want %q", output, expected)
	}
}

func TestURLFlag_Inboxes(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "get", "5", "--url"})
		if err != nil {
			t.Fatalf("inboxes get --url failed: %v", err)
		}
	})

	output = strings.TrimSpace(output)
	expected := "/app/accounts/1/inboxes/5"
	if !strings.HasSuffix(output, expected) {
		t.Errorf("output = %q, want suffix %q", output, expected)
	}
}

func TestURLFlag_Teams(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"teams", "get", "3", "--url"})
		if err != nil {
			t.Fatalf("teams get --url failed: %v", err)
		}
	})

	output = strings.TrimSpace(output)
	expected := "/app/accounts/1/teams/3"
	if !strings.HasSuffix(output, expected) {
		t.Errorf("output = %q, want suffix %q", output, expected)
	}
}

func TestURLFlag_Agents(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"agents", "get", "10", "--url"})
		if err != nil {
			t.Fatalf("agents get --url failed: %v", err)
		}
	})

	output = strings.TrimSpace(output)
	expected := "/app/accounts/1/agents/10"
	if !strings.HasSuffix(output, expected) {
		t.Errorf("output = %q, want suffix %q", output, expected)
	}
}

func TestURLFlag_Campaigns(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"campaigns", "get", "7", "--url"})
		if err != nil {
			t.Fatalf("campaigns get --url failed: %v", err)
		}
	})

	output = strings.TrimSpace(output)
	expected := "/app/accounts/1/campaigns/7"
	if !strings.HasSuffix(output, expected) {
		t.Errorf("output = %q, want suffix %q", output, expected)
	}
}

func TestURLFlag_OutputsOnlyURL(t *testing.T) {
	// Verify --url outputs ONLY the URL (one line, no extra text)
	setupTestEnv(t, jsonResponse(200, `{}`))

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "get", "123", "--url"})
		if err != nil {
			t.Fatalf("command failed: %v", err)
		}
	})

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Errorf("expected exactly 1 line of output, got %d: %q", len(lines), output)
	}
}

func TestURLFlag_SkipsAPICall(t *testing.T) {
	// Set up a handler that would fail if called - proving --url skips the API
	handler := newRouteHandler() // No routes registered, any API call returns 404
	setupTestEnvWithHandler(t, handler)

	// If --url tried to make an API call, the 404 would cause an error
	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "get", "99", "--url"})
		if err != nil {
			t.Fatalf("inboxes get --url should not make API calls, but got error: %v", err)
		}
	})

	output = strings.TrimSpace(output)
	if !strings.Contains(output, "/app/accounts/1/inboxes/99") {
		t.Errorf("output = %q, want URL containing /app/accounts/1/inboxes/99", output)
	}
}

func TestURLFlag_WithoutFlag(t *testing.T) {
	// Verify that without --url, the command still works normally (makes API call)
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/teams/5", jsonResponse(200, `{
			"id": 5,
			"name": "Support Team",
			"description": "Primary support team"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"teams", "get", "5"})
		if err != nil {
			t.Fatalf("teams get failed: %v", err)
		}
	})

	// Should show team details, not a URL
	if !strings.Contains(output, "Support Team") {
		t.Errorf("expected normal output with team name, got: %s", output)
	}
	if strings.Contains(output, "/app/accounts/") {
		t.Errorf("without --url, output should not contain URL path, got: %s", output)
	}
}

func TestURLFlag_ContactsShow(t *testing.T) {
	// Verify the 'show' alias also supports --url
	handler := newRouteHandler()
	env := setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "show", "456", "--url"})
		if err != nil {
			t.Fatalf("contacts show --url failed: %v", err)
		}
	})

	output = strings.TrimSpace(output)
	expected := env.server.URL + "/app/accounts/1/contacts/456"
	if output != expected {
		t.Errorf("output = %q, want %q", output, expected)
	}
}

// TestResourceURL tests the resourceURL function directly
func TestResourceURL(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		resourceID   int
		wantSuffix   string // We check suffix since base URL varies per test server
	}{
		{
			name:         "conversations",
			resourceType: "conversations",
			resourceID:   123,
			wantSuffix:   "/app/accounts/1/conversations/123",
		},
		{
			name:         "contacts",
			resourceType: "contacts",
			resourceID:   456,
			wantSuffix:   "/app/accounts/1/contacts/456",
		},
		{
			name:         "inboxes",
			resourceType: "inboxes",
			resourceID:   7,
			wantSuffix:   "/app/accounts/1/inboxes/7",
		},
		{
			name:         "teams",
			resourceType: "teams",
			resourceID:   99,
			wantSuffix:   "/app/accounts/1/teams/99",
		},
		{
			name:         "agents",
			resourceType: "agents",
			resourceID:   42,
			wantSuffix:   "/app/accounts/1/agents/42",
		},
		{
			name:         "campaigns",
			resourceType: "campaigns",
			resourceID:   1,
			wantSuffix:   "/app/accounts/1/campaigns/1",
		},
		{
			name:         "labels",
			resourceType: "labels",
			resourceID:   55,
			wantSuffix:   "/app/accounts/1/labels/55",
		},
		{
			name:         "large resource ID",
			resourceType: "conversations",
			resourceID:   999999999,
			wantSuffix:   "/app/accounts/1/conversations/999999999",
		},
		{
			name:         "zero resource ID",
			resourceType: "contacts",
			resourceID:   0,
			wantSuffix:   "/app/accounts/1/contacts/0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupTestEnv(t, jsonResponse(200, `{}`))

			url, err := resourceURL(tt.resourceType, tt.resourceID)
			if err != nil {
				t.Fatalf("resourceURL() error = %v", err)
			}

			// Verify the URL starts with the test server URL
			if !strings.HasPrefix(url, env.server.URL) {
				t.Errorf("resourceURL() = %q, should start with %q", url, env.server.URL)
			}

			// Verify the URL ends with the expected path
			if !strings.HasSuffix(url, tt.wantSuffix) {
				t.Errorf("resourceURL() = %q, want suffix %q", url, tt.wantSuffix)
			}

			// Verify the full URL matches expected format
			expected := env.server.URL + tt.wantSuffix
			if url != expected {
				t.Errorf("resourceURL() = %q, want %q", url, expected)
			}
		})
	}
}

// TestResourceURL_URLFormat verifies the URL format is correct
func TestResourceURL_URLFormat(t *testing.T) {
	env := setupTestEnv(t, jsonResponse(200, `{}`))

	url, err := resourceURL("conversations", 123)
	if err != nil {
		t.Fatalf("resourceURL() error = %v", err)
	}

	// Verify no double slashes (except in http://)
	urlWithoutScheme := strings.TrimPrefix(url, "http://")
	urlWithoutScheme = strings.TrimPrefix(urlWithoutScheme, "https://")
	if strings.Contains(urlWithoutScheme, "//") {
		t.Errorf("resourceURL() = %q, contains double slashes in path", url)
	}

	// Verify no trailing slash
	if strings.HasSuffix(url, "/") {
		t.Errorf("resourceURL() = %q, has trailing slash", url)
	}

	// Verify expected format: {baseURL}/app/accounts/{accountID}/{resourceType}/{resourceID}
	expectedFormat := env.server.URL + "/app/accounts/1/conversations/123"
	if url != expectedFormat {
		t.Errorf("resourceURL() = %q, want %q", url, expectedFormat)
	}
}
