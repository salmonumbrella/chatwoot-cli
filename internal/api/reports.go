package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ReportSummary represents a report summary from /reports/summary
type ReportSummary struct {
	AvgFirstResponseTime  string         `json:"avg_first_response_time,omitempty"`
	AvgResolutionTime     string         `json:"avg_resolution_time,omitempty"`
	ConversationsCount    int            `json:"conversations_count,omitempty"`
	IncomingMessagesCount int            `json:"incoming_messages_count,omitempty"`
	OutgoingMessagesCount int            `json:"outgoing_messages_count,omitempty"`
	ResolutionsCount      int            `json:"resolutions_count,omitempty"`
	Previous              *ReportSummary `json:"previous,omitempty"`
}

// ReportDataPoint represents a single data point in a time-series report from /reports
type ReportDataPoint struct {
	Value     FlexString `json:"value"`
	Timestamp int64      `json:"timestamp"`
}

// ConversationMetrics represents account-level conversation metrics from /reports/conversations?type=account
type ConversationMetrics struct {
	Open       int `json:"open"`
	Unattended int `json:"unattended"`
	Unassigned int `json:"unassigned"`
}

// AgentMetrics represents an agent's conversation metrics from /reports/conversations?type=agent
type AgentMetrics struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Thumbnail    string `json:"thumbnail,omitempty"`
	Availability string `json:"availability,omitempty"`
	Metric       struct {
		Open       int `json:"open"`
		Unattended int `json:"unattended"`
	} `json:"metric,omitempty"`
}

// v2ReportPath returns the base path for v2 reports API
func (c *Client) v2ReportPath(path string) string {
	return fmt.Sprintf("%s/api/v2/accounts/%d%s", c.BaseURL, c.AccountID, path)
}

// GetReportSummary gets report summary.
func (s ReportsService) GetReportSummary(ctx context.Context, reportType, since, until, id string) (*ReportSummary, error) {
	params := url.Values{}
	params.Set("type", reportType)
	params.Set("since", since)
	params.Set("until", until)
	if id != "" {
		params.Set("id", id)
	}

	reqURL := s.v2ReportPath("/reports/summary?" + params.Encode())

	var result ReportSummary
	if err := s.do(ctx, http.MethodGet, reqURL, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Summary gets report summary.
func (s ReportsService) Summary(ctx context.Context, reportType, since, until, id string) (*ReportSummary, error) {
	return s.GetReportSummary(ctx, reportType, since, until, id)
}

// GetReportTimeSeries gets time-series report data for a specific metric.
func (s ReportsService) GetReportTimeSeries(ctx context.Context, metric, reportType, since, until, id string) ([]ReportDataPoint, error) {
	params := url.Values{}
	params.Set("metric", metric)
	params.Set("type", reportType)
	params.Set("since", since)
	params.Set("until", until)
	if id != "" {
		params.Set("id", id)
	}

	reqURL := s.v2ReportPath("/reports?" + params.Encode())

	var result []ReportDataPoint
	if err := s.do(ctx, http.MethodGet, reqURL, nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// TimeSeries gets time-series report data for a specific metric.
func (s ReportsService) TimeSeries(ctx context.Context, metric, reportType, since, until, id string) ([]ReportDataPoint, error) {
	return s.GetReportTimeSeries(ctx, metric, reportType, since, until, id)
}

// GetConversationMetrics gets account-level conversation metrics.
func (s ReportsService) GetConversationMetrics(ctx context.Context) (*ConversationMetrics, error) {
	params := url.Values{}
	params.Set("type", "account")

	reqURL := s.v2ReportPath("/reports/conversations?" + params.Encode())

	var result ConversationMetrics
	if err := s.do(ctx, http.MethodGet, reqURL, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ConversationMetrics gets account-level conversation metrics.
func (s ReportsService) ConversationMetrics(ctx context.Context) (*ConversationMetrics, error) {
	return s.GetConversationMetrics(ctx)
}

// GetAgentMetrics gets conversation metrics for agents.
func (s ReportsService) GetAgentMetrics(ctx context.Context, userID string) ([]AgentMetrics, error) {
	params := url.Values{}
	params.Set("type", "agent")
	if userID != "" {
		params.Set("user_id", userID)
	}

	reqURL := s.v2ReportPath("/reports/conversations?" + params.Encode())

	var result []AgentMetrics
	if err := s.do(ctx, http.MethodGet, reqURL, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// AgentMetrics gets conversation metrics for agents.
func (s ReportsService) AgentMetrics(ctx context.Context, userID string) ([]AgentMetrics, error) {
	return s.GetAgentMetrics(ctx, userID)
}

// ReportingEvent represents a reporting event
type ReportingEvent struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Value     any    `json:"value"`
	AccountID int    `json:"account_id"`
	InboxID   int    `json:"inbox_id,omitempty"`
	UserID    int    `json:"user_id,omitempty"`
	CreatedAt string `json:"created_at"`
	EventType string `json:"event_type,omitempty"`
}

// ListReportingEvents lists account-level reporting events.
func (s ReportsService) ListReportingEvents(ctx context.Context, since, until string, eventType string) ([]ReportingEvent, error) {
	params := url.Values{}
	if since != "" {
		params.Set("since", since)
	}
	if until != "" {
		params.Set("until", until)
	}
	if eventType != "" {
		params.Set("type", eventType)
	}

	path := s.accountPath("/reporting_events")
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var result []ReportingEvent
	if err := s.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ListEvents lists account-level reporting events.
func (s ReportsService) ListEvents(ctx context.Context, since, until string, eventType string) ([]ReportingEvent, error) {
	return s.ListReportingEvents(ctx, since, until, eventType)
}

// GetConversationReportingEvents gets reporting events for a conversation.
func (s ReportsService) GetConversationReportingEvents(ctx context.Context, conversationID int) ([]ReportingEvent, error) {
	path := s.accountPath(fmt.Sprintf("/conversations/%d/reporting_events", conversationID))

	var result []ReportingEvent
	if err := s.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ConversationEvents gets reporting events for a conversation.
func (s ReportsService) ConversationEvents(ctx context.Context, conversationID int) ([]ReportingEvent, error) {
	return s.GetConversationReportingEvents(ctx, conversationID)
}
