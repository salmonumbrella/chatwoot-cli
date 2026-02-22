package cmd

import (
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

type lightConversationContact struct {
	ID   *int    `json:"id"`
	Name *string `json:"nm"`
}

type lightConversationPayload struct {
	ID      int                      `json:"id"`
	Status  string                   `json:"st"`
	InboxID int                      `json:"ib"`
	Contact lightConversationContact `json:"ct"`
	Msgs    []string                 `json:"msgs"`
}

// buildLightConversationContext returns a minimal, stable payload for fast triage:
// conversation id/status/inbox, contact name, and non-activity message content.
func buildLightConversationContext(conversationID int, ctx *api.ConversationContext) lightConversationPayload {
	payload := lightConversationPayload{
		ID:   conversationID,
		Msgs: make([]string, 0),
	}

	if ctx == nil {
		return payload
	}

	if ctx.Conversation != nil {
		payload.Status = shortStatus(ctx.Conversation.Status)
		payload.InboxID = ctx.Conversation.InboxID
	}

	// Prefer ctx.Contact (from dedicated API call) but fall back to
	// meta.sender embedded in the conversation response. The Chatwoot API
	// often returns contact_id: null, so the dedicated call is skipped and
	// ctx.Contact stays nil.
	if ctx.Contact != nil {
		payload.Contact.ID = nullableInt(ctx.Contact.ID)
		payload.Contact.Name = nullableString(ctx.Contact.Name)
	} else if ctx.Conversation != nil {
		extractSenderFromMeta(ctx.Conversation.Meta, &payload.Contact)
	}

	payload.Msgs = buildSmartTail(ctx.Messages)

	return payload
}

// buildSmartTail returns the trailing conversation exchange: the last agent
// reply plus any subsequent customer messages (or vice versa). This gives
// just enough context to understand the current conversation state.
//
// Role prefixes: "> " for public agent replies, "* " for private notes,
// bare for customer messages.
func buildSmartTail(messages []api.MessageWithEmbeddings) []string {
	// First pass: filter to non-activity messages only.
	var msgs []api.MessageWithEmbeddings
	for _, m := range messages {
		if m.MessageType == api.MessageTypeIncoming || m.MessageType == api.MessageTypeOutgoing {
			msgs = append(msgs, m)
		}
	}

	if len(msgs) == 0 {
		return []string{}
	}

	// Safety cap: in conversations with many customer messages and no public
	// agent reply (or a very old one), limit the tail to avoid returning the
	// entire conversation history.
	const maxSmartTailMessages = 20

	// Walk backward, collecting messages until we've seen at least one
	// public outgoing AND at least one incoming message.
	seenPublicOutgoing := false
	seenIncoming := false
	startIdx := len(msgs) - 1

	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		startIdx = i

		if len(msgs)-i >= maxSmartTailMessages {
			break
		}

		if m.MessageType == api.MessageTypeOutgoing && !m.Private {
			seenPublicOutgoing = true
		} else if m.MessageType == api.MessageTypeIncoming {
			seenIncoming = true
		}

		if seenPublicOutgoing && seenIncoming {
			break
		}
	}

	// Build result with role prefixes.
	result := make([]string, 0, len(msgs)-startIdx)
	for _, m := range msgs[startIdx:] {
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		switch {
		case m.MessageType == api.MessageTypeOutgoing && m.Private:
			result = append(result, "* "+content)
		case m.MessageType == api.MessageTypeOutgoing:
			result = append(result, "> "+content)
		default:
			result = append(result, content)
		}
	}

	return result
}

// extractSenderFromMeta populates light contact fields from conversation meta.sender.
func extractSenderFromMeta(meta map[string]any, contact *lightConversationContact) {
	sender, ok := meta["sender"].(map[string]any)
	if !ok {
		return
	}
	if id, ok := senderInt(sender["id"]); ok && id > 0 {
		contact.ID = &id
	}
	if name, ok := sender["name"].(string); ok {
		name = strings.TrimSpace(name)
		if name != "" {
			contact.Name = &name
		}
	}
}

// senderInt extracts an int from a JSON value that may be float64 or json.Number.
func senderInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	}
	return 0, false
}

func nullableInt(v int) *int {
	if v <= 0 {
		return nil
	}
	x := v
	return &x
}

func nullableString(v string) *string {
	s := strings.TrimSpace(v)
	if s == "" {
		return nil
	}
	return &s
}
