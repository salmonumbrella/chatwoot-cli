package agentfmt

import (
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

func TestTransformListItems_SupportedSlices(t *testing.T) {
	conversations := []api.Conversation{{ID: 1, InboxID: 2, Status: "open", CreatedAt: 1700000000}}
	convSummaries, ok := TransformListItems(conversations).([]ConversationSummary)
	if !ok {
		t.Fatalf("expected []ConversationSummary")
	}
	if len(convSummaries) != 1 || convSummaries[0].ID != 1 {
		t.Fatalf("unexpected conversation summaries: %#v", convSummaries)
	}

	contacts := []api.Contact{{ID: 2, Name: "Customer", CreatedAt: 1700000000}}
	contactSummaries, ok := TransformListItems(contacts).([]ContactSummary)
	if !ok {
		t.Fatalf("expected []ContactSummary")
	}
	if len(contactSummaries) != 1 || contactSummaries[0].ID != 2 {
		t.Fatalf("unexpected contact summaries: %#v", contactSummaries)
	}

	messages := []api.Message{{ID: 3, ConversationID: 1, MessageType: api.MessageTypeIncoming, CreatedAt: 1700000000}}
	messageSummaries, ok := TransformListItems(messages).([]MessageSummary)
	if !ok {
		t.Fatalf("expected []MessageSummary")
	}
	if len(messageSummaries) != 1 || messageSummaries[0].ID != 3 {
		t.Fatalf("unexpected message summaries: %#v", messageSummaries)
	}
}
