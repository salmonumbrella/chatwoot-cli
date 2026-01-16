package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DashboardClient is a client for external dashboard APIs
type DashboardClient struct {
	Endpoint  string
	AuthEmail string
	HTTP      *http.Client
}

// DashboardRequest is the request body for dashboard queries
type DashboardRequest struct {
	ContactID int `json:"contact_id"`
	Page      int `json:"page"`
	PerPage   int `json:"per_page"`
}

// NewDashboardClient creates a new dashboard API client
func NewDashboardClient(endpoint, authEmail string) *DashboardClient {
	return &DashboardClient{
		Endpoint:  endpoint,
		AuthEmail: authEmail,
		HTTP: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Query sends a request to the dashboard endpoint and returns the response
func (c *DashboardClient) Query(ctx context.Context, req DashboardRequest) (map[string]any, error) {
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PerPage == 0 {
		req.PerPage = 100
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.AuthEmail)))

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("dashboard API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}
