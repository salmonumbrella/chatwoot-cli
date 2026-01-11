package api

import (
	"context"
	"fmt"
	"net/http"
)

// PublicContact represents a contact created via public API
// Fields are limited to those documented in the public API.
type PublicContact struct {
	ID          int    `json:"id"`
	SourceID    string `json:"source_id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	PubsubToken string `json:"pubsub_token"`
}

// PublicInbox represents inbox info from public API
type PublicInbox struct {
	Name                string `json:"name"`
	WorkingHoursEnabled bool   `json:"working_hours_enabled"`
	Timezone            string `json:"timezone,omitempty"`
	WorkingHours        []any  `json:"working_hours,omitempty"`
	CsatSurveyEnabled   bool   `json:"csat_survey_enabled"`
}

// PublicUpdateContactRequest represents a request to update a public contact
type PublicUpdateContactRequest struct {
	Identifier       string         `json:"identifier,omitempty"`
	IdentifierHash   string         `json:"identifier_hash,omitempty"`
	Email            string         `json:"email,omitempty"`
	Name             string         `json:"name,omitempty"`
	PhoneNumber      string         `json:"phone_number,omitempty"`
	AvatarURL        string         `json:"avatar_url,omitempty"`
	CustomAttributes map[string]any `json:"custom_attributes,omitempty"`
}

// PublicCreateContactRequest represents a request to create a public contact
type PublicCreateContactRequest struct {
	Identifier       string         `json:"identifier,omitempty"`
	IdentifierHash   string         `json:"identifier_hash,omitempty"`
	Email            string         `json:"email,omitempty"`
	Name             string         `json:"name,omitempty"`
	PhoneNumber      string         `json:"phone_number,omitempty"`
	AvatarURL        string         `json:"avatar_url,omitempty"`
	CustomAttributes map[string]any `json:"custom_attributes,omitempty"`
}

// CreateContact creates a contact via public API.
func (s PublicService) CreateContact(ctx context.Context, inboxIdentifier string, req PublicCreateContactRequest) (*PublicContact, error) {
	return publicCreateContact(ctx, s, inboxIdentifier, req)
}

func publicCreateContact(ctx context.Context, r Requester, inboxIdentifier string, req PublicCreateContactRequest) (*PublicContact, error) {
	var result PublicContact
	path := fmt.Sprintf("/inboxes/%s/contacts", inboxIdentifier)
	if err := r.do(ctx, http.MethodPost, r.publicPath(path), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetContact retrieves a contact via public API.
func (s PublicService) GetContact(ctx context.Context, inboxIdentifier, contactIdentifier string) (*PublicContact, error) {
	return publicGetContact(ctx, s, inboxIdentifier, contactIdentifier)
}

func publicGetContact(ctx context.Context, r Requester, inboxIdentifier, contactIdentifier string) (*PublicContact, error) {
	var result PublicContact
	path := fmt.Sprintf("/inboxes/%s/contacts/%s", inboxIdentifier, contactIdentifier)
	if err := r.do(ctx, http.MethodGet, r.publicPath(path), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateConversation creates a conversation via public API.
func (s PublicService) CreateConversation(ctx context.Context, inboxIdentifier, contactIdentifier string, customAttributes map[string]any) (map[string]any, error) {
	return publicCreateConversation(ctx, s, inboxIdentifier, contactIdentifier, customAttributes)
}

func publicCreateConversation(ctx context.Context, r Requester, inboxIdentifier, contactIdentifier string, customAttributes map[string]any) (map[string]any, error) {
	body := map[string]any{}
	if customAttributes != nil {
		body["custom_attributes"] = customAttributes
	}

	path := fmt.Sprintf("/inboxes/%s/contacts/%s/conversations", inboxIdentifier, contactIdentifier)
	var result map[string]any
	if err := r.do(ctx, http.MethodPost, r.publicPath(path), body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ListConversations lists conversations via public API.
func (s PublicService) ListConversations(ctx context.Context, inboxIdentifier, contactIdentifier string) ([]map[string]any, error) {
	return publicListConversations(ctx, s, inboxIdentifier, contactIdentifier)
}

func publicListConversations(ctx context.Context, r Requester, inboxIdentifier, contactIdentifier string) ([]map[string]any, error) {
	path := fmt.Sprintf("/inboxes/%s/contacts/%s/conversations", inboxIdentifier, contactIdentifier)
	var result []map[string]any
	if err := r.do(ctx, http.MethodGet, r.publicPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetConversation retrieves a single conversation via public API.
func (s PublicService) GetConversation(ctx context.Context, inboxIdentifier, contactIdentifier string, conversationID int) (map[string]any, error) {
	return publicGetConversation(ctx, s, inboxIdentifier, contactIdentifier, conversationID)
}

func publicGetConversation(ctx context.Context, r Requester, inboxIdentifier, contactIdentifier string, conversationID int) (map[string]any, error) {
	path := fmt.Sprintf("/inboxes/%s/contacts/%s/conversations/%d", inboxIdentifier, contactIdentifier, conversationID)
	var result map[string]any
	if err := r.do(ctx, http.MethodGet, r.publicPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ResolveConversation resolves a conversation via public API.
func (s PublicService) ResolveConversation(ctx context.Context, inboxIdentifier, contactIdentifier string, conversationID int) (map[string]any, error) {
	return publicResolveConversation(ctx, s, inboxIdentifier, contactIdentifier, conversationID)
}

func publicResolveConversation(ctx context.Context, r Requester, inboxIdentifier, contactIdentifier string, conversationID int) (map[string]any, error) {
	path := fmt.Sprintf("/inboxes/%s/contacts/%s/conversations/%d/toggle_status", inboxIdentifier, contactIdentifier, conversationID)
	var result map[string]any
	if err := r.do(ctx, http.MethodPost, r.publicPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ToggleTyping toggles typing status via public API.
func (s PublicService) ToggleTyping(ctx context.Context, inboxIdentifier, contactIdentifier string, conversationID int, status string) error {
	return publicToggleTyping(ctx, s, inboxIdentifier, contactIdentifier, conversationID, status)
}

func publicToggleTyping(ctx context.Context, r Requester, inboxIdentifier, contactIdentifier string, conversationID int, status string) error {
	path := fmt.Sprintf("/inboxes/%s/contacts/%s/conversations/%d/toggle_typing", inboxIdentifier, contactIdentifier, conversationID)
	body := map[string]string{"typing_status": status}
	return r.do(ctx, http.MethodPost, r.publicPath(path), body, nil)
}

// UpdateLastSeen updates last seen via public API.
func (s PublicService) UpdateLastSeen(ctx context.Context, inboxIdentifier, contactIdentifier string, conversationID int) error {
	return publicUpdateLastSeen(ctx, s, inboxIdentifier, contactIdentifier, conversationID)
}

func publicUpdateLastSeen(ctx context.Context, r Requester, inboxIdentifier, contactIdentifier string, conversationID int) error {
	path := fmt.Sprintf("/inboxes/%s/contacts/%s/conversations/%d/update_last_seen", inboxIdentifier, contactIdentifier, conversationID)
	return r.do(ctx, http.MethodPost, r.publicPath(path), nil, nil)
}

// CreateMessage creates a message via public API.
func (s PublicService) CreateMessage(ctx context.Context, inboxIdentifier, contactIdentifier string, conversationID int, content, echoID string) (map[string]any, error) {
	return publicCreateMessage(ctx, s, inboxIdentifier, contactIdentifier, conversationID, content, echoID)
}

func publicCreateMessage(ctx context.Context, r Requester, inboxIdentifier, contactIdentifier string, conversationID int, content, echoID string) (map[string]any, error) {
	path := fmt.Sprintf("/inboxes/%s/contacts/%s/conversations/%d/messages", inboxIdentifier, contactIdentifier, conversationID)
	body := map[string]any{
		"content": content,
	}
	if echoID != "" {
		body["echo_id"] = echoID
	}
	var result map[string]any
	if err := r.do(ctx, http.MethodPost, r.publicPath(path), body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetInbox retrieves inbox info via public API.
func (s PublicService) GetInbox(ctx context.Context, inboxIdentifier string) (*PublicInbox, error) {
	return publicGetInbox(ctx, s, inboxIdentifier)
}

func publicGetInbox(ctx context.Context, r Requester, inboxIdentifier string) (*PublicInbox, error) {
	var result PublicInbox
	path := fmt.Sprintf("/inboxes/%s", inboxIdentifier)
	if err := r.do(ctx, http.MethodGet, r.publicPath(path), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateContact updates a contact via public API.
func (s PublicService) UpdateContact(ctx context.Context, inboxIdentifier, contactIdentifier string, req PublicUpdateContactRequest) (*PublicContact, error) {
	return publicUpdateContact(ctx, s, inboxIdentifier, contactIdentifier, req)
}

func publicUpdateContact(ctx context.Context, r Requester, inboxIdentifier, contactIdentifier string, req PublicUpdateContactRequest) (*PublicContact, error) {
	var result PublicContact
	path := fmt.Sprintf("/inboxes/%s/contacts/%s", inboxIdentifier, contactIdentifier)
	if err := r.do(ctx, http.MethodPatch, r.publicPath(path), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListMessages lists messages via public API.
func (s PublicService) ListMessages(ctx context.Context, inboxIdentifier, contactIdentifier string, conversationID int) ([]map[string]any, error) {
	return publicListMessages(ctx, s, inboxIdentifier, contactIdentifier, conversationID)
}

func publicListMessages(ctx context.Context, r Requester, inboxIdentifier, contactIdentifier string, conversationID int) ([]map[string]any, error) {
	path := fmt.Sprintf("/inboxes/%s/contacts/%s/conversations/%d/messages", inboxIdentifier, contactIdentifier, conversationID)
	var result []map[string]any
	if err := r.do(ctx, http.MethodGet, r.publicPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateMessage updates a message via public API.
func (s PublicService) UpdateMessage(ctx context.Context, inboxIdentifier, contactIdentifier string, conversationID, messageID int, content string) (map[string]any, error) {
	return publicUpdateMessage(ctx, s, inboxIdentifier, contactIdentifier, conversationID, messageID, content)
}

func publicUpdateMessage(ctx context.Context, r Requester, inboxIdentifier, contactIdentifier string, conversationID, messageID int, content string) (map[string]any, error) {
	path := fmt.Sprintf("/inboxes/%s/contacts/%s/conversations/%d/messages/%d", inboxIdentifier, contactIdentifier, conversationID, messageID)
	body := map[string]any{"content": content}
	var result map[string]any
	if err := r.do(ctx, http.MethodPatch, r.publicPath(path), body, &result); err != nil {
		return nil, err
	}
	return result, nil
}
