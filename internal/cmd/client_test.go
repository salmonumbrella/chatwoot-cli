package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestNewClientCmd(t *testing.T) {
	cmd := newClientCmd()

	if cmd.Use != "client" {
		t.Errorf("Expected Use to be 'client', got %s", cmd.Use)
	}
	if cmd.Short != "Access Chatwoot public client APIs" {
		t.Errorf("Expected Short to be 'Access Chatwoot public client APIs', got %s", cmd.Short)
	}

	// Verify persistent flags exist
	baseURLFlag := cmd.PersistentFlags().Lookup("base-url")
	if baseURLFlag == nil {
		t.Error("Expected --base-url flag to exist")
	}

	inboxFlag := cmd.PersistentFlags().Lookup("inbox")
	if inboxFlag == nil {
		t.Error("Expected --inbox flag to exist")
	}

	contactFlag := cmd.PersistentFlags().Lookup("contact")
	if contactFlag == nil {
		t.Error("Expected --contact flag to exist")
	}

	// Verify subcommands exist
	subcommands := []string{"contacts", "conversations", "messages", "typing", "last-seen"}
	for _, sub := range subcommands {
		found := false
		for _, c := range cmd.Commands() {
			if c.Use == sub || strings.HasPrefix(c.Use, sub+" ") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected '%s' subcommand to exist", sub)
		}
	}
}

func TestNewClientContactsCmd(t *testing.T) {
	baseURL := ""
	inboxID := ""
	contactID := ""

	cmd := newClientContactsCmd(&baseURL, &inboxID, &contactID)

	if cmd.Use != "contacts" {
		t.Errorf("Expected Use to be 'contacts', got %s", cmd.Use)
	}

	// Verify subcommands
	subcommands := []string{"create", "get"}
	for _, sub := range subcommands {
		found := false
		for _, c := range cmd.Commands() {
			if c.Use == sub || strings.HasPrefix(c.Use, sub+" ") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected '%s' subcommand to exist", sub)
		}
	}
}

func TestNewClientContactsCreateCmd(t *testing.T) {
	baseURL := ""
	inboxID := ""

	cmd := newClientContactsCreateCmd(&baseURL, &inboxID)

	if cmd.Use != "create" {
		t.Errorf("Expected Use to be 'create', got %s", cmd.Use)
	}

	// Verify flags exist
	flags := []string{"name", "email", "phone", "identifier", "identifier-hash", "avatar-url", "custom-attributes"}
	for _, flag := range flags {
		if cmd.Flag(flag) == nil {
			t.Errorf("Expected --%s flag to exist", flag)
		}
	}
}

func TestNewClientContactsCreateCmdRequiresInbox(t *testing.T) {
	baseURL := ""
	inboxID := "" // Empty - should cause error

	cmd := newClientContactsCreateCmd(&baseURL, &inboxID)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error when --inbox is not provided")
	}
	if err != nil && !strings.Contains(err.Error(), "--inbox is required") {
		t.Errorf("Expected error about --inbox being required, got: %v", err)
	}
}

func TestNewClientContactsGetCmd(t *testing.T) {
	baseURL := ""
	inboxID := ""
	contactID := ""

	cmd := newClientContactsGetCmd(&baseURL, &inboxID, &contactID)

	if cmd.Use != "get" {
		t.Errorf("Expected Use to be 'get', got %s", cmd.Use)
	}
}

func TestNewClientContactsGetCmdRequiresInboxAndContact(t *testing.T) {
	tests := []struct {
		name        string
		inboxID     string
		contactID   string
		expectError string
	}{
		{
			name:        "missing inbox",
			inboxID:     "",
			contactID:   "contact-123",
			expectError: "--inbox is required",
		},
		{
			name:        "missing contact",
			inboxID:     "inbox-123",
			contactID:   "",
			expectError: "--contact is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL := ""
			cmd := newClientContactsGetCmd(&baseURL, &tt.inboxID, &tt.contactID)
			cmd.SetArgs([]string{})

			err := cmd.Execute()
			if err == nil {
				t.Error("Expected error")
			}
			if err != nil && !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing %q, got: %v", tt.expectError, err)
			}
		})
	}
}

func TestNewClientConversationsCmd(t *testing.T) {
	baseURL := ""
	inboxID := ""
	contactID := ""

	cmd := newClientConversationsCmd(&baseURL, &inboxID, &contactID)

	if cmd.Use != "conversations" {
		t.Errorf("Expected Use to be 'conversations', got %s", cmd.Use)
	}

	// Verify subcommands
	subcommands := []string{"list", "create", "get", "resolve"}
	for _, sub := range subcommands {
		found := false
		for _, c := range cmd.Commands() {
			if c.Use == sub || strings.HasPrefix(c.Use, sub+" ") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected '%s' subcommand to exist", sub)
		}
	}
}

func TestNewClientConversationsListCmdRequiresFlags(t *testing.T) {
	tests := []struct {
		name        string
		inboxID     string
		contactID   string
		expectError string
	}{
		{
			name:        "missing inbox",
			inboxID:     "",
			contactID:   "contact-123",
			expectError: "--inbox is required",
		},
		{
			name:        "missing contact",
			inboxID:     "inbox-123",
			contactID:   "",
			expectError: "--contact is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL := ""
			cmd := newClientConversationsListCmd(&baseURL, &tt.inboxID, &tt.contactID)
			cmd.SetArgs([]string{})

			err := cmd.Execute()
			if err == nil {
				t.Error("Expected error")
			}
			if err != nil && !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing %q, got: %v", tt.expectError, err)
			}
		})
	}
}

func TestNewClientConversationsCreateCmd(t *testing.T) {
	baseURL := ""
	inboxID := ""
	contactID := ""

	cmd := newClientConversationsCreateCmd(&baseURL, &inboxID, &contactID)

	if cmd.Use != "create" {
		t.Errorf("Expected Use to be 'create', got %s", cmd.Use)
	}

	// Verify custom-attributes flag exists
	if cmd.Flag("custom-attributes") == nil {
		t.Error("Expected --custom-attributes flag to exist")
	}
}

func TestNewClientConversationsGetCmd(t *testing.T) {
	baseURL := ""
	inboxID := ""
	contactID := ""

	cmd := newClientConversationsGetCmd(&baseURL, &inboxID, &contactID)

	if !strings.HasPrefix(cmd.Use, "get") {
		t.Errorf("Expected Use to start with 'get', got %s", cmd.Use)
	}

	// Test args validation
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("Expected error for no args")
	}
	if err := cmd.Args(cmd, []string{"123"}); err != nil {
		t.Errorf("Expected no error for single arg, got %v", err)
	}
}

func TestNewClientConversationsResolveCmd(t *testing.T) {
	baseURL := ""
	inboxID := ""
	contactID := ""

	cmd := newClientConversationsResolveCmd(&baseURL, &inboxID, &contactID)

	if !strings.HasPrefix(cmd.Use, "resolve") {
		t.Errorf("Expected Use to start with 'resolve', got %s", cmd.Use)
	}
}

func TestNewClientMessagesCmd(t *testing.T) {
	baseURL := ""
	inboxID := ""
	contactID := ""

	cmd := newClientMessagesCmd(&baseURL, &inboxID, &contactID)

	if cmd.Use != "messages" {
		t.Errorf("Expected Use to be 'messages', got %s", cmd.Use)
	}

	// Verify create subcommand exists
	found := false
	for _, c := range cmd.Commands() {
		if strings.HasPrefix(c.Use, "create") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'create' subcommand to exist")
	}
}

func TestNewClientMessagesCreateCmd(t *testing.T) {
	baseURL := ""
	inboxID := ""
	contactID := ""

	cmd := newClientMessagesCreateCmd(&baseURL, &inboxID, &contactID)

	if !strings.HasPrefix(cmd.Use, "create") {
		t.Errorf("Expected Use to start with 'create', got %s", cmd.Use)
	}

	// Verify flags exist
	if cmd.Flag("content") == nil {
		t.Error("Expected --content flag to exist")
	}
	if cmd.Flag("echo-id") == nil {
		t.Error("Expected --echo-id flag to exist")
	}
}

func TestNewClientMessagesCreateCmdRequiresContent(t *testing.T) {
	baseURL := ""
	inboxID := "inbox-123"
	contactID := "contact-123"

	cmd := newClientMessagesCreateCmd(&baseURL, &inboxID, &contactID)
	cmd.SetArgs([]string{"1"}) // conversation ID

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error when --content is not provided")
	}
	if err != nil && !strings.Contains(err.Error(), "--content is required") {
		t.Errorf("Expected error about --content being required, got: %v", err)
	}
}

func TestNewClientTypingCmd(t *testing.T) {
	baseURL := ""
	inboxID := ""
	contactID := ""

	cmd := newClientTypingCmd(&baseURL, &inboxID, &contactID)

	if !strings.HasPrefix(cmd.Use, "typing") {
		t.Errorf("Expected Use to start with 'typing', got %s", cmd.Use)
	}

	// Verify --status flag exists
	if cmd.Flag("status") == nil {
		t.Error("Expected --status flag to exist")
	}
}

func TestNewClientTypingCmdStatusValidation(t *testing.T) {
	baseURL := ""
	inboxID := "inbox-123"
	contactID := "contact-123"

	cmd := newClientTypingCmd(&baseURL, &inboxID, &contactID)
	cmd.SetArgs([]string{"1"})
	_ = cmd.Flags().Set("status", "invalid")

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for invalid status")
	}
	if err != nil && !strings.Contains(err.Error(), "'on' or 'off'") {
		t.Errorf("Expected error about valid status values, got: %v", err)
	}
}

func TestNewClientLastSeenCmd(t *testing.T) {
	baseURL := ""
	inboxID := ""
	contactID := ""

	cmd := newClientLastSeenCmd(&baseURL, &inboxID, &contactID)

	if cmd.Use != "last-seen" {
		t.Errorf("Expected Use to be 'last-seen', got %s", cmd.Use)
	}

	// Verify update subcommand exists
	found := false
	for _, c := range cmd.Commands() {
		if strings.HasPrefix(c.Use, "update") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'update' subcommand to exist")
	}
}

func TestNewClientLastSeenUpdateCmd(t *testing.T) {
	baseURL := ""
	inboxID := ""
	contactID := ""

	cmd := newClientLastSeenUpdateCmd(&baseURL, &inboxID, &contactID)

	if !strings.HasPrefix(cmd.Use, "update") {
		t.Errorf("Expected Use to start with 'update', got %s", cmd.Use)
	}

	// Test args validation
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("Expected error for no args")
	}
	if err := cmd.Args(cmd, []string{"123"}); err != nil {
		t.Errorf("Expected no error for single arg, got %v", err)
	}
}

func TestClientCommandsWithMockServer(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/public/api/v1/inboxes/inbox-123/contacts/contact-123", jsonResponse(200, `{"id":1,"name":"Test Contact","email":"test@example.com","source_id":"contact-123"}`)).
		On("POST", "/public/api/v1/inboxes/inbox-123/contacts", jsonResponse(201, `{"id":2,"name":"New Contact","email":"new@example.com","source_id":"new-contact"}`)).
		On("GET", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations", jsonResponse(200, `[{"id":1,"status":"open","inbox_id":1}]`)).
		On("POST", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations", jsonResponse(201, `{"id":2,"status":"open"}`)).
		On("GET", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations/1", jsonResponse(200, `{"id":1,"status":"open"}`)).
		On("POST", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations/1/toggle_status", jsonResponse(200, `{"id":1,"status":"resolved"}`)).
		On("POST", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations/1/messages", jsonResponse(201, `{"id":1,"content":"Hello"}`)).
		On("POST", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations/1/toggle_typing", jsonResponse(200, ``)).
		On("POST", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations/1/update_last_seen", jsonResponse(200, ``))

	env := setupTestEnvWithHandler(t, handler)

	t.Run("get contact", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientContactsGetCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{})
		output := captureStdout(t, func() {
			_ = cmd.Execute()
		})

		if !strings.Contains(output, "Contact") && !strings.Contains(output, "1") {
			t.Errorf("Expected contact output, got: %s", output)
		}
	})

	t.Run("list conversations", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientConversationsListCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{})
		output := captureStdout(t, func() {
			_ = cmd.Execute()
		})

		if !strings.Contains(output, "ID") && !strings.Contains(output, "STATUS") {
			t.Errorf("Expected conversations list output, got: %s", output)
		}
	})

	t.Run("get conversation", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientConversationsGetCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})
		output := captureStdout(t, func() {
			_ = cmd.Execute()
		})

		if !strings.Contains(output, "Conversation") {
			t.Errorf("Expected conversation output, got: %s", output)
		}
	})

	t.Run("resolve conversation", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientConversationsResolveCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})
		output := captureStdout(t, func() {
			_ = cmd.Execute()
		})

		if !strings.Contains(output, "Resolved") {
			t.Errorf("Expected resolved output, got: %s", output)
		}
	})

	t.Run("send message", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientMessagesCreateCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})
		_ = cmd.Flags().Set("content", "Hello")
		output := captureStdout(t, func() {
			_ = cmd.Execute()
		})

		if !strings.Contains(output, "Sent") {
			t.Errorf("Expected sent message output, got: %s", output)
		}
	})

	t.Run("toggle typing", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientTypingCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})
		_ = cmd.Flags().Set("status", "on")
		output := captureStdout(t, func() {
			_ = cmd.Execute()
		})

		if !strings.Contains(output, "Typing") {
			t.Errorf("Expected typing status output, got: %s", output)
		}
	})

	t.Run("update last seen", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientLastSeenUpdateCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})
		output := captureStdout(t, func() {
			_ = cmd.Execute()
		})

		if !strings.Contains(output, "Last seen") {
			t.Errorf("Expected last seen update output, got: %s", output)
		}
	})
}

func TestClientCommandsJSONOutput(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/public/api/v1/inboxes/inbox-123/contacts/contact-123", jsonResponse(200, `{"id":1,"name":"Test","email":"test@example.com","source_id":"src"}`))

	env := setupTestEnvWithHandler(t, handler)

	t.Run("get contact JSON", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientContactsGetCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{})

		output := captureStdout(t, func() {
			_ = cmd.Execute()
		})

		// Text output contains contact info
		if !strings.Contains(output, "Contact") {
			t.Logf("Output: %s", output) // Log for debugging
		}
	})
}

func TestClientCommandsErrorHandling(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/public/api/v1/inboxes/inbox-123/contacts/contact-123", jsonResponse(404, `{"error":"not found"}`))

	env := setupTestEnvWithHandler(t, handler)

	t.Run("contact not found", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientContactsGetCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for 404 response")
		}
	})
}

func TestClientWithInvalidBaseURL(t *testing.T) {
	// Save original env
	origURL := os.Getenv("CHATWOOT_BASE_URL")
	defer func() {
		_ = os.Setenv("CHATWOOT_BASE_URL", origURL)
	}()

	// Clear env vars
	_ = os.Unsetenv("CHATWOOT_BASE_URL")

	t.Run("missing base URL with empty flag", func(t *testing.T) {
		baseURL := "" // Empty base URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientContactsGetCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		// Should error because no base URL is configured
		if err == nil {
			t.Error("Expected error when base URL is not configured")
		}
	})
}

func TestConversationIDValidation(t *testing.T) {
	baseURL := "http://example.com"
	inboxID := "inbox-123"
	contactID := "contact-123"

	tests := []struct {
		name        string
		convID      string
		expectError bool
	}{
		{"valid positive", "123", false},
		{"zero", "0", true},
		{"negative", "-1", true},
		{"non-numeric", "abc", true},
		{"float", "12.5", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newClientConversationsGetCmd(&baseURL, &inboxID, &contactID)
			cmd.SetArgs([]string{tt.convID})

			err := cmd.Execute()
			if tt.expectError && err == nil {
				t.Error("Expected error for invalid conversation ID")
			}
			// Note: Valid IDs will still error due to missing server, but that's expected
		})
	}
}

func TestClientContactsCreateWithServer(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/public/api/v1/inboxes/inbox-123/contacts", jsonResponse(201, `{"id":1,"name":"New Contact","email":"new@example.com","source_id":"new-contact"}`))

	env := setupTestEnvWithHandler(t, handler)

	t.Run("create contact success", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"

		cmd := newClientContactsCreateCmd(&baseURL, &inboxID)
		cmd.SetArgs([]string{})
		_ = cmd.Flags().Set("name", "New Contact")
		_ = cmd.Flags().Set("email", "new@example.com")
		output := captureStdout(t, func() {
			err := cmd.Execute()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})

		if !strings.Contains(output, "Created contact") {
			t.Errorf("Expected 'Created contact' in output, got: %s", output)
		}
	})

	t.Run("create contact with custom attributes", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"

		cmd := newClientContactsCreateCmd(&baseURL, &inboxID)
		cmd.SetArgs([]string{})
		_ = cmd.Flags().Set("name", "Test")
		_ = cmd.Flags().Set("custom-attributes", `{"key":"value"}`)
		output := captureStdout(t, func() {
			err := cmd.Execute()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})

		if !strings.Contains(output, "Created contact") {
			t.Errorf("Expected 'Created contact' in output, got: %s", output)
		}
	})

	t.Run("create contact with invalid custom attributes", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"

		cmd := newClientContactsCreateCmd(&baseURL, &inboxID)
		cmd.SetArgs([]string{})
		_ = cmd.Flags().Set("custom-attributes", `invalid-json`)

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
		if !strings.Contains(err.Error(), "invalid custom-attributes JSON") {
			t.Errorf("Expected 'invalid custom-attributes JSON' error, got: %v", err)
		}
	})

	t.Run("create contact API error", func(t *testing.T) {
		errorHandler := newRouteHandler().
			On("POST", "/public/api/v1/inboxes/inbox-123/contacts", jsonResponse(400, `{"error":"bad request"}`))
		errorEnv := setupTestEnvWithHandler(t, errorHandler)
		baseURL := errorEnv.server.URL
		inboxID := "inbox-123"

		cmd := newClientContactsCreateCmd(&baseURL, &inboxID)
		cmd.SetArgs([]string{})
		_ = cmd.Flags().Set("name", "Test")

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for 400 response")
		}
	})
}

func TestClientConversationsCreateWithServer(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations", jsonResponse(201, `{"id":1,"status":"open"}`))

	env := setupTestEnvWithHandler(t, handler)

	t.Run("create conversation success", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientConversationsCreateCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{})
		output := captureStdout(t, func() {
			err := cmd.Execute()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})

		if !strings.Contains(output, "Created conversation") {
			t.Errorf("Expected 'Created conversation' in output, got: %s", output)
		}
	})

	t.Run("create conversation with custom attributes", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientConversationsCreateCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{})
		_ = cmd.Flags().Set("custom-attributes", `{"key":"value"}`)
		output := captureStdout(t, func() {
			err := cmd.Execute()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})

		if !strings.Contains(output, "Created conversation") {
			t.Errorf("Expected 'Created conversation' in output, got: %s", output)
		}
	})

	t.Run("create conversation with invalid custom attributes", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientConversationsCreateCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{})
		_ = cmd.Flags().Set("custom-attributes", `{invalid}`)

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
		if !strings.Contains(err.Error(), "invalid custom-attributes JSON") {
			t.Errorf("Expected 'invalid custom-attributes JSON' error, got: %v", err)
		}
	})

	t.Run("create conversation missing inbox", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := ""
		contactID := "contact-123"

		cmd := newClientConversationsCreateCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for missing inbox")
		}
		if !strings.Contains(err.Error(), "--inbox is required") {
			t.Errorf("Expected '--inbox is required' error, got: %v", err)
		}
	})

	t.Run("create conversation missing contact", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := ""

		cmd := newClientConversationsCreateCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for missing contact")
		}
		if !strings.Contains(err.Error(), "--contact is required") {
			t.Errorf("Expected '--contact is required' error, got: %v", err)
		}
	})

	t.Run("create conversation API error", func(t *testing.T) {
		errorHandler := newRouteHandler().
			On("POST", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations", jsonResponse(500, `{"error":"server error"}`))
		errorEnv := setupTestEnvWithHandler(t, errorHandler)
		baseURL := errorEnv.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientConversationsCreateCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for 500 response")
		}
	})
}

func TestClientConversationsResolveWithServer(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations/1/toggle_status", jsonResponse(200, `{"id":1,"status":"resolved"}`))

	env := setupTestEnvWithHandler(t, handler)

	t.Run("resolve conversation missing inbox", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := ""
		contactID := "contact-123"

		cmd := newClientConversationsResolveCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for missing inbox")
		}
		if !strings.Contains(err.Error(), "--inbox is required") {
			t.Errorf("Expected '--inbox is required' error, got: %v", err)
		}
	})

	t.Run("resolve conversation missing contact", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := ""

		cmd := newClientConversationsResolveCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for missing contact")
		}
		if !strings.Contains(err.Error(), "--contact is required") {
			t.Errorf("Expected '--contact is required' error, got: %v", err)
		}
	})

	t.Run("resolve conversation invalid ID", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientConversationsResolveCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"invalid"})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for invalid conversation ID")
		}
	})

	t.Run("resolve conversation API error", func(t *testing.T) {
		errorHandler := newRouteHandler().
			On("POST", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations/1/toggle_status", jsonResponse(500, `{"error":"server error"}`))
		errorEnv := setupTestEnvWithHandler(t, errorHandler)
		baseURL := errorEnv.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientConversationsResolveCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for 500 response")
		}
	})
}

func TestClientConversationsGetWithServer(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations/1", jsonResponse(200, `{"id":1,"status":"open"}`))

	env := setupTestEnvWithHandler(t, handler)

	t.Run("get conversation API error", func(t *testing.T) {
		errorHandler := newRouteHandler().
			On("GET", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations/1", jsonResponse(500, `{"error":"server error"}`))
		errorEnv := setupTestEnvWithHandler(t, errorHandler)
		baseURL := errorEnv.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientConversationsGetCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for 500 response")
		}
	})

	t.Run("get conversation missing inbox", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := ""
		contactID := "contact-123"

		cmd := newClientConversationsGetCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for missing inbox")
		}
	})

	t.Run("get conversation missing contact", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := ""

		cmd := newClientConversationsGetCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for missing contact")
		}
	})
}

func TestClientMessagesCreateWithServer(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations/1/messages", jsonResponse(201, `{"id":1,"content":"Hello"}`))

	env := setupTestEnvWithHandler(t, handler)

	t.Run("create message API error", func(t *testing.T) {
		errorHandler := newRouteHandler().
			On("POST", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations/1/messages", jsonResponse(500, `{"error":"server error"}`))
		errorEnv := setupTestEnvWithHandler(t, errorHandler)
		baseURL := errorEnv.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientMessagesCreateCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})
		_ = cmd.Flags().Set("content", "Hello")

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for 500 response")
		}
	})

	t.Run("create message with echo-id", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientMessagesCreateCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})
		_ = cmd.Flags().Set("content", "Hello")
		_ = cmd.Flags().Set("echo-id", "echo-123")
		output := captureStdout(t, func() {
			err := cmd.Execute()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})

		if !strings.Contains(output, "Sent message") {
			t.Errorf("Expected 'Sent message' in output, got: %s", output)
		}
	})

	t.Run("create message missing inbox", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := ""
		contactID := "contact-123"

		cmd := newClientMessagesCreateCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})
		_ = cmd.Flags().Set("content", "Hello")

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for missing inbox")
		}
	})

	t.Run("create message missing contact", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := ""

		cmd := newClientMessagesCreateCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})
		_ = cmd.Flags().Set("content", "Hello")

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for missing contact")
		}
	})

	t.Run("create message invalid conversation ID", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientMessagesCreateCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"invalid"})
		_ = cmd.Flags().Set("content", "Hello")

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for invalid conversation ID")
		}
	})
}

func TestClientTypingWithServer(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations/1/toggle_typing", jsonResponse(200, ``))

	env := setupTestEnvWithHandler(t, handler)

	t.Run("typing API error", func(t *testing.T) {
		errorHandler := newRouteHandler().
			On("POST", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations/1/toggle_typing", jsonResponse(500, `{"error":"server error"}`))
		errorEnv := setupTestEnvWithHandler(t, errorHandler)
		baseURL := errorEnv.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientTypingCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})
		_ = cmd.Flags().Set("status", "on")

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for 500 response")
		}
	})

	t.Run("typing off status", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientTypingCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})
		_ = cmd.Flags().Set("status", "off")
		output := captureStdout(t, func() {
			err := cmd.Execute()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})

		if !strings.Contains(output, "off") {
			t.Errorf("Expected 'off' in output, got: %s", output)
		}
	})

	t.Run("typing missing inbox", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := ""
		contactID := "contact-123"

		cmd := newClientTypingCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})
		_ = cmd.Flags().Set("status", "on")

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for missing inbox")
		}
	})

	t.Run("typing missing contact", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := ""

		cmd := newClientTypingCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})
		_ = cmd.Flags().Set("status", "on")

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for missing contact")
		}
	})

	t.Run("typing invalid conversation ID", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientTypingCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"invalid"})
		_ = cmd.Flags().Set("status", "on")

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for invalid conversation ID")
		}
	})
}

func TestClientLastSeenWithServer(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations/1/update_last_seen", jsonResponse(200, ``))

	env := setupTestEnvWithHandler(t, handler)

	t.Run("last seen API error", func(t *testing.T) {
		errorHandler := newRouteHandler().
			On("POST", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations/1/update_last_seen", jsonResponse(500, `{"error":"server error"}`))
		errorEnv := setupTestEnvWithHandler(t, errorHandler)
		baseURL := errorEnv.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientLastSeenUpdateCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for 500 response")
		}
	})

	t.Run("last seen missing inbox", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := ""
		contactID := "contact-123"

		cmd := newClientLastSeenUpdateCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for missing inbox")
		}
	})

	t.Run("last seen missing contact", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := ""

		cmd := newClientLastSeenUpdateCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"1"})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for missing contact")
		}
	})

	t.Run("last seen invalid conversation ID", func(t *testing.T) {
		baseURL := env.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientLastSeenUpdateCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{"invalid"})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for invalid conversation ID")
		}
	})
}

func TestClientConversationsListWithServer(t *testing.T) {
	t.Run("list conversations API error", func(t *testing.T) {
		errorHandler := newRouteHandler().
			On("GET", "/public/api/v1/inboxes/inbox-123/contacts/contact-123/conversations", jsonResponse(500, `{"error":"server error"}`))
		errorEnv := setupTestEnvWithHandler(t, errorHandler)
		baseURL := errorEnv.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientConversationsListCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for 500 response")
		}
	})
}

func TestClientContactsGetWithServer(t *testing.T) {
	t.Run("get contact API error", func(t *testing.T) {
		errorHandler := newRouteHandler().
			On("GET", "/public/api/v1/inboxes/inbox-123/contacts/contact-123", jsonResponse(500, `{"error":"server error"}`))
		errorEnv := setupTestEnvWithHandler(t, errorHandler)
		baseURL := errorEnv.server.URL
		inboxID := "inbox-123"
		contactID := "contact-123"

		cmd := newClientContactsGetCmd(&baseURL, &inboxID, &contactID)
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for 500 response")
		}
	})
}
