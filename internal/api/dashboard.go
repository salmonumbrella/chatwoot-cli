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
	"net/url"
	"path"
	"strings"
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

// LinkOrderRequest is the request body for linking an order to a contact.
type LinkOrderRequest struct {
	OrderNumber string `json:"order_number"`
	ContactID   int    `json:"contact_id"`
}

// LinkOrderResponse is the response payload for a successful link operation.
type LinkOrderResponse struct {
	CustomerID        string `json:"customer_id"`
	ChatwootContactID int    `json:"chatwoot_contact_id"`
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

	return c.doJSON(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body), "application/json")
}

// QueryOrderDetail fetches order-level detail payload (line_items, metadata) by order ID.
func (c *DashboardClient) QueryOrderDetail(ctx context.Context, orderID string) (map[string]any, error) {
	detailURL, err := c.orderDetailURL(orderID)
	if err != nil {
		return nil, err
	}
	return c.doJSON(ctx, http.MethodGet, detailURL, nil, "")
}

// LinkOrderToContact links an order number to the provided Chatwoot contact ID.
func (c *DashboardClient) LinkOrderToContact(ctx context.Context, orderNumber string, contactID int) (*LinkOrderResponse, error) {
	orderNumber = strings.TrimSpace(orderNumber)
	if orderNumber == "" {
		return nil, fmt.Errorf("order number is required")
	}
	if contactID <= 0 {
		return nil, fmt.Errorf("contact id must be positive")
	}

	linkURL, err := c.orderLinkURL()
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(LinkOrderRequest{
		OrderNumber: orderNumber,
		ContactID:   contactID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal link request: %w", err)
	}

	raw, err := c.doRaw(ctx, http.MethodPost, linkURL, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, err
	}

	var resp LinkOrderResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse link response: %w", err)
	}

	return &resp, nil
}

func (c *DashboardClient) doJSON(ctx context.Context, method, endpoint string, body io.Reader, contentType string) (map[string]any, error) {
	raw, err := c.doRaw(ctx, method, endpoint, body, contentType)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

func (c *DashboardClient) doRaw(ctx context.Context, method, endpoint string, body io.Reader, contentType string) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}
	if authHeader := c.authorizationHeader(); authHeader != "" {
		httpReq.Header.Set("Authorization", authHeader)
	}
	httpReq.Header.Set("Accept", "application/json")

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

	return respBody, nil
}

func (c *DashboardClient) authorizationHeader() string {
	token := strings.TrimSpace(c.AuthToken)
	if token == "" {
		return ""
	}
	lower := strings.ToLower(token)
	if strings.HasPrefix(lower, "bearer ") {
		return "Bearer " + token[7:]
	}
	if strings.HasPrefix(lower, "basic ") {
		return "Basic " + token[6:]
	}
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(token))
}

func (c *DashboardClient) orderDetailURL(orderID string) (string, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return "", fmt.Errorf("order id is required")
	}

	parsed, err := url.Parse(c.Endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid dashboard endpoint %q: %w", c.Endpoint, err)
	}

	escaped := url.PathEscape(orderID)
	p := strings.TrimRight(parsed.Path, "/")

	switch {
	case strings.HasSuffix(p, "/chatwoot/contact/orders"):
		p = strings.TrimSuffix(p, "/chatwoot/contact/orders") + "/chatwoot/orders/" + escaped
	case strings.HasSuffix(p, "/contact/orders"):
		p = strings.TrimSuffix(p, "/contact/orders") + "/orders/" + escaped
	case strings.HasSuffix(p, "/orders"):
		p = p + "/" + escaped
	case p == "":
		p = "/orders/" + escaped
	default:
		return "", fmt.Errorf("cannot derive order detail URL from endpoint path %q: expected path ending in /orders or /contact/orders", p)
	}

	parsed.Path = path.Clean(p)
	parsed.RawPath = "" // let url.URL re-escape from Path
	// Order detail is a clean path-only URL; drop any query params or fragment
	// that may exist on the base endpoint — they belong to the listing endpoint.
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return parsed.String(), nil
}

func (c *DashboardClient) orderLinkURL() (string, error) {
	parsed, err := url.Parse(c.Endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid dashboard endpoint %q: %w", c.Endpoint, err)
	}

	p := strings.TrimRight(parsed.Path, "/")

	switch {
	case strings.HasSuffix(p, "/chatwoot/contact/orders"):
		p = strings.TrimSuffix(p, "/chatwoot/contact/orders") + "/chatwoot/orders/link"
	case strings.HasSuffix(p, "/contact/orders"):
		p = strings.TrimSuffix(p, "/contact/orders") + "/orders/link"
	case strings.HasSuffix(p, "/orders"):
		p = p + "/link"
	case p == "":
		p = "/orders/link"
	default:
		return "", fmt.Errorf("cannot derive order link URL from endpoint path %q: expected path ending in /orders or /contact/orders", p)
	}

	parsed.Path = path.Clean(p)
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return parsed.String(), nil
}
