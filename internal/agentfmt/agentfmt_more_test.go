package agentfmt

import (
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

type fakePayload struct {
	v any
}

func (f fakePayload) AgentPayload() any { return f.v }

func TestEnvelopeAgentPayloadMethods(t *testing.T) {
	l := ListEnvelope{Kind: "k1", Items: []int{1}}
	if got := l.AgentPayload().(ListEnvelope); got.Kind != "k1" {
		t.Fatalf("ListEnvelope AgentPayload kind = %q", got.Kind)
	}

	i := ItemEnvelope{Kind: "k2", Item: 1}
	if got := i.AgentPayload().(ItemEnvelope); got.Kind != "k2" {
		t.Fatalf("ItemEnvelope AgentPayload kind = %q", got.Kind)
	}

	s := SearchEnvelope{Kind: "k3", Query: "q"}
	if got := s.AgentPayload().(SearchEnvelope); got.Kind != "k3" {
		t.Fatalf("SearchEnvelope AgentPayload kind = %q", got.Kind)
	}

	d := DataEnvelope{Kind: "k4", Data: map[string]any{"ok": true}}
	if got := d.AgentPayload().(DataEnvelope); got.Kind != "k4" {
		t.Fatalf("DataEnvelope AgentPayload kind = %q", got.Kind)
	}

	e := ErrorEnvelope{Kind: "k5", Error: &api.StructuredError{Message: "oops"}}
	if got := e.AgentPayload().(ErrorEnvelope); got.Kind != "k5" {
		t.Fatalf("ErrorEnvelope AgentPayload kind = %q", got.Kind)
	}
}

func TestKindFromCommandPath_Unknown(t *testing.T) {
	if got := KindFromCommandPath(""); got != "unknown" {
		t.Fatalf("KindFromCommandPath(\"\") = %q, want unknown", got)
	}
	if got := KindFromCommandPath("   "); got != "unknown" {
		t.Fatalf("KindFromCommandPath(\"   \") = %q, want unknown", got)
	}
}

func TestTransform_CoversTypedCases(t *testing.T) {
	se := api.StructuredError{Message: "bad"}
	if got := Transform("err.kind", se); got.(ErrorEnvelope).Kind != "err.kind" {
		t.Fatalf("Transform(StructuredError) kind mismatch")
	}
	if got := Transform("err.kind", &se); got.(ErrorEnvelope).Error.Message != "bad" {
		t.Fatalf("Transform(*StructuredError) message mismatch")
	}

	conv := api.Conversation{ID: 10, CreatedAt: 1700000000}
	if got := Transform("conv.get", conv); got.(ItemEnvelope).Kind != "conv.get" {
		t.Fatalf("Transform(Conversation) kind mismatch")
	}
	if got := Transform("conv.get", &conv); got.(ItemEnvelope).Kind != "conv.get" {
		t.Fatalf("Transform(*Conversation) kind mismatch")
	}
	if got := Transform("conv.get", (*api.Conversation)(nil)); got.(ItemEnvelope).Item != nil {
		t.Fatalf("Transform(nil *Conversation) expected nil item")
	}
	if got := Transform("conv.list", []api.Conversation{conv}); got.(ListEnvelope).Kind != "conv.list" {
		t.Fatalf("Transform([]Conversation) kind mismatch")
	}

	contact := api.Contact{ID: 20, Name: "Jane", CreatedAt: 1700000000}
	if got := Transform("contact.get", contact); got.(ItemEnvelope).Kind != "contact.get" {
		t.Fatalf("Transform(Contact) kind mismatch")
	}
	if got := Transform("contact.get", &contact); got.(ItemEnvelope).Kind != "contact.get" {
		t.Fatalf("Transform(*Contact) kind mismatch")
	}
	if got := Transform("contact.get", (*api.Contact)(nil)); got.(ItemEnvelope).Item != nil {
		t.Fatalf("Transform(nil *Contact) expected nil item")
	}
	if got := Transform("contact.list", []api.Contact{contact}); got.(ListEnvelope).Kind != "contact.list" {
		t.Fatalf("Transform([]Contact) kind mismatch")
	}

	msg := api.Message{ID: 30, ConversationID: 10, MessageType: api.MessageTypeIncoming, CreatedAt: 1700000000}
	if got := Transform("msg.get", msg); got.(ItemEnvelope).Kind != "msg.get" {
		t.Fatalf("Transform(Message) kind mismatch")
	}
	if got := Transform("msg.get", &msg); got.(ItemEnvelope).Kind != "msg.get" {
		t.Fatalf("Transform(*Message) kind mismatch")
	}
	if got := Transform("msg.get", (*api.Message)(nil)); got.(ItemEnvelope).Item != nil {
		t.Fatalf("Transform(nil *Message) expected nil item")
	}
	if got := Transform("msg.list", []api.Message{msg}); got.(ListEnvelope).Kind != "msg.list" {
		t.Fatalf("Transform([]Message) kind mismatch")
	}

	already := fakePayload{v: map[string]any{"wrapped": true}}
	got := Transform("ignored", already).(map[string]any)
	if wrapped, ok := got["wrapped"].(bool); !ok || !wrapped {
		t.Fatalf("Transform(Payload) did not return AgentPayload value: %#v", got)
	}
}

func TestTransformListItems_DefaultCase(t *testing.T) {
	input := []int{1, 2, 3}
	got := TransformListItems(input).([]int)
	if len(got) != 3 || got[2] != 3 {
		t.Fatalf("TransformListItems default mismatch: %#v", got)
	}
}

func TestConversationAndContactAndMessageSummaries(t *testing.T) {
	priority := "high"
	assigneeID := 9
	teamID := 8
	lastActivity := int64(1700000200)
	conv := api.Conversation{
		ID:             7,
		DisplayID:      nil,
		Status:         "open",
		Priority:       &priority,
		InboxID:        2,
		ContactID:      3,
		AssigneeID:     &assigneeID,
		TeamID:         &teamID,
		Unread:         4,
		MessagesCount:  6,
		Labels:         []string{"vip"},
		CreatedAt:      1700000000,
		LastActivityAt: 1700000200,
		Muted:          true,
		Meta: map[string]any{
			"contact": map[string]any{
				"id":           "3",
				"name":         "Bob",
				"email":        "bob@example.com",
				"phone_number": "+1",
			},
		},
		CustomAttributes: map[string]any{"tier": "gold"},
	}

	convList := ConversationSummaries([]api.Conversation{conv})
	if len(convList) != 1 {
		t.Fatalf("ConversationSummaries len = %d", len(convList))
	}
	if convList[0].DisplayID != 7 {
		t.Fatalf("DisplayID should fallback to ID, got %d", convList[0].DisplayID)
	}
	if convList[0].Contact == nil || convList[0].Contact.Name != "Bob" {
		t.Fatalf("contact ref missing: %#v", convList[0].Contact)
	}

	detail := ConversationDetailFromConversation(conv)
	if !detail.Muted {
		t.Fatalf("ConversationDetail muted mismatch")
	}
	if detail.CustomAttributes["tier"] != "gold" {
		t.Fatalf("ConversationDetail custom attrs mismatch: %#v", detail.CustomAttributes)
	}

	if got := ConversationSummaries(nil); got != nil {
		t.Fatalf("ConversationSummaries(nil) should be nil, got %#v", got)
	}

	contact := api.Contact{
		ID:               3,
		Name:             "Bob",
		Email:            "bob@example.com",
		PhoneNumber:      "+1",
		Identifier:       "ext-1",
		Thumbnail:        "thumb.png",
		CustomAttributes: map[string]any{"tier": "gold"},
		CreatedAt:        1700000000,
		LastActivityAt:   &lastActivity,
	}
	contactSummaries := ContactSummaries([]api.Contact{contact})
	if len(contactSummaries) != 1 || contactSummaries[0].Name != "Bob" {
		t.Fatalf("ContactSummaries unexpected: %#v", contactSummaries)
	}
	contactDetail := ContactDetailFromContact(contact)
	if contactDetail.Thumbnail != "thumb.png" {
		t.Fatalf("ContactDetail thumbnail mismatch")
	}
	if got := ContactSummaries(nil); got != nil {
		t.Fatalf("ContactSummaries(nil) should be nil, got %#v", got)
	}

	msgWithSender := api.Message{
		ID:             1,
		ConversationID: 7,
		MessageType:    api.MessageTypeOutgoing,
		Private:        true,
		Content:        "hi",
		CreatedAt:      1700000000,
		Sender:         &api.MessageSender{ID: 11, Name: "Agent", Type: "user"},
		Attachments: []api.Attachment{
			{ID: 99, FileType: "image/png", DataURL: "https://x", ThumbURL: "https://t", FileSize: 10},
		},
	}
	msgSummary := MessageSummaryFromMessage(msgWithSender)
	if msgSummary.Sender == nil || msgSummary.Sender.Name != "Agent" {
		t.Fatalf("MessageSummary sender mismatch: %#v", msgSummary.Sender)
	}
	if len(msgSummary.Attachments) != 1 || msgSummary.Attachments[0].ID != 99 {
		t.Fatalf("MessageSummary attachments mismatch: %#v", msgSummary.Attachments)
	}

	sid := 12
	msgWithFallbackSender := api.Message{
		ID:             2,
		ConversationID: 7,
		MessageType:    api.MessageTypeIncoming,
		SenderID:       &sid,
		SenderType:     "contact",
		CreatedAt:      1700000000,
	}
	msgSummary2 := MessageSummaryFromMessage(msgWithFallbackSender)
	if msgSummary2.Sender == nil || msgSummary2.Sender.Type != "contact" {
		t.Fatalf("fallback sender mismatch: %#v", msgSummary2.Sender)
	}

	msgList := MessageSummaries([]api.Message{msgWithSender, msgWithFallbackSender})
	if len(msgList) != 2 {
		t.Fatalf("MessageSummaries len = %d", len(msgList))
	}
	if got := MessageSummaries(nil); got != nil {
		t.Fatalf("MessageSummaries(nil) should be nil, got %#v", got)
	}
}

func TestInternalHelperConversions(t *testing.T) {
	if timestampOrNil(0) != nil {
		t.Fatalf("timestampOrNil(0) should be nil")
	}
	ts := timestampOrNil(1700000000)
	if ts == nil || ts.Unix != 1700000000 || ts.ISO == "" {
		t.Fatalf("timestampOrNil nonzero mismatch: %#v", ts)
	}

	if timestampPtrOrNil(nil) != nil {
		t.Fatalf("timestampPtrOrNil(nil) should be nil")
	}
	zero := int64(0)
	if timestampPtrOrNil(&zero) != nil {
		t.Fatalf("timestampPtrOrNil(&0) should be nil")
	}
	nonzero := int64(1700000000)
	if got := timestampPtrOrNil(&nonzero); got == nil || got.Unix != nonzero {
		t.Fatalf("timestampPtrOrNil nonzero mismatch: %#v", got)
	}

	if senderRefFromMessage(api.Message{}) != nil {
		t.Fatalf("senderRefFromMessage(empty) should be nil")
	}

	if got := attachmentSummaries(nil); len(got) != 0 {
		t.Fatalf("attachmentSummaries(nil) len = %d", len(got))
	}

	if got := contactRefFromMeta(nil); got != nil {
		t.Fatalf("contactRefFromMeta(nil) should be nil")
	}
	if got := contactRefFromMeta(map[string]any{"x": "y"}); got != nil {
		t.Fatalf("contactRefFromMeta(no contact/sender) should be nil")
	}

	if got := contactRefFromMap(map[string]any{}); got != nil {
		t.Fatalf("contactRefFromMap(empty) should be nil")
	}
	if got := contactRefFromMap(map[string]any{"id": float64(1)}); got == nil || got.ID != 1 {
		t.Fatalf("contactRefFromMap id mismatch: %#v", got)
	}

	if got := mapFromAny(map[string]any{"ok": true}); got["ok"] != true {
		t.Fatalf("mapFromAny map mismatch: %#v", got)
	}
	if got := mapFromAny("not-map"); got != nil {
		t.Fatalf("mapFromAny(non-map) should be nil")
	}

	intTests := []struct {
		in   any
		want int
		ok   bool
	}{
		{in: int(5), want: 5, ok: true},
		{in: int64(6), want: 6, ok: true},
		{in: float64(7.9), want: 7, ok: true},
		{in: "8", want: 8, ok: true},
		{in: "", want: 0, ok: false},
		{in: "bad", want: 0, ok: false},
		{in: true, want: 0, ok: false},
	}
	for _, tt := range intTests {
		got, ok := toInt(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Fatalf("toInt(%#v) = (%d,%v), want (%d,%v)", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}
