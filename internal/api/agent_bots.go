package api

import (
	"context"
	"fmt"
	"net/http"
)

// List returns all agent bots for the account.
func (s AgentBotsService) List(ctx context.Context) ([]AgentBot, error) {
	return listAgentBots(ctx, s)
}

func listAgentBots(ctx context.Context, r Requester) ([]AgentBot, error) {
	var bots []AgentBot
	if err := r.do(ctx, http.MethodGet, r.accountPath("/agent_bots"), nil, &bots); err != nil {
		return nil, err
	}
	return bots, nil
}

// Get returns a specific agent bot by ID.
func (s AgentBotsService) Get(ctx context.Context, id int) (*AgentBot, error) {
	return getAgentBot(ctx, s, id)
}

func getAgentBot(ctx context.Context, r Requester, id int) (*AgentBot, error) {
	var bot AgentBot
	path := fmt.Sprintf("/agent_bots/%d", id)
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &bot); err != nil {
		return nil, err
	}
	return &bot, nil
}

// Create creates a new agent bot.
func (s AgentBotsService) Create(ctx context.Context, name, outgoingURL string) (*AgentBot, error) {
	return createAgentBot(ctx, s, name, outgoingURL)
}

func createAgentBot(ctx context.Context, r Requester, name, outgoingURL string) (*AgentBot, error) {
	body := map[string]any{
		"name":         name,
		"outgoing_url": outgoingURL,
	}
	var bot AgentBot
	if err := r.do(ctx, http.MethodPost, r.accountPath("/agent_bots"), body, &bot); err != nil {
		return nil, err
	}
	return &bot, nil
}

// Update updates an existing agent bot.
func (s AgentBotsService) Update(ctx context.Context, id int, name, outgoingURL string) (*AgentBot, error) {
	return updateAgentBot(ctx, s, id, name, outgoingURL)
}

func updateAgentBot(ctx context.Context, r Requester, id int, name, outgoingURL string) (*AgentBot, error) {
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if outgoingURL != "" {
		body["outgoing_url"] = outgoingURL
	}

	var bot AgentBot
	path := fmt.Sprintf("/agent_bots/%d", id)
	if err := r.do(ctx, http.MethodPatch, r.accountPath(path), body, &bot); err != nil {
		return nil, err
	}
	return &bot, nil
}

// Delete deletes an agent bot.
func (s AgentBotsService) Delete(ctx context.Context, id int) error {
	return deleteAgentBot(ctx, s, id)
}

func deleteAgentBot(ctx context.Context, r Requester, id int) error {
	path := fmt.Sprintf("/agent_bots/%d", id)
	return r.do(ctx, http.MethodDelete, r.accountPath(path), nil, nil)
}

// DeleteAvatar removes the avatar from an agent bot.
func (s AgentBotsService) DeleteAvatar(ctx context.Context, id int) error {
	return deleteAgentBotAvatar(ctx, s, id)
}

func deleteAgentBotAvatar(ctx context.Context, r Requester, id int) error {
	return r.do(ctx, http.MethodDelete, r.accountPath(fmt.Sprintf("/agent_bots/%d/avatar", id)), nil, nil)
}

// ResetAccessToken resets the access token for an agent bot.
func (s AgentBotsService) ResetAccessToken(ctx context.Context, id int) (string, error) {
	return resetAgentBotAccessToken(ctx, s, id)
}

func resetAgentBotAccessToken(ctx context.Context, r Requester, id int) (string, error) {
	path := fmt.Sprintf("/agent_bots/%d/reset_access_token", id)
	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := r.do(ctx, http.MethodPost, r.accountPath(path), nil, &result); err != nil {
		return "", err
	}
	return result.AccessToken, nil
}
