package cmd

import (
	"strconv"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

// lightLookupContact keeps only stable contact identity fields for lookup flows.
type lightLookupContact struct {
	ID   *int    `json:"id,omitempty"`
	Name *string `json:"nm,omitempty"`
}

// lightConversationLookup is a compact conversation summary optimized for triage.
type lightConversationLookup struct {
	ID      int    `json:"id"`
	Status  string `json:"st,omitempty"`
	InboxID int    `json:"ib,omitempty"`
	// UnreadCount always serializes (no omitempty) — zero unread is meaningful for triage.
	UnreadCount    int                 `json:"ur"`
	LastActivityAt int64               `json:"la,omitempty"`
	MessagesCount  int                 `json:"mc,omitempty"`
	Contact        *lightLookupContact `json:"ct,omitempty"`
	LastMessage    *string             `json:"lm,omitempty"`
}

// lightMessageLookup is a minimal message payload for agent reads.
type lightMessageLookup struct {
	ID          int                 `json:"id"`
	MessageType int                 `json:"mt"`
	Private     bool                `json:"prv,omitempty"`
	Content     *string             `json:"ct,omitempty"`
	CreatedAt   int64               `json:"ts,omitempty"`
	Sender      *lightMessageSender `json:"sn,omitempty"`
	Attachments []string            `json:"att,omitempty"`
}

type lightMessageSender struct {
	Name *string `json:"nm,omitempty"`
}

// lightSearchPayload is a compact multi-resource search result.
type lightSearchPayload struct {
	Query   string              `json:"q"`
	Results []lightSearchResult `json:"rs"`
	Summary map[string]int      `json:"sm"`
}

type lightSearchResult struct {
	Type           string  `json:"type"`
	ID             int     `json:"id"`
	Name           *string `json:"nm,omitempty"`
	Email          *string `json:"em,omitempty"`
	Status         *string `json:"st,omitempty"`
	InboxID        *int    `json:"ib,omitempty"`
	ContactID      *int    `json:"cid,omitempty"`
	ContactName    *string `json:"cnm,omitempty"`
	LastActivityAt *int64  `json:"la,omitempty"`
	MessageCount   *int    `json:"mc,omitempty"`
	Snippet        *string `json:"snip,omitempty"`
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
		Status:         shortStatus(conv.Status),
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
			Content:     nullableString(normalizeMessagePreview(msg.Content)),
			CreatedAt:   msg.CreatedAt,
		}

		if msg.Sender != nil {
			sender := lightMessageSender{
				Name: nullableString(msg.Sender.Name),
			}
			if sender.Name != nil {
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
				item.Status = nullableString(shortStatus(r.Conversation.Status))
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

// lightConversationGet is a compact conversation detail for agent reads.
type lightConversationGet struct {
	ID             int                 `json:"id"`
	Status         string              `json:"st"`
	InboxID        int                 `json:"ib"`
	UnreadCount    int                 `json:"ur"`
	LastActivityAt int64               `json:"la,omitempty"`
	Contact        *lightLookupContact `json:"ct,omitempty"`
	Assignee       *lightLookupContact `json:"ag,omitempty"`
	LastMessage    *string             `json:"lm,omitempty"`
}

func buildLightConversationGet(conv api.Conversation) lightConversationGet {
	item := lightConversationGet{
		ID:             conv.ID,
		Status:         shortStatus(conv.Status),
		InboxID:        conv.InboxID,
		UnreadCount:    conv.Unread,
		LastActivityAt: conv.LastActivityAt,
	}
	item.Contact = extractLightLookupContact(conv)
	if conv.Meta != nil {
		if assignee, ok := conv.Meta["assignee"].(map[string]any); ok {
			var a lightLookupContact
			if id, ok := senderInt(assignee["id"]); ok && id > 0 {
				a.ID = nullableInt(id)
			}
			if name, ok := assignee["name"].(string); ok {
				a.Name = nullableString(name)
			}
			if a.ID != nil || a.Name != nil {
				item.Assignee = &a
			}
		}
	}
	item.LastMessage = nullableString(extractLastNonActivityMessage(conv))
	return item
}

// lightInbox is a compact inbox summary.
type lightInbox struct {
	ID          int    `json:"id"`
	Name        string `json:"nm"`
	ChannelType string `json:"ch"`
}

func buildLightInboxes(inboxes []api.Inbox) []lightInbox {
	if len(inboxes) == 0 {
		return []lightInbox{}
	}
	out := make([]lightInbox, 0, len(inboxes))
	for _, inbox := range inboxes {
		out = append(out, lightInbox{
			ID:          inbox.ID,
			Name:        inbox.Name,
			ChannelType: inbox.ChannelType,
		})
	}
	return out
}

// lightAgent is a compact agent summary.
type lightAgent struct {
	ID    int    `json:"id"`
	Name  string `json:"nm"`
	Avail string `json:"av"`
}

func buildLightAgents(agents []api.Agent) []lightAgent {
	if len(agents) == 0 {
		return []lightAgent{}
	}
	out := make([]lightAgent, 0, len(agents))
	for _, agent := range agents {
		out = append(out, lightAgent{
			ID:    agent.ID,
			Name:  agent.Name,
			Avail: agent.AvailabilityStatus,
		})
	}
	return out
}

// lightTeam is a compact team summary.
type lightTeam struct {
	ID   int    `json:"id"`
	Name string `json:"nm"`
}

func buildLightTeams(teams []api.Team) []lightTeam {
	if len(teams) == 0 {
		return []lightTeam{}
	}
	out := make([]lightTeam, 0, len(teams))
	for _, team := range teams {
		out = append(out, lightTeam{
			ID:   team.ID,
			Name: team.Name,
		})
	}
	return out
}

// lightLabel is a compact label summary.
type lightLabel struct {
	ID    int    `json:"id"`
	Title string `json:"t"`
}

func buildLightLabels(labels []api.Label) []lightLabel {
	if len(labels) == 0 {
		return []lightLabel{}
	}
	out := make([]lightLabel, 0, len(labels))
	for _, label := range labels {
		out = append(out, lightLabel{
			ID:    label.ID,
			Title: label.Title,
		})
	}
	return out
}

// lightCannedResponse is a compact canned response summary.
type lightCannedResponse struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
}

func buildLightCannedResponses(responses []api.CannedResponse) []lightCannedResponse {
	if len(responses) == 0 {
		return []lightCannedResponse{}
	}
	out := make([]lightCannedResponse, 0, len(responses))
	for _, r := range responses {
		out = append(out, lightCannedResponse{
			ID:   r.ID,
			Code: r.ShortCode,
		})
	}
	return out
}

// lightAutomationRule is a compact automation rule summary.
type lightAutomationRule struct {
	ID     int    `json:"id"`
	Name   string `json:"nm"`
	Event  string `json:"ev"`
	Active bool   `json:"on"`
}

func buildLightAutomationRules(rules []api.AutomationRule) []lightAutomationRule {
	if len(rules) == 0 {
		return []lightAutomationRule{}
	}
	out := make([]lightAutomationRule, 0, len(rules))
	for _, r := range rules {
		out = append(out, lightAutomationRule{
			ID:     r.ID,
			Name:   r.Name,
			Event:  r.EventName,
			Active: r.Active,
		})
	}
	return out
}

// lightIntegration is a compact integration app summary.
type lightIntegration struct {
	ID      string `json:"id"`
	Name    string `json:"nm"`
	Enabled bool   `json:"on"`
}

func buildLightIntegrations(apps []api.Integration) []lightIntegration {
	if len(apps) == 0 {
		return []lightIntegration{}
	}
	out := make([]lightIntegration, 0, len(apps))
	for _, app := range apps {
		out = append(out, lightIntegration{
			ID:      app.ID,
			Name:    app.Name,
			Enabled: app.Enabled,
		})
	}
	return out
}

// lightCustomFilter is a compact custom filter summary.
type lightCustomFilter struct {
	ID   int    `json:"id"`
	Name string `json:"nm"`
	Type string `json:"type"`
}

func buildLightCustomFilters(filters []api.CustomFilter) []lightCustomFilter {
	if len(filters) == 0 {
		return []lightCustomFilter{}
	}
	out := make([]lightCustomFilter, 0, len(filters))
	for _, f := range filters {
		out = append(out, lightCustomFilter{
			ID:   f.ID,
			Name: f.Name,
			Type: f.FilterType,
		})
	}
	return out
}
