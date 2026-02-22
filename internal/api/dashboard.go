package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// maxResponseSize is the maximum response body size allowed from dashboard endpoints (10MB)
const maxResponseSize = 10 * 1024 * 1024

// ErrResponseTooLarge is returned when the dashboard response exceeds maxResponseSize
var ErrResponseTooLarge = errors.New("dashboard response exceeds maximum allowed size of 10MB")

// DashboardClient is a client for external dashboard APIs
type DashboardClient struct {
	Endpoint  string
	AuthToken string
	HTTP      *http.Client
}

// DashboardRequest is the request body for dashboard queries
type DashboardRequest struct {
	ContactID int `json:"contact_id"`
	Page      int `json:"page"`
	PerPage   int `json:"per_page"`
}

// NewDashboardClient creates a new dashboard API client
func NewDashboardClient(endpoint, authToken string) *DashboardClient {
	return &DashboardClient{
		Endpoint:  endpoint,
		AuthToken: authToken,
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
	httpReq.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.AuthToken)))

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Early rejection if Content-Length is known and exceeds limit (avoids reading any bytes)
	if resp.ContentLength > maxResponseSize {
		return nil, fmt.Errorf("dashboard response too large: %d bytes exceeds %d", resp.ContentLength, maxResponseSize)
	}

	// Limit response body size to prevent memory exhaustion from malicious/misconfigured endpoints
	// (handles cases where Content-Length is unknown or missing)
	limitedReader := io.LimitReader(resp.Body, maxResponseSize+1)
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if len(respBody) > maxResponseSize {
		return nil, ErrResponseTooLarge
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
