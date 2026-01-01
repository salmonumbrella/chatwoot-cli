package api

import (
	"context"
	"fmt"
)

// PlatformAgentBot represents a platform-level agent bot
type PlatformAgentBot struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	OutgoingURL string `json:"outgoing_url,omitempty"`
	BotType     string `json:"bot_type,omitempty"`
	BotConfig   any    `json:"bot_config,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
}

// CreatePlatformAgentBotRequest represents a request to create a platform agent bot
type CreatePlatformAgentBotRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	OutgoingURL string `json:"outgoing_url,omitempty"`
}

// UpdatePlatformAgentBotRequest represents a request to update a platform agent bot
type UpdatePlatformAgentBotRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	OutgoingURL string `json:"outgoing_url,omitempty"`
}

// ListPlatformAgentBots lists all platform agent bots
func (c *Client) ListPlatformAgentBots(ctx context.Context) ([]PlatformAgentBot, error) {
	var result []PlatformAgentBot
	if err := c.do(ctx, "GET", c.platformPath("/agent_bots"), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetPlatformAgentBot retrieves a platform agent bot by ID
func (c *Client) GetPlatformAgentBot(ctx context.Context, id int) (*PlatformAgentBot, error) {
	var result PlatformAgentBot
	if err := c.do(ctx, "GET", c.platformPath(fmt.Sprintf("/agent_bots/%d", id)), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreatePlatformAgentBot creates a new platform agent bot
func (c *Client) CreatePlatformAgentBot(ctx context.Context, req CreatePlatformAgentBotRequest) (*PlatformAgentBot, error) {
	var result PlatformAgentBot
	if err := c.do(ctx, "POST", c.platformPath("/agent_bots"), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdatePlatformAgentBot updates a platform agent bot
func (c *Client) UpdatePlatformAgentBot(ctx context.Context, id int, req UpdatePlatformAgentBotRequest) (*PlatformAgentBot, error) {
	var result PlatformAgentBot
	if err := c.do(ctx, "PATCH", c.platformPath(fmt.Sprintf("/agent_bots/%d", id)), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeletePlatformAgentBot deletes a platform agent bot
func (c *Client) DeletePlatformAgentBot(ctx context.Context, id int) error {
	return c.do(ctx, "DELETE", c.platformPath(fmt.Sprintf("/agent_bots/%d", id)), nil, nil)
}
