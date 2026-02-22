package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListTeams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/teams" {
			t.Errorf("Expected path /api/v1/accounts/1/teams, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"id": 1, "name": "Support Team", "description": "Customer support", "allow_auto_assign": true, "account_id": 1},
			{"id": 2, "name": "Sales Team", "description": "Sales team", "allow_auto_assign": false, "account_id": 1}
		]`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Teams().List(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 teams, got %d", len(result))
	}
	if result[0].Name != "Support Team" {
		t.Errorf("Expected name 'Support Team', got %s", result[0].Name)
	}
	if result[0].Description != "Customer support" {
		t.Errorf("Expected description 'Customer support', got %s", result[0].Description)
	}
	if !result[0].AllowAutoAssign {
		t.Error("Expected allow_auto_assign to be true")
	}
	if result[1].Name != "Sales Team" {
		t.Errorf("Expected name 'Sales Team', got %s", result[1].Name)
	}
}

func TestGetTeam(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/teams/1" {
			t.Errorf("Expected path /api/v1/accounts/1/teams/1, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": 1, "name": "Support Team", "description": "Customer support", "allow_auto_assign": true, "account_id": 1}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Teams().Get(context.Background(), 1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.ID != 1 {
		t.Errorf("Expected ID 1, got %d", result.ID)
	}
	if result.Name != "Support Team" {
		t.Errorf("Expected name 'Support Team', got %s", result.Name)
	}
	if result.Description != "Customer support" {
		t.Errorf("Expected description 'Customer support', got %s", result.Description)
	}
	if !result.AllowAutoAssign {
		t.Error("Expected allow_auto_assign to be true")
	}
}

func TestCreateTeam(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/teams" {
			t.Errorf("Expected path /api/v1/accounts/1/teams, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": 3, "name": "New Team", "description": "A new team", "allow_auto_assign": false, "account_id": 1}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Teams().Create(context.Background(), "New Team", "A new team")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.ID != 3 {
		t.Errorf("Expected ID 3, got %d", result.ID)
	}
	if result.Name != "New Team" {
		t.Errorf("Expected name 'New Team', got %s", result.Name)
	}
	if result.Description != "A new team" {
		t.Errorf("Expected description 'A new team', got %s", result.Description)
	}
}

func TestUpdateTeam(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/teams/1" {
			t.Errorf("Expected path /api/v1/accounts/1/teams/1, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": 1, "name": "Updated Team", "description": "Updated description", "allow_auto_assign": true, "account_id": 1}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Teams().Update(context.Background(), 1, "Updated Team", "Updated description")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.ID != 1 {
		t.Errorf("Expected ID 1, got %d", result.ID)
	}
	if result.Name != "Updated Team" {
		t.Errorf("Expected name 'Updated Team', got %s", result.Name)
	}
	if result.Description != "Updated description" {
		t.Errorf("Expected description 'Updated description', got %s", result.Description)
	}
}

func TestUpdateTeamPartial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": 1, "name": "Updated Name Only", "description": "Original description", "allow_auto_assign": false, "account_id": 1}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Teams().Update(context.Background(), 1, "Updated Name Only", "")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.Name != "Updated Name Only" {
		t.Errorf("Expected name 'Updated Name Only', got %s", result.Name)
	}
}

func TestDeleteTeam(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/teams/1" {
			t.Errorf("Expected path /api/v1/accounts/1/teams/1, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.Teams().Delete(context.Background(), 1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestListTeamMembers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/teams/1/team_members" {
			t.Errorf("Expected path /api/v1/accounts/1/teams/1/team_members, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"id": 1, "name": "Agent One", "email": "agent1@example.com", "role": "agent"},
			{"id": 2, "name": "Agent Two", "email": "agent2@example.com", "role": "agent"}
		]`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Teams().ListMembers(context.Background(), 1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 members, got %d", len(result))
	}
	if result[0].Name != "Agent One" {
		t.Errorf("Expected name 'Agent One', got %s", result[0].Name)
	}
	if result[0].Email != "agent1@example.com" {
		t.Errorf("Expected email 'agent1@example.com', got %s", result[0].Email)
	}
	if result[1].Name != "Agent Two" {
		t.Errorf("Expected name 'Agent Two', got %s", result[1].Name)
	}
}

func TestAddTeamMembers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/teams/1/team_members" {
			t.Errorf("Expected path /api/v1/accounts/1/teams/1/team_members, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.Teams().AddMembers(context.Background(), 1, []int{10, 20})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRemoveTeamMembers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/teams/1/team_members" {
			t.Errorf("Expected path /api/v1/accounts/1/teams/1/team_members, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.Teams().RemoveMembers(context.Background(), 1, []int{10, 20})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
