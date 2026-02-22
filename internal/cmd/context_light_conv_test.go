package cmd

import (
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

func makeMsg(id int, msgType int, private bool, content string) api.MessageWithEmbeddings {
	return api.MessageWithEmbeddings{
		ID:          id,
		MessageType: msgType,
		Private:     private,
		Content:     content,
	}
}

func TestBuildLightConversationContext_SmartTail_CustomerWaiting(t *testing.T) {
	ctx := &api.ConversationContext{
		Conversation: &api.Conversation{Status: "open", InboxID: 48},
		Messages: []api.MessageWithEmbeddings{
			makeMsg(1, api.MessageTypeIncoming, false, "First question"),
			makeMsg(2, api.MessageTypeOutgoing, false, "Our reply"),
			makeMsg(3, api.MessageTypeActivity, false, "Assigned to agent"),
			makeMsg(4, api.MessageTypeIncoming, false, "Follow-up 1"),
			makeMsg(5, api.MessageTypeIncoming, false, "Follow-up 2"),
		},
	}
	payload := buildLightConversationContext(123, ctx)
	expected := []string{"> Our reply", "Follow-up 1", "Follow-up 2"}
	assertMsgs(t, payload.Msgs, expected)
}

func TestBuildLightConversationContext_SmartTail_WeJustReplied(t *testing.T) {
	ctx := &api.ConversationContext{
		Conversation: &api.Conversation{Status: "resolved", InboxID: 48},
		Messages: []api.MessageWithEmbeddings{
			makeMsg(1, api.MessageTypeIncoming, false, "Their question"),
			makeMsg(2, api.MessageTypeIncoming, false, "More detail"),
			makeMsg(3, api.MessageTypeOutgoing, false, "Our reply"),
		},
	}
	payload := buildLightConversationContext(123, ctx)
	expected := []string{"More detail", "> Our reply"}
	assertMsgs(t, payload.Msgs, expected)
}

func TestBuildLightConversationContext_SmartTail_PrivateNoteInMiddle(t *testing.T) {
	ctx := &api.ConversationContext{
		Conversation: &api.Conversation{Status: "pending", InboxID: 48},
		Messages: []api.MessageWithEmbeddings{
			makeMsg(1, api.MessageTypeIncoming, false, "Question"),
			makeMsg(2, api.MessageTypeOutgoing, false, "Public reply"),
			makeMsg(3, api.MessageTypeOutgoing, true, "Internal note for team"),
			makeMsg(4, api.MessageTypeIncoming, false, "Customer follow-up"),
		},
	}
	payload := buildLightConversationContext(123, ctx)
	expected := []string{"> Public reply", "* Internal note for team", "Customer follow-up"}
	assertMsgs(t, payload.Msgs, expected)
}

func TestBuildLightConversationContext_SmartTail_OnlyCustomerMsgs(t *testing.T) {
	ctx := &api.ConversationContext{
		Conversation: &api.Conversation{Status: "open", InboxID: 48},
		Messages: []api.MessageWithEmbeddings{
			makeMsg(1, api.MessageTypeIncoming, false, "Hello"),
			makeMsg(2, api.MessageTypeIncoming, false, "Anyone there?"),
		},
	}
	payload := buildLightConversationContext(123, ctx)
	expected := []string{"Hello", "Anyone there?"}
	assertMsgs(t, payload.Msgs, expected)
}

func TestBuildLightConversationContext_SmartTail_OnlyAgentMsgs(t *testing.T) {
	ctx := &api.ConversationContext{
		Conversation: &api.Conversation{Status: "open", InboxID: 15},
		Messages: []api.MessageWithEmbeddings{
			makeMsg(1, api.MessageTypeOutgoing, false, "Outreach msg"),
			makeMsg(2, api.MessageTypeOutgoing, false, "Follow-up"),
		},
	}
	payload := buildLightConversationContext(123, ctx)
	expected := []string{"> Outreach msg", "> Follow-up"}
	assertMsgs(t, payload.Msgs, expected)
}

func TestBuildLightConversationContext_SmartTail_SingleMessage(t *testing.T) {
	ctx := &api.ConversationContext{
		Conversation: &api.Conversation{Status: "open", InboxID: 48},
		Messages: []api.MessageWithEmbeddings{
			makeMsg(1, api.MessageTypeIncoming, false, "Just one message"),
		},
	}
	payload := buildLightConversationContext(123, ctx)
	expected := []string{"Just one message"}
	assertMsgs(t, payload.Msgs, expected)
}

func TestBuildLightConversationContext_SmartTail_Empty(t *testing.T) {
	ctx := &api.ConversationContext{
		Conversation: &api.Conversation{Status: "open", InboxID: 48},
		Messages:     []api.MessageWithEmbeddings{},
	}
	payload := buildLightConversationContext(123, ctx)
	if len(payload.Msgs) != 0 {
		t.Fatalf("expected empty msgs, got %v", payload.Msgs)
	}
}

func TestBuildLightConversationContext_SmartTail_NilMessages(t *testing.T) {
	ctx := &api.ConversationContext{
		Conversation: &api.Conversation{Status: "open", InboxID: 48},
		// Messages is nil (not initialized)
	}
	payload := buildLightConversationContext(123, ctx)
	if len(payload.Msgs) != 0 {
		t.Fatalf("expected empty msgs for nil messages, got %v", payload.Msgs)
	}
}

func TestBuildLightConversationContext_SmartTail_EmptyContent(t *testing.T) {
	ctx := &api.ConversationContext{
		Conversation: &api.Conversation{Status: "open", InboxID: 48},
		Messages: []api.MessageWithEmbeddings{
			makeMsg(1, api.MessageTypeIncoming, false, "   "),
			makeMsg(2, api.MessageTypeOutgoing, false, "Our reply"),
			makeMsg(3, api.MessageTypeIncoming, false, "  \n  "),
			makeMsg(4, api.MessageTypeIncoming, false, "Real question"),
		},
	}
	payload := buildLightConversationContext(123, ctx)
	// Empty/whitespace messages are skipped
	expected := []string{"> Our reply", "Real question"}
	assertMsgs(t, payload.Msgs, expected)
}

func assertMsgs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("msgs length: got %d, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("msgs[%d]: got %q, want %q\nfull got:  %v\nfull want: %v", i, got[i], want[i], got, want)
		}
	}
}
