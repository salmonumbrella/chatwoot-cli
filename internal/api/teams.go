package api

import (
	"context"
	"fmt"
)

// ListTeams retrieves all teams for the account
func (c *Client) ListTeams(ctx context.Context) ([]Team, error) {
	var teams []Team
	err := c.Get(ctx, "/teams", &teams)
	return teams, err
}

// GetTeam retrieves a specific team by ID
func (c *Client) GetTeam(ctx context.Context, id int) (*Team, error) {
	var team Team
	err := c.Get(ctx, fmt.Sprintf("/teams/%d", id), &team)
	if err != nil {
		return nil, err
	}
	return &team, nil
}

// CreateTeam creates a new team
func (c *Client) CreateTeam(ctx context.Context, name, description string) (*Team, error) {
	body := map[string]any{
		"name":        name,
		"description": description,
	}
	var team Team
	err := c.Post(ctx, "/teams", body, &team)
	if err != nil {
		return nil, err
	}
	return &team, nil
}

// UpdateTeam updates an existing team
func (c *Client) UpdateTeam(ctx context.Context, id int, name, description string) (*Team, error) {
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if description != "" {
		body["description"] = description
	}

	var team Team
	err := c.Patch(ctx, fmt.Sprintf("/teams/%d", id), body, &team)
	if err != nil {
		return nil, err
	}
	return &team, nil
}

// DeleteTeam deletes a team by ID
func (c *Client) DeleteTeam(ctx context.Context, id int) error {
	return c.Delete(ctx, fmt.Sprintf("/teams/%d", id))
}

// ListTeamMembers retrieves all members of a team
func (c *Client) ListTeamMembers(ctx context.Context, teamID int) ([]Agent, error) {
	var agents []Agent
	err := c.Get(ctx, fmt.Sprintf("/teams/%d/team_members", teamID), &agents)
	return agents, err
}

// AddTeamMembers adds users to a team
func (c *Client) AddTeamMembers(ctx context.Context, teamID int, userIDs []int) error {
	body := map[string]any{
		"user_ids": userIDs,
	}
	return c.Post(ctx, fmt.Sprintf("/teams/%d/team_members", teamID), body, nil)
}

// RemoveTeamMembers removes users from a team
func (c *Client) RemoveTeamMembers(ctx context.Context, teamID int, userIDs []int) error {
	body := map[string]any{
		"user_ids": userIDs,
	}
	return c.DeleteWithBody(ctx, fmt.Sprintf("/teams/%d/team_members", teamID), body)
}
