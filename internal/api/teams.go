package api

import (
	"context"
	"fmt"
	"net/http"
)

// List retrieves all teams for the account.
func (s TeamsService) List(ctx context.Context) ([]Team, error) {
	return listTeams(ctx, s)
}

func listTeams(ctx context.Context, r Requester) ([]Team, error) {
	var teams []Team
	err := r.do(ctx, http.MethodGet, r.accountPath("/teams"), nil, &teams)
	return teams, err
}

// Get retrieves a specific team by ID.
func (s TeamsService) Get(ctx context.Context, id int) (*Team, error) {
	return getTeam(ctx, s, id)
}

func getTeam(ctx context.Context, r Requester, id int) (*Team, error) {
	var team Team
	err := r.do(ctx, http.MethodGet, r.accountPath(fmt.Sprintf("/teams/%d", id)), nil, &team)
	if err != nil {
		return nil, err
	}
	return &team, nil
}

// Create creates a new team.
func (s TeamsService) Create(ctx context.Context, name, description string) (*Team, error) {
	return createTeam(ctx, s, name, description)
}

func createTeam(ctx context.Context, r Requester, name, description string) (*Team, error) {
	body := map[string]any{
		"name":        name,
		"description": description,
	}
	var team Team
	err := r.do(ctx, http.MethodPost, r.accountPath("/teams"), body, &team)
	if err != nil {
		return nil, err
	}
	return &team, nil
}

// Update updates an existing team.
func (s TeamsService) Update(ctx context.Context, id int, name, description string) (*Team, error) {
	return updateTeam(ctx, s, id, name, description)
}

func updateTeam(ctx context.Context, r Requester, id int, name, description string) (*Team, error) {
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if description != "" {
		body["description"] = description
	}

	var team Team
	err := r.do(ctx, http.MethodPatch, r.accountPath(fmt.Sprintf("/teams/%d", id)), body, &team)
	if err != nil {
		return nil, err
	}
	return &team, nil
}

// Delete deletes a team by ID.
func (s TeamsService) Delete(ctx context.Context, id int) error {
	return deleteTeam(ctx, s, id)
}

func deleteTeam(ctx context.Context, r Requester, id int) error {
	return r.do(ctx, http.MethodDelete, r.accountPath(fmt.Sprintf("/teams/%d", id)), nil, nil)
}

// ListMembers retrieves all members of a team.
func (s TeamsService) ListMembers(ctx context.Context, teamID int) ([]Agent, error) {
	return listTeamMembers(ctx, s, teamID)
}

func listTeamMembers(ctx context.Context, r Requester, teamID int) ([]Agent, error) {
	var agents []Agent
	err := r.do(ctx, http.MethodGet, r.accountPath(fmt.Sprintf("/teams/%d/team_members", teamID)), nil, &agents)
	return agents, err
}

// AddMembers adds users to a team.
func (s TeamsService) AddMembers(ctx context.Context, teamID int, userIDs []int) error {
	return addTeamMembers(ctx, s, teamID, userIDs)
}

func addTeamMembers(ctx context.Context, r Requester, teamID int, userIDs []int) error {
	body := map[string]any{
		"user_ids": userIDs,
	}
	return r.do(ctx, http.MethodPost, r.accountPath(fmt.Sprintf("/teams/%d/team_members", teamID)), body, nil)
}

// RemoveMembers removes users from a team.
func (s TeamsService) RemoveMembers(ctx context.Context, teamID int, userIDs []int) error {
	return removeTeamMembers(ctx, s, teamID, userIDs)
}

func removeTeamMembers(ctx context.Context, r Requester, teamID int, userIDs []int) error {
	body := map[string]any{
		"user_ids": userIDs,
	}
	return r.do(ctx, http.MethodDelete, r.accountPath(fmt.Sprintf("/teams/%d/team_members", teamID)), body, nil)
}
