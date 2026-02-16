package cmd

import (
	"strconv"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

// lightLookupContact keeps only stable contact identity fields for lookup flows.
type lightLookupContact struct {
	ID   *int    `json:"id,omitempty"`
	Name *string `json:"name,omitempty"`
}

// lightConversationLookup is a compact conversation summary optimized for triage.
type lightConversationLookup struct {
	ID      int    `json:"id"`
	Status  string `json:"status,omitempty"`
	InboxID int    `json:"inbox_id,omitempty"`
	// UnreadCount always serializes (no omitempty) — zero unread is meaningful for triage.
	UnreadCount    int                 `json:"unread_count"`
	LastActivityAt int64               `json:"last_activity_at,omitempty"`
	MessagesCount  int                 `json:"messages_count,omitempty"`
	Contact        *lightLookupContact `json:"contact,omitempty"`
	LastMessage    *string             `json:"last_message,omitempty"`
}

// lightMessageLookup is a minimal message payload for agent reads.
type lightMessageLookup struct {
	ID          int                 `json:"id"`
	MessageType int                 `json:"message_type"`
	Private     bool                `json:"private,omitempty"`
	Content     *string             `json:"content,omitempty"`
	CreatedAt   int64               `json:"created_at,omitempty"`
	Sender      *lightLookupContact `json:"sender,omitempty"`
	Attachments []string            `json:"attachments,omitempty"`
}

// lightSearchPayload is a compact multi-resource search result.
type lightSearchPayload struct {
	Query   string              `json:"query"`
	Results []lightSearchResult `json:"results"`
	Summary map[string]int      `json:"summary"`
}

type lightSearchResult struct {
	Type           string  `json:"type"`
	ID             int     `json:"id"`
	Name           *string `json:"name,omitempty"`
	Email          *string `json:"email,omitempty"`
	Status         *string `json:"status,omitempty"`
	InboxID        *int    `json:"inbox_id,omitempty"`
	ContactID      *int    `json:"contact_id,omitempty"`
	ContactName    *string `json:"contact_name,omitempty"`
	LastActivityAt *int64  `json:"last_activity_at,omitempty"`
	MessageCount   *int    `json:"message_count,omitempty"`
	Snippet        *string `json:"snippet,omitempty"`
}

func buildLightConversationLookups(conversations []api.Conversation) []lightConversationLookup {
	if len(conversations) == 0 {
		return []lightConversationLookup{}
	}

	items := make([]lightConversationLookup, 0, len(conversations))
	for _, conv := range conversations {
		items = append(items, buildLightConversationLookup(conv))
	}
	return items
}

func buildLightConversationLookup(conv api.Conversation) lightConversationLookup {
	item := lightConversationLookup{
		ID:             conv.ID,
		Status:         strings.TrimSpace(conv.Status),
		InboxID:        conv.InboxID,
		UnreadCount:    conv.Unread,
		LastActivityAt: conv.LastActivityAt,
		MessagesCount:  conv.MessagesCount,
	}

	item.LastMessage = nullableString(extractLastNonActivityMessage(conv))
	item.Contact = extractLightLookupContact(conv)
	return item
}

func extractLightLookupContact(conv api.Conversation) *lightLookupContact {
	var contact lightLookupContact

	if conv.ContactID > 0 {
		contact.ID = nullableInt(conv.ContactID)
	}

	// Meta sender takes precedence over top-level ContactID because the
	// sender reflects the current contact after merges or reassignments.
	if conv.Meta != nil {
		sender, ok := conv.Meta["sender"].(map[string]any)
		if ok {
			if id, ok := senderInt(sender["id"]); ok && id > 0 {
				contact.ID = nullableInt(id)
			}
			if name, ok := sender["name"].(string); ok {
				contact.Name = nullableString(name)
			}
		}
	}

	if contact.ID == nil && contact.Name == nil {
		return nil
	}
	return &contact
}

func buildLightMessageLookups(messages []api.Message) []lightMessageLookup {
	if len(messages) == 0 {
		return []lightMessageLookup{}
	}

	out := make([]lightMessageLookup, 0, len(messages))
	for _, msg := range messages {
		// Keep only customer/agent messages for compact lookup context.
		if msg.MessageType != api.MessageTypeIncoming && msg.MessageType != api.MessageTypeOutgoing {
			continue
		}

		item := lightMessageLookup{
			ID:          msg.ID,
			MessageType: msg.MessageType,
			Private:     msg.Private,
			Content:     nullableString(msg.Content),
			CreatedAt:   msg.CreatedAt,
		}

		if msg.Sender != nil {
			sender := lightLookupContact{
				ID:   nullableInt(msg.Sender.ID),
				Name: nullableString(msg.Sender.Name),
			}
			if sender.ID != nil || sender.Name != nil {
				item.Sender = &sender
			}
		}

		if len(msg.Attachments) > 0 {
			types := make([]string, 0, len(msg.Attachments))
			for _, att := range msg.Attachments {
				ft := strings.TrimSpace(att.FileType)
				if ft == "" {
					ft = "file"
				}
				types = append(types, ft)
			}
			if len(types) > 0 {
				item.Attachments = types
			}
		}

		out = append(out, item)
	}

	return out
}

func buildLightSearchPayload(results SearchResults) lightSearchPayload {
	payload := lightSearchPayload{
		Query:   results.Query,
		Results: make([]lightSearchResult, 0, len(results.Results)),
		Summary: results.Summary,
	}

	if payload.Summary == nil {
		payload.Summary = map[string]int{}
	}

	for _, r := range results.Results {
		item := lightSearchResult{Type: r.Type}

		switch r.Type {
		case "contact":
			if r.Contact != nil {
				item.ID = r.Contact.ID
				item.Name = nullableString(r.Contact.Name)
				item.Email = nullableString(r.Contact.Email)
				if r.Contact.LastActivityAt != nil {
					last := *r.Contact.LastActivityAt
					item.LastActivityAt = &last
				}
			}

		case "conversation":
			if r.Conversation != nil {
				item.ID = r.Conversation.ID
				item.Status = nullableString(r.Conversation.Status)
				if r.Conversation.InboxID > 0 {
					inboxID := r.Conversation.InboxID
					item.InboxID = &inboxID
				}
				if r.Conversation.LastActivityAt > 0 {
					last := r.Conversation.LastActivityAt
					item.LastActivityAt = &last
				}
				if r.Conversation.Meta != nil {
					if sender, ok := r.Conversation.Meta["sender"].(map[string]any); ok {
						if name, ok := sender["name"].(string); ok {
							item.ContactName = nullableString(name)
						}
					}
				}
				if snippet, ok := results.Snippets[strconv.Itoa(r.Conversation.ID)]; ok {
					item.Snippet = nullableString(snippet.Content)
				}
			}

		case "sender":
			if r.Sender != nil {
				item.ID = r.Sender.ConversationID
				item.Name = nullableString(r.Sender.Name)
				item.ContactName = nullableString(r.Sender.ContactName)
				if r.Sender.ContactID > 0 {
					cid := r.Sender.ContactID
					item.ContactID = &cid
				}
				if r.Sender.LastMessageAt > 0 {
					last := r.Sender.LastMessageAt
					item.LastActivityAt = &last
				}
				if r.Sender.MessageCount > 0 {
					count := r.Sender.MessageCount
					item.MessageCount = &count
				}
			}
		}

		payload.Results = append(payload.Results, item)
	}

	if payload.Results == nil {
		payload.Results = []lightSearchResult{}
	}
	return payload
}
