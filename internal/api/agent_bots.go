package api

import (
	"context"
	"fmt"
)

// ListAgentBots returns all agent bots for the account
func (c *Client) ListAgentBots(ctx context.Context) ([]AgentBot, error) {
	var bots []AgentBot
	if err := c.Get(ctx, "/agent_bots", &bots); err != nil {
		return nil, err
	}
	return bots, nil
}

// GetAgentBot returns a specific agent bot by ID
func (c *Client) GetAgentBot(ctx context.Context, id int) (*AgentBot, error) {
	var bot AgentBot
	path := fmt.Sprintf("/agent_bots/%d", id)
	if err := c.Get(ctx, path, &bot); err != nil {
		return nil, err
	}
	return &bot, nil
}

// CreateAgentBot creates a new agent bot
func (c *Client) CreateAgentBot(ctx context.Context, name, outgoingURL string) (*AgentBot, error) {
	body := map[string]any{
		"name":         name,
		"outgoing_url": outgoingURL,
	}
	var bot AgentBot
	if err := c.Post(ctx, "/agent_bots", body, &bot); err != nil {
		return nil, err
	}
	return &bot, nil
}

// UpdateAgentBot updates an existing agent bot
func (c *Client) UpdateAgentBot(ctx context.Context, id int, name, outgoingURL string) (*AgentBot, error) {
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if outgoingURL != "" {
		body["outgoing_url"] = outgoingURL
	}

	var bot AgentBot
	path := fmt.Sprintf("/agent_bots/%d", id)
	if err := c.Patch(ctx, path, body, &bot); err != nil {
		return nil, err
	}
	return &bot, nil
}

// DeleteAgentBot deletes an agent bot
func (c *Client) DeleteAgentBot(ctx context.Context, id int) error {
	path := fmt.Sprintf("/agent_bots/%d", id)
	return c.Delete(ctx, path)
}
