package cmd

import (
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

type lightConversationContact struct {
	ID    *int    `json:"id"`
	Name  *string `json:"name"`
	Email *string `json:"email"`
	Phone *string `json:"phone"`
}

type lightConversationPayload struct {
	ID      int                      `json:"id"`
	Status  string                   `json:"st"`
	InboxID int                      `json:"inbox"`
	Contact lightConversationContact `json:"contact"`
	Msgs    []string                 `json:"msgs"`
}

// buildLightConversationContext returns a minimal, stable payload for fast triage:
// conversation id/status/inbox, contact core fields, and non-activity message content.
func buildLightConversationContext(conversationID int, ctx *api.ConversationContext) lightConversationPayload {
	payload := lightConversationPayload{
		ID:      conversationID,
		Status:  "",
		InboxID: 0,
		Contact: lightConversationContact{
			ID:    nil,
			Name:  nil,
			Email: nil,
			Phone: nil,
		},
		Msgs: make([]string, 0),
	}

	if ctx == nil {
		return payload
	}

	if ctx.Conversation != nil {
		payload.Status = ctx.Conversation.Status
		payload.InboxID = ctx.Conversation.InboxID
		payload.Contact.ID = nullableInt(ctx.Conversation.ContactID)
	}

	if ctx.Contact != nil {
		payload.Contact.ID = nullableInt(ctx.Contact.ID)
		payload.Contact.Name = nullableString(ctx.Contact.Name)
		payload.Contact.Email = nullableString(ctx.Contact.Email)
		payload.Contact.Phone = nullableString(ctx.Contact.PhoneNumber)
	}

	for _, msg := range ctx.Messages {
		if msg.MessageType != api.MessageTypeIncoming && msg.MessageType != api.MessageTypeOutgoing {
			continue
		}
		payload.Msgs = append(payload.Msgs, strings.TrimSpace(msg.Content))
	}

	return payload
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
