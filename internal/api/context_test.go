package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestGetConversationContext(t *testing.T) {
	tests := []struct {
		name                 string
		conversationID       int
		embedImages          bool
		conversationResponse string
		messagesResponse     string
		contactResponse      string
		expectError          bool
		validateFunc         func(*testing.T, *ConversationContext)
	}{
		{
			name:           "successful context with contact and messages",
			conversationID: 123,
			embedImages:    false,
			conversationResponse: `{
				"id": 123,
				"account_id": 1,
				"inbox_id": 5,
				"status": "open",
				"contact_id": 456,
				"created_at": 1700000000,
				"meta": {"channel": "email"}
			}`,
			messagesResponse: `{
				"payload": [
					{
						"id": 1,
						"conversation_id": 123,
						"content": "Hello, I need help",
						"content_type": "text",
						"message_type": 0,
						"private": false,
						"created_at": 1700000000,
						"sender_type": "Contact"
					},
					{
						"id": 2,
						"conversation_id": 123,
						"content": "Sure, how can I assist?",
						"content_type": "text",
						"message_type": 1,
						"private": false,
						"created_at": 1700001000,
						"sender_type": "User"
					}
				]
			}`,
			contactResponse: `{
				"payload": {
					"id": 456,
					"name": "John Doe",
					"email": "john@example.com",
					"created_at": 1699900000
				}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, ctx *ConversationContext) {
				if ctx.Conversation == nil {
					t.Fatal("Expected conversation, got nil")
				}
				if ctx.Conversation.ID != 123 {
					t.Errorf("Expected conversation ID 123, got %d", ctx.Conversation.ID)
				}
				if ctx.Contact == nil {
					t.Fatal("Expected contact, got nil")
				}
				if ctx.Contact.Name != "John Doe" {
					t.Errorf("Expected contact name 'John Doe', got %s", ctx.Contact.Name)
				}
				if len(ctx.Messages) != 2 {
					t.Errorf("Expected 2 messages, got %d", len(ctx.Messages))
				}
				if ctx.Messages[0].Content != "Hello, I need help" {
					t.Errorf("Expected message content 'Hello, I need help', got %s", ctx.Messages[0].Content)
				}
				if ctx.Summary == "" {
					t.Error("Expected summary to be generated")
				}
				// Check summary contains expected info
				if !strings.Contains(ctx.Summary, "John Doe") {
					t.Error("Expected summary to contain customer name")
				}
				if !strings.Contains(ctx.Summary, "open") {
					t.Error("Expected summary to contain status")
				}
				if !strings.Contains(ctx.Summary, "Messages: 2") {
					t.Error("Expected summary to contain message count")
				}
			},
		},
		{
			name:           "context without contact",
			conversationID: 123,
			embedImages:    false,
			conversationResponse: `{
				"id": 123,
				"account_id": 1,
				"inbox_id": 5,
				"status": "pending",
				"contact_id": 0,
				"created_at": 1700000000
			}`,
			messagesResponse: `{
				"payload": [
					{
						"id": 1,
						"conversation_id": 123,
						"content": "Hello",
						"content_type": "text",
						"message_type": 0,
						"private": false,
						"created_at": 1700000000
					}
				]
			}`,
			contactResponse: "",
			expectError:     false,
			validateFunc: func(t *testing.T, ctx *ConversationContext) {
				if ctx.Contact != nil {
					t.Error("Expected no contact, got one")
				}
				if len(ctx.Messages) != 1 {
					t.Errorf("Expected 1 message, got %d", len(ctx.Messages))
				}
			},
		},
		{
			name:           "context with attachments no embedding",
			conversationID: 123,
			embedImages:    false,
			conversationResponse: `{
				"id": 123,
				"account_id": 1,
				"inbox_id": 5,
				"status": "open",
				"contact_id": 0,
				"created_at": 1700000000
			}`,
			messagesResponse: `{
				"payload": [
					{
						"id": 1,
						"conversation_id": 123,
						"content": "See attached",
						"content_type": "text",
						"message_type": 0,
						"private": false,
						"created_at": 1700000000,
						"attachments": [
							{
								"id": 10,
								"file_type": "image",
								"data_url": "https://example.com/image.jpg",
								"file_size": 12345
							}
						]
					}
				]
			}`,
			contactResponse: "",
			expectError:     false,
			validateFunc: func(t *testing.T, ctx *ConversationContext) {
				if len(ctx.Messages) != 1 {
					t.Fatalf("Expected 1 message, got %d", len(ctx.Messages))
				}
				if len(ctx.Messages[0].Attachments) != 1 {
					t.Fatalf("Expected 1 attachment, got %d", len(ctx.Messages[0].Attachments))
				}
				att := ctx.Messages[0].Attachments[0]
				if att.ID != 10 {
					t.Errorf("Expected attachment ID 10, got %d", att.ID)
				}
				if att.FileType != "image" {
					t.Errorf("Expected file type 'image', got %s", att.FileType)
				}
				if att.Embedded != "" {
					t.Error("Expected no embedded data when embedImages=false")
				}
				// Check summary mentions attachments
				if !strings.Contains(ctx.Summary, "Attachments: 1") {
					t.Error("Expected summary to contain attachment count")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				path := r.URL.Path
				switch {
				case strings.Contains(path, "/messages"):
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(tt.messagesResponse))
				case strings.Contains(path, "/contacts/"):
					if tt.contactResponse == "" {
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(`{"error": "Not found"}`))
					} else {
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(tt.contactResponse))
					}
				case strings.Contains(path, "/conversations/"):
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(tt.conversationResponse))
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.GetConversationContext(context.Background(), tt.conversationID, tt.embedImages)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.validateFunc != nil && result != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestGetConversationContextPaginatesMessages(t *testing.T) {
	conversationResponse := `{
		"id": 123,
		"account_id": 1,
		"inbox_id": 5,
		"status": "open",
		"contact_id": 0,
		"created_at": 1700000000
	}`

	page1 := `{"payload":[{"id":3,"conversation_id":123,"content":"first","content_type":"text","message_type":0,"private":false,"created_at":1700000001},{"id":2,"conversation_id":123,"content":"second","content_type":"text","message_type":0,"private":false,"created_at":1700000002}]}`
	page2 := `{"payload":[{"id":1,"conversation_id":123,"content":"third","content_type":"text","message_type":0,"private":false,"created_at":1700000003}]}`
	empty := `{"payload":[]}`

	var mu sync.Mutex
	var befores []string

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.Path, "/messages"):
			before := r.URL.Query().Get("before")
			mu.Lock()
			befores = append(befores, before)
			mu.Unlock()
			switch before {
			case "":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(page1))
			case "2":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(page2))
			default:
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(empty))
			}
		case strings.Contains(r.URL.Path, "/conversations/"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(conversationResponse))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer apiServer.Close()

	client := newTestClient(apiServer.URL, "test-token", 1)
	result, err := client.GetConversationContext(context.Background(), 123, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result.Messages))
	}

	mu.Lock()
	defer mu.Unlock()
	if len(befores) < 2 || befores[0] != "" || befores[1] != "2" {
		t.Fatalf("unexpected pagination sequence: %v", befores)
	}
}

func TestGetConversationContextWithImageEmbedding(t *testing.T) {
	// Create a test image server
	imageData := []byte("fake image data for testing")
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imageData)
	}))
	defer imageServer.Close()

	conversationResponse := `{
		"id": 123,
		"account_id": 1,
		"inbox_id": 5,
		"status": "open",
		"contact_id": 0,
		"created_at": 1700000000
	}`
	messagesResponse := `{
		"payload": [
			{
				"id": 1,
				"conversation_id": 123,
				"content": "See attached",
				"content_type": "text",
				"message_type": 0,
				"private": false,
				"created_at": 1700000000,
				"attachments": [
					{
						"id": 10,
						"file_type": "image",
						"data_url": "` + imageServer.URL + `/image.jpg",
						"file_size": 12345
					}
				]
			}
		]
	}`

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		path := r.URL.Path
		switch {
		case strings.Contains(path, "/messages"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(messagesResponse))
		case strings.Contains(path, "/conversations/"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(conversationResponse))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer apiServer.Close()

	client := newTestClient(apiServer.URL, "test-token", 1)
	result, err := client.GetConversationContext(context.Background(), 123, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(result.Messages))
	}
	if len(result.Messages[0].Attachments) != 1 {
		t.Fatalf("Expected 1 attachment, got %d", len(result.Messages[0].Attachments))
	}

	att := result.Messages[0].Attachments[0]
	if att.Embedded == "" {
		t.Error("Expected embedded data to be present")
	}
	if !strings.HasPrefix(att.Embedded, "data:image/jpeg;base64,") {
		t.Errorf("Expected data URI format, got %s", att.Embedded[:50])
	}

	// Verify the base64 encoded data
	expectedEncoded := base64.StdEncoding.EncodeToString(imageData)
	expectedDataURI := "data:image/jpeg;base64," + expectedEncoded
	if att.Embedded != expectedDataURI {
		t.Errorf("Embedded data mismatch")
	}
}

func TestDownloadAndEncode(t *testing.T) {
	tests := []struct {
		name         string
		contentType  string
		fileType     string
		responseCode int
		responseBody []byte
		expectError  bool
		validateFunc func(*testing.T, string)
	}{
		{
			name:         "successful image download",
			contentType:  "image/png",
			fileType:     "image",
			responseCode: http.StatusOK,
			responseBody: []byte("fake png data"),
			expectError:  false,
			validateFunc: func(t *testing.T, result string) {
				if !strings.HasPrefix(result, "data:image/png;base64,") {
					t.Errorf("Expected data:image/png;base64, prefix, got %s", result[:30])
				}
				encoded := base64.StdEncoding.EncodeToString([]byte("fake png data"))
				if !strings.HasSuffix(result, encoded) {
					t.Error("Base64 encoding mismatch")
				}
			},
		},
		{
			name:         "jpeg with content-type",
			contentType:  "image/jpeg",
			fileType:     "image",
			responseCode: http.StatusOK,
			responseBody: []byte{0xFF, 0xD8, 0xFF},
			expectError:  false,
			validateFunc: func(t *testing.T, result string) {
				if !strings.HasPrefix(result, "data:image/jpeg;base64,") {
					t.Errorf("Expected data:image/jpeg;base64, prefix, got %s", result)
				}
			},
		},
		{
			name:         "fallback mime type for image without content-type",
			contentType:  "",
			fileType:     "image",
			responseCode: http.StatusOK,
			responseBody: []byte("data"),
			expectError:  false,
			validateFunc: func(t *testing.T, result string) {
				if !strings.HasPrefix(result, "data:image/jpeg;base64,") {
					t.Errorf("Expected fallback to image/jpeg, got %s", result)
				}
			},
		},
		{
			name:         "unknown file type fallback",
			contentType:  "",
			fileType:     "document",
			responseCode: http.StatusOK,
			responseBody: []byte("pdf data"),
			expectError:  false,
			validateFunc: func(t *testing.T, result string) {
				if !strings.HasPrefix(result, "data:application/octet-stream;base64,") {
					t.Errorf("Expected fallback to application/octet-stream, got %s", result)
				}
			},
		},
		{
			name:         "download failure - 404",
			contentType:  "",
			fileType:     "image",
			responseCode: http.StatusNotFound,
			responseBody: nil,
			expectError:  true,
		},
		{
			name:         "download failure - server error",
			contentType:  "",
			fileType:     "image",
			responseCode: http.StatusInternalServerError,
			responseBody: nil,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.contentType != "" {
					w.Header().Set("Content-Type", tt.contentType)
				}
				w.WriteHeader(tt.responseCode)
				if tt.responseBody != nil {
					_, _ = w.Write(tt.responseBody)
				}
			}))
			defer server.Close()

			client := newTestClient("https://api.example.com", "test-token", 1)
			result, err := client.downloadAndEncode(context.Background(), server.URL+"/image.jpg", tt.fileType)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.validateFunc != nil && !tt.expectError {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestDownloadAndEncodeRejectsInvalidURL(t *testing.T) {
	client := New("https://example.com", "test-token", 1)
	_, err := client.downloadAndEncode(context.Background(), "http://127.0.0.1/image.jpg", "image")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestDownloadAndEncodeRejectsLargeAttachment(t *testing.T) {
	data := bytes.Repeat([]byte("a"), maxEmbeddedAttachmentBytes+1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(data)
	}))
	defer server.Close()

	client := newTestClient("https://api.example.com", "test-token", 1)
	_, err := client.downloadAndEncode(context.Background(), server.URL+"/image.jpg", "image")
	if err == nil {
		t.Fatal("expected error for oversized attachment")
	}
}

func TestIsImageType(t *testing.T) {
	tests := []struct {
		fileType string
		expected bool
	}{
		{"image", true},
		{"image/jpeg", true},
		{"image/png", true},
		{"image/gif", true},
		{"image/webp", true},
		{"IMAGE", true},      // Case insensitive
		{"Image/JPEG", true}, // Case insensitive
		{"document", false},
		{"video", false},
		{"audio", false},
		{"application/pdf", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.fileType, func(t *testing.T) {
			result := isImageType(tt.fileType)
			if result != tt.expected {
				t.Errorf("isImageType(%q) = %v, want %v", tt.fileType, result, tt.expected)
			}
		})
	}
}

func TestGetMimeType(t *testing.T) {
	tests := []struct {
		fileType    string
		contentType string
		expected    string
	}{
		{"image", "image/jpeg", "image/jpeg"},
		{"image", "image/png", "image/png"},
		{"image", "", "image/jpeg"},                        // Fallback for "image" type
		{"document", "", "application/octet-stream"},       // Fallback for unknown
		{"video", "video/mp4", "application/octet-stream"}, // Non-image content-type ignored
		{"", "image/gif", "image/gif"},
	}

	for _, tt := range tests {
		name := tt.fileType + "_" + tt.contentType
		if name == "_" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			result := getMimeType(tt.fileType, tt.contentType)
			if result != tt.expected {
				t.Errorf("getMimeType(%q, %q) = %q, want %q", tt.fileType, tt.contentType, result, tt.expected)
			}
		})
	}
}

func TestGenerateContextSummary(t *testing.T) {
	tests := []struct {
		name     string
		ctx      *ConversationContext
		contains []string
	}{
		{
			name: "full context with contact and channel",
			ctx: &ConversationContext{
				Conversation: &Conversation{
					Status: "open",
					Meta:   map[string]any{"channel": "email"},
				},
				Contact: &Contact{
					Name:  "Jane Doe",
					Email: "jane@example.com",
				},
				Messages: []MessageWithEmbeddings{
					{ID: 1, Content: "Hello"},
					{ID: 2, Content: "Hi there"},
					{ID: 3, Content: "Thanks", Attachments: []EmbeddedAttachment{{ID: 1}}},
				},
			},
			contains: []string{"Jane Doe", "jane@example.com", "open", "email", "Messages: 3", "Attachments: 1"},
		},
		{
			name: "context without contact",
			ctx: &ConversationContext{
				Conversation: &Conversation{
					Status: "pending",
				},
				Messages: []MessageWithEmbeddings{
					{ID: 1, Content: "Test"},
				},
			},
			contains: []string{"pending", "Messages: 1"},
		},
		{
			name: "context without channel in meta",
			ctx: &ConversationContext{
				Conversation: &Conversation{
					Status: "resolved",
					Meta:   map[string]any{},
				},
				Contact: &Contact{
					Name: "Bob",
				},
				Messages: []MessageWithEmbeddings{},
			},
			contains: []string{"Bob", "resolved", "Messages: 0"},
		},
		{
			name: "nil conversation and contact",
			ctx: &ConversationContext{
				Messages: []MessageWithEmbeddings{
					{ID: 1, Content: "Orphan message"},
				},
			},
			contains: []string{"Messages: 1"},
		},
		{
			name: "multiple attachments across messages",
			ctx: &ConversationContext{
				Conversation: &Conversation{Status: "open"},
				Messages: []MessageWithEmbeddings{
					{ID: 1, Attachments: []EmbeddedAttachment{{ID: 1}, {ID: 2}}},
					{ID: 2, Attachments: []EmbeddedAttachment{{ID: 3}}},
				},
			},
			contains: []string{"Attachments: 3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateContextSummary(tt.ctx)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected summary to contain %q, got %q", expected, result)
				}
			}
		})
	}
}

func TestGetConversationContextError(t *testing.T) {
	tests := []struct {
		name             string
		conversationErr  bool
		messagesErr      bool
		expectedContains string
	}{
		{
			name:             "conversation fetch error",
			conversationErr:  true,
			messagesErr:      false,
			expectedContains: "failed to get conversation",
		},
		{
			name:             "messages fetch error",
			conversationErr:  false,
			messagesErr:      true,
			expectedContains: "failed to get messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path

				if tt.conversationErr && strings.Contains(path, "/conversations/") && !strings.Contains(path, "/messages") {
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"error": "Not found"}`))
					return
				}

				if tt.messagesErr && strings.Contains(path, "/messages") {
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"error": "Server error"}`))
					return
				}

				w.Header().Set("Content-Type", "application/json")
				if strings.Contains(path, "/messages") {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"payload": []}`))
				} else if strings.Contains(path, "/conversations/") {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"id": 123, "status": "open", "created_at": 1700000000}`))
				}
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			_, err := client.GetConversationContext(context.Background(), 123, false)

			if err == nil {
				t.Fatal("Expected error but got nil")
			}

			if !strings.Contains(err.Error(), tt.expectedContains) {
				t.Errorf("Expected error to contain %q, got %q", tt.expectedContains, err.Error())
			}
		})
	}
}
