package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Message type constants
const (
	MessageTypeIncoming = 0 // Customer message
	MessageTypeOutgoing = 1 // Agent reply
	MessageTypeActivity = 2 // System activity (status changes, assignments)
	MessageTypeTemplate = 3 // Template message (WhatsApp, etc.)
)

// Attachment limits
const (
	MaxAttachments    = 10               // Maximum attachments per message
	MaxAttachmentSize = 40 * 1024 * 1024 // 40MB per attachment
)

// FlexInt handles JSON numbers that may come as strings or integers
type FlexInt int

func (fi *FlexInt) UnmarshalJSON(data []byte) error {
	// Try as int first
	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		*fi = FlexInt(i)
		return nil
	}
	// Try as string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if s == "" {
			*fi = 0
			return nil
		}
		i, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		*fi = FlexInt(i)
		return nil
	}
	return fmt.Errorf("cannot unmarshal %s into FlexInt", data)
}

// FlexFloat handles JSON numbers that may come as strings or numbers
type FlexFloat float64

func (ff *FlexFloat) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	// Try as float first
	var f float64
	if err := json.Unmarshal(data, &f); err == nil {
		*ff = FlexFloat(f)
		return nil
	}
	// Try as string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if s == "" {
			*ff = 0
			return nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		*ff = FlexFloat(f)
		return nil
	}
	return fmt.Errorf("cannot unmarshal %s into FlexFloat", data)
}

// FlexString handles JSON values that may come as strings or numbers
// and stores them as strings
type FlexString string

func (fs *FlexString) UnmarshalJSON(data []byte) error {
	// Try as string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*fs = FlexString(s)
		return nil
	}
	// Try as float64 (JSON numbers are float64)
	var f float64
	if err := json.Unmarshal(data, &f); err == nil {
		// Format as integer if it's a whole number
		if f == float64(int64(f)) {
			*fs = FlexString(strconv.FormatInt(int64(f), 10))
		} else {
			*fs = FlexString(strconv.FormatFloat(f, 'f', -1, 64))
		}
		return nil
	}
	return fmt.Errorf("cannot unmarshal %s into FlexString", data)
}

// String returns the string value
func (fs FlexString) String() string {
	return string(fs)
}

// Conversation represents a Chatwoot conversation
type Conversation struct {
	ID                     int                     `json:"id"`
	AccountID              int                     `json:"account_id"`
	InboxID                int                     `json:"inbox_id"`
	Status                 string                  `json:"status"`
	Priority               *string                 `json:"priority,omitempty"`
	AssigneeID             *int                    `json:"assignee_id,omitempty"`
	TeamID                 *int                    `json:"team_id,omitempty"`
	ContactID              int                     `json:"contact_id,omitempty"`
	DisplayID              *int                    `json:"display_id,omitempty"`
	Muted                  bool                    `json:"muted"`
	Unread                 int                     `json:"unread_count"`
	MessagesCount          int                     `json:"messages_count,omitempty"`
	FirstReplyCreatedAt    *int64                  `json:"first_reply_created_at,omitempty"`
	CreatedAt              int64                   `json:"created_at"`
	LastActivityAt         int64                   `json:"last_activity_at,omitempty"`
	Labels                 []string                `json:"labels,omitempty"`
	Meta                   map[string]any          `json:"meta,omitempty"`
	CustomAttributes       map[string]any          `json:"custom_attributes,omitempty"`
	LastNonActivityMessage *LastNonActivityMessage `json:"last_non_activity_message,omitempty"`
}

// LastNonActivityMessage is the most recent non-activity message in a conversation.
type LastNonActivityMessage struct {
	Content string `json:"content"`
}

// CreatedAtTime returns CreatedAt as time.Time
func (c *Conversation) CreatedAtTime() time.Time {
	return time.Unix(c.CreatedAt, 0)
}

// LastActivityAtTime returns LastActivityAt as time.Time
func (c *Conversation) LastActivityAtTime() time.Time {
	return time.Unix(c.LastActivityAt, 0)
}

// ConversationList is a paginated list of conversations
// Note: The nested Data structure matches the Chatwoot API response format exactly:
// {"data": {"meta": {...}, "payload": [...]}}
// This differs from ContactList which has a flatter structure without the "data" wrapper.
// See: app/views/api/v1/accounts/conversations/index.json.jbuilder in Chatwoot source
type ConversationList struct {
	Data struct {
		Meta    PaginationMeta `json:"meta"`
		Payload []Conversation `json:"payload"`
	} `json:"data"`
}

// MessageSender represents the sender of a message
type MessageSender struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

// Message represents a message in a conversation
type Message struct {
	ID             int            `json:"id"`
	ConversationID int            `json:"conversation_id"`
	Content        string         `json:"content"`
	ContentType    string         `json:"content_type"`
	MessageType    int            `json:"message_type"`
	Private        bool           `json:"private"`
	SenderID       *int           `json:"sender_id,omitempty"`
	SenderType     string         `json:"sender_type,omitempty"`
	Sender         *MessageSender `json:"sender,omitempty"`
	CreatedAt      int64          `json:"created_at"`
	Attachments    []Attachment   `json:"attachments,omitempty"`
}

// CreatedAtTime returns CreatedAt as time.Time
func (m *Message) CreatedAtTime() time.Time {
	return time.Unix(m.CreatedAt, 0)
}

// MessageTypeName returns the human-readable message type
func (m *Message) MessageTypeName() string {
	switch m.MessageType {
	case MessageTypeIncoming:
		return "incoming"
	case MessageTypeOutgoing:
		return "outgoing"
	case MessageTypeActivity:
		return "activity"
	case MessageTypeTemplate:
		return "template"
	default:
		return "unknown"
	}
}

// Attachment represents a message attachment
type Attachment struct {
	ID       int    `json:"id"`
	FileType string `json:"file_type"`
	DataURL  string `json:"data_url"`
	ThumbURL string `json:"thumb_url,omitempty"`
	FileSize int    `json:"file_size,omitempty"`
}

// Contact represents a Chatwoot contact
type Contact struct {
	ID               int            `json:"id"`
	Name             string         `json:"name"`
	Email            string         `json:"email,omitempty"`
	PhoneNumber      string         `json:"phone_number,omitempty"`
	Identifier       string         `json:"identifier,omitempty"`
	Thumbnail        string         `json:"thumbnail,omitempty"`
	CustomAttributes map[string]any `json:"custom_attributes,omitempty"`
	CreatedAt        int64          `json:"created_at"`
	LastActivityAt   *int64         `json:"last_activity_at,omitempty"`
}

// CreatedAtTime returns CreatedAt as time.Time
func (c *Contact) CreatedAtTime() time.Time {
	return time.Unix(c.CreatedAt, 0)
}

// ContactList is a paginated list of contacts
type ContactList struct {
	Payload []Contact      `json:"payload"`
	Meta    PaginationMeta `json:"meta"`
}

// ContactResponse wraps a single contact response (for show/update endpoints)
type ContactResponse struct {
	Payload Contact `json:"payload"`
}

// ContactCreateResponse wraps the create contact response
type ContactCreateResponse struct {
	Payload struct {
		Contact      Contact        `json:"contact"`
		ContactInbox map[string]any `json:"contact_inbox"`
	} `json:"payload"`
}

// Inbox represents a Chatwoot inbox
type Inbox struct {
	ID                   int    `json:"id"`
	Name                 string `json:"name"`
	ChannelType          string `json:"channel_type"`
	AvatarURL            string `json:"avatar_url,omitempty"`
	WebsiteURL           string `json:"website_url,omitempty"`
	GreetingEnabled      bool   `json:"greeting_enabled"`
	GreetingMessage      string `json:"greeting_message,omitempty"`
	EnableAutoAssignment bool   `json:"enable_auto_assignment"`
}

// ContactInbox represents a contact's association with an inbox
type ContactInbox struct {
	SourceID string `json:"source_id"`
	Inbox    Inbox  `json:"inbox"`
}

// Agent represents a Chatwoot agent/user
type Agent struct {
	ID                 int       `json:"id"`
	Name               string    `json:"name"`
	Email              string    `json:"email"`
	Role               string    `json:"role"`
	AvailabilityStatus string    `json:"availability_status,omitempty"`
	Thumbnail          string    `json:"thumbnail,omitempty"`
	ConfirmedAt        time.Time `json:"confirmed_at,omitempty"`
}

// Team represents a Chatwoot team
type Team struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description,omitempty"`
	AllowAutoAssign bool   `json:"allow_auto_assign"`
	AccountID       int    `json:"account_id"`
}

// CannedResponse represents a saved response template
type CannedResponse struct {
	ID        int    `json:"id"`
	ShortCode string `json:"short_code"`
	Content   string `json:"content"`
	AccountID int    `json:"account_id"`
}

// CustomAttribute represents a custom attribute definition
type CustomAttribute struct {
	ID                   int      `json:"id"`
	AttributeDisplayName string   `json:"attribute_display_name"`
	AttributeKey         string   `json:"attribute_key"`
	AttributeModel       string   `json:"attribute_model"`
	AttributeDisplayType string   `json:"attribute_display_type"`
	DefaultValue         any      `json:"default_value,omitempty"`
	AttributeValues      []string `json:"attribute_values,omitempty"`
}

// CustomFilter represents a saved filter
type CustomFilter struct {
	ID         int            `json:"id"`
	Name       string         `json:"name"`
	FilterType string         `json:"filter_type"`
	Query      map[string]any `json:"query"`
}

// Webhook represents a webhook subscription
type Webhook struct {
	ID            int      `json:"id"`
	URL           string   `json:"url"`
	Subscriptions []string `json:"subscriptions"`
	AccountID     int      `json:"account_id"`
}

// AutomationRule represents an automation rule
type AutomationRule struct {
	ID          int              `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	EventName   string           `json:"event_name"`
	Conditions  []map[string]any `json:"conditions"`
	Actions     []map[string]any `json:"actions"`
	Active      bool             `json:"active"`
	AccountID   int              `json:"account_id"`
}

// AgentBot represents an agent bot
type AgentBot struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	OutgoingURL string `json:"outgoing_url,omitempty"`
	AccountID   int    `json:"account_id"`
}

// Portal represents a help center portal
type Portal struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	CustomDomain string `json:"custom_domain,omitempty"`
	Color        string `json:"color,omitempty"`
	HomepageLink string `json:"homepage_link,omitempty"`
	PageTitle    string `json:"page_title,omitempty"`
	HeaderText   string `json:"header_text,omitempty"`
	AccountID    int    `json:"account_id"`
}

// Account represents the Chatwoot account
type Account struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Locale string `json:"locale"`
	Domain string `json:"domain,omitempty"`
}

// Profile represents the current user's profile
type Profile struct {
	ID                int       `json:"id"`
	Name              string    `json:"name"`
	Email             string    `json:"email"`
	PubsubToken       string    `json:"pubsub_token,omitempty"`
	AvailableAccounts []Account `json:"accounts,omitempty"`
}

// PaginationMeta contains pagination info
type PaginationMeta struct {
	CurrentPage FlexInt `json:"current_page,omitempty"`
	PerPage     FlexInt `json:"per_page,omitempty"`
	TotalPages  FlexInt `json:"total_pages,omitempty"`
	TotalCount  FlexInt `json:"total_count,omitempty"`
	HasMore     *bool   `json:"has_more,omitempty"`
}

// PortalListResponse wraps the portals list response
type PortalListResponse struct {
	Payload []Portal `json:"payload"`
	Meta    struct {
		CurrentPage  FlexInt `json:"current_page"`
		PortalsCount FlexInt `json:"portals_count"`
	} `json:"meta"`
}

// Integration represents an integration app
type Integration struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	Description        string            `json:"description,omitempty"`
	HookType           string            `json:"hook_type,omitempty"`
	Enabled            bool              `json:"enabled"`
	AllowMultipleHooks bool              `json:"allow_multiple_hooks,omitempty"`
	Hooks              []IntegrationHook `json:"hooks,omitempty"`
}

// IntegrationHook represents an integration hook
type IntegrationHook struct {
	ID        int            `json:"id"`
	AppID     string         `json:"app_id"`
	InboxID   int            `json:"inbox_id,omitempty"`
	AccountID int            `json:"account_id"`
	Settings  map[string]any `json:"settings,omitempty"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID             int            `json:"id"`
	Action         string         `json:"action"`
	AuditableType  string         `json:"auditable_type"`
	AuditableID    int            `json:"auditable_id"`
	UserID         int            `json:"user_id"`
	Username       string         `json:"username,omitempty"`
	AuditedChanges map[string]any `json:"audited_changes,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

// Report represents report data
type Report struct {
	Value     float64 `json:"value"`
	Timestamp string  `json:"timestamp,omitempty"`
}

// IntegrationAppsResponse wraps the integration apps response
type IntegrationAppsResponse struct {
	Payload []Integration `json:"payload"`
}

// WebhookListResponse wraps the webhook list response
type WebhookListResponse struct {
	Payload struct {
		Webhooks []Webhook `json:"webhooks"`
	} `json:"payload"`
}

// WebhookResponse wraps a single webhook response
type WebhookResponse struct {
	Payload struct {
		Webhook Webhook `json:"webhook"`
	} `json:"payload"`
}

// AutomationRuleListResponse wraps the automation rules list response
type AutomationRuleListResponse struct {
	Payload []AutomationRule `json:"payload"`
}

// AutomationRuleResponse wraps a single automation rule response
type AutomationRuleResponse struct {
	Payload AutomationRule `json:"payload"`
}

// ToggleStatusResponse represents the response from toggle_status endpoint
type ToggleStatusResponse struct {
	Meta    map[string]any `json:"meta"`
	Payload struct {
		Success        bool   `json:"success"`
		ConversationID int    `json:"conversation_id"`
		CurrentStatus  string `json:"current_status"`
		SnoozedUntil   *int64 `json:"snoozed_until"`
	} `json:"payload"`
}

// TogglePriorityResponse represents the response from toggle_priority endpoint
type TogglePriorityResponse struct {
	Meta    map[string]any `json:"meta"`
	Payload struct {
		Success         bool    `json:"success"`
		ConversationID  int     `json:"conversation_id"`
		CurrentPriority *string `json:"current_priority"`
	} `json:"payload"`
}

// Campaign represents a Chatwoot campaign (ongoing or one-off SMS).
// Note: The Campaigns API returns raw arrays/objects, not wrapped in response containers.
type Campaign struct {
	ID                             int                `json:"id"`
	Title                          string             `json:"title"`
	Description                    string             `json:"description,omitempty"`
	Message                        string             `json:"message"`
	Enabled                        bool               `json:"enabled"`
	CampaignType                   string             `json:"campaign_type"` // "ongoing" or "one_off"
	CampaignStatus                 string             `json:"campaign_status,omitempty"`
	InboxID                        int                `json:"inbox_id"`
	SenderID                       int                `json:"sender_id,omitempty"`
	ScheduledAt                    int64              `json:"scheduled_at,omitempty"`
	TriggerOnlyDuringBusinessHours bool               `json:"trigger_only_during_business_hours"`
	Audience                       []CampaignAudience `json:"audience,omitempty"`
	TriggerRules                   map[string]any     `json:"trigger_rules,omitempty"`
	CreatedAt                      int64              `json:"created_at"`
	UpdatedAt                      int64              `json:"updated_at,omitempty"`
	AccountID                      int                `json:"account_id"`
}

// CampaignAudience defines targeting for one-off campaigns.
type CampaignAudience struct {
	Type string `json:"type"` // "Label"
	ID   int    `json:"id"`
}

// ScheduledAtTime returns the scheduled time as time.Time.
func (c *Campaign) ScheduledAtTime() time.Time {
	if c.ScheduledAt == 0 {
		return time.Time{}
	}
	return time.Unix(c.ScheduledAt, 0)
}

// CreatedAtTime returns the created time as time.Time.
func (c *Campaign) CreatedAtTime() time.Time {
	return time.Unix(c.CreatedAt, 0)
}

// CSATResponse represents a customer satisfaction survey response
type CSATResponse struct {
	ID              int     `json:"id"`
	ConversationID  int     `json:"conversation_id"`
	Rating          int     `json:"rating"`
	FeedbackMessage string  `json:"feedback_message"`
	ContactID       int     `json:"contact_id"`
	AssignedAgentID *int    `json:"assigned_agent_id,omitempty"`
	CreatedAt       float64 `json:"created_at"`
	UpdatedAt       float64 `json:"updated_at,omitempty"`
}

// CreatedAtTime returns CreatedAt as time.Time
func (c *CSATResponse) CreatedAtTime() time.Time {
	return time.Unix(int64(c.CreatedAt), 0)
}

// CSATListResponse wraps the CSAT responses list
type CSATListResponse struct {
	Payload []CSATResponse `json:"payload"`
	Meta    PaginationMeta `json:"meta"`
}

// InboxTriage is the response for inbox triage command
type InboxTriage struct {
	InboxID       int                  `json:"inbox_id"`
	InboxName     string               `json:"inbox_name"`
	Summary       TriageSummary        `json:"summary"`
	Conversations []TriageConversation `json:"conversations"`
}

// TriageSummary contains counts for triage overview
type TriageSummary struct {
	Open    int `json:"open"`
	Pending int `json:"pending"`
	Unread  int `json:"unread"`
}

// TriageConversation represents an enriched conversation for triage
type TriageConversation struct {
	ID          int            `json:"id"`
	DisplayID   *int           `json:"display_id,omitempty"`
	Contact     TriageContact  `json:"contact"`
	Status      string         `json:"status"`
	Priority    *string        `json:"priority,omitempty"`
	UnreadCount int            `json:"unread_count"`
	LastMessage *TriageMessage `json:"last_message,omitempty"`
	Labels      []string       `json:"labels,omitempty"`
	AssigneeID  *int           `json:"assignee_id,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

// TriageContact contains essential contact info for triage
type TriageContact struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

// TriageMessage represents the last message in a conversation
type TriageMessage struct {
	Content string    `json:"content"`
	Type    string    `json:"type"` // "incoming" | "outgoing"
	At      time.Time `json:"at"`
}

// ReplyResult is the response for the reply command
type ReplyResult struct {
	Action         string         `json:"action"` // "replied" | "disambiguation_needed"
	ConversationID int            `json:"conversation_id,omitempty"`
	Contact        *TriageContact `json:"contact,omitempty"`
	MessageID      int            `json:"message_id,omitempty"`
	Resolved       bool           `json:"resolved,omitempty"`
	Pending        bool           `json:"pending,omitempty"`

	// For disambiguation
	Type    string `json:"type,omitempty"` // "multiple_contacts" | "multiple_conversations"
	Matches any    `json:"matches,omitempty"`
	Hint    string `json:"hint,omitempty"`
}

// Mention represents a mention of the current user in a private note
type Mention struct {
	ConversationID int       `json:"conversation_id"`
	MessageID      int       `json:"message_id"`
	Content        string    `json:"content"`
	SenderName     string    `json:"sender_name"`
	CreatedAt      time.Time `json:"created_at"`
}
