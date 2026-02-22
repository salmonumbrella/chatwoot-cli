package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// List retrieves all agents in the account.
func (s AgentsService) List(ctx context.Context) ([]Agent, error) {
	return listAgents(ctx, s)
}

func listAgents(ctx context.Context, r Requester) ([]Agent, error) {
	var agents []Agent
	if err := r.do(ctx, http.MethodGet, r.accountPath("/agents"), nil, &agents); err != nil {
		return nil, err
	}
	return agents, nil
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
	if err := r.do(ctx, http.MethodPost, r.accountPath("/agents"), body, &agent); err != nil {
		return nil, err
	}
	return &agent, nil
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
	if err := r.do(ctx, http.MethodPatch, r.accountPath(path), body, &agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// Delete deletes an agent.
func (s AgentsService) Delete(ctx context.Context, id int) error {
	return deleteAgent(ctx, s, id)
}

func deleteAgent(ctx context.Context, r Requester, id int) error {
	path := fmt.Sprintf("/agents/%d", id)
	return r.do(ctx, http.MethodDelete, r.accountPath(path), nil, nil)
}

// BulkCreateAgentsRequest represents a request to create multiple agents
type BulkCreateAgentsRequest struct {
	Emails []string `json:"emails"`
}

// BulkCreate creates multiple agents at once.
func (s AgentsService) BulkCreate(ctx context.Context, emails []string) ([]Agent, error) {
	return bulkCreateAgents(ctx, s, emails)
}

func bulkCreateAgents(ctx context.Context, r Requester, emails []string) ([]Agent, error) {
	body := BulkCreateAgentsRequest{Emails: emails}
	var result []Agent
	if err := r.do(ctx, http.MethodPost, r.accountPath("/agents/bulk_create"), body, &result); err != nil {
		return nil, err
	}
	return result, nil
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
