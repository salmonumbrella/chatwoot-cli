package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// ReportSummary represents a report summary from /reports/summary
type ReportSummary struct {
	AvgFirstResponseTime  FlexString     `json:"avg_first_response_time,omitempty"`
	AvgResolutionTime     FlexString     `json:"avg_resolution_time,omitempty"`
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

// ChannelSummary represents conversation counts grouped by channel type.
type ChannelSummary struct {
	Open     int `json:"open"`
	Resolved int `json:"resolved"`
	Pending  int `json:"pending"`
	Snoozed  int `json:"snoozed"`
	Total    int `json:"total"`
}

// SummaryReportEntry represents a summary report entry grouped by inbox, agent, or team.
type SummaryReportEntry struct {
	ID                         int        `json:"id"`
	ConversationsCount         FlexInt    `json:"conversations_count"`
	ResolvedConversationsCount FlexInt    `json:"resolved_conversations_count"`
	AvgResolutionTime          *FlexFloat `json:"avg_resolution_time,omitempty"`
	AvgFirstResponseTime       *FlexFloat `json:"avg_first_response_time,omitempty"`
	AvgReplyTime               *FlexFloat `json:"avg_reply_time,omitempty"`
}

// v2ReportPath returns the base path for v2 reports API
func (c *Client) v2ReportPath(path string) string {
	return fmt.Sprintf("%s/api/v2/accounts/%d%s", c.BaseURL, c.AccountID, path)
}

// Summary gets report summary.
func (s ReportsService) Summary(ctx context.Context, reportType, since, until, id string) (*ReportSummary, error) {
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

// TimeSeries gets time-series report data for a specific metric.
func (s ReportsService) TimeSeries(ctx context.Context, metric, reportType, since, until, id string) ([]ReportDataPoint, error) {
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

// ConversationMetrics gets account-level conversation metrics.
func (s ReportsService) ConversationMetrics(ctx context.Context) (*ConversationMetrics, error) {
	params := url.Values{}
	params.Set("type", "account")

	reqURL := s.v2ReportPath("/reports/conversations?" + params.Encode())

	var result ConversationMetrics
	if err := s.do(ctx, http.MethodGet, reqURL, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// AgentMetrics gets conversation metrics for agents.
func (s ReportsService) AgentMetrics(ctx context.Context, userID string) ([]AgentMetrics, error) {
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

// ChannelSummary gets conversation statistics grouped by channel type.
func (s ReportsService) ChannelSummary(ctx context.Context, since, until string, businessHours *bool) (map[string]ChannelSummary, error) {
	params := url.Values{}
	if since != "" {
		params.Set("since", since)
	}
	if until != "" {
		params.Set("until", until)
	}
	if businessHours != nil {
		params.Set("business_hours", strconv.FormatBool(*businessHours))
	}

	reqURL := s.v2ReportPath("/summary_reports/channel")
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	var result map[string]ChannelSummary
	if err := s.do(ctx, http.MethodGet, reqURL, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s ReportsService) summaryReportEntries(ctx context.Context, path, since, until string, businessHours *bool) ([]SummaryReportEntry, error) {
	params := url.Values{}
	if since != "" {
		params.Set("since", since)
	}
	if until != "" {
		params.Set("until", until)
	}
	if businessHours != nil {
		params.Set("business_hours", strconv.FormatBool(*businessHours))
	}

	reqURL := s.v2ReportPath(path)
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	var result []SummaryReportEntry
	if err := s.do(ctx, http.MethodGet, reqURL, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// SummaryByInbox gets summary report entries grouped by inbox.
func (s ReportsService) SummaryByInbox(ctx context.Context, since, until string, businessHours *bool) ([]SummaryReportEntry, error) {
	return s.summaryReportEntries(ctx, "/summary_reports/inbox", since, until, businessHours)
}

// SummaryByAgent gets summary report entries grouped by agent.
func (s ReportsService) SummaryByAgent(ctx context.Context, since, until string, businessHours *bool) ([]SummaryReportEntry, error) {
	return s.summaryReportEntries(ctx, "/summary_reports/agent", since, until, businessHours)
}

// SummaryByTeam gets summary report entries grouped by team.
func (s ReportsService) SummaryByTeam(ctx context.Context, since, until string, businessHours *bool) ([]SummaryReportEntry, error) {
	return s.summaryReportEntries(ctx, "/summary_reports/team", since, until, businessHours)
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

// ListEvents lists account-level reporting events.
func (s ReportsService) ListEvents(ctx context.Context, since, until string, eventType string) ([]ReportingEvent, error) {
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

// ConversationEvents gets reporting events for a conversation.
func (s ReportsService) ConversationEvents(ctx context.Context, conversationID int) ([]ReportingEvent, error) {
	path := s.accountPath(fmt.Sprintf("/conversations/%d/reporting_events", conversationID))

	var result []ReportingEvent
	if err := s.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// InboxLabelMatrixEntry represents a single cell in the inbox-label matrix report.
type InboxLabelMatrixEntry struct {
	InboxID int `json:"inbox_id"`
	LabelID int `json:"label_id"`
	Count   int `json:"count"`
}

// InboxLabelMatrix gets conversation counts grouped by inbox and label.
func (s ReportsService) InboxLabelMatrix(ctx context.Context, since, until string, inboxIDs, labelIDs []int) ([]InboxLabelMatrixEntry, error) {
	params := url.Values{}
	params.Set("since", since)
	params.Set("until", until)
	for _, id := range inboxIDs {
		params.Add("inbox_ids[]", strconv.Itoa(id))
	}
	for _, id := range labelIDs {
		params.Add("label_ids[]", strconv.Itoa(id))
	}

	reqURL := s.v2ReportPath("/reports/inbox_label_matrix?" + params.Encode())

	var result []InboxLabelMatrixEntry
	if err := s.do(ctx, http.MethodGet, reqURL, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// FirstResponseTimeDistribution gets conversation counts grouped by channel type and response time buckets.
func (s ReportsService) FirstResponseTimeDistribution(ctx context.Context, since, until string) (map[string]map[string]int, error) {
	params := url.Values{}
	params.Set("since", since)
	params.Set("until", until)

	reqURL := s.v2ReportPath("/reports/first_response_time_distribution?" + params.Encode())

	var result map[string]map[string]int
	if err := s.do(ctx, http.MethodGet, reqURL, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// OutgoingMessagesEntry represents an outgoing messages count entry.
type OutgoingMessagesEntry struct {
	ID    int `json:"id"`
	Count int `json:"count"`
}

// OutgoingMessagesCount gets outgoing message counts grouped by agent, team, inbox, or label.
func (s ReportsService) OutgoingMessagesCount(ctx context.Context, since, until, groupBy string) ([]OutgoingMessagesEntry, error) {
	params := url.Values{}
	params.Set("since", since)
	params.Set("until", until)
	if groupBy != "" {
		params.Set("group_by", groupBy)
	}

	reqURL := s.v2ReportPath("/reports/outgoing_messages_count?" + params.Encode())

	var result []OutgoingMessagesEntry
	if err := s.do(ctx, http.MethodGet, reqURL, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}
