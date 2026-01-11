package api

import (
	"context"
	"fmt"
	"net/url"
	"strings"
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

// ListConversationsParams defines filters for listing conversations
type ListConversationsParams struct {
	Status       string
	InboxID      string
	AssigneeType string
	TeamID       string
	Labels       []string
	Query        string
	Page         int
}

// ListConversations retrieves conversations filtered by params
func (c *Client) ListConversations(ctx context.Context, params ListConversationsParams) (*ConversationList, error) {
	return listConversations(ctx, c, params)
}

// List retrieves conversations filtered by params.
func (s ConversationsService) List(ctx context.Context, params ListConversationsParams) (*ConversationList, error) {
	return listConversations(ctx, s, params)
}

func listConversations(ctx context.Context, r Requester, params ListConversationsParams) (*ConversationList, error) {
	path := "/conversations"
	query := url.Values{}

	if params.Status != "" && params.Status != "all" {
		query.Set("status", params.Status)
	}
	if params.InboxID != "" {
		query.Set("inbox_id", params.InboxID)
	}
	if params.AssigneeType != "" {
		query.Set("assignee_type", params.AssigneeType)
	}
	if params.TeamID != "" {
		query.Set("team_id", params.TeamID)
	}
	if len(params.Labels) > 0 {
		query.Set("labels", strings.Join(params.Labels, ","))
	}
	if params.Query != "" {
		query.Set("q", params.Query)
	}
	if params.Page > 0 {
		query.Set("page", fmt.Sprintf("%d", params.Page))
	}

	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	var result ConversationList
	if err := r.do(ctx, "GET", r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetConversation retrieves a specific conversation by ID
func (c *Client) GetConversation(ctx context.Context, id int) (*Conversation, error) {
	return getConversation(ctx, c, id)
}

// Get retrieves a specific conversation by ID.
func (s ConversationsService) Get(ctx context.Context, id int) (*Conversation, error) {
	return getConversation(ctx, s, id)
}

func getConversation(ctx context.Context, r Requester, id int) (*Conversation, error) {
	var result Conversation
	if err := r.do(ctx, "GET", r.accountPath(fmt.Sprintf("/conversations/%d", id)), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateConversation creates a new conversation
func (c *Client) CreateConversation(ctx context.Context, req CreateConversationRequest) (*Conversation, error) {
	return createConversation(ctx, c, req)
}

// Create creates a new conversation.
func (s ConversationsService) Create(ctx context.Context, req CreateConversationRequest) (*Conversation, error) {
	return createConversation(ctx, s, req)
}

func createConversation(ctx context.Context, r Requester, req CreateConversationRequest) (*Conversation, error) {
	var result Conversation
	if err := r.do(ctx, "POST", r.accountPath("/conversations"), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// FilterConversations filters conversations based on custom query payload
// Note: The filter API returns {"meta": ..., "payload": [...]} without the "data" wrapper
// that ListConversations returns, so we use a different response type
func (c *Client) FilterConversations(ctx context.Context, payload map[string]any) (*ConversationList, error) {
	return filterConversations(ctx, c, payload)
}

// Filter filters conversations based on custom query payload.
func (s ConversationsService) Filter(ctx context.Context, payload map[string]any) (*ConversationList, error) {
	return filterConversations(ctx, s, payload)
}

func filterConversations(ctx context.Context, r Requester, payload map[string]any) (*ConversationList, error) {
	var raw struct {
		Meta    PaginationMeta `json:"meta"`
		Payload []Conversation `json:"payload"`
	}
	if err := r.do(ctx, "POST", r.accountPath("/conversations/filter"), payload, &raw); err != nil {
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
func (c *Client) GetConversationsMeta(ctx context.Context, params ListConversationsParams) (map[string]any, error) {
	return getConversationsMeta(ctx, c, params)
}

// Meta retrieves metadata about conversations.
func (s ConversationsService) Meta(ctx context.Context, params ListConversationsParams) (map[string]any, error) {
	return getConversationsMeta(ctx, s, params)
}

func getConversationsMeta(ctx context.Context, r Requester, params ListConversationsParams) (map[string]any, error) {
	path := "/conversations/meta"
	query := url.Values{}

	if params.Status != "" && params.Status != "all" {
		query.Set("status", params.Status)
	}
	if params.InboxID != "" {
		query.Set("inbox_id", params.InboxID)
	}
	if params.TeamID != "" {
		query.Set("team_id", params.TeamID)
	}
	if len(params.Labels) > 0 {
		query.Set("labels", strings.Join(params.Labels, ","))
	}
	if params.Query != "" {
		query.Set("q", params.Query)
	}

	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	var result map[string]any
	if err := r.do(ctx, "GET", r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ToggleConversationStatus toggles the status of a conversation
// If snoozedUntil is provided (non-zero), it will be included in the request when status is "snoozed"
func (c *Client) ToggleConversationStatus(ctx context.Context, id int, status string, snoozedUntil int64) (*ToggleStatusResponse, error) {
	return toggleConversationStatus(ctx, c, id, status, snoozedUntil)
}

// ToggleStatus toggles the status of a conversation.
func (s ConversationsService) ToggleStatus(ctx context.Context, id int, status string, snoozedUntil int64) (*ToggleStatusResponse, error) {
	return toggleConversationStatus(ctx, s, id, status, snoozedUntil)
}

func toggleConversationStatus(ctx context.Context, r Requester, id int, status string, snoozedUntil int64) (*ToggleStatusResponse, error) {
	payload := map[string]any{"status": status}
	if snoozedUntil > 0 {
		payload["snoozed_until"] = snoozedUntil
	}
	var result ToggleStatusResponse
	if err := r.do(ctx, "POST", r.accountPath(fmt.Sprintf("/conversations/%d/toggle_status", id)), payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ToggleConversationPriority toggles the priority of a conversation
// Note: This endpoint returns HTTP 200 with no body, so we fetch the conversation after to get updated data
func (c *Client) ToggleConversationPriority(ctx context.Context, id int, priority string) error {
	return toggleConversationPriority(ctx, c, id, priority)
}

// TogglePriority toggles the priority of a conversation.
func (s ConversationsService) TogglePriority(ctx context.Context, id int, priority string) error {
	return toggleConversationPriority(ctx, s, id, priority)
}

func toggleConversationPriority(ctx context.Context, r Requester, id int, priority string) error {
	payload := map[string]string{"priority": priority}
	return r.do(ctx, "POST", r.accountPath(fmt.Sprintf("/conversations/%d/toggle_priority", id)), payload, nil)
}

// AssignConversation assigns a conversation to an agent and/or team
// Note: This endpoint returns the assigned agent/team object, not the conversation
func (c *Client) AssignConversation(ctx context.Context, id, agentID, teamID int) (any, error) {
	return assignConversation(ctx, c, id, agentID, teamID)
}

// Assign assigns a conversation to an agent and/or team.
func (s ConversationsService) Assign(ctx context.Context, id, agentID, teamID int) (any, error) {
	return assignConversation(ctx, s, id, agentID, teamID)
}

func assignConversation(ctx context.Context, r Requester, id, agentID, teamID int) (any, error) {
	payload := make(map[string]any)
	if agentID > 0 {
		payload["assignee_id"] = agentID
	}
	if teamID > 0 {
		payload["team_id"] = teamID
	}

	var result any
	if err := r.do(ctx, "POST", r.accountPath(fmt.Sprintf("/conversations/%d/assignments", id)), payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetConversationLabels retrieves labels for a conversation
func (c *Client) GetConversationLabels(ctx context.Context, id int) ([]string, error) {
	return getConversationLabels(ctx, c, id)
}

// Labels retrieves labels for a conversation.
func (s ConversationsService) Labels(ctx context.Context, id int) ([]string, error) {
	return getConversationLabels(ctx, s, id)
}

func getConversationLabels(ctx context.Context, r Requester, id int) ([]string, error) {
	var result struct {
		Payload []string `json:"payload"`
	}
	if err := r.do(ctx, "GET", r.accountPath(fmt.Sprintf("/conversations/%d/labels", id)), nil, &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// AddConversationLabels adds labels to a conversation
func (c *Client) AddConversationLabels(ctx context.Context, id int, labels []string) ([]string, error) {
	return addConversationLabels(ctx, c, id, labels)
}

// AddLabels adds labels to a conversation.
func (s ConversationsService) AddLabels(ctx context.Context, id int, labels []string) ([]string, error) {
	return addConversationLabels(ctx, s, id, labels)
}

func addConversationLabels(ctx context.Context, r Requester, id int, labels []string) ([]string, error) {
	payload := map[string][]string{"labels": labels}
	var result struct {
		Payload []string `json:"payload"`
	}
	if err := r.do(ctx, "POST", r.accountPath(fmt.Sprintf("/conversations/%d/labels", id)), payload, &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// UpdateConversationCustomAttributes updates custom attributes for a conversation
func (c *Client) UpdateConversationCustomAttributes(ctx context.Context, id int, attrs map[string]any) error {
	return updateConversationCustomAttributes(ctx, c, id, attrs)
}

// UpdateCustomAttributes updates custom attributes for a conversation.
func (s ConversationsService) UpdateCustomAttributes(ctx context.Context, id int, attrs map[string]any) error {
	return updateConversationCustomAttributes(ctx, s, id, attrs)
}

func updateConversationCustomAttributes(ctx context.Context, r Requester, id int, attrs map[string]any) error {
	payload := map[string]any{"custom_attributes": attrs}
	return r.do(ctx, "POST", r.accountPath(fmt.Sprintf("/conversations/%d/custom_attributes", id)), payload, nil)
}

// MarkConversationUnread marks a conversation as unread for all agents
// This resets the agent_last_seen_at timestamp, making the conversation appear unread globally
func (c *Client) MarkConversationUnread(ctx context.Context, id int) error {
	return markConversationUnread(ctx, c, id)
}

// MarkUnread marks a conversation as unread for all agents.
func (s ConversationsService) MarkUnread(ctx context.Context, id int) error {
	return markConversationUnread(ctx, s, id)
}

func markConversationUnread(ctx context.Context, r Requester, id int) error {
	return r.do(ctx, "POST", r.accountPath(fmt.Sprintf("/conversations/%d/unread", id)), nil, nil)
}

// SearchConversations searches conversations by message content
func (c *Client) SearchConversations(ctx context.Context, query string, page int) (*ConversationList, error) {
	return searchConversations(ctx, c, query, page)
}

// Search searches conversations by message content.
func (s ConversationsService) Search(ctx context.Context, query string, page int) (*ConversationList, error) {
	return searchConversations(ctx, s, query, page)
}

func searchConversations(ctx context.Context, r Requester, query string, page int) (*ConversationList, error) {
	path := fmt.Sprintf("/conversations/search?q=%s", url.QueryEscape(query))
	if page > 0 {
		path = fmt.Sprintf("%s&page=%d", path, page)
	}

	var result ConversationList
	if err := r.do(ctx, "GET", r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetConversationAttachments retrieves all attachments for a conversation
func (c *Client) GetConversationAttachments(ctx context.Context, id int) ([]Attachment, error) {
	return getConversationAttachments(ctx, c, id)
}

// Attachments retrieves all attachments for a conversation.
func (s ConversationsService) Attachments(ctx context.Context, id int) ([]Attachment, error) {
	return getConversationAttachments(ctx, s, id)
}

func getConversationAttachments(ctx context.Context, r Requester, id int) ([]Attachment, error) {
	path := fmt.Sprintf("/conversations/%d/attachments", id)
	var result []Attachment
	if err := r.do(ctx, "GET", r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ToggleMuteConversation sets the mute status of a conversation
func (c *Client) ToggleMuteConversation(ctx context.Context, id int, mute bool) error {
	return toggleMuteConversation(ctx, c, id, mute)
}

// ToggleMute sets the mute status of a conversation.
func (s ConversationsService) ToggleMute(ctx context.Context, id int, mute bool) error {
	return toggleMuteConversation(ctx, s, id, mute)
}

func toggleMuteConversation(ctx context.Context, r Requester, id int, mute bool) error {
	payload := map[string]bool{"status": mute}
	return r.do(ctx, "POST", r.accountPath(fmt.Sprintf("/conversations/%d/toggle_mute", id)), payload, nil)
}

// MuteConversation mutes a conversation
func (c *Client) MuteConversation(ctx context.Context, id int) error {
	return muteConversation(ctx, c, id)
}

// Mute mutes a conversation.
func (s ConversationsService) Mute(ctx context.Context, id int) error {
	return muteConversation(ctx, s, id)
}

func muteConversation(ctx context.Context, r Requester, id int) error {
	return r.do(ctx, "POST", r.accountPath(fmt.Sprintf("/conversations/%d/mute", id)), nil, nil)
}

// UnmuteConversation unmutes a conversation
func (c *Client) UnmuteConversation(ctx context.Context, id int) error {
	return unmuteConversation(ctx, c, id)
}

// Unmute unmutes a conversation.
func (s ConversationsService) Unmute(ctx context.Context, id int) error {
	return unmuteConversation(ctx, s, id)
}

func unmuteConversation(ctx context.Context, r Requester, id int) error {
	return r.do(ctx, "POST", r.accountPath(fmt.Sprintf("/conversations/%d/unmute", id)), nil, nil)
}

// SendTranscript sends conversation transcript via email
func (c *Client) SendTranscript(ctx context.Context, id int, email string) error {
	return sendTranscript(ctx, c, id, email)
}

// Transcript sends conversation transcript via email.
func (s ConversationsService) Transcript(ctx context.Context, id int, email string) error {
	return sendTranscript(ctx, s, id, email)
}

func sendTranscript(ctx context.Context, r Requester, id int, email string) error {
	body := map[string]string{"email": email}
	return r.do(ctx, "POST", r.accountPath(fmt.Sprintf("/conversations/%d/transcript", id)), body, nil)
}

// ToggleTypingStatus toggles typing indicator for a conversation
func (c *Client) ToggleTypingStatus(ctx context.Context, id int, typingOn bool, isPrivate bool) error {
	return toggleTypingStatus(ctx, c, id, typingOn, isPrivate)
}

// ToggleTyping toggles typing indicator for a conversation.
func (s ConversationsService) ToggleTyping(ctx context.Context, id int, typingOn bool, isPrivate bool) error {
	return toggleTypingStatus(ctx, s, id, typingOn, isPrivate)
}

func toggleTypingStatus(ctx context.Context, r Requester, id int, typingOn bool, isPrivate bool) error {
	status := "off"
	if typingOn {
		status = "on"
	}
	body := map[string]any{
		"typing_status": status,
		"is_private":    isPrivate,
	}
	return r.do(ctx, "POST", r.accountPath(fmt.Sprintf("/conversations/%d/toggle_typing_status", id)), body, nil)
}

// UpdateConversation updates conversation attributes via PATCH endpoint
// Both priority and slaPolicyID are optional, but at least one must be provided
func (c *Client) UpdateConversation(ctx context.Context, id int, priority string, slaPolicyID int) (*Conversation, error) {
	return updateConversation(ctx, c, id, priority, slaPolicyID)
}

// Update updates conversation attributes via PATCH endpoint.
func (s ConversationsService) Update(ctx context.Context, id int, priority string, slaPolicyID int) (*Conversation, error) {
	return updateConversation(ctx, s, id, priority, slaPolicyID)
}

func updateConversation(ctx context.Context, r Requester, id int, priority string, slaPolicyID int) (*Conversation, error) {
	payload := make(map[string]any)

	if priority != "" {
		payload["priority"] = priority
	}
	if slaPolicyID > 0 {
		payload["sla_policy_id"] = slaPolicyID
	}

	var result Conversation
	if err := r.do(ctx, "PATCH", r.accountPath(fmt.Sprintf("/conversations/%d", id)), payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
