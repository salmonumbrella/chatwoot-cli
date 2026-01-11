package api

import (
	"context"
	"fmt"
	"net/http"
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

// List lists all platform agent bots.
func (s PlatformAgentBotsService) List(ctx context.Context) ([]PlatformAgentBot, error) {
	return listPlatformAgentBots(ctx, s)
}

func listPlatformAgentBots(ctx context.Context, r Requester) ([]PlatformAgentBot, error) {
	var result []PlatformAgentBot
	if err := r.do(ctx, http.MethodGet, r.platformPath("/agent_bots"), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Get retrieves a platform agent bot by ID.
func (s PlatformAgentBotsService) Get(ctx context.Context, id int) (*PlatformAgentBot, error) {
	return getPlatformAgentBot(ctx, s, id)
}

func getPlatformAgentBot(ctx context.Context, r Requester, id int) (*PlatformAgentBot, error) {
	var result PlatformAgentBot
	if err := r.do(ctx, http.MethodGet, r.platformPath(fmt.Sprintf("/agent_bots/%d", id)), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Create creates a new platform agent bot.
func (s PlatformAgentBotsService) Create(ctx context.Context, req CreatePlatformAgentBotRequest) (*PlatformAgentBot, error) {
	return createPlatformAgentBot(ctx, s, req)
}

func createPlatformAgentBot(ctx context.Context, r Requester, req CreatePlatformAgentBotRequest) (*PlatformAgentBot, error) {
	var result PlatformAgentBot
	if err := r.do(ctx, http.MethodPost, r.platformPath("/agent_bots"), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Update updates a platform agent bot.
func (s PlatformAgentBotsService) Update(ctx context.Context, id int, req UpdatePlatformAgentBotRequest) (*PlatformAgentBot, error) {
	return updatePlatformAgentBot(ctx, s, id, req)
}

func updatePlatformAgentBot(ctx context.Context, r Requester, id int, req UpdatePlatformAgentBotRequest) (*PlatformAgentBot, error) {
	var result PlatformAgentBot
	if err := r.do(ctx, http.MethodPatch, r.platformPath(fmt.Sprintf("/agent_bots/%d", id)), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes a platform agent bot.
func (s PlatformAgentBotsService) Delete(ctx context.Context, id int) error {
	return deletePlatformAgentBot(ctx, s, id)
}

func deletePlatformAgentBot(ctx context.Context, r Requester, id int) error {
	return r.do(ctx, http.MethodDelete, r.platformPath(fmt.Sprintf("/agent_bots/%d", id)), nil, nil)
}
