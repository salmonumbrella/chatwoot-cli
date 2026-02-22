package agentfmt

import (
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

func TestKindFromCommandPath(t *testing.T) {
	kind := KindFromCommandPath("cw conversations list")
	if kind != "conversations.list" {
		t.Fatalf("expected kind conversations.list, got %s", kind)
	}
}

func TestConversationSummaryFromConversation(t *testing.T) {
	display := 42
	conv := api.Conversation{
		ID:             10,
		DisplayID:      &display,
		Status:         "open",
		InboxID:        2,
		ContactID:      7,
		Unread:         3,
		MessagesCount:  5,
		CreatedAt:      1700000000,
		LastActivityAt: 1700001000,
		Meta: map[string]any{
			"sender": map[string]any{
				"id":    7,
				"name":  "Jane Doe",
				"email": "jane@example.com",
			},
		},
	}

	summary := ConversationSummaryFromConversation(conv)
	if summary.DisplayID != 42 {
		t.Fatalf("expected display_id 42, got %d", summary.DisplayID)
	}
	if summary.CreatedAt == nil || summary.CreatedAt.Unix != 1700000000 {
		t.Fatalf("expected created_at unix 1700000000, got %#v", summary.CreatedAt)
	}
	if summary.LastActivity == nil || summary.LastActivity.Unix != 1700001000 {
		t.Fatalf("expected last_activity_at unix 1700001000, got %#v", summary.LastActivity)
	}
	if summary.Contact == nil || summary.Contact.Name != "Jane Doe" {
		t.Fatalf("expected contact name Jane Doe, got %#v", summary.Contact)
	}
	if len(summary.Path) == 0 {
		t.Fatalf("expected path entries")
	}
}

func TestTransformListItems(t *testing.T) {
	contacts := []api.Contact{
		{ID: 1, Name: "Test", CreatedAt: 1700000000},
	}
	items := TransformListItems(contacts)
	list, ok := items.([]ContactSummary)
	if !ok {
		t.Fatalf("expected contact summaries, got %T", items)
	}
	if len(list) != 1 || list[0].ID != 1 {
		t.Fatalf("unexpected contact summary: %#v", list)
	}
}

func TestTransformUnknown(t *testing.T) {
	payload := Transform("unknown.kind", map[string]any{"ok": true})
	wrapped, ok := payload.(DataEnvelope)
	if !ok {
		t.Fatalf("expected DataEnvelope, got %T", payload)
	}
	if wrapped.Kind != "unknown.kind" {
		t.Fatalf("unexpected kind: %s", wrapped.Kind)
	}
}

func TestComputeRelationshipSummary(t *testing.T) {
	now := int64(1700000000)
	lastWeek := now - 86400*7

	conversations := []api.Conversation{
		{ID: 1, Status: "open", CreatedAt: lastWeek, LastActivityAt: now},
		{ID: 2, Status: "resolved", CreatedAt: lastWeek - 86400, LastActivityAt: lastWeek},
		{ID: 3, Status: "pending", CreatedAt: now - 3600, LastActivityAt: now - 1800},
	}

	summary := ComputeRelationshipSummary(conversations)

	if summary.TotalConversations != 3 {
		t.Errorf("expected 3 total, got %d", summary.TotalConversations)
	}
	if summary.OpenConversations != 2 {
		t.Errorf("expected 2 open (open+pending), got %d", summary.OpenConversations)
	}
	if summary.FirstContact == nil {
		t.Error("expected first_contact")
	} else if summary.FirstContact.Unix != lastWeek-86400 {
		t.Errorf("expected first_contact unix %d, got %d", lastWeek-86400, summary.FirstContact.Unix)
	}
	if summary.LastActivity == nil {
		t.Error("expected last_activity")
	} else if summary.LastActivity.Unix != now {
		t.Errorf("expected last_activity unix %d, got %d", now, summary.LastActivity.Unix)
	}
}

func TestComputeRelationshipSummaryEmpty(t *testing.T) {
	summary := ComputeRelationshipSummary([]api.Conversation{})
	if summary.TotalConversations != 0 {
		t.Errorf("expected 0 total, got %d", summary.TotalConversations)
	}
	if summary.OpenConversations != 0 {
		t.Errorf("expected 0 open, got %d", summary.OpenConversations)
	}
	if summary.FirstContact != nil {
		t.Errorf("expected nil first_contact, got %v", summary.FirstContact)
	}
	if summary.LastActivity != nil {
		t.Errorf("expected nil last_activity, got %v", summary.LastActivity)
	}
}
