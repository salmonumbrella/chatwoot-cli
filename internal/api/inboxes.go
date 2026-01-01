package api

import (
	"context"
	"fmt"
)

// InboxSettings represents configurable inbox attributes
type InboxSettings struct {
	GreetingEnabled            *bool          `json:"greeting_enabled,omitempty"`
	GreetingMessage            string         `json:"greeting_message,omitempty"`
	EnableEmailCollect         *bool          `json:"enable_email_collect,omitempty"`
	CSATSurveyEnabled          *bool          `json:"csat_survey_enabled,omitempty"`
	EnableAutoAssignment       *bool          `json:"enable_auto_assignment,omitempty"`
	AutoAssignmentConfig       map[string]any `json:"auto_assignment_config,omitempty"`
	WorkingHoursEnabled        *bool          `json:"working_hours_enabled,omitempty"`
	Timezone                   string         `json:"timezone,omitempty"`
	AllowMessagesAfterResolved *bool          `json:"allow_messages_after_resolved,omitempty"`
	LockToSingleConversation   *bool          `json:"lock_to_single_conversation,omitempty"`
	PortalID                   *int           `json:"portal_id,omitempty"`
	SenderNameType             string         `json:"sender_name_type,omitempty"`
	OutOfOfficeMessage         string         `json:"out_of_office_message,omitempty"`
	OutOfOfficeEnabled         *bool          `json:"out_of_office_enabled,omitempty"`
}

// CreateInboxRequest represents a request to create an inbox
type CreateInboxRequest struct {
	Name        string
	ChannelType string
	InboxSettings
}

// UpdateInboxRequest represents a request to update an inbox
type UpdateInboxRequest struct {
	Name string
	InboxSettings
}

func applyInboxSettings(body map[string]any, settings InboxSettings) {
	if settings.GreetingEnabled != nil {
		body["greeting_enabled"] = *settings.GreetingEnabled
	}
	if settings.GreetingMessage != "" {
		body["greeting_message"] = settings.GreetingMessage
	}
	if settings.EnableEmailCollect != nil {
		body["enable_email_collect"] = *settings.EnableEmailCollect
	}
	if settings.CSATSurveyEnabled != nil {
		body["csat_survey_enabled"] = *settings.CSATSurveyEnabled
	}
	if settings.EnableAutoAssignment != nil {
		body["enable_auto_assignment"] = *settings.EnableAutoAssignment
	}
	if settings.AutoAssignmentConfig != nil {
		body["auto_assignment_config"] = settings.AutoAssignmentConfig
	}
	if settings.WorkingHoursEnabled != nil {
		body["working_hours_enabled"] = *settings.WorkingHoursEnabled
	}
	if settings.Timezone != "" {
		body["timezone"] = settings.Timezone
	}
	if settings.AllowMessagesAfterResolved != nil {
		body["allow_messages_after_resolved"] = *settings.AllowMessagesAfterResolved
	}
	if settings.LockToSingleConversation != nil {
		body["lock_to_single_conversation"] = *settings.LockToSingleConversation
	}
	if settings.PortalID != nil {
		body["portal_id"] = *settings.PortalID
	}
	if settings.SenderNameType != "" {
		body["sender_name_type"] = settings.SenderNameType
	}
	if settings.OutOfOfficeMessage != "" {
		body["out_of_office_message"] = settings.OutOfOfficeMessage
	}
	if settings.OutOfOfficeEnabled != nil {
		body["out_of_office_enabled"] = *settings.OutOfOfficeEnabled
	}
}

// ListInboxes retrieves all inboxes for the account
func (c *Client) ListInboxes(ctx context.Context) ([]Inbox, error) {
	var result struct {
		Payload []Inbox `json:"payload"`
	}
	if err := c.Get(ctx, "/inboxes", &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// GetInbox retrieves a specific inbox by ID
func (c *Client) GetInbox(ctx context.Context, id int) (*Inbox, error) {
	var result Inbox
	if err := c.Get(ctx, fmt.Sprintf("/inboxes/%d", id), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateInbox creates a new inbox
func (c *Client) CreateInbox(ctx context.Context, req CreateInboxRequest) (*Inbox, error) {
	body := map[string]any{}
	if req.Name != "" {
		body["name"] = req.Name
	}
	if req.ChannelType != "" {
		body["channel"] = map[string]string{
			"type": req.ChannelType,
		}
	}
	applyInboxSettings(body, req.InboxSettings)
	var result Inbox
	if err := c.Post(ctx, "/inboxes", body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateInbox updates an existing inbox
func (c *Client) UpdateInbox(ctx context.Context, id int, req UpdateInboxRequest) (*Inbox, error) {
	body := map[string]any{}
	if req.Name != "" {
		body["name"] = req.Name
	}
	applyInboxSettings(body, req.InboxSettings)
	var result Inbox
	if err := c.Patch(ctx, fmt.Sprintf("/inboxes/%d", id), body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteInbox deletes an inbox
func (c *Client) DeleteInbox(ctx context.Context, id int) error {
	return c.Delete(ctx, fmt.Sprintf("/inboxes/%d", id))
}

// GetInboxAgentBot retrieves the agent bot assigned to an inbox
func (c *Client) GetInboxAgentBot(ctx context.Context, id int) (*AgentBot, error) {
	var result AgentBot
	if err := c.Get(ctx, fmt.Sprintf("/inboxes/%d/agent_bot", id), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetInboxAgentBot assigns an agent bot to an inbox
func (c *Client) SetInboxAgentBot(ctx context.Context, inboxID, botID int) error {
	body := map[string]any{
		"agent_bot_id": botID,
	}
	return c.Post(ctx, fmt.Sprintf("/inboxes/%d/set_agent_bot", inboxID), body, nil)
}

// ListInboxMembers retrieves all agents assigned to an inbox
func (c *Client) ListInboxMembers(ctx context.Context, inboxID int) ([]Agent, error) {
	var result struct {
		Payload []Agent `json:"payload"`
	}
	if err := c.Get(ctx, fmt.Sprintf("/inbox_members/%d", inboxID), &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// AddInboxMembers adds agents to an inbox
func (c *Client) AddInboxMembers(ctx context.Context, inboxID int, userIDs []int) error {
	body := map[string]any{
		"inbox_id": inboxID,
		"user_ids": userIDs,
	}
	return c.Post(ctx, "/inbox_members", body, nil)
}

// RemoveInboxMembers removes agents from an inbox
func (c *Client) RemoveInboxMembers(ctx context.Context, inboxID int, userIDs []int) error {
	body := map[string]any{
		"inbox_id": inboxID,
		"user_ids": userIDs,
	}
	return c.DeleteWithBody(ctx, "/inbox_members", body)
}

// UpdateInboxMembers updates inbox members (replaces the list)
func (c *Client) UpdateInboxMembers(ctx context.Context, inboxID int, userIDs []int) error {
	body := map[string]any{
		"inbox_id": inboxID,
		"user_ids": userIDs,
	}
	return c.Patch(ctx, "/inbox_members", body, nil)
}
