package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/debug"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
)

const (
	DefaultTimeout      = 30 * time.Second
	DefaultWaitTimeout  = 5 * time.Minute
	DefaultWaitInterval = 2 * time.Second
)

// Client is the Chatwoot API client.
//
// The client includes a circuit breaker that tracks server failures across requests.
// Circuit breaker state persists for the lifetime of the client, which may affect
// unrelated requests if the client is reused across different logical sessions.
//
// Use ResetCircuitBreaker() to clear the circuit breaker state when reusing a client
// between test runs, logical sessions, or after recovering from a known transient failure.
type Client struct {
	BaseURL            string
	APIToken           string
	AccountID          int
	HTTP               *http.Client
	UserAgent          string
	IdempotencyKey     string
	IdempotencyKeyFunc func() string
	RetryConfig        RetryConfig     // retry and circuit breaker configuration
	skipURLValidation  bool            // internal flag for testing only
	circuitBreaker     *circuitBreaker // circuit breaker for retry logic
	validatedBaseURL   bool
	validateMu         sync.Mutex
	WaitForAsync       bool
	WaitTimeout        time.Duration
	WaitInterval       time.Duration
	rateLimitMu        sync.Mutex
	lastRateLimit      *RateLimitInfo
}

// Compile-time interface implementation checks
var (
	_ Requester    = (*Client)(nil)
	_ PathResolver = (*Client)(nil)
	_ HTTPExecutor = (*Client)(nil)
)

var validateChatwootURL = validation.ValidateChatwootURL

// New creates a new Chatwoot API client
func New(baseURL, token string, accountID int) *Client {
	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		baseTransport = &http.Transport{}
	}
	transport := baseTransport.Clone()
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	} else {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	}
	transport.TLSClientConfig.MinVersion = tls.VersionTLS12
	transport.TLSClientConfig.InsecureSkipVerify = false

	// Allow localhost URLs when CHATWOOT_TESTING=1 is set (for integration tests)
	skipValidation := os.Getenv("CHATWOOT_TESTING") == "1"

	retryCfg := DefaultRetryConfig()
	return &Client{
		BaseURL:           baseURL,
		APIToken:          token,
		AccountID:         accountID,
		RetryConfig:       retryCfg,
		skipURLValidation: skipValidation,
		HTTP: &http.Client{
			Timeout:   DefaultTimeout,
			Transport: transport,
		},
		circuitBreaker: &circuitBreaker{
			threshold: retryCfg.CircuitBreakerThreshold,
			resetTime: retryCfg.CircuitBreakerResetTime,
		},
		WaitTimeout:  DefaultWaitTimeout,
		WaitInterval: DefaultWaitInterval,
	}
}

// newTestClient creates a client with URL validation disabled for testing
func newTestClient(baseURL, token string, accountID int) *Client {
	c := New(baseURL, token, accountID)
	c.skipURLValidation = true
	return c
}

// ResetCircuitBreaker clears the circuit breaker state, resetting failure counts
// and closing the circuit. This is useful when reusing a client across logical
// sessions (e.g., between test runs) to prevent stale failure state from affecting
// new requests.
func (c *Client) ResetCircuitBreaker() {
	if c.circuitBreaker != nil {
		c.circuitBreaker.reset()
	}
}

// SetRetryConfig updates the retry configuration and aligns circuit breaker settings.
func (c *Client) SetRetryConfig(cfg RetryConfig) {
	c.RetryConfig = cfg
	if c.circuitBreaker != nil {
		c.circuitBreaker.threshold = cfg.CircuitBreakerThreshold
		c.circuitBreaker.resetTime = cfg.CircuitBreakerResetTime
	}
}

func (c *Client) ensureBaseURLValidated() error {
	if c.skipURLValidation {
		return nil
	}

	c.validateMu.Lock()
	defer c.validateMu.Unlock()

	if c.validatedBaseURL {
		return nil
	}

	if err := validateChatwootURL(c.BaseURL); err != nil {
		return fmt.Errorf("URL validation failed: %w", err)
	}

	c.validatedBaseURL = true
	return nil
}

// accountPath returns the base path for account-scoped API calls
func (c *Client) accountPath(path string) string {
	if path != "" && path[0] != '/' {
		path = "/" + path
	}
	return fmt.Sprintf("%s/api/v1/accounts/%d%s", c.BaseURL, c.AccountID, path)
}

// platformPath returns the base path for platform API calls
func (c *Client) platformPath(path string) string {
	if path != "" && path[0] != '/' {
		path = "/" + path
	}
	return fmt.Sprintf("%s/platform/api/v1%s", c.BaseURL, path)
}

// publicPath returns the base path for public client API calls
func (c *Client) publicPath(path string) string {
	if path != "" && path[0] != '/' {
		path = "/" + path
	}
	return fmt.Sprintf("%s/public/api/v1%s", c.BaseURL, path)
}

// do performs an HTTP request and decodes the response
func (c *Client) do(ctx context.Context, method, url string, body any, result any) error {
	respBody, _, _, err := c.executeRequest(ctx, method, url, body)
	if err != nil {
		return err
	}
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unexpected API response format (JSON decode failed): %w", err)
		}
	}
	return nil
}

// executeRequest is the internal helper that performs HTTP requests with retry and circuit breaker logic.
// It returns the response body, headers, status code, and any error.
// Both doRaw and DoRaw delegate to this method.
func (c *Client) executeRequest(ctx context.Context, method, url string, body any) ([]byte, http.Header, int, error) {
	// Marshal body to JSON once (will be reused for retries)
	var jsonBody []byte
	if body != nil {
		var err error
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	return c.executeRequestWithBody(ctx, method, url, jsonBody, "application/json")
}

// executeRequestWithBody performs HTTP requests with retry logic for raw request bodies.
// contentType can be empty to omit the Content-Type header.
func (c *Client) executeRequestWithBody(ctx context.Context, method, url string, body []byte, contentType string) ([]byte, http.Header, int, error) {
	return c.executeRequestWithBodyInternal(ctx, method, url, body, contentType, true)
}

// executeRequestWithBodyInternal performs HTTP requests with retry logic and optional async waiting.
// allowWait controls whether 202 responses trigger async polling.
func (c *Client) executeRequestWithBodyInternal(ctx context.Context, method, url string, body []byte, contentType string, allowWait bool) ([]byte, http.Header, int, error) {
	// Check circuit breaker at start
	if c.circuitBreaker != nil && c.circuitBreaker.isOpen() {
		return nil, nil, 0, &CircuitBreakerError{}
	}

	// Validate BaseURL at request time to prevent DNS rebinding attacks
	// Skip validation in tests to allow httptest.Server localhost URLs
	if err := c.ensureBaseURLValidated(); err != nil {
		return nil, nil, 0, err
	}

	idempotencyKey := c.IdempotencyKey
	if idempotencyKey == "" && c.IdempotencyKeyFunc != nil {
		idempotencyKey = c.IdempotencyKeyFunc()
	}

	// Determine if method is idempotent for retry logic
	isIdempotent := method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
	if !isIdempotent && idempotencyKey != "" {
		isIdempotent = true
	}

	var retries429, retries5xx int
	attempt := 0

	for {
		attempt++
		start := time.Now()
		// Create fresh body reader for each attempt
		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to create request: %w", err)
		}

		if c.APIToken != "" {
			req.Header.Set("api_access_token", c.APIToken)
		}
		if c.UserAgent != "" {
			req.Header.Set("User-Agent", c.UserAgent)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		req.Header.Set("Accept", "application/json")
		if idempotencyKey != "" && method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}

		resp, err := c.HTTP.Do(req)
		if err != nil {
			if debug.IsEnabled(ctx) {
				slog.Debug("request failed", "method", method, "url", url, "attempt", attempt, "error", err)
			}
			return nil, nil, 0, fmt.Errorf("request failed: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to read response: %w", err)
		}
		c.recordRateLimit(resp.Header)
		if debug.IsEnabled(ctx) {
			slog.Debug("request complete", "method", method, "url", url, "status", resp.StatusCode, "attempt", attempt, "duration", time.Since(start))
		}

		// Handle async operations (202 Accepted) when requested.
		if resp.StatusCode == http.StatusAccepted && allowWait && c.WaitForAsync {
			location := strings.TrimSpace(resp.Header.Get("Location"))
			if location != "" {
				return c.waitForAsync(ctx, location, resp.Header)
			}
		}

		// Handle 429 rate limiting with exponential backoff (idempotent only)
		if resp.StatusCode == 429 {
			retryAfter, hasRetryAfter := retryAfterDuration(resp.Header)
			baseDelay := c.RetryConfig.RateLimitBaseDelay
			if !isIdempotent {
				if hasRetryAfter {
					return nil, nil, resp.StatusCode, &RateLimitError{RetryAfter: retryAfter}
				}
				return nil, nil, resp.StatusCode, &RateLimitError{RetryAfter: baseDelay}
			}
			if retries429 >= c.RetryConfig.MaxRateLimitRetries {
				if hasRetryAfter {
					return nil, nil, resp.StatusCode, &RateLimitError{RetryAfter: retryAfter}
				}
				return nil, nil, resp.StatusCode, &RateLimitError{RetryAfter: baseDelay}
			}
			delay := retryAfter
			if !hasRetryAfter {
				delay = baseDelay * time.Duration(1<<retries429)
			}
			slog.Info("rate limited, retrying", "delay", delay, "attempt", retries429+1)
			if err := sleepWithContext(ctx, delay); err != nil {
				return nil, nil, 0, err
			}
			retries429++
			continue
		}

		// Handle 5xx server errors
		if resp.StatusCode >= 500 {
			if c.circuitBreaker != nil {
				c.circuitBreaker.recordFailure()
			}
			if isIdempotent && retries5xx < c.RetryConfig.Max5xxRetries {
				slog.Info("server error, retrying", "status", resp.StatusCode)
				if err := sleepWithContext(ctx, c.RetryConfig.ServerErrorRetryDelay); err != nil {
					return nil, nil, 0, err
				}
				retries5xx++
				continue
			}
		}

		// Handle other 4xx errors - return body and headers for debugging
		if resp.StatusCode >= 400 {
			return respBody, resp.Header, resp.StatusCode, &APIError{
				StatusCode: resp.StatusCode,
				Body:       sanitizeErrorBody(string(respBody)),
				RequestID:  requestIDFromHeader(resp.Header),
			}
		}

		// Success (2xx) - record to circuit breaker
		if resp.StatusCode >= 200 && resp.StatusCode < 300 && c.circuitBreaker != nil {
			c.circuitBreaker.recordSuccess()
		}

		return respBody, resp.Header, resp.StatusCode, nil
	}
}

// doRaw performs an HTTP request and returns the raw response body
func (c *Client) doRaw(ctx context.Context, method, url string, body any) ([]byte, error) {
	respBody, _, _, err := c.executeRequest(ctx, method, url, body)
	return respBody, err
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, path string, result any) error {
	return c.do(ctx, http.MethodGet, c.accountPath(path), nil, result)
}

// GetRaw performs a GET request and returns the raw response body
func (c *Client) GetRaw(ctx context.Context, path string) ([]byte, error) {
	return c.doRaw(ctx, http.MethodGet, c.accountPath(path), nil)
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, path string, body any, result any) error {
	return c.do(ctx, http.MethodPost, c.accountPath(path), body, result)
}

// Patch performs a PATCH request
func (c *Client) Patch(ctx context.Context, path string, body any, result any) error {
	return c.do(ctx, http.MethodPatch, c.accountPath(path), body, result)
}

// Put performs a PUT request
func (c *Client) Put(ctx context.Context, path string, body any, result any) error {
	return c.do(ctx, http.MethodPut, c.accountPath(path), body, result)
}

// Delete performs a DELETE request
func (c *Client) Delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, c.accountPath(path), nil, nil)
}

// DeleteWithBody performs a DELETE request with a request body
func (c *Client) DeleteWithBody(ctx context.Context, path string, body any) error {
	return c.do(ctx, http.MethodDelete, c.accountPath(path), body, nil)
}

// DoRaw performs an HTTP request with the given method and path, returning raw response body,
// headers, and status code. This is designed for the raw API command that needs full response details.
// The path is relative to the account API path (e.g., "/conversations/123").
func (c *Client) DoRaw(ctx context.Context, method, path string, body any) ([]byte, http.Header, int, error) {
	url := c.accountPath(path)
	return c.executeRequest(ctx, method, url, body)
}

// PostMultipart performs a multipart POST request with files and form fields
func (c *Client) PostMultipart(ctx context.Context, path string, fields map[string]string, files map[string][]byte, result any) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return fmt.Errorf("failed to write field %s: %w", key, err)
		}
	}

	// Add files
	for filename, content := range files {
		part, err := writer.CreateFormFile("attachments[]", filename)
		if err != nil {
			return fmt.Errorf("failed to create form file %s: %w", filename, err)
		}
		if _, err := part.Write(content); err != nil {
			return fmt.Errorf("failed to write file content %s: %w", filename, err)
		}
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := c.accountPath(path)
	respBody, _, _, err := c.executeRequestWithBody(ctx, http.MethodPost, url, body.Bytes(), writer.FormDataContentType())
	if err != nil {
		return err
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unexpected API response format (JSON decode failed): %w", err)
		}
	}

	return nil
}

func requestIDFromHeader(header http.Header) string {
	if header == nil {
		return ""
	}
	if id := header.Get("X-Request-Id"); id != "" {
		return id
	}
	if id := header.Get("X-Request-ID"); id != "" {
		return id
	}
	return ""
}

// sanitizeErrorBody extracts safe error message from API response
// without exposing potentially sensitive data like tokens or user info
func sanitizeErrorBody(body string) string {
	// Try to extract error/message and validation errors from JSON response
	var errResp struct {
		Error   string      `json:"error"`
		Message string      `json:"message"`
		Errors  interface{} `json:"errors"` // Can be map[string]string or map[string][]string
	}
	if err := json.Unmarshal([]byte(body), &errResp); err != nil {
		// If we can't parse JSON, return generic message
		return "API request failed (response body redacted for security)"
	}

	// Extract field-specific validation errors if present
	validationErrors := formatValidationErrors(errResp.Errors)

	// Build result from error/message and validation errors
	var result string
	if errResp.Error != "" {
		result = errResp.Error
	} else if errResp.Message != "" {
		result = errResp.Message
	}

	// Append validation errors if present
	if validationErrors != "" {
		if result != "" {
			return result + "\nValidation errors:\n" + validationErrors
		}
		return "Validation errors:\n" + validationErrors
	}

	if result != "" {
		return result
	}

	// No recognized fields found
	return "API request failed (response body redacted for security)"
}

// formatValidationErrors formats the errors field from API validation responses.
// Handles both map[string]string and map[string][]string formats.
func formatValidationErrors(errors interface{}) string {
	if errors == nil {
		return ""
	}

	errMap, ok := errors.(map[string]interface{})
	if !ok {
		return ""
	}

	if len(errMap) == 0 {
		return ""
	}

	// Collect formatted field errors
	var lines []string
	for field, value := range errMap {
		switch v := value.(type) {
		case string:
			// Format: {"errors": {"email": "is invalid"}}
			lines = append(lines, fmt.Sprintf("  %s: %s", field, v))
		case []interface{}:
			// Format: {"errors": {"email": ["is invalid", "can't be blank"]}}
			for _, msg := range v {
				if msgStr, ok := msg.(string); ok {
					lines = append(lines, fmt.Sprintf("  %s: %s", field, msgStr))
				}
			}
		}
	}

	if len(lines) == 0 {
		return ""
	}

	// Sort for consistent output (important for testing)
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// APIError represents an error response from the API
type APIError struct {
	StatusCode int
	Body       string
	RequestID  string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Body)
}

// HealthCheck checks if the Chatwoot server is reachable via GET /health.
// Returns true if the server responds with 200, false otherwise.
func (c *Client) HealthCheck(ctx context.Context) (bool, error) {
	reqURL := c.BaseURL + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return false, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode == http.StatusOK, nil
}
