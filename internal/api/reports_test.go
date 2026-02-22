package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestV2ReportPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "reports summary path",
			path:     "/reports/summary",
			expected: "https://example.com/api/v2/accounts/123/reports/summary",
		},
		{
			name:     "reports base path",
			path:     "/reports",
			expected: "https://example.com/api/v2/accounts/123/reports",
		},
		{
			name:     "reports conversations path",
			path:     "/reports/conversations",
			expected: "https://example.com/api/v2/accounts/123/reports/conversations",
		},
		{
			name:     "empty path",
			path:     "",
			expected: "https://example.com/api/v2/accounts/123",
		},
	}

	client := newTestClient("https://example.com", "token", 123)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.v2ReportPath(tt.path)
			if result != tt.expected {
				t.Errorf("v2ReportPath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGetReportSummary(t *testing.T) {
	tests := []struct {
		name                  string
		reportType            string
		since                 string
		until                 string
		id                    string
		statusCode            int
		responseBody          string
		expectError           bool
		expectConversations   int
		expectPreviousPresent bool
		validateURL           func(*testing.T, string)
	}{
		{
			name:       "account summary",
			reportType: "account",
			since:      "1609459200",
			until:      "1609545600",
			id:         "",
			statusCode: http.StatusOK,
			responseBody: `{
				"avg_first_response_time": "2.5",
				"avg_resolution_time": "24.0",
				"conversations_count": 150,
				"incoming_messages_count": 500,
				"outgoing_messages_count": 450,
				"resolutions_count": 120,
				"previous": {
					"conversations_count": 140,
					"resolutions_count": 110
				}
			}`,
			expectError:           false,
			expectConversations:   150,
			expectPreviousPresent: true,
			validateURL: func(t *testing.T, url string) {
				if !strings.Contains(url, "type=account") {
					t.Error("URL should contain type=account")
				}
				if !strings.Contains(url, "since=1609459200") {
					t.Error("URL should contain since param")
				}
				if !strings.Contains(url, "until=1609545600") {
					t.Error("URL should contain until param")
				}
				if strings.Contains(url, "id=") {
					t.Error("URL should not contain id param for account type")
				}
			},
		},
		{
			name:       "agent summary with id",
			reportType: "agent",
			since:      "1609459200",
			until:      "1609545600",
			id:         "42",
			statusCode: http.StatusOK,
			responseBody: `{
				"avg_first_response_time": "1.8",
				"avg_resolution_time": "12.0",
				"conversations_count": 25,
				"incoming_messages_count": 100,
				"outgoing_messages_count": 95,
				"resolutions_count": 20
			}`,
			expectError:           false,
			expectConversations:   25,
			expectPreviousPresent: false,
			validateURL: func(t *testing.T, url string) {
				if !strings.Contains(url, "type=agent") {
					t.Error("URL should contain type=agent")
				}
				if !strings.Contains(url, "id=42") {
					t.Error("URL should contain id=42 for agent type")
				}
			},
		},
		{
			name:         "unauthorized error",
			reportType:   "account",
			since:        "1609459200",
			until:        "1609545600",
			id:           "",
			statusCode:   http.StatusUnauthorized,
			responseBody: `{"error": "Unauthorized"}`,
			expectError:  true,
		},
		{
			name:       "inbox summary with id",
			reportType: "inbox",
			since:      "1609459200",
			until:      "1609545600",
			id:         "5",
			statusCode: http.StatusOK,
			responseBody: `{
				"conversations_count": 50,
				"resolutions_count": 45
			}`,
			expectError:         false,
			expectConversations: 50,
			validateURL: func(t *testing.T, url string) {
				if !strings.Contains(url, "type=inbox") {
					t.Error("URL should contain type=inbox")
				}
				if !strings.Contains(url, "id=5") {
					t.Error("URL should contain id=5")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedURL string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedURL = r.URL.String()
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				if r.Header.Get("api_access_token") != "test-token" {
					t.Error("Missing or wrong api_access_token header")
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Reports().Summary(context.Background(), tt.reportType, tt.since, tt.until, tt.id)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				if result == nil {
					t.Fatal("Expected result, got nil")
				}
				if result.ConversationsCount != tt.expectConversations {
					t.Errorf("Expected conversations_count %d, got %d", tt.expectConversations, result.ConversationsCount)
				}
				if tt.expectPreviousPresent && result.Previous == nil {
					t.Error("Expected previous to be present")
				}
				if !tt.expectPreviousPresent && result.Previous != nil {
					t.Error("Expected previous to be nil")
				}
			}

			if tt.validateURL != nil {
				tt.validateURL(t, capturedURL)
			}
		})
	}
}

func TestGetReportTimeSeries(t *testing.T) {
	tests := []struct {
		name            string
		metric          string
		reportType      string
		since           string
		until           string
		id              string
		statusCode      int
		responseBody    string
		expectError     bool
		expectCount     int
		expectedFirstTS int64
		expectedFirstV  string
	}{
		{
			name:       "successful retrieval with data points",
			metric:     "conversations_count",
			reportType: "account",
			since:      "1609459200",
			until:      "1609545600",
			id:         "",
			statusCode: http.StatusOK,
			responseBody: `[
				{"value": 10, "timestamp": 1609459200},
				{"value": "15", "timestamp": 1609462800},
				{"value": 20.5, "timestamp": 1609466400}
			]`,
			expectError:     false,
			expectCount:     3,
			expectedFirstTS: 1609459200,
			expectedFirstV:  "10",
		},
		{
			name:       "FlexString handles string value",
			metric:     "avg_first_response_time",
			reportType: "agent",
			since:      "1609459200",
			until:      "1609545600",
			id:         "5",
			statusCode: http.StatusOK,
			responseBody: `[
				{"value": "2.5", "timestamp": 1609459200}
			]`,
			expectError:     false,
			expectCount:     1,
			expectedFirstTS: 1609459200,
			expectedFirstV:  "2.5",
		},
		{
			name:       "FlexString handles numeric value",
			metric:     "resolutions_count",
			reportType: "account",
			since:      "1609459200",
			until:      "1609545600",
			id:         "",
			statusCode: http.StatusOK,
			responseBody: `[
				{"value": 42, "timestamp": 1609459200}
			]`,
			expectError:     false,
			expectCount:     1,
			expectedFirstTS: 1609459200,
			expectedFirstV:  "42",
		},
		{
			name:         "empty time series",
			metric:       "conversations_count",
			reportType:   "inbox",
			since:        "1609459200",
			until:        "1609545600",
			id:           "999",
			statusCode:   http.StatusOK,
			responseBody: `[]`,
			expectError:  false,
			expectCount:  0,
		},
		{
			name:         "server error",
			metric:       "conversations_count",
			reportType:   "account",
			since:        "1609459200",
			until:        "1609545600",
			id:           "",
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": "Internal server error"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}

				// Verify URL params
				query := r.URL.Query()
				if query.Get("metric") != tt.metric {
					t.Errorf("Expected metric=%s, got %s", tt.metric, query.Get("metric"))
				}
				if query.Get("type") != tt.reportType {
					t.Errorf("Expected type=%s, got %s", tt.reportType, query.Get("type"))
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Reports().TimeSeries(context.Background(), tt.metric, tt.reportType, tt.since, tt.until, tt.id)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				if len(result) != tt.expectCount {
					t.Errorf("Expected %d data points, got %d", tt.expectCount, len(result))
				}
				if tt.expectCount > 0 {
					if result[0].Timestamp != tt.expectedFirstTS {
						t.Errorf("Expected first timestamp %d, got %d", tt.expectedFirstTS, result[0].Timestamp)
					}
					if result[0].Value.String() != tt.expectedFirstV {
						t.Errorf("Expected first value %q, got %q", tt.expectedFirstV, result[0].Value.String())
					}
				}
			}
		})
	}
}

func TestGetConversationMetrics(t *testing.T) {
	tests := []struct {
		name             string
		statusCode       int
		responseBody     string
		expectError      bool
		expectOpen       int
		expectUnattended int
		expectUnassigned int
	}{
		{
			name:       "successful metrics",
			statusCode: http.StatusOK,
			responseBody: `{
				"open": 25,
				"unattended": 10,
				"unassigned": 5
			}`,
			expectError:      false,
			expectOpen:       25,
			expectUnattended: 10,
			expectUnassigned: 5,
		},
		{
			name:       "zero metrics",
			statusCode: http.StatusOK,
			responseBody: `{
				"open": 0,
				"unattended": 0,
				"unassigned": 0
			}`,
			expectError:      false,
			expectOpen:       0,
			expectUnattended: 0,
			expectUnassigned: 0,
		},
		{
			name:         "unauthorized error",
			statusCode:   http.StatusUnauthorized,
			responseBody: `{"error": "Unauthorized"}`,
			expectError:  true,
		},
		{
			name:         "forbidden error",
			statusCode:   http.StatusForbidden,
			responseBody: `{"error": "Access denied"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}

				// Verify type=account is in URL
				query := r.URL.Query()
				if query.Get("type") != "account" {
					t.Errorf("Expected type=account, got %s", query.Get("type"))
				}

				// Verify path contains /reports/conversations
				if !strings.Contains(r.URL.Path, "/reports/conversations") {
					t.Errorf("Expected path to contain /reports/conversations, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Reports().ConversationMetrics(context.Background())

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				if result == nil {
					t.Fatal("Expected result, got nil")
				}
				if result.Open != tt.expectOpen {
					t.Errorf("Expected open=%d, got %d", tt.expectOpen, result.Open)
				}
				if result.Unattended != tt.expectUnattended {
					t.Errorf("Expected unattended=%d, got %d", tt.expectUnattended, result.Unattended)
				}
				if result.Unassigned != tt.expectUnassigned {
					t.Errorf("Expected unassigned=%d, got %d", tt.expectUnassigned, result.Unassigned)
				}
			}
		})
	}
}

func TestGetAgentMetrics(t *testing.T) {
	tests := []struct {
		name         string
		userID       string
		statusCode   int
		responseBody string
		expectError  bool
		expectCount  int
		validateURL  func(*testing.T, string)
	}{
		{
			name:       "all agents",
			userID:     "",
			statusCode: http.StatusOK,
			responseBody: `[
				{
					"id": 1,
					"name": "Agent One",
					"email": "agent1@example.com",
					"availability": "online",
					"metric": {
						"open": 15,
						"unattended": 3
					}
				},
				{
					"id": 2,
					"name": "Agent Two",
					"email": "agent2@example.com",
					"availability": "busy",
					"metric": {
						"open": 10,
						"unattended": 2
					}
				}
			]`,
			expectError: false,
			expectCount: 2,
			validateURL: func(t *testing.T, url string) {
				if !strings.Contains(url, "type=agent") {
					t.Error("URL should contain type=agent")
				}
				if strings.Contains(url, "user_id=") {
					t.Error("URL should not contain user_id when not specified")
				}
			},
		},
		{
			name:       "specific agent with user_id",
			userID:     "42",
			statusCode: http.StatusOK,
			responseBody: `[
				{
					"id": 42,
					"name": "Specific Agent",
					"email": "specific@example.com",
					"thumbnail": "https://example.com/avatar.png",
					"availability": "online",
					"metric": {
						"open": 8,
						"unattended": 1
					}
				}
			]`,
			expectError: false,
			expectCount: 1,
			validateURL: func(t *testing.T, url string) {
				if !strings.Contains(url, "type=agent") {
					t.Error("URL should contain type=agent")
				}
				if !strings.Contains(url, "user_id=42") {
					t.Error("URL should contain user_id=42")
				}
			},
		},
		{
			name:         "empty result",
			userID:       "999",
			statusCode:   http.StatusOK,
			responseBody: `[]`,
			expectError:  false,
			expectCount:  0,
		},
		{
			name:         "unauthorized error",
			userID:       "",
			statusCode:   http.StatusUnauthorized,
			responseBody: `{"error": "Unauthorized"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedURL string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedURL = r.URL.String()
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}

				// Verify path contains /reports/conversations
				if !strings.Contains(r.URL.Path, "/reports/conversations") {
					t.Errorf("Expected path to contain /reports/conversations, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Reports().AgentMetrics(context.Background(), tt.userID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				if len(result) != tt.expectCount {
					t.Errorf("Expected %d agents, got %d", tt.expectCount, len(result))
				}

				// Verify first agent details if present
				if tt.expectCount > 0 {
					agent := result[0]
					if agent.Name == "" {
						t.Error("Expected agent name to be populated")
					}
					if agent.Email == "" {
						t.Error("Expected agent email to be populated")
					}
				}
			}

			if tt.validateURL != nil {
				tt.validateURL(t, capturedURL)
			}
		})
	}
}

func TestGetChannelSummary(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }
	tests := []struct {
		name          string
		since         string
		until         string
		businessHours *bool
		statusCode    int
		responseBody  string
		expectError   bool
		expectCount   int
		validateURL   func(*testing.T, string)
	}{
		{
			name:          "with params and business hours",
			since:         "1609459200",
			until:         "1609545600",
			businessHours: boolPtr(true),
			statusCode:    http.StatusOK,
			responseBody: `{
				"Channel::WebWidget": {"open": 5, "resolved": 10, "pending": 2, "snoozed": 1, "total": 18},
				"Channel::Email": {"open": 3, "resolved": 4, "pending": 1, "snoozed": 0, "total": 8}
			}`,
			expectError: false,
			expectCount: 2,
			validateURL: func(t *testing.T, url string) {
				if !strings.Contains(url, "since=1609459200") {
					t.Error("URL should contain since param")
				}
				if !strings.Contains(url, "until=1609545600") {
					t.Error("URL should contain until param")
				}
				if !strings.Contains(url, "business_hours=true") {
					t.Error("URL should contain business_hours=true")
				}
			},
		},
		{
			name:          "no params",
			since:         "",
			until:         "",
			businessHours: nil,
			statusCode:    http.StatusOK,
			responseBody:  `{}`,
			expectError:   false,
			expectCount:   0,
			validateURL: func(t *testing.T, url string) {
				if strings.Contains(url, "since=") || strings.Contains(url, "until=") || strings.Contains(url, "business_hours=") {
					t.Error("URL should not contain query params")
				}
			},
		},
		{
			name:          "unauthorized error",
			since:         "1609459200",
			until:         "1609545600",
			businessHours: nil,
			statusCode:    http.StatusUnauthorized,
			responseBody:  `{"error": "Unauthorized"}`,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedURL string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedURL = r.URL.String()
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/summary_reports/channel") {
					t.Errorf("Expected path to contain /summary_reports/channel, got %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Reports().ChannelSummary(context.Background(), tt.since, tt.until, tt.businessHours)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				if len(result) != tt.expectCount {
					t.Errorf("Expected %d channel summaries, got %d", tt.expectCount, len(result))
				}
			}

			if tt.validateURL != nil {
				tt.validateURL(t, capturedURL)
			}
		})
	}
}

func TestSummaryReportEntries(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }

	tests := []struct {
		name          string
		expectedPath  string
		call          func(ReportsService, context.Context, string, string, *bool) ([]SummaryReportEntry, error)
		businessHours *bool
	}{
		{
			name:          "summary by inbox",
			expectedPath:  "/api/v2/accounts/1/summary_reports/inbox",
			call:          ReportsService.SummaryByInbox,
			businessHours: boolPtr(true),
		},
		{
			name:          "summary by agent",
			expectedPath:  "/api/v2/accounts/1/summary_reports/agent",
			call:          ReportsService.SummaryByAgent,
			businessHours: boolPtr(true),
		},
		{
			name:          "summary by team",
			expectedPath:  "/api/v2/accounts/1/summary_reports/team",
			call:          ReportsService.SummaryByTeam,
			businessHours: boolPtr(true),
		},
	}

	responseBody := `[
		{"id": 1, "conversations_count": 12, "resolved_conversations_count": 10, "avg_resolution_time": 3600, "avg_first_response_time": 120, "avg_reply_time": 240},
		{"id": 2, "conversations_count": 5, "resolved_conversations_count": 3, "avg_resolution_time": null, "avg_first_response_time": 60, "avg_reply_time": null}
	]`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedURL string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedURL = r.URL.String()
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				if r.URL.Path != tt.expectedPath {
					t.Errorf("Expected path %s, got %s", tt.expectedPath, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := tt.call(client.Reports(), context.Background(), "1609459200", "1609545600", tt.businessHours)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			if len(result) != 2 {
				t.Errorf("Expected 2 summary entries, got %d", len(result))
			}
			if len(result) > 0 && result[0].ID != 1 {
				t.Errorf("Expected first entry ID 1, got %d", result[0].ID)
			}
			if len(result) > 1 && result[1].AvgResolutionTime != nil {
				t.Error("Expected second entry avg_resolution_time to be nil")
			}
			if !strings.Contains(capturedURL, "since=1609459200") || !strings.Contains(capturedURL, "until=1609545600") {
				t.Error("Expected URL to include since/until query params")
			}
			if !strings.Contains(capturedURL, "business_hours=true") {
				t.Error("Expected URL to include business_hours=true")
			}
		})
	}
}

func TestListReportingEvents(t *testing.T) {
	tests := []struct {
		name         string
		since        string
		until        string
		eventType    string
		statusCode   int
		responseBody string
		expectError  bool
		expectCount  int
	}{
		{
			name:       "list events with filters",
			since:      "1609459200",
			until:      "1609545600",
			eventType:  "conversation.created",
			statusCode: http.StatusOK,
			responseBody: `[
				{"id": 1, "name": "conversation_created", "value": 1, "account_id": 1, "created_at": "2021-01-01T00:00:00Z"},
				{"id": 2, "name": "conversation_created", "value": 1, "account_id": 1, "created_at": "2021-01-01T01:00:00Z"}
			]`,
			expectError: false,
			expectCount: 2,
		},
		{
			name:         "empty events",
			since:        "",
			until:        "",
			eventType:    "",
			statusCode:   http.StatusOK,
			responseBody: `[]`,
			expectError:  false,
			expectCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}

				// Verify path contains /reporting_events (v1 API)
				if !strings.Contains(r.URL.Path, "/reporting_events") {
					t.Errorf("Expected path to contain /reporting_events, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Reports().ListEvents(context.Background(), tt.since, tt.until, tt.eventType)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError && len(result) != tt.expectCount {
				t.Errorf("Expected %d events, got %d", tt.expectCount, len(result))
			}
		})
	}
}

func TestInboxLabelMatrix(t *testing.T) {
	tests := []struct {
		name         string
		since        string
		until        string
		inboxIDs     []int
		labelIDs     []int
		statusCode   int
		responseBody string
		expectError  bool
		validateURL  func(*testing.T, string)
	}{
		{
			name:       "basic matrix",
			since:      "1609459200",
			until:      "1609545600",
			statusCode: http.StatusOK,
			responseBody: `[
				{"inbox_id": 1, "label_id": 2, "count": 15},
				{"inbox_id": 1, "label_id": 3, "count": 8}
			]`,
			validateURL: func(t *testing.T, url string) {
				if !strings.Contains(url, "since=1609459200") {
					t.Error("URL should contain since param")
				}
				if !strings.Contains(url, "until=1609545600") {
					t.Error("URL should contain until param")
				}
			},
		},
		{
			name:         "with inbox and label filters",
			since:        "1609459200",
			until:        "1609545600",
			inboxIDs:     []int{1, 2},
			labelIDs:     []int{3},
			statusCode:   http.StatusOK,
			responseBody: `[{"inbox_id": 1, "label_id": 3, "count": 5}]`,
			validateURL: func(t *testing.T, url string) {
				if !strings.Contains(url, "inbox_ids") {
					t.Error("URL should contain inbox_ids param")
				}
				if !strings.Contains(url, "label_ids") {
					t.Error("URL should contain label_ids param")
				}
			},
		},
		{
			name:         "server error",
			since:        "1609459200",
			until:        "1609545600",
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": "server error"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedURL string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedURL = r.URL.String()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 123)
			result, err := client.Reports().InboxLabelMatrix(context.Background(), tt.since, tt.until, tt.inboxIDs, tt.labelIDs)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result == nil {
				t.Error("expected result but got nil")
				return
			}
			if tt.validateURL != nil {
				tt.validateURL(t, capturedURL)
			}
		})
	}
}

func TestGetConversationReportingEvents(t *testing.T) {
	tests := []struct {
		name           string
		conversationID int
		statusCode     int
		responseBody   string
		expectError    bool
		expectCount    int
	}{
		{
			name:           "conversation with events",
			conversationID: 123,
			statusCode:     http.StatusOK,
			responseBody: `[
				{"id": 1, "name": "first_response", "value": 120, "account_id": 1, "created_at": "2021-01-01T00:02:00Z"},
				{"id": 2, "name": "conversation_resolved", "value": 1, "account_id": 1, "created_at": "2021-01-01T01:00:00Z"}
			]`,
			expectError: false,
			expectCount: 2,
		},
		{
			name:           "conversation without events",
			conversationID: 456,
			statusCode:     http.StatusOK,
			responseBody:   `[]`,
			expectError:    false,
			expectCount:    0,
		},
		{
			name:           "conversation not found",
			conversationID: 999,
			statusCode:     http.StatusNotFound,
			responseBody:   `{"error": "Conversation not found"}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}

				// Verify path contains conversation ID and reporting_events
				expectedPathPart := "/conversations/"
				if !strings.Contains(r.URL.Path, expectedPathPart) {
					t.Errorf("Expected path to contain %s, got %s", expectedPathPart, r.URL.Path)
				}
				if !strings.Contains(r.URL.Path, "/reporting_events") {
					t.Errorf("Expected path to contain /reporting_events, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Reports().ConversationEvents(context.Background(), tt.conversationID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError && len(result) != tt.expectCount {
				t.Errorf("Expected %d events, got %d", tt.expectCount, len(result))
			}
		})
	}
}

func TestOutgoingMessagesCount(t *testing.T) {
	tests := []struct {
		name         string
		since        string
		until        string
		groupBy      string
		statusCode   int
		responseBody string
		expectError  bool
		validateURL  func(*testing.T, string)
	}{
		{
			name:       "group by agent",
			since:      "1609459200",
			until:      "1609545600",
			groupBy:    "agent",
			statusCode: http.StatusOK,
			responseBody: `[
				{"id": 1, "count": 42},
				{"id": 2, "count": 35}
			]`,
			validateURL: func(t *testing.T, url string) {
				if !strings.Contains(url, "group_by=agent") {
					t.Error("URL should contain group_by=agent")
				}
			},
		},
		{
			name:         "no group_by",
			since:        "1609459200",
			until:        "1609545600",
			statusCode:   http.StatusOK,
			responseBody: `[{"id": 0, "count": 100}]`,
			validateURL: func(t *testing.T, url string) {
				if strings.Contains(url, "group_by") {
					t.Error("URL should not contain group_by when empty")
				}
			},
		},
		{
			name:         "server error",
			since:        "1609459200",
			until:        "1609545600",
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": "server error"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedURL string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedURL = r.URL.String()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 123)
			result, err := client.Reports().OutgoingMessagesCount(context.Background(), tt.since, tt.until, tt.groupBy)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result == nil {
				t.Error("expected result but got nil")
			}
			if tt.validateURL != nil {
				tt.validateURL(t, capturedURL)
			}
		})
	}
}

func TestFirstResponseTimeDistribution(t *testing.T) {
	tests := []struct {
		name         string
		since        string
		until        string
		statusCode   int
		responseBody string
		expectError  bool
		validateURL  func(*testing.T, string)
	}{
		{
			name:       "basic distribution",
			since:      "1609459200",
			until:      "1609545600",
			statusCode: http.StatusOK,
			responseBody: `{
				"Channel::WebWidget": {"0-1h": 10, "1-4h": 5, "4-8h": 2, "8-24h": 1, "24h+": 0},
				"Channel::Email": {"0-1h": 3, "1-4h": 8, "4-8h": 4, "8-24h": 6, "24h+": 2}
			}`,
			validateURL: func(t *testing.T, url string) {
				if !strings.Contains(url, "since=1609459200") {
					t.Error("URL should contain since param")
				}
				if !strings.Contains(url, "until=1609545600") {
					t.Error("URL should contain until param")
				}
			},
		},
		{
			name:         "server error",
			since:        "1609459200",
			until:        "1609545600",
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": "server error"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedURL string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedURL = r.URL.String()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 123)
			result, err := client.Reports().FirstResponseTimeDistribution(context.Background(), tt.since, tt.until)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result == nil {
				t.Error("expected result but got nil")
				return
			}
			if tt.validateURL != nil {
				tt.validateURL(t, capturedURL)
			}
		})
	}
}
