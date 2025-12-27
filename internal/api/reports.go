package api

import (
	"context"
	"fmt"
	"net/http"
)

// ReportSummary represents a report summary
type ReportSummary struct {
	AvgFirstResponseTime  string         `json:"avg_first_response_time,omitempty"`
	AvgResolutionTime     string         `json:"avg_resolution_time,omitempty"`
	ConversationsCount    int            `json:"conversations_count,omitempty"`
	IncomingMessagesCount int            `json:"incoming_messages_count,omitempty"`
	OutgoingMessagesCount int            `json:"outgoing_messages_count,omitempty"`
	ResolutionsCount      int            `json:"resolutions_count,omitempty"`
	Previous              *ReportSummary `json:"previous,omitempty"`
}

// ConversationsReport represents a conversations report
type ConversationsReport struct {
	Data []map[string]any `json:"data"`
}

// MetricsReport represents a metrics report
type MetricsReport struct {
	Metrics map[string]any `json:"metrics"`
}

// GetReportSummary gets report summary
func (c *Client) GetReportSummary(ctx context.Context, reportType, from, to string) (*ReportSummary, error) {
	path := fmt.Sprintf("/reports/summary?type=%s&since=%s&until=%s", reportType, from, to)

	url := fmt.Sprintf("%s/api/v2/accounts/%d%s", c.BaseURL, c.AccountID, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("api_access_token", c.APIToken)
	req.Header.Set("Accept", "application/json")

	var result ReportSummary
	if err := c.do(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetConversationsReport gets conversations report
func (c *Client) GetConversationsReport(ctx context.Context, reportType, from, to string) (*ConversationsReport, error) {
	path := fmt.Sprintf("/reports/conversations?type=%s&since=%s&until=%s", reportType, from, to)

	url := fmt.Sprintf("%s/api/v2/accounts/%d%s", c.BaseURL, c.AccountID, path)

	var result ConversationsReport
	if err := c.do(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetMetricsReport gets metrics report
func (c *Client) GetMetricsReport(ctx context.Context, from, to string) (*MetricsReport, error) {
	path := fmt.Sprintf("/reports/metrics?since=%s&until=%s", from, to)

	url := fmt.Sprintf("%s/api/v2/accounts/%d%s", c.BaseURL, c.AccountID, path)

	var result MetricsReport
	if err := c.do(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// AgentReport represents an agent's performance metrics
type AgentReport struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	// Metrics are dynamic based on time range
	Metric map[string]any `json:"metric,omitempty"`
}

// InboxReport represents an inbox's metrics
type InboxReport struct {
	ID     int            `json:"id"`
	Name   string         `json:"name"`
	Metric map[string]any `json:"metric,omitempty"`
}

// TeamReport represents a team's metrics
type TeamReport struct {
	ID     int            `json:"id"`
	Name   string         `json:"name"`
	Metric map[string]any `json:"metric,omitempty"`
}

// LabelReport represents a label's analytics
type LabelReport struct {
	ID     int            `json:"id"`
	Title  string         `json:"title"`
	Metric map[string]any `json:"metric,omitempty"`
}

// LiveMetrics represents real-time conversation metrics
type LiveMetrics struct {
	Open       int `json:"open"`
	Unassigned int `json:"unassigned"`
	Pending    int `json:"pending"`
}

// GetAgentsReport gets agent performance report
func (c *Client) GetAgentsReport(ctx context.Context, from, to string) ([]AgentReport, error) {
	path := fmt.Sprintf("/reports/agents?since=%s&until=%s", from, to)
	url := fmt.Sprintf("%s/api/v2/accounts/%d%s", c.BaseURL, c.AccountID, path)

	var result []AgentReport
	if err := c.do(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetInboxesReport gets inbox metrics report
func (c *Client) GetInboxesReport(ctx context.Context, from, to string) ([]InboxReport, error) {
	path := fmt.Sprintf("/reports/inboxes?since=%s&until=%s", from, to)
	url := fmt.Sprintf("%s/api/v2/accounts/%d%s", c.BaseURL, c.AccountID, path)

	var result []InboxReport
	if err := c.do(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetTeamsReport gets team metrics report
func (c *Client) GetTeamsReport(ctx context.Context, from, to string) ([]TeamReport, error) {
	path := fmt.Sprintf("/reports/teams?since=%s&until=%s", from, to)
	url := fmt.Sprintf("%s/api/v2/accounts/%d%s", c.BaseURL, c.AccountID, path)

	var result []TeamReport
	if err := c.do(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetLabelsReport gets label analytics report
func (c *Client) GetLabelsReport(ctx context.Context, from, to string) ([]LabelReport, error) {
	path := fmt.Sprintf("/reports/labels?since=%s&until=%s", from, to)
	url := fmt.Sprintf("%s/api/v2/accounts/%d%s", c.BaseURL, c.AccountID, path)

	var result []LabelReport
	if err := c.do(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetBotSummaryReport gets bot performance summary
func (c *Client) GetBotSummaryReport(ctx context.Context, from, to string) (map[string]any, error) {
	path := fmt.Sprintf("/reports/bot_summary?since=%s&until=%s", from, to)
	url := fmt.Sprintf("%s/api/v2/accounts/%d%s", c.BaseURL, c.AccountID, path)

	var result map[string]any
	if err := c.do(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetConversationTrafficReport gets conversation traffic time-series data
func (c *Client) GetConversationTrafficReport(ctx context.Context, from, to string) (map[string]any, error) {
	path := fmt.Sprintf("/reports/conversation_traffic?since=%s&until=%s", from, to)
	url := fmt.Sprintf("%s/api/v2/accounts/%d%s", c.BaseURL, c.AccountID, path)

	var result map[string]any
	if err := c.do(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetLiveMetrics gets real-time conversation metrics
func (c *Client) GetLiveMetrics(ctx context.Context) (*LiveMetrics, error) {
	url := fmt.Sprintf("%s/api/v2/accounts/%d/live_reports/conversation_metrics", c.BaseURL, c.AccountID)

	var result LiveMetrics
	if err := c.do(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
