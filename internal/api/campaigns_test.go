package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestListCampaigns(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, []Campaign)
	}{
		{
			name: "successful list with results",
			page: 1,
			responseBody: `[
				{
					"id": 1,
					"title": "Summer Promo",
					"description": "50% off sale",
					"message": "Get 50% off today!",
					"enabled": true,
					"campaign_type": "one_off",
					"campaign_status": "completed",
					"inbox_id": 5,
					"sender_id": 10,
					"scheduled_at": 1700000000,
					"trigger_only_during_business_hours": false,
					"audience": [{"type": "Label", "id": 1}],
					"created_at": 1700000000,
					"account_id": 1
				},
				{
					"id": 2,
					"title": "Ongoing Campaign",
					"message": "Welcome message",
					"enabled": false,
					"campaign_type": "ongoing",
					"inbox_id": 3,
					"trigger_only_during_business_hours": true,
					"created_at": 1700001000,
					"account_id": 1
				}
			]`,
			expectError: false,
			validateFunc: func(t *testing.T, campaigns []Campaign) {
				if len(campaigns) != 2 {
					t.Errorf("Expected 2 campaigns, got %d", len(campaigns))
				}
				if campaigns[0].ID != 1 {
					t.Errorf("Expected first campaign ID 1, got %d", campaigns[0].ID)
				}
				if campaigns[0].Title != "Summer Promo" {
					t.Errorf("Expected first campaign title 'Summer Promo', got %s", campaigns[0].Title)
				}
				if campaigns[0].CampaignType != "one_off" {
					t.Errorf("Expected campaign_type 'one_off', got %s", campaigns[0].CampaignType)
				}
				if !campaigns[0].Enabled {
					t.Error("Expected first campaign to be enabled")
				}
				if len(campaigns[0].Audience) != 1 {
					t.Errorf("Expected 1 audience item, got %d", len(campaigns[0].Audience))
				}
				if campaigns[1].TriggerOnlyDuringBusinessHours != true {
					t.Error("Expected second campaign to have business hours enabled")
				}
			},
		},
		{
			name:         "empty results",
			page:         1,
			responseBody: `[]`,
			expectError:  false,
			validateFunc: func(t *testing.T, campaigns []Campaign) {
				if len(campaigns) != 0 {
					t.Errorf("Expected 0 campaigns, got %d", len(campaigns))
				}
			},
		},
		{
			name: "page 2 with results",
			page: 2,
			responseBody: `[
				{
					"id": 3,
					"title": "Third Campaign",
					"message": "Test",
					"enabled": true,
					"campaign_type": "one_off",
					"inbox_id": 1,
					"trigger_only_during_business_hours": false,
					"created_at": 1700002000,
					"account_id": 1
				}
			]`,
			expectError: false,
			validateFunc: func(t *testing.T, campaigns []Campaign) {
				if len(campaigns) != 1 {
					t.Errorf("Expected 1 campaign, got %d", len(campaigns))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET request, got %s", r.Method)
				}

				if !strings.Contains(r.URL.Path, "/campaigns") {
					t.Errorf("Expected path to contain /campaigns, got %s", r.URL.Path)
				}

				if tt.page > 0 {
					query := r.URL.Query()
					expectedPage := strconv.Itoa(tt.page)
					if actualPage := query.Get("page"); actualPage != expectedPage {
						t.Errorf("Expected page=%s, got page=%s", expectedPage, actualPage)
					}
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Campaigns().List(context.Background(), tt.page)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.validateFunc != nil && result != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestGetCampaign(t *testing.T) {
	tests := []struct {
		name         string
		campaignID   int
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *Campaign)
	}{
		{
			name:       "successful get",
			campaignID: 123,
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 123,
				"title": "Test Campaign",
				"description": "Test description",
				"message": "Hello world",
				"enabled": true,
				"campaign_type": "one_off",
				"campaign_status": "active",
				"inbox_id": 5,
				"sender_id": 10,
				"scheduled_at": 1700000000,
				"trigger_only_during_business_hours": true,
				"audience": [
					{"type": "Label", "id": 1},
					{"type": "Label", "id": 2}
				],
				"trigger_rules": {"url": "example.com"},
				"created_at": 1700000000,
				"updated_at": 1700001000,
				"account_id": 1
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, campaign *Campaign) {
				if campaign.ID != 123 {
					t.Errorf("Expected ID 123, got %d", campaign.ID)
				}
				if campaign.Title != "Test Campaign" {
					t.Errorf("Expected title 'Test Campaign', got %s", campaign.Title)
				}
				if campaign.Description != "Test description" {
					t.Errorf("Expected description 'Test description', got %s", campaign.Description)
				}
				if campaign.Message != "Hello world" {
					t.Errorf("Expected message 'Hello world', got %s", campaign.Message)
				}
				if !campaign.Enabled {
					t.Error("Expected campaign to be enabled")
				}
				if campaign.CampaignType != "one_off" {
					t.Errorf("Expected campaign_type 'one_off', got %s", campaign.CampaignType)
				}
				if campaign.InboxID != 5 {
					t.Errorf("Expected inbox_id 5, got %d", campaign.InboxID)
				}
				if campaign.SenderID != 10 {
					t.Errorf("Expected sender_id 10, got %d", campaign.SenderID)
				}
				if !campaign.TriggerOnlyDuringBusinessHours {
					t.Error("Expected trigger_only_during_business_hours to be true")
				}
				if len(campaign.Audience) != 2 {
					t.Errorf("Expected 2 audience items, got %d", len(campaign.Audience))
				}
				if campaign.Audience[0].Type != "Label" || campaign.Audience[0].ID != 1 {
					t.Errorf("Expected first audience {Type: Label, ID: 1}, got %+v", campaign.Audience[0])
				}
			},
		},
		{
			name:         "campaign not found",
			campaignID:   999,
			statusCode:   http.StatusNotFound,
			responseBody: `{"error":"Not found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Campaigns().Get(context.Background(), tt.campaignID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.validateFunc != nil && result != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestCreateCampaign(t *testing.T) {
	tests := []struct {
		name         string
		request      CreateCampaignRequest
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *Campaign)
	}{
		{
			name: "successful create with all fields",
			request: CreateCampaignRequest{
				Title:                          "New Campaign",
				Description:                    "Test description",
				Message:                        "Hello",
				Enabled:                        true,
				InboxID:                        5,
				SenderID:                       10,
				ScheduledAt:                    1700000000,
				TriggerOnlyDuringBusinessHours: true,
				Audience: []CampaignAudience{
					{Type: "Label", ID: 1},
					{Type: "Label", ID: 2},
				},
				TriggerRules: map[string]any{"url": "example.com"},
			},
			responseBody: `{
				"id": 456,
				"title": "New Campaign",
				"description": "Test description",
				"message": "Hello",
				"enabled": true,
				"campaign_type": "one_off",
				"inbox_id": 5,
				"sender_id": 10,
				"scheduled_at": 1700000000,
				"trigger_only_during_business_hours": true,
				"audience": [
					{"type": "Label", "id": 1},
					{"type": "Label", "id": 2}
				],
				"trigger_rules": {"url": "example.com"},
				"created_at": 1700000000,
				"account_id": 1
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, campaign *Campaign) {
				if campaign.ID != 456 {
					t.Errorf("Expected ID 456, got %d", campaign.ID)
				}
				if campaign.Title != "New Campaign" {
					t.Errorf("Expected title 'New Campaign', got %s", campaign.Title)
				}
				if len(campaign.Audience) != 2 {
					t.Errorf("Expected 2 audience items, got %d", len(campaign.Audience))
				}
			},
		},
		{
			name: "minimal create request",
			request: CreateCampaignRequest{
				Title:   "Minimal Campaign",
				Message: "Test",
				InboxID: 1,
			},
			responseBody: `{
				"id": 789,
				"title": "Minimal Campaign",
				"message": "Test",
				"enabled": false,
				"campaign_type": "one_off",
				"inbox_id": 1,
				"trigger_only_during_business_hours": false,
				"created_at": 1700000000,
				"account_id": 1
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, campaign *Campaign) {
				if campaign.ID != 789 {
					t.Errorf("Expected ID 789, got %d", campaign.ID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}

				var payload CreateCampaignRequest
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Errorf("Failed to decode request body: %v", err)
				}

				// Validate request payload matches
				if payload.Title != tt.request.Title {
					t.Errorf("Expected title %s, got %s", tt.request.Title, payload.Title)
				}
				if payload.Message != tt.request.Message {
					t.Errorf("Expected message %s, got %s", tt.request.Message, payload.Message)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Campaigns().Create(context.Background(), tt.request)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.validateFunc != nil && result != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestUpdateCampaign(t *testing.T) {
	enabledTrue := true
	enabledFalse := false
	businessHoursTrue := true

	tests := []struct {
		name         string
		campaignID   int
		request      UpdateCampaignRequest
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *Campaign, *http.Request)
	}{
		{
			name:       "update with pointer bools",
			campaignID: 123,
			request: UpdateCampaignRequest{
				Title:                          "Updated Title",
				Enabled:                        &enabledTrue,
				TriggerOnlyDuringBusinessHours: &businessHoursTrue,
			},
			responseBody: `{
				"id": 123,
				"title": "Updated Title",
				"message": "Original message",
				"enabled": true,
				"campaign_type": "one_off",
				"inbox_id": 5,
				"trigger_only_during_business_hours": true,
				"created_at": 1700000000,
				"account_id": 1
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, campaign *Campaign, r *http.Request) {
				if campaign.Enabled != true {
					t.Error("Expected enabled to be true")
				}
				if campaign.TriggerOnlyDuringBusinessHours != true {
					t.Error("Expected trigger_only_during_business_hours to be true")
				}
			},
		},
		{
			name:       "update with enabled=false",
			campaignID: 123,
			request: UpdateCampaignRequest{
				Enabled: &enabledFalse,
			},
			responseBody: `{
				"id": 123,
				"title": "Test",
				"message": "Test",
				"enabled": false,
				"campaign_type": "one_off",
				"inbox_id": 5,
				"trigger_only_during_business_hours": false,
				"created_at": 1700000000,
				"account_id": 1
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, campaign *Campaign, r *http.Request) {
				if campaign.Enabled != false {
					t.Error("Expected enabled to be false")
				}
			},
		},
		{
			name:       "update audience",
			campaignID: 123,
			request: UpdateCampaignRequest{
				Audience: []CampaignAudience{
					{Type: "Label", ID: 5},
					{Type: "Label", ID: 6},
					{Type: "Label", ID: 7},
				},
			},
			responseBody: `{
				"id": 123,
				"title": "Test",
				"message": "Test",
				"enabled": true,
				"campaign_type": "one_off",
				"inbox_id": 5,
				"trigger_only_during_business_hours": false,
				"audience": [
					{"type": "Label", "id": 5},
					{"type": "Label", "id": 6},
					{"type": "Label", "id": 7}
				],
				"created_at": 1700000000,
				"account_id": 1
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, campaign *Campaign, r *http.Request) {
				if len(campaign.Audience) != 3 {
					t.Errorf("Expected 3 audience items, got %d", len(campaign.Audience))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedRequest *http.Request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPatch {
					t.Errorf("Expected PATCH request, got %s", r.Method)
				}

				// Store request for validation
				capturedRequest = r

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Campaigns().Update(context.Background(), tt.campaignID, tt.request)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.validateFunc != nil && result != nil && capturedRequest != nil {
				tt.validateFunc(t, result, capturedRequest)
			}
		})
	}
}

func TestDeleteCampaign(t *testing.T) {
	tests := []struct {
		name        string
		campaignID  int
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful delete",
			campaignID:  123,
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "campaign not found",
			campaignID:  999,
			statusCode:  http.StatusNotFound,
			expectError: true,
		},
		{
			name:        "unauthorized",
			campaignID:  123,
			statusCode:  http.StatusUnauthorized,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("Expected DELETE request, got %s", r.Method)
				}

				if !strings.Contains(r.URL.Path, "/campaigns/") {
					t.Errorf("Expected path to contain /campaigns/, got %s", r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			err := client.Campaigns().Delete(context.Background(), tt.campaignID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.expectError && err != nil {
				apiErr, ok := err.(*APIError)
				if !ok {
					t.Errorf("Expected APIError, got %T", err)
				} else if apiErr.StatusCode != tt.statusCode {
					t.Errorf("Expected status code %d, got %d", tt.statusCode, apiErr.StatusCode)
				}
			}
		})
	}
}

func TestCreateCampaignRequestJSONMarshaling(t *testing.T) {
	tests := []struct {
		name     string
		request  CreateCampaignRequest
		expected map[string]any
	}{
		{
			name: "all fields populated",
			request: CreateCampaignRequest{
				Title:                          "Test Campaign",
				Description:                    "Test description",
				Message:                        "Hello world",
				Enabled:                        true,
				InboxID:                        5,
				SenderID:                       10,
				ScheduledAt:                    1700000000,
				TriggerOnlyDuringBusinessHours: true,
				Audience: []CampaignAudience{
					{Type: "Label", ID: 1},
				},
				TriggerRules: map[string]any{"url": "example.com"},
			},
			expected: map[string]any{
				"title":                              "Test Campaign",
				"description":                        "Test description",
				"message":                            "Hello world",
				"enabled":                            true,
				"inbox_id":                           float64(5),
				"sender_id":                          float64(10),
				"scheduled_at":                       float64(1700000000),
				"trigger_only_during_business_hours": true,
			},
		},
		{
			name: "minimal fields",
			request: CreateCampaignRequest{
				Title:   "Minimal",
				Message: "Test",
				InboxID: 1,
			},
			expected: map[string]any{
				"title":                              "Minimal",
				"message":                            "Test",
				"enabled":                            false,
				"inbox_id":                           float64(1),
				"trigger_only_during_business_hours": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			var result map[string]any
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			for key, expectedValue := range tt.expected {
				actualValue, exists := result[key]
				if !exists {
					t.Errorf("Expected key %s to exist in JSON", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("Expected %s to be %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestUpdateCampaignRequestPointerBools(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name         string
		request      UpdateCampaignRequest
		expectFields []string
		expectValues map[string]any
	}{
		{
			name: "enabled=true sets field",
			request: UpdateCampaignRequest{
				Enabled: &trueVal,
			},
			expectFields: []string{"enabled"},
			expectValues: map[string]any{"enabled": true},
		},
		{
			name: "enabled=false sets field",
			request: UpdateCampaignRequest{
				Enabled: &falseVal,
			},
			expectFields: []string{"enabled"},
			expectValues: map[string]any{"enabled": false},
		},
		{
			name: "business_hours=true sets field",
			request: UpdateCampaignRequest{
				TriggerOnlyDuringBusinessHours: &trueVal,
			},
			expectFields: []string{"trigger_only_during_business_hours"},
			expectValues: map[string]any{"trigger_only_during_business_hours": true},
		},
		{
			name:         "nil pointers omit fields",
			request:      UpdateCampaignRequest{},
			expectFields: []string{},
			expectValues: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			var result map[string]any
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			for _, field := range tt.expectFields {
				if _, exists := result[field]; !exists {
					t.Errorf("Expected field %s to exist in JSON", field)
				}
			}

			for key, expectedValue := range tt.expectValues {
				actualValue, exists := result[key]
				if !exists {
					t.Errorf("Expected key %s to exist", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("Expected %s to be %v, got %v", key, expectedValue, actualValue)
				}
			}

			// Verify nil pointer fields are omitted
			if tt.request.Enabled == nil {
				if _, exists := result["enabled"]; exists {
					t.Error("Expected enabled to be omitted when nil")
				}
			}
			if tt.request.TriggerOnlyDuringBusinessHours == nil {
				if _, exists := result["trigger_only_during_business_hours"]; exists {
					t.Error("Expected trigger_only_during_business_hours to be omitted when nil")
				}
			}
		})
	}
}
