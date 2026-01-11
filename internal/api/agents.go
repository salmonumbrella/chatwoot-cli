package api

import (
	"context"
	"fmt"
	"strings"
)

// ListAgents retrieves all agents in the account
func (c *Client) ListAgents(ctx context.Context) ([]Agent, error) {
	return listAgents(ctx, c)
}

// List retrieves all agents in the account.
func (s AgentsService) List(ctx context.Context) ([]Agent, error) {
	return listAgents(ctx, s)
}

func listAgents(ctx context.Context, r Requester) ([]Agent, error) {
	var agents []Agent
	if err := r.do(ctx, "GET", r.accountPath("/agents"), nil, &agents); err != nil {
		return nil, err
	}
	return agents, nil
}

// GetAgent retrieves a specific agent by ID
// Note: The Chatwoot API doesn't expose a show endpoint for individual agents,
// so this fetches all agents and filters by ID client-side
func (c *Client) GetAgent(ctx context.Context, id int) (*Agent, error) {
	return getAgent(ctx, c, id)
}

// Get retrieves a specific agent by ID.
func (s AgentsService) Get(ctx context.Context, id int) (*Agent, error) {
	return getAgent(ctx, s, id)
}

func getAgent(ctx context.Context, r Requester, id int) (*Agent, error) {
	agents, err := listAgents(ctx, r)
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
	return createAgent(ctx, c, name, email, role)
}

// Create creates a new agent.
func (s AgentsService) Create(ctx context.Context, name, email, role string) (*Agent, error) {
	return createAgent(ctx, s, name, email, role)
}

func createAgent(ctx context.Context, r Requester, name, email, role string) (*Agent, error) {
	body := map[string]any{
		"name":  name,
		"email": email,
		"role":  role,
	}
	var agent Agent
	if err := r.do(ctx, "POST", r.accountPath("/agents"), body, &agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// UpdateAgent updates an existing agent
func (c *Client) UpdateAgent(ctx context.Context, id int, name, role string) (*Agent, error) {
	return updateAgent(ctx, c, id, name, role)
}

// Update updates an existing agent.
func (s AgentsService) Update(ctx context.Context, id int, name, role string) (*Agent, error) {
	return updateAgent(ctx, s, id, name, role)
}

func updateAgent(ctx context.Context, r Requester, id int, name, role string) (*Agent, error) {
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if role != "" {
		body["role"] = role
	}
	var agent Agent
	path := fmt.Sprintf("/agents/%d", id)
	if err := r.do(ctx, "PATCH", r.accountPath(path), body, &agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// DeleteAgent deletes an agent
func (c *Client) DeleteAgent(ctx context.Context, id int) error {
	return deleteAgent(ctx, c, id)
}

// Delete deletes an agent.
func (s AgentsService) Delete(ctx context.Context, id int) error {
	return deleteAgent(ctx, s, id)
}

func deleteAgent(ctx context.Context, r Requester, id int) error {
	path := fmt.Sprintf("/agents/%d", id)
	return r.do(ctx, "DELETE", r.accountPath(path), nil, nil)
}

// BulkCreateAgentsRequest represents a request to create multiple agents
type BulkCreateAgentsRequest struct {
	Emails []string `json:"emails"`
}

// BulkCreateAgents creates multiple agents at once
func (c *Client) BulkCreateAgents(ctx context.Context, emails []string) ([]Agent, error) {
	return bulkCreateAgents(ctx, c, emails)
}

// BulkCreate creates multiple agents at once.
func (s AgentsService) BulkCreate(ctx context.Context, emails []string) ([]Agent, error) {
	return bulkCreateAgents(ctx, s, emails)
}

func bulkCreateAgents(ctx context.Context, r Requester, emails []string) ([]Agent, error) {
	body := BulkCreateAgentsRequest{Emails: emails}
	var result []Agent
	if err := r.do(ctx, "POST", r.accountPath("/agents/bulk_create"), body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// FindAgentByNameOrEmail searches for an agent by name or email (case-insensitive partial match)
// Returns the first matching agent, or an error if no match or multiple ambiguous matches found
func (c *Client) FindAgentByNameOrEmail(ctx context.Context, query string) (*Agent, error) {
	return findAgentByNameOrEmail(ctx, c, query)
}

// Find searches for an agent by name or email (case-insensitive partial match).
func (s AgentsService) Find(ctx context.Context, query string) (*Agent, error) {
	return findAgentByNameOrEmail(ctx, s, query)
}

func findAgentByNameOrEmail(ctx context.Context, r Requester, query string) (*Agent, error) {
	agents, err := listAgents(ctx, r)
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
