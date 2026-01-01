package api

import (
	"context"
	"fmt"
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

// PublicCreateContact creates a contact via public API
func (c *Client) PublicCreateContact(ctx context.Context, inboxIdentifier string, req PublicCreateContactRequest) (*PublicContact, error) {
	var result PublicContact
	path := fmt.Sprintf("/inboxes/%s/contacts", inboxIdentifier)
	if err := c.do(ctx, "POST", c.publicPath(path), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PublicGetContact retrieves a contact via public API
func (c *Client) PublicGetContact(ctx context.Context, inboxIdentifier, contactIdentifier string) (*PublicContact, error) {
	var result PublicContact
	path := fmt.Sprintf("/inboxes/%s/contacts/%s", inboxIdentifier, contactIdentifier)
	if err := c.do(ctx, "GET", c.publicPath(path), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PublicCreateConversation creates a conversation via public API
func (c *Client) PublicCreateConversation(ctx context.Context, inboxIdentifier, contactIdentifier string, customAttributes map[string]any) (map[string]any, error) {
	body := map[string]any{}
	if customAttributes != nil {
		body["custom_attributes"] = customAttributes
	}

	path := fmt.Sprintf("/inboxes/%s/contacts/%s/conversations", inboxIdentifier, contactIdentifier)
	var result map[string]any
	if err := c.do(ctx, "POST", c.publicPath(path), body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// PublicListConversations lists conversations via public API
func (c *Client) PublicListConversations(ctx context.Context, inboxIdentifier, contactIdentifier string) ([]map[string]any, error) {
	path := fmt.Sprintf("/inboxes/%s/contacts/%s/conversations", inboxIdentifier, contactIdentifier)
	var result []map[string]any
	if err := c.do(ctx, "GET", c.publicPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// PublicGetConversation retrieves a single conversation via public API
func (c *Client) PublicGetConversation(ctx context.Context, inboxIdentifier, contactIdentifier string, conversationID int) (map[string]any, error) {
	path := fmt.Sprintf("/inboxes/%s/contacts/%s/conversations/%d", inboxIdentifier, contactIdentifier, conversationID)
	var result map[string]any
	if err := c.do(ctx, "GET", c.publicPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// PublicResolveConversation resolves a conversation via public API
func (c *Client) PublicResolveConversation(ctx context.Context, inboxIdentifier, contactIdentifier string, conversationID int) (map[string]any, error) {
	path := fmt.Sprintf("/inboxes/%s/contacts/%s/conversations/%d/toggle_status", inboxIdentifier, contactIdentifier, conversationID)
	var result map[string]any
	if err := c.do(ctx, "POST", c.publicPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// PublicToggleTyping toggles typing status via public API
func (c *Client) PublicToggleTyping(ctx context.Context, inboxIdentifier, contactIdentifier string, conversationID int, status string) error {
	path := fmt.Sprintf("/inboxes/%s/contacts/%s/conversations/%d/toggle_typing", inboxIdentifier, contactIdentifier, conversationID)
	body := map[string]string{"typing_status": status}
	return c.do(ctx, "POST", c.publicPath(path), body, nil)
}

// PublicUpdateLastSeen updates last seen via public API
func (c *Client) PublicUpdateLastSeen(ctx context.Context, inboxIdentifier, contactIdentifier string, conversationID int) error {
	path := fmt.Sprintf("/inboxes/%s/contacts/%s/conversations/%d/update_last_seen", inboxIdentifier, contactIdentifier, conversationID)
	return c.do(ctx, "POST", c.publicPath(path), nil, nil)
}

// PublicCreateMessage creates a message via public API
func (c *Client) PublicCreateMessage(ctx context.Context, inboxIdentifier, contactIdentifier string, conversationID int, content, echoID string) (map[string]any, error) {
	path := fmt.Sprintf("/inboxes/%s/contacts/%s/conversations/%d/messages", inboxIdentifier, contactIdentifier, conversationID)
	body := map[string]any{
		"content": content,
	}
	if echoID != "" {
		body["echo_id"] = echoID
	}
	var result map[string]any
	if err := c.do(ctx, "POST", c.publicPath(path), body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// PublicGetInbox retrieves inbox info via public API
func (c *Client) PublicGetInbox(ctx context.Context, inboxIdentifier string) (*PublicInbox, error) {
	var result PublicInbox
	path := fmt.Sprintf("/inboxes/%s", inboxIdentifier)
	if err := c.do(ctx, "GET", c.publicPath(path), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PublicUpdateContact updates a contact via public API
func (c *Client) PublicUpdateContact(ctx context.Context, inboxIdentifier, contactIdentifier string, req PublicUpdateContactRequest) (*PublicContact, error) {
	var result PublicContact
	path := fmt.Sprintf("/inboxes/%s/contacts/%s", inboxIdentifier, contactIdentifier)
	if err := c.do(ctx, "PATCH", c.publicPath(path), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PublicListMessages lists messages via public API
func (c *Client) PublicListMessages(ctx context.Context, inboxIdentifier, contactIdentifier string, conversationID int) ([]map[string]any, error) {
	path := fmt.Sprintf("/inboxes/%s/contacts/%s/conversations/%d/messages", inboxIdentifier, contactIdentifier, conversationID)
	var result []map[string]any
	if err := c.do(ctx, "GET", c.publicPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// PublicUpdateMessage updates a message via public API
func (c *Client) PublicUpdateMessage(ctx context.Context, inboxIdentifier, contactIdentifier string, conversationID, messageID int, content string) (map[string]any, error) {
	path := fmt.Sprintf("/inboxes/%s/contacts/%s/conversations/%d/messages/%d", inboxIdentifier, contactIdentifier, conversationID, messageID)
	body := map[string]any{"content": content}
	var result map[string]any
	if err := c.do(ctx, "PATCH", c.publicPath(path), body, &result); err != nil {
		return nil, err
	}
	return result, nil
}
