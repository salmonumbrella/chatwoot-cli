package api

import (
	"context"
	"fmt"
	"net/url"
)

// CreateConversationRequest represents a request to create a conversation
type CreateConversationRequest struct {
	InboxID          int            `json:"inbox_id"`
	ContactID        int            `json:"contact_id"`
	Message          string         `json:"message,omitempty"`
	Status           string         `json:"status,omitempty"`
	Assignee         *int           `json:"assignee_id,omitempty"`
	TeamID           *int           `json:"team_id,omitempty"`
	CustomAttributes map[string]any `json:"custom_attributes,omitempty"`
}

// ListConversations retrieves conversations filtered by status and inbox
func (c *Client) ListConversations(ctx context.Context, status, inboxID string, page int) (*ConversationList, error) {
	path := "/conversations"
	params := url.Values{}

	if status != "" && status != "all" {
		params.Set("status", status)
	}
	if inboxID != "" {
		params.Set("inbox_id", inboxID)
	}
	if page > 0 {
		params.Set("page", fmt.Sprintf("%d", page))
	}

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var result ConversationList
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetConversation retrieves a specific conversation by ID
func (c *Client) GetConversation(ctx context.Context, id int) (*Conversation, error) {
	var result Conversation
	if err := c.Get(ctx, fmt.Sprintf("/conversations/%d", id), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateConversation creates a new conversation
func (c *Client) CreateConversation(ctx context.Context, req CreateConversationRequest) (*Conversation, error) {
	var result Conversation
	if err := c.Post(ctx, "/conversations", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// FilterConversations filters conversations based on custom query payload
// Note: The filter API returns {"meta": ..., "payload": [...]} without the "data" wrapper
// that ListConversations returns, so we use a different response type
func (c *Client) FilterConversations(ctx context.Context, payload map[string]any) (*ConversationList, error) {
	var raw struct {
		Meta    PaginationMeta `json:"meta"`
		Payload []Conversation `json:"payload"`
	}
	if err := c.Post(ctx, "/conversations/filter", payload, &raw); err != nil {
		return nil, err
	}
	// Convert to ConversationList format for consistency
	return &ConversationList{
		Data: struct {
			Meta    PaginationMeta `json:"meta"`
			Payload []Conversation `json:"payload"`
		}{
			Meta:    raw.Meta,
			Payload: raw.Payload,
		},
	}, nil
}

// GetConversationsMeta retrieves metadata about conversations
func (c *Client) GetConversationsMeta(ctx context.Context) (map[string]any, error) {
	var result map[string]any
	if err := c.Get(ctx, "/conversations/meta", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ToggleConversationStatus toggles the status of a conversation
// If snoozedUntil is provided (non-zero), it will be included in the request when status is "snoozed"
func (c *Client) ToggleConversationStatus(ctx context.Context, id int, status string, snoozedUntil int64) (*ToggleStatusResponse, error) {
	payload := map[string]any{"status": status}
	if snoozedUntil > 0 {
		payload["snoozed_until"] = snoozedUntil
	}
	var result ToggleStatusResponse
	if err := c.Post(ctx, fmt.Sprintf("/conversations/%d/toggle_status", id), payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ToggleConversationPriority toggles the priority of a conversation
// Note: This endpoint returns HTTP 200 with no body, so we fetch the conversation after to get updated data
func (c *Client) ToggleConversationPriority(ctx context.Context, id int, priority string) error {
	payload := map[string]string{"priority": priority}
	return c.Post(ctx, fmt.Sprintf("/conversations/%d/toggle_priority", id), payload, nil)
}

// AssignConversation assigns a conversation to an agent and/or team
// Note: This endpoint returns the assigned agent/team object, not the conversation
func (c *Client) AssignConversation(ctx context.Context, id, agentID, teamID int) (any, error) {
	payload := make(map[string]any)
	if agentID > 0 {
		payload["assignee_id"] = agentID
	}
	if teamID > 0 {
		payload["team_id"] = teamID
	}

	var result any
	if err := c.Post(ctx, fmt.Sprintf("/conversations/%d/assignments", id), payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetConversationLabels retrieves labels for a conversation
func (c *Client) GetConversationLabels(ctx context.Context, id int) ([]string, error) {
	var result struct {
		Payload []string `json:"payload"`
	}
	if err := c.Get(ctx, fmt.Sprintf("/conversations/%d/labels", id), &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// AddConversationLabels adds labels to a conversation
func (c *Client) AddConversationLabels(ctx context.Context, id int, labels []string) ([]string, error) {
	payload := map[string][]string{"labels": labels}
	var result struct {
		Payload []string `json:"payload"`
	}
	if err := c.Post(ctx, fmt.Sprintf("/conversations/%d/labels", id), payload, &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// UpdateConversationCustomAttributes updates custom attributes for a conversation
func (c *Client) UpdateConversationCustomAttributes(ctx context.Context, id int, attrs map[string]any) error {
	payload := map[string]any{"custom_attributes": attrs}
	return c.Post(ctx, fmt.Sprintf("/conversations/%d/custom_attributes", id), payload, nil)
}

// MarkConversationUnread marks a conversation as unread for all agents
// This resets the agent_last_seen_at timestamp, making the conversation appear unread globally
func (c *Client) MarkConversationUnread(ctx context.Context, id int) error {
	return c.Post(ctx, fmt.Sprintf("/conversations/%d/unread", id), nil, nil)
}

// SearchConversations searches conversations by message content
func (c *Client) SearchConversations(ctx context.Context, query string, page int) (*ConversationList, error) {
	path := fmt.Sprintf("/conversations/search?q=%s", url.QueryEscape(query))
	if page > 0 {
		path = fmt.Sprintf("%s&page=%d", path, page)
	}

	var result ConversationList
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetConversationAttachments retrieves all attachments for a conversation
func (c *Client) GetConversationAttachments(ctx context.Context, id int) ([]Attachment, error) {
	path := fmt.Sprintf("/conversations/%d/attachments", id)
	var result []Attachment
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ToggleMuteConversation sets the mute status of a conversation
func (c *Client) ToggleMuteConversation(ctx context.Context, id int, mute bool) error {
	payload := map[string]bool{"status": mute}
	return c.Post(ctx, fmt.Sprintf("/conversations/%d/toggle_mute", id), payload, nil)
}

// UpdateConversation updates conversation attributes via PATCH endpoint
// Both priority and slaPolicyID are optional, but at least one must be provided
func (c *Client) UpdateConversation(ctx context.Context, id int, priority string, slaPolicyID int) (*Conversation, error) {
	payload := make(map[string]any)

	if priority != "" {
		payload["priority"] = priority
	}
	if slaPolicyID > 0 {
		payload["sla_policy_id"] = slaPolicyID
	}

	var result Conversation
	if err := c.Patch(ctx, fmt.Sprintf("/conversations/%d", id), payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
