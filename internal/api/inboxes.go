package api

import (
	"context"
	"fmt"
)

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
func (c *Client) CreateInbox(ctx context.Context, name, channelType string) (*Inbox, error) {
	body := map[string]any{
		"name": name,
		"channel": map[string]string{
			"type": channelType,
		},
	}
	var result Inbox
	if err := c.Post(ctx, "/inboxes", body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateInbox updates an existing inbox
func (c *Client) UpdateInbox(ctx context.Context, id int, name string) (*Inbox, error) {
	body := map[string]any{
		"name": name,
	}
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
