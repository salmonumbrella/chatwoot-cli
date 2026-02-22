package agentfmt

import (
	"strconv"
	"strings"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

// Payload marks a value as already agent-formatted.
type Payload interface {
	AgentPayload() any
}

// Timestamp provides both Unix and ISO-8601 representations.
type Timestamp struct {
	Unix int64  `json:"unix"`
	ISO  string `json:"iso"`
}

// PathEntry describes a breadcrumb-style path to a resource.
type PathEntry struct {
	Type  string `json:"type"`
	ID    int    `json:"id"`
	Label string `json:"label,omitempty"`
}

// ContactRef summarizes a contact reference.
type ContactRef struct {
	ID    int    `json:"id"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}

// ConversationSummary is a compact, agent-friendly view of a conversation.
type ConversationSummary struct {
	ID            int         `json:"id"`
	DisplayID     int         `json:"display_id"`
	Status        string      `json:"status"`
	Priority      *string     `json:"priority,omitempty"`
	InboxID       int         `json:"inbox_id"`
	ContactID     int         `json:"contact_id,omitempty"`
	AssigneeID    *int        `json:"assignee_id,omitempty"`
	TeamID        *int        `json:"team_id,omitempty"`
	UnreadCount   int         `json:"unread_count"`
	MessagesCount int         `json:"messages_count,omitempty"`
	Labels        []string    `json:"labels,omitempty"`
	CreatedAt     *Timestamp  `json:"created_at,omitempty"`
	LastActivity  *Timestamp  `json:"last_activity_at,omitempty"`
	Path          []PathEntry `json:"path,omitempty"`
	Contact       *ContactRef `json:"contact,omitempty"`
	URL           string      `json:"url,omitempty"`
}

// ConversationDetail expands the summary with additional context.
type ConversationDetail struct {
	ConversationSummary
	Muted            bool           `json:"muted"`
	Meta             map[string]any `json:"meta,omitempty"`
	CustomAttributes map[string]any `json:"custom_attributes,omitempty"`
}

// ConversationDetailWithMessages includes recent messages inline.
type ConversationDetailWithMessages struct {
	ConversationDetail
	Messages []MessageSummary `json:"messages,omitempty"`
}

// SenderRef summarizes a message sender.
type SenderRef struct {
	ID   *int   `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

// AttachmentSummary is a compact attachment view.
type AttachmentSummary struct {
	ID       int    `json:"id"`
	FileType string `json:"file_type,omitempty"`
	DataURL  string `json:"data_url,omitempty"`
	ThumbURL string `json:"thumb_url,omitempty"`
	FileSize int    `json:"file_size,omitempty"`
}

// MessageSummary is a compact, agent-friendly view of a message.
type MessageSummary struct {
	ID             int                 `json:"id"`
	ConversationID int                 `json:"conversation_id"`
	Type           string              `json:"type"`
	Private        bool                `json:"private"`
	Content        string              `json:"content"`
	Sender         *SenderRef          `json:"sender,omitempty"`
	CreatedAt      *Timestamp          `json:"created_at,omitempty"`
	Attachments    []AttachmentSummary `json:"attachments,omitempty"`
}

// MessageSummaryWithPosition includes position metadata for list outputs.
type MessageSummaryWithPosition struct {
	MessageSummary
	Position      int `json:"position"`
	TotalMessages int `json:"total_messages"`
}

// ContactSummary is a compact, agent-friendly view of a contact.
type ContactSummary struct {
	ID             int        `json:"id"`
	Name           string     `json:"name"`
	Email          string     `json:"email,omitempty"`
	PhoneNumber    string     `json:"phone_number,omitempty"`
	Identifier     string     `json:"identifier,omitempty"`
	CreatedAt      *Timestamp `json:"created_at,omitempty"`
	LastActivityAt *Timestamp `json:"last_activity_at,omitempty"`
	URL            string     `json:"url,omitempty"`
}

// ContactDetail expands the summary with additional context.
type ContactDetail struct {
	ContactSummary
	Thumbnail        string         `json:"thumbnail,omitempty"`
	CustomAttributes map[string]any `json:"custom_attributes,omitempty"`
}

// RelationshipSummary provides context about a contact's history.
type RelationshipSummary struct {
	FirstContact       *Timestamp `json:"first_contact,omitempty"`
	TotalConversations int        `json:"total_conversations"`
	OpenConversations  int        `json:"open_conversations"`
	LastActivity       *Timestamp `json:"last_activity,omitempty"`
}

// ContactDetailWithRelationship extends ContactDetail with relationship data.
type ContactDetailWithRelationship struct {
	ContactDetail
	Relationship      *RelationshipSummary  `json:"relationship,omitempty"`
	OpenConversations []ConversationSummary `json:"open_conversations,omitempty"`
}

// ConversationContext provides comprehensive context for a conversation.
type ConversationContext struct {
	Conversation ConversationDetail             `json:"conversation"`
	Messages     []MessageSummary               `json:"messages,omitempty"`
	Contact      *ContactDetailWithRelationship `json:"contact,omitempty"`
}

// ListEnvelope wraps list outputs.
type ListEnvelope struct {
	Kind    string         `json:"kind"`
	Items   any            `json:"items"`
	HasMore *bool          `json:"has_more,omitempty"`
	Meta    map[string]any `json:"meta,omitempty"`
}

// ItemEnvelope wraps single-item outputs.
type ItemEnvelope struct {
	Kind string `json:"kind"`
	Item any    `json:"item"`
}

// SearchEnvelope wraps search outputs.
type SearchEnvelope struct {
	Kind    string         `json:"kind"`
	Query   string         `json:"query"`
	Results any            `json:"results"`
	Summary map[string]int `json:"summary,omitempty"`
	Meta    map[string]any `json:"meta,omitempty"`
}

// DataEnvelope wraps untyped outputs.
type DataEnvelope struct {
	Kind string `json:"kind"`
	Data any    `json:"data"`
}

// ErrorEnvelope wraps structured errors in agent mode.
type ErrorEnvelope struct {
	Kind  string               `json:"kind"`
	Error *api.StructuredError `json:"error"`
}

func (e ListEnvelope) AgentPayload() any   { return e }
func (e ItemEnvelope) AgentPayload() any   { return e }
func (e SearchEnvelope) AgentPayload() any { return e }
func (e DataEnvelope) AgentPayload() any   { return e }
func (e ErrorEnvelope) AgentPayload() any  { return e }

// KindFromCommandPath converts a cobra CommandPath to a dotted kind string.
func KindFromCommandPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "cw ")
	parts := strings.Fields(path)
	if len(parts) == 0 {
		return "unknown"
	}
	return strings.Join(parts, ".")
}

// Transform wraps known API types into agent-friendly structures.
func Transform(kind string, v any) any {
	if payload, ok := v.(Payload); ok {
		return payload.AgentPayload()
	}

	switch val := v.(type) {
	case api.StructuredError:
		return ErrorEnvelope{Kind: kind, Error: &val}
	case *api.StructuredError:
		return ErrorEnvelope{Kind: kind, Error: val}
	case api.Conversation:
		return ItemEnvelope{Kind: kind, Item: ConversationDetailFromConversation(val)}
	case *api.Conversation:
		if val == nil {
			return ItemEnvelope{Kind: kind, Item: nil}
		}
		return ItemEnvelope{Kind: kind, Item: ConversationDetailFromConversation(*val)}
	case []api.Conversation:
		return ListEnvelope{Kind: kind, Items: ConversationSummaries(val)}
	case api.Contact:
		return ItemEnvelope{Kind: kind, Item: ContactDetailFromContact(val)}
	case *api.Contact:
		if val == nil {
			return ItemEnvelope{Kind: kind, Item: nil}
		}
		return ItemEnvelope{Kind: kind, Item: ContactDetailFromContact(*val)}
	case []api.Contact:
		return ListEnvelope{Kind: kind, Items: ContactSummaries(val)}
	case api.Message:
		return ItemEnvelope{Kind: kind, Item: MessageSummaryFromMessage(val)}
	case *api.Message:
		if val == nil {
			return ItemEnvelope{Kind: kind, Item: nil}
		}
		return ItemEnvelope{Kind: kind, Item: MessageSummaryFromMessage(*val)}
	case []api.Message:
		return ListEnvelope{Kind: kind, Items: MessageSummaries(val)}
	default:
		return DataEnvelope{Kind: kind, Data: v}
	}
}

// TransformListItems converts list item slices to agent summaries when supported.
func TransformListItems(items any) any {
	switch val := items.(type) {
	case []api.Conversation:
		return ConversationSummaries(val)
	case []api.Contact:
		return ContactSummaries(val)
	case []api.Message:
		return MessageSummaries(val)
	default:
		return items
	}
}

func ConversationSummaries(conversations []api.Conversation) []ConversationSummary {
	if len(conversations) == 0 {
		return nil
	}
	out := make([]ConversationSummary, 0, len(conversations))
	for _, conv := range conversations {
		out = append(out, ConversationSummaryFromConversation(conv))
	}
	return out
}

func ConversationSummaryFromConversation(conv api.Conversation) ConversationSummary {
	displayID := conv.ID
	if conv.DisplayID != nil {
		displayID = *conv.DisplayID
	}

	createdAt := timestampOrNil(conv.CreatedAt)
	lastActivity := timestampOrNil(conv.LastActivityAt)

	summary := ConversationSummary{
		ID:            conv.ID,
		DisplayID:     displayID,
		Status:        conv.Status,
		Priority:      conv.Priority,
		InboxID:       conv.InboxID,
		ContactID:     conv.ContactID,
		AssigneeID:    conv.AssigneeID,
		TeamID:        conv.TeamID,
		UnreadCount:   conv.Unread,
		MessagesCount: conv.MessagesCount,
		Labels:        conv.Labels,
		CreatedAt:     createdAt,
		LastActivity:  lastActivity,
	}

	contact := contactRefFromMeta(conv.Meta)
	if contact != nil {
		summary.Contact = contact
	}
	summary.Path = conversationPath(conv, contact)

	return summary
}

func ConversationDetailFromConversation(conv api.Conversation) ConversationDetail {
	summary := ConversationSummaryFromConversation(conv)
	return ConversationDetail{
		ConversationSummary: summary,
		Muted:               conv.Muted,
		Meta:                conv.Meta,
		CustomAttributes:    conv.CustomAttributes,
	}
}

func MessageSummaries(messages []api.Message) []MessageSummary {
	if len(messages) == 0 {
		return nil
	}
	out := make([]MessageSummary, 0, len(messages))
	for _, msg := range messages {
		out = append(out, MessageSummaryFromMessage(msg))
	}
	return out
}

func MessageSummaryFromMessage(msg api.Message) MessageSummary {
	summary := MessageSummary{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		Type:           msg.MessageTypeName(),
		Private:        msg.Private,
		Content:        msg.Content,
		CreatedAt:      timestampOrNil(msg.CreatedAt),
	}
	if sender := senderRefFromMessage(msg); sender != nil {
		summary.Sender = sender
	}
	if len(msg.Attachments) > 0 {
		summary.Attachments = attachmentSummaries(msg.Attachments)
	}
	return summary
}

func ContactSummaries(contacts []api.Contact) []ContactSummary {
	if len(contacts) == 0 {
		return nil
	}
	out := make([]ContactSummary, 0, len(contacts))
	for _, contact := range contacts {
		out = append(out, ContactSummaryFromContact(contact))
	}
	return out
}

func ContactSummaryFromContact(contact api.Contact) ContactSummary {
	return ContactSummary{
		ID:             contact.ID,
		Name:           contact.Name,
		Email:          contact.Email,
		PhoneNumber:    contact.PhoneNumber,
		Identifier:     contact.Identifier,
		CreatedAt:      timestampOrNil(contact.CreatedAt),
		LastActivityAt: timestampPtrOrNil(contact.LastActivityAt),
	}
}

func ContactDetailFromContact(contact api.Contact) ContactDetail {
	summary := ContactSummaryFromContact(contact)
	return ContactDetail{
		ContactSummary:   summary,
		Thumbnail:        contact.Thumbnail,
		CustomAttributes: contact.CustomAttributes,
	}
}

// ComputeRelationshipSummary calculates relationship stats from conversations.
func ComputeRelationshipSummary(conversations []api.Conversation) *RelationshipSummary {
	if len(conversations) == 0 {
		return &RelationshipSummary{
			TotalConversations: 0,
			OpenConversations:  0,
		}
	}

	summary := &RelationshipSummary{
		TotalConversations: len(conversations),
	}

	var earliest, latest int64
	for _, conv := range conversations {
		if conv.Status == "open" || conv.Status == "pending" {
			summary.OpenConversations++
		}
		if earliest == 0 || conv.CreatedAt < earliest {
			earliest = conv.CreatedAt
		}
		if conv.LastActivityAt > latest {
			latest = conv.LastActivityAt
		}
	}

	if earliest > 0 {
		summary.FirstContact = timestampOrNil(earliest)
	}
	if latest > 0 {
		summary.LastActivity = timestampOrNil(latest)
	}

	return summary
}

func timestampOrNil(unix int64) *Timestamp {
	if unix == 0 {
		return nil
	}
	return &Timestamp{
		Unix: unix,
		ISO:  time.Unix(unix, 0).UTC().Format(time.RFC3339),
	}
}

func timestampPtrOrNil(unix *int64) *Timestamp {
	if unix == nil || *unix == 0 {
		return nil
	}
	return timestampOrNil(*unix)
}

func senderRefFromMessage(msg api.Message) *SenderRef {
	if msg.Sender != nil {
		return &SenderRef{
			ID:   &msg.Sender.ID,
			Name: msg.Sender.Name,
			Type: msg.Sender.Type,
		}
	}
	if msg.SenderID != nil || msg.SenderType != "" {
		return &SenderRef{
			ID:   msg.SenderID,
			Type: msg.SenderType,
		}
	}
	return nil
}

func attachmentSummaries(attachments []api.Attachment) []AttachmentSummary {
	out := make([]AttachmentSummary, 0, len(attachments))
	for _, att := range attachments {
		out = append(out, AttachmentSummary{
			ID:       att.ID,
			FileType: att.FileType,
			DataURL:  att.DataURL,
			ThumbURL: att.ThumbURL,
			FileSize: att.FileSize,
		})
	}
	return out
}

func conversationPath(conv api.Conversation, contact *ContactRef) []PathEntry {
	var path []PathEntry
	if conv.InboxID > 0 {
		path = append(path, PathEntry{Type: "inbox", ID: conv.InboxID})
	}
	if conv.ContactID > 0 {
		entry := PathEntry{Type: "contact", ID: conv.ContactID}
		if contact != nil && contact.Name != "" {
			entry.Label = contact.Name
		}
		path = append(path, entry)
	}
	path = append(path, PathEntry{Type: "conversation", ID: conv.ID})
	return path
}

func contactRefFromMeta(meta map[string]any) *ContactRef {
	if meta == nil {
		return nil
	}
	// Common keys seen in Chatwoot payloads.
	if sender := mapFromAny(meta["sender"]); sender != nil {
		return contactRefFromMap(sender)
	}
	if contact := mapFromAny(meta["contact"]); contact != nil {
		return contactRefFromMap(contact)
	}
	return nil
}

func contactRefFromMap(m map[string]any) *ContactRef {
	ref := ContactRef{}
	if id, ok := toInt(m["id"]); ok {
		ref.ID = id
	}
	if name, ok := m["name"].(string); ok {
		ref.Name = name
	}
	if email, ok := m["email"].(string); ok {
		ref.Email = email
	}
	if phone, ok := m["phone_number"].(string); ok {
		ref.Phone = phone
	}
	if ref.ID == 0 && ref.Name == "" && ref.Email == "" && ref.Phone == "" {
		return nil
	}
	return &ref
}

func mapFromAny(value any) map[string]any {
	switch v := value.(type) {
	case map[string]any:
		return v
	default:
		return nil
	}
}

func toInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		if v == "" {
			return 0, false
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}
