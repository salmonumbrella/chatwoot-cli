package cmd

import (
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

type lightContact struct {
	ID    int                `json:"id"`
	Name  *string            `json:"nm,omitempty"`
	Email *string            `json:"em,omitempty"`
	Phone *string            `json:"ph,omitempty"`
	Convs []lightContactConv `json:"convs,omitempty"`
}

type lightContactConv struct {
	ID      int     `json:"id"`
	Status  string  `json:"st"`
	InboxID int     `json:"inbox"`
	Last    *string `json:"last,omitempty"`
}

func buildLightContact(c *api.Contact) lightContact {
	if c == nil {
		return lightContact{}
	}
	return lightContact{
		ID:    c.ID,
		Name:  nullableString(c.Name),
		Email: nullableString(c.Email),
		Phone: nullableString(c.PhoneNumber),
	}
}

func buildLightContactConversation(conv api.Conversation, lastMsg string) lightContactConv {
	return lightContactConv{
		ID:      conv.ID,
		Status:  conv.Status,
		InboxID: conv.InboxID,
		Last:    nullableString(lastMsg),
	}
}

func extractLastNonActivityMessage(conv api.Conversation) string {
	if conv.LastNonActivityMessage != nil {
		return strings.TrimSpace(conv.LastNonActivityMessage.Content)
	}
	return ""
}
