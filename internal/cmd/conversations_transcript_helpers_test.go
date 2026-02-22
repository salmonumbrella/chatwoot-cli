package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func TestTranscriptActor(t *testing.T) {
	tests := []struct {
		name string
		msg  api.Message
		want string
	}{
		{name: "private with sender", msg: api.Message{Private: true, Sender: &api.MessageSender{Name: "Agent A"}}, want: "Private note (Agent A)"},
		{name: "private", msg: api.Message{Private: true}, want: "Private note"},
		{name: "incoming sender", msg: api.Message{MessageType: api.MessageTypeIncoming, Sender: &api.MessageSender{Name: "Customer A"}}, want: "Customer (Customer A)"},
		{name: "incoming", msg: api.Message{MessageType: api.MessageTypeIncoming}, want: "Customer"},
		{name: "outgoing sender", msg: api.Message{MessageType: api.MessageTypeOutgoing, Sender: &api.MessageSender{Name: "Agent B"}}, want: "Agent (Agent B)"},
		{name: "activity", msg: api.Message{MessageType: api.MessageTypeActivity}, want: "Activity"},
		{name: "template", msg: api.Message{MessageType: api.MessageTypeTemplate}, want: "Template"},
		{name: "fallback sender", msg: api.Message{MessageType: 99, Sender: &api.MessageSender{Name: "System"}}, want: "System"},
		{name: "fallback type", msg: api.Message{MessageType: 99}, want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transcriptActor(tt.msg); got != tt.want {
				t.Fatalf("transcriptActor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteTranscript(t *testing.T) {
	priority := "high"
	displayID := 77
	assigneeID := 9
	teamID := 4
	conv := &api.Conversation{
		ID:             33,
		DisplayID:      &displayID,
		Status:         "open",
		Priority:       &priority,
		InboxID:        2,
		ContactID:      11,
		AssigneeID:     &assigneeID,
		TeamID:         &teamID,
		CreatedAt:      1700000000,
		LastActivityAt: 1700000200,
		Meta: map[string]any{
			"contact": map[string]any{
				"id":           11,
				"name":         "Jane Doe",
				"email":        "jane@example.com",
				"phone_number": "+1-555-0100",
			},
		},
	}

	messages := []api.Message{
		{
			ID:          1,
			Content:     "hello\nworld",
			MessageType: api.MessageTypeIncoming,
			CreatedAt:   1700000001,
			Sender:      &api.MessageSender{Name: "Jane Doe"},
		},
		{
			ID:          2,
			Content:     "",
			MessageType: api.MessageTypeOutgoing,
			Private:     true,
			CreatedAt:   1700000002,
			Sender:      &api.MessageSender{Name: "Agent One"},
			Attachments: []api.Attachment{{FileType: "image/png", DataURL: "https://example.com/a.png"}},
		},
	}

	var out bytes.Buffer
	writeTranscript(&out, conv, messages, true, 1, 1, 25)
	text := out.String()

	checks := []string{
		"Conversation #77",
		"ID: 33",
		"Status: open",
		"Priority: high",
		"Inbox ID: 2",
		"Contact: Jane Doe <jane@example.com> (+1-555-0100) [id 11]",
		"Assignee ID: 9",
		"Team ID: 4",
		"Messages: 2 (public 1, private 1)",
		"Public only: true",
		"Limit: 25",
		"--- Transcript ---",
		"Customer (Jane Doe)",
		"Private note (Agent One)",
		"(no content)",
		"[attachment] image/png https://example.com/a.png",
	}
	for _, want := range checks {
		if !strings.Contains(text, want) {
			t.Fatalf("transcript missing %q\n%s", want, text)
		}
	}
}

func TestFetchAndDisplayConversations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/conversations" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"data": {
				"meta": {"current_page": 1, "total_pages": 1},
				"payload": [
					{"id": 1, "display_id": 101, "inbox_id": 1, "status": "open", "unread_count": 2, "created_at": 1700000000, "last_activity_at": 1700000100, "priority": "high"},
					{"id": 2, "display_id": 102, "inbox_id": 1, "status": "pending", "unread_count": 0, "created_at": 1700000200, "last_activity_at": 1700000300}
				]
			}
		}`))
	}))
	defer server.Close()

	t.Setenv("CHATWOOT_TESTING", "1")
	client := api.New(server.URL, "token", 1)
	seen := map[int]int64{}

	t.Run("json output respects limit and seen", func(t *testing.T) {
		cmd := &cobra.Command{}
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetContext(outfmt.WithMode(context.Background(), outfmt.JSON))

		if err := fetchAndDisplayConversations(context.Background(), cmd, client, "open", 0, 1, seen); err != nil {
			t.Fatalf("first fetchAndDisplayConversations error: %v", err)
		}

		var payload map[string]any
		if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &payload); err != nil {
			t.Fatalf("decode json output: %v\n%s", err, out.String())
		}
		items, ok := payload["items"].([]any)
		if !ok {
			t.Fatalf("expected items list in payload: %#v", payload)
		}
		if len(items) != 1 {
			t.Fatalf("expected 1 limited item, got %d", len(items))
		}

		out.Reset()
		if err := fetchAndDisplayConversations(context.Background(), cmd, client, "open", 0, 1, seen); err != nil {
			t.Fatalf("second fetchAndDisplayConversations error: %v", err)
		}
		if strings.TrimSpace(out.String()) != "" {
			t.Fatalf("expected no output for unchanged seen set, got %q", out.String())
		}
	})

	t.Run("text output prints summary", func(t *testing.T) {
		cmd := &cobra.Command{}
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetContext(outfmt.WithMode(context.Background(), outfmt.Text))
		localSeen := map[int]int64{}

		if err := fetchAndDisplayConversations(context.Background(), cmd, client, "open", 0, 0, localSeen); err != nil {
			t.Fatalf("fetchAndDisplayConversations text mode error: %v", err)
		}
		text := out.String()
		if !strings.Contains(text, "update(s):") {
			t.Fatalf("expected summary output, got %q", text)
		}
		if !strings.Contains(text, "#101 [open] priority=high") {
			t.Fatalf("expected first conversation line, got %q", text)
		}
	})
}
