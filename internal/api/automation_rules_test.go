package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListAutomationRules(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, []AutomationRule)
	}{
		{
			name:       "successful list",
			statusCode: http.StatusOK,
			responseBody: `{
				"payload": [
					{"id": 1, "name": "Auto Assign", "event_name": "conversation_created", "active": true, "account_id": 1, "conditions": [], "actions": []},
					{"id": 2, "name": "Add Label", "event_name": "message_created", "active": false, "account_id": 1, "conditions": [], "actions": []}
				]
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, rules []AutomationRule) {
				if len(rules) != 2 {
					t.Errorf("Expected 2 rules, got %d", len(rules))
				}
				if rules[0].Name != "Auto Assign" {
					t.Errorf("Expected name 'Auto Assign', got %s", rules[0].Name)
				}
				if rules[0].EventName != "conversation_created" {
					t.Errorf("Expected event_name 'conversation_created', got %s", rules[0].EventName)
				}
				if !rules[0].Active {
					t.Error("Expected first rule to be active")
				}
			},
		},
		{
			name:         "empty list",
			statusCode:   http.StatusOK,
			responseBody: `{"payload": []}`,
			expectError:  false,
			validateFunc: func(t *testing.T, rules []AutomationRule) {
				if len(rules) != 0 {
					t.Errorf("Expected 0 rules, got %d", len(rules))
				}
			},
		},
		{
			name:         "server error",
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": "internal error"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.AutomationRules().List(context.Background())

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.validateFunc != nil && result != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestGetAutomationRule(t *testing.T) {
	tests := []struct {
		name         string
		ruleID       int
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *AutomationRule)
	}{
		{
			name:       "successful get",
			ruleID:     1,
			statusCode: http.StatusOK,
			responseBody: `{
				"payload": {
					"id": 1,
					"name": "Auto Assign",
					"description": "Assigns conversations automatically",
					"event_name": "conversation_created",
					"active": true,
					"account_id": 1,
					"conditions": [{"attribute_key": "status", "filter_operator": "equal_to", "values": ["open"]}],
					"actions": [{"action_name": "assign_agent", "action_params": [1]}]
				}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, rule *AutomationRule) {
				if rule.ID != 1 {
					t.Errorf("Expected ID 1, got %d", rule.ID)
				}
				if rule.Description != "Assigns conversations automatically" {
					t.Errorf("Expected description, got %s", rule.Description)
				}
				if len(rule.Conditions) != 1 {
					t.Errorf("Expected 1 condition, got %d", len(rule.Conditions))
				}
				if len(rule.Actions) != 1 {
					t.Errorf("Expected 1 action, got %d", len(rule.Actions))
				}
			},
		},
		{
			name:         "not found",
			ruleID:       999,
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "not found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.AutomationRules().Get(context.Background(), tt.ruleID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.validateFunc != nil && result != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestCreateAutomationRule(t *testing.T) {
	tests := []struct {
		name         string
		ruleName     string
		eventName    string
		conditions   []map[string]any
		actions      []map[string]any
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *AutomationRule, map[string]any)
	}{
		{
			name:       "successful create",
			ruleName:   "New Rule",
			eventName:  "conversation_created",
			conditions: []map[string]any{{"attribute_key": "status", "filter_operator": "equal_to", "values": []string{"open"}}},
			actions:    []map[string]any{{"action_name": "add_label", "action_params": []string{"urgent"}}},
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 1,
				"name": "New Rule",
				"event_name": "conversation_created",
				"active": true,
				"account_id": 1,
				"conditions": [{"attribute_key": "status", "filter_operator": "equal_to", "values": ["open"]}],
				"actions": [{"action_name": "add_label", "action_params": ["urgent"]}]
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, rule *AutomationRule, body map[string]any) {
				if rule.Name != "New Rule" {
					t.Errorf("Expected name 'New Rule', got %s", rule.Name)
				}
				if body["name"] != "New Rule" {
					t.Errorf("Expected name in body, got %v", body["name"])
				}
				if body["event_name"] != "conversation_created" {
					t.Errorf("Expected event_name in body, got %v", body["event_name"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				_ = json.NewDecoder(r.Body).Decode(&capturedBody)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.AutomationRules().Create(context.Background(), tt.ruleName, tt.eventName, tt.conditions, tt.actions)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.validateFunc != nil && result != nil {
				tt.validateFunc(t, result, capturedBody)
			}
		})
	}
}

func TestUpdateAutomationRule(t *testing.T) {
	tests := []struct {
		name         string
		ruleID       int
		ruleName     string
		conditions   []map[string]any
		actions      []map[string]any
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *AutomationRule, map[string]any)
	}{
		{
			name:       "update all fields",
			ruleID:     1,
			ruleName:   "Updated Rule",
			conditions: []map[string]any{{"attribute_key": "priority", "filter_operator": "equal_to", "values": []string{"high"}}},
			actions:    []map[string]any{{"action_name": "send_email", "action_params": []string{"admin@example.com"}}},
			statusCode: http.StatusOK,
			responseBody: `{
				"payload": {
					"id": 1,
					"name": "Updated Rule",
					"event_name": "conversation_created",
					"active": true,
					"account_id": 1,
					"conditions": [{"attribute_key": "priority"}],
					"actions": [{"action_name": "send_email"}]
				}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, rule *AutomationRule, body map[string]any) {
				if rule.Name != "Updated Rule" {
					t.Errorf("Expected name 'Updated Rule', got %s", rule.Name)
				}
			},
		},
		{
			name:         "partial update - name only",
			ruleID:       1,
			ruleName:     "Only Name Update",
			conditions:   nil,
			actions:      nil,
			statusCode:   http.StatusOK,
			responseBody: `{"payload": {"id": 1, "name": "Only Name Update", "event_name": "conversation_created", "active": true, "account_id": 1, "conditions": [], "actions": []}}`,
			expectError:  false,
			validateFunc: func(t *testing.T, rule *AutomationRule, body map[string]any) {
				if _, ok := body["conditions"]; ok {
					t.Error("Expected no conditions in body when nil")
				}
				if _, ok := body["actions"]; ok {
					t.Error("Expected no actions in body when nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPatch {
					t.Errorf("Expected PATCH, got %s", r.Method)
				}
				_ = json.NewDecoder(r.Body).Decode(&capturedBody)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.AutomationRules().Update(context.Background(), tt.ruleID, tt.ruleName, tt.conditions, tt.actions, nil)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.validateFunc != nil && result != nil {
				tt.validateFunc(t, result, capturedBody)
			}
		})
	}
}

func TestDeleteAutomationRule(t *testing.T) {
	tests := []struct {
		name        string
		ruleID      int
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful delete",
			ruleID:      1,
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "not found",
			ruleID:      999,
			statusCode:  http.StatusNotFound,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("Expected DELETE, got %s", r.Method)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			err := client.AutomationRules().Delete(context.Background(), tt.ruleID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestCloneAutomationRule(t *testing.T) {
	tests := []struct {
		name         string
		ruleID       int
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *AutomationRule)
	}{
		{
			name:       "successful clone",
			ruleID:     1,
			statusCode: http.StatusOK,
			responseBody: `{
				"payload": {
					"id": 2,
					"name": "Auto Assign (Copy)",
					"event_name": "conversation_created",
					"active": false,
					"account_id": 1,
					"conditions": [{"attribute_key": "status"}],
					"actions": [{"action_name": "assign_agent"}]
				}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, rule *AutomationRule) {
				if rule.ID != 2 {
					t.Errorf("Expected ID 2, got %d", rule.ID)
				}
				if rule.Name != "Auto Assign (Copy)" {
					t.Errorf("Expected name with (Copy), got %s", rule.Name)
				}
				if rule.Active {
					t.Error("Expected cloned rule to be inactive")
				}
			},
		},
		{
			name:         "not found",
			ruleID:       999,
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "not found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				expectedPath := "/api/v1/accounts/1/automation_rules/1/clone"
				if tt.ruleID == 999 {
					expectedPath = "/api/v1/accounts/1/automation_rules/999/clone"
				}
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.AutomationRules().Clone(context.Background(), tt.ruleID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.validateFunc != nil && result != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}
