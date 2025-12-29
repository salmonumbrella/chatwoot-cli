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
	Value     string `json:"value"`
	Timestamp int64  `json:"timestamp"`
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

// GetReportSummary gets report summary
// Valid types: account, agent, inbox, label, team
// If type is agent/inbox/label, id parameter specifies which one
func (c *Client) GetReportSummary(ctx context.Context, reportType, since, until, id string) (*ReportSummary, error) {
	params := url.Values{}
	params.Set("type", reportType)
	params.Set("since", since)
	params.Set("until", until)
	if id != "" {
		params.Set("id", id)
	}

	reqURL := c.v2ReportPath("/reports/summary?" + params.Encode())

	var result ReportSummary
	if err := c.do(ctx, http.MethodGet, reqURL, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetReportTimeSeries gets time-series report data for a specific metric
// Valid metrics: conversations_count, incoming_messages_count, outgoing_messages_count,
//
//	avg_first_response_time, avg_resolution_time, resolutions_count
//
// Valid types: account, agent, inbox, label, team
// If type is agent/inbox/label/team, id parameter specifies which one
func (c *Client) GetReportTimeSeries(ctx context.Context, metric, reportType, since, until, id string) ([]ReportDataPoint, error) {
	params := url.Values{}
	params.Set("metric", metric)
	params.Set("type", reportType)
	params.Set("since", since)
	params.Set("until", until)
	if id != "" {
		params.Set("id", id)
	}

	reqURL := c.v2ReportPath("/reports?" + params.Encode())

	var result []ReportDataPoint
	if err := c.do(ctx, http.MethodGet, reqURL, nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetConversationMetrics gets account-level conversation metrics (open/unattended/unassigned counts)
func (c *Client) GetConversationMetrics(ctx context.Context) (*ConversationMetrics, error) {
	params := url.Values{}
	params.Set("type", "account")

	reqURL := c.v2ReportPath("/reports/conversations?" + params.Encode())

	var result ConversationMetrics
	if err := c.do(ctx, http.MethodGet, reqURL, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetAgentMetrics gets conversation metrics for all agents or a specific agent
// Returns array of agents with their open/unattended conversation counts
func (c *Client) GetAgentMetrics(ctx context.Context, userID string) ([]AgentMetrics, error) {
	params := url.Values{}
	params.Set("type", "agent")
	if userID != "" {
		params.Set("user_id", userID)
	}

	reqURL := c.v2ReportPath("/reports/conversations?" + params.Encode())

	var result []AgentMetrics
	if err := c.do(ctx, http.MethodGet, reqURL, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}
