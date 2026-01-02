package api

import (
	"context"
	"fmt"
	"strings"
)

// ListAgents retrieves all agents in the account
func (c *Client) ListAgents(ctx context.Context) ([]Agent, error) {
	var agents []Agent
	if err := c.Get(ctx, "/agents", &agents); err != nil {
		return nil, err
	}
	return agents, nil
}

// GetAgent retrieves a specific agent by ID
// Note: The Chatwoot API doesn't expose a show endpoint for individual agents,
// so this fetches all agents and filters by ID client-side
func (c *Client) GetAgent(ctx context.Context, id int) (*Agent, error) {
	agents, err := c.ListAgents(ctx)
	if err != nil {
		return nil, err
	}

	for _, agent := range agents {
		if agent.ID == id {
			return &agent, nil
		}
	}

	return nil, &APIError{
		StatusCode: 404,
		Body:       fmt.Sprintf("agent with ID %d not found", id),
	}
}

// CreateAgent creates a new agent
func (c *Client) CreateAgent(ctx context.Context, name, email, role string) (*Agent, error) {
	body := map[string]any{
		"name":  name,
		"email": email,
		"role":  role,
	}
	var agent Agent
	if err := c.Post(ctx, "/agents", body, &agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// UpdateAgent updates an existing agent
func (c *Client) UpdateAgent(ctx context.Context, id int, name, role string) (*Agent, error) {
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if role != "" {
		body["role"] = role
	}
	var agent Agent
	path := fmt.Sprintf("/agents/%d", id)
	if err := c.Patch(ctx, path, body, &agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// DeleteAgent deletes an agent
func (c *Client) DeleteAgent(ctx context.Context, id int) error {
	path := fmt.Sprintf("/agents/%d", id)
	return c.Delete(ctx, path)
}

// FindAgentByNameOrEmail searches for an agent by name or email (case-insensitive partial match)
// Returns the first matching agent, or an error if no match or multiple ambiguous matches found
func (c *Client) FindAgentByNameOrEmail(ctx context.Context, query string) (*Agent, error) {
	agents, err := c.ListAgents(ctx)
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var matches []Agent

	for _, agent := range agents {
		nameLower := strings.ToLower(agent.Name)
		emailLower := strings.ToLower(agent.Email)

		// Exact match on name or email takes priority
		if nameLower == query || emailLower == query {
			return &agent, nil
		}

		// Partial match on name or email prefix (before @)
		emailPrefix := emailLower
		if idx := strings.Index(emailLower, "@"); idx > 0 {
			emailPrefix = emailLower[:idx]
		}
		if strings.Contains(nameLower, query) || strings.HasPrefix(emailPrefix, query) {
			matches = append(matches, agent)
		}
	}

	if len(matches) == 0 {
		return nil, &APIError{
			StatusCode: 404,
			Body:       fmt.Sprintf("no agent found matching '%s'", query),
		}
	}

	if len(matches) == 1 {
		return &matches[0], nil
	}

	// Multiple matches - return error with suggestions
	var names []string
	for _, m := range matches {
		names = append(names, fmt.Sprintf("%s (%s)", m.Name, m.Email))
	}
	return nil, &APIError{
		StatusCode: 400,
		Body:       fmt.Sprintf("ambiguous match for '%s': %s", query, strings.Join(names, ", ")),
	}
}
