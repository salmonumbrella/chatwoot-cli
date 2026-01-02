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
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
)

const defaultTimeout = 30 * time.Second

// Client is the Chatwoot API client
type Client struct {
	BaseURL           string
	APIToken          string
	AccountID         int
	HTTP              *http.Client
	skipURLValidation bool            // internal flag for testing only
	circuitBreaker    *circuitBreaker // circuit breaker for retry logic
}

// New creates a new Chatwoot API client
func New(baseURL, token string, accountID int) *Client {
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false,
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	// Allow localhost URLs when CHATWOOT_TESTING=1 is set (for integration tests)
	skipValidation := os.Getenv("CHATWOOT_TESTING") == "1"

	return &Client{
		BaseURL:           baseURL,
		APIToken:          token,
		AccountID:         accountID,
		skipURLValidation: skipValidation,
		HTTP: &http.Client{
			Timeout:   defaultTimeout,
			Transport: transport,
		},
		circuitBreaker: &circuitBreaker{},
	}
}

// newTestClient creates a client with URL validation disabled for testing
func newTestClient(baseURL, token string, accountID int) *Client {
	c := New(baseURL, token, accountID)
	c.skipURLValidation = true
	return c
}

// accountPath returns the base path for account-scoped API calls
func (c *Client) accountPath(path string) string {
	return fmt.Sprintf("%s/api/v1/accounts/%d%s", c.BaseURL, c.AccountID, path)
}

// platformPath returns the base path for platform API calls
func (c *Client) platformPath(path string) string {
	return fmt.Sprintf("%s/platform/api/v1%s", c.BaseURL, path)
}

// publicPath returns the base path for public client API calls
func (c *Client) publicPath(path string) string {
	return fmt.Sprintf("%s/public/api/v1%s", c.BaseURL, path)
}

// do performs an HTTP request and decodes the response
func (c *Client) do(ctx context.Context, method, url string, body any, result any) error {
	// Check circuit breaker at start
	if c.circuitBreaker != nil && c.circuitBreaker.isOpen() {
		return &CircuitBreakerError{}
	}

	// Validate BaseURL at request time to prevent DNS rebinding attacks
	// Skip validation in tests to allow httptest.Server localhost URLs
	if !c.skipURLValidation {
		if err := validation.ValidateChatwootURL(c.BaseURL); err != nil {
			return fmt.Errorf("URL validation failed: %w", err)
		}
	}

	// Marshal body to JSON once (will be reused for retries)
	var jsonBody []byte
	if body != nil {
		var err error
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	// Determine if method is idempotent for retry logic
	isIdempotent := method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions

	var retries429, retries5xx int

	for {
		// Create fresh body reader for each attempt
		var bodyReader io.Reader
		if jsonBody != nil {
			bodyReader = bytes.NewReader(jsonBody)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		if c.APIToken != "" {
			req.Header.Set("api_access_token", c.APIToken)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := c.HTTP.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		// Handle 429 rate limiting with exponential backoff
		if resp.StatusCode == 429 {
			if retries429 >= MaxRateLimitRetries {
				return &RateLimitError{RetryAfter: RateLimitBaseDelay}
			}
			delay := RateLimitBaseDelay * time.Duration(1<<retries429)
			slog.Info("rate limited, retrying", "delay", delay, "attempt", retries429+1)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
			retries429++
			continue
		}

		// Handle 5xx server errors
		if resp.StatusCode >= 500 {
			if c.circuitBreaker != nil {
				c.circuitBreaker.recordFailure()
			}
			if isIdempotent && retries5xx < Max5xxRetries {
				slog.Info("server error, retrying", "status", resp.StatusCode)
				time.Sleep(ServerErrorRetryDelay)
				retries5xx++
				continue
			}
		}

		// Handle other 4xx errors
		if resp.StatusCode >= 400 {
			return &APIError{
				StatusCode: resp.StatusCode,
				Body:       sanitizeErrorBody(string(respBody)),
			}
		}

		// Success (2xx) - record to circuit breaker
		if resp.StatusCode >= 200 && resp.StatusCode < 300 && c.circuitBreaker != nil {
			c.circuitBreaker.recordSuccess()
		}

		if result != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("unexpected API response format (JSON decode failed): %w", err)
			}
		}

		return nil
	}
}

// doRaw performs an HTTP request and returns the raw response body
func (c *Client) doRaw(ctx context.Context, method, url string, body any) ([]byte, error) {
	// Check circuit breaker at start
	if c.circuitBreaker != nil && c.circuitBreaker.isOpen() {
		return nil, &CircuitBreakerError{}
	}

	// Validate BaseURL at request time to prevent DNS rebinding attacks
	// Skip validation in tests to allow httptest.Server localhost URLs
	if !c.skipURLValidation {
		if err := validation.ValidateChatwootURL(c.BaseURL); err != nil {
			return nil, fmt.Errorf("URL validation failed: %w", err)
		}
	}

	// Marshal body to JSON once (will be reused for retries)
	var jsonBody []byte
	if body != nil {
		var err error
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	// Determine if method is idempotent for retry logic
	isIdempotent := method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions

	var retries429, retries5xx int

	for {
		// Create fresh body reader for each attempt
		var bodyReader io.Reader
		if jsonBody != nil {
			bodyReader = bytes.NewReader(jsonBody)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		if c.APIToken != "" {
			req.Header.Set("api_access_token", c.APIToken)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := c.HTTP.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		// Handle 429 rate limiting with exponential backoff
		if resp.StatusCode == 429 {
			if retries429 >= MaxRateLimitRetries {
				return nil, &RateLimitError{RetryAfter: RateLimitBaseDelay}
			}
			delay := RateLimitBaseDelay * time.Duration(1<<retries429)
			slog.Info("rate limited, retrying", "delay", delay, "attempt", retries429+1)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			retries429++
			continue
		}

		// Handle 5xx server errors
		if resp.StatusCode >= 500 {
			if c.circuitBreaker != nil {
				c.circuitBreaker.recordFailure()
			}
			if isIdempotent && retries5xx < Max5xxRetries {
				slog.Info("server error, retrying", "status", resp.StatusCode)
				time.Sleep(ServerErrorRetryDelay)
				retries5xx++
				continue
			}
		}

		// Handle other 4xx errors
		if resp.StatusCode >= 400 {
			return nil, &APIError{
				StatusCode: resp.StatusCode,
				Body:       sanitizeErrorBody(string(respBody)),
			}
		}

		// Success (2xx) - record to circuit breaker
		if resp.StatusCode >= 200 && resp.StatusCode < 300 && c.circuitBreaker != nil {
			c.circuitBreaker.recordSuccess()
		}

		return respBody, nil
	}
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
	// Check circuit breaker at start
	if c.circuitBreaker != nil && c.circuitBreaker.isOpen() {
		return nil, nil, 0, &CircuitBreakerError{}
	}

	// Validate BaseURL at request time to prevent DNS rebinding attacks
	// Skip validation in tests to allow httptest.Server localhost URLs
	if !c.skipURLValidation {
		if err := validation.ValidateChatwootURL(c.BaseURL); err != nil {
			return nil, nil, 0, fmt.Errorf("URL validation failed: %w", err)
		}
	}

	// Marshal body to JSON once (will be reused for retries)
	var jsonBody []byte
	if body != nil {
		var err error
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	// Determine if method is idempotent for retry logic
	isIdempotent := method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions

	var retries429, retries5xx int

	for {
		// Create fresh body reader for each attempt
		var bodyReader io.Reader
		if jsonBody != nil {
			bodyReader = bytes.NewReader(jsonBody)
		}

		url := c.accountPath(path)
		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to create request: %w", err)
		}

		if c.APIToken != "" {
			req.Header.Set("api_access_token", c.APIToken)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := c.HTTP.Do(req)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("request failed: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to read response: %w", err)
		}

		// Handle 429 rate limiting with exponential backoff
		if resp.StatusCode == 429 {
			if retries429 >= MaxRateLimitRetries {
				return nil, nil, resp.StatusCode, &RateLimitError{RetryAfter: RateLimitBaseDelay}
			}
			delay := RateLimitBaseDelay * time.Duration(1<<retries429)
			slog.Info("rate limited, retrying", "delay", delay, "attempt", retries429+1)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, nil, 0, ctx.Err()
			}
			retries429++
			continue
		}

		// Handle 5xx server errors
		if resp.StatusCode >= 500 {
			if c.circuitBreaker != nil {
				c.circuitBreaker.recordFailure()
			}
			if isIdempotent && retries5xx < Max5xxRetries {
				slog.Info("server error, retrying", "status", resp.StatusCode)
				time.Sleep(ServerErrorRetryDelay)
				retries5xx++
				continue
			}
		}

		// Handle other 4xx errors - return as error but still include body for debugging
		if resp.StatusCode >= 400 {
			return respBody, resp.Header, resp.StatusCode, &APIError{
				StatusCode: resp.StatusCode,
				Body:       sanitizeErrorBody(string(respBody)),
			}
		}

		// Success (2xx) - record to circuit breaker
		if resp.StatusCode >= 200 && resp.StatusCode < 300 && c.circuitBreaker != nil {
			c.circuitBreaker.recordSuccess()
		}

		return respBody, resp.Header, resp.StatusCode, nil
	}
}

// PostMultipart performs a multipart POST request with files and form fields
func (c *Client) PostMultipart(ctx context.Context, path string, fields map[string]string, files map[string][]byte, result any) error {
	if !c.skipURLValidation {
		if err := validation.ValidateChatwootURL(c.BaseURL); err != nil {
			return fmt.Errorf("URL validation failed: %w", err)
		}
	}

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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("api_access_token", c.APIToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Body:       sanitizeErrorBody(string(respBody)),
		}
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unexpected API response format (JSON decode failed): %w", err)
		}
	}

	return nil
}

// sanitizeErrorBody extracts safe error message from API response
// without exposing potentially sensitive data like tokens or user info
func sanitizeErrorBody(body string) string {
	// Try to extract just the error/message field from JSON response
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(body), &errResp); err == nil {
		if errResp.Error != "" {
			return errResp.Error
		}
		if errResp.Message != "" {
			return errResp.Message
		}
	}
	// If we can't parse JSON or no error field found, return generic message
	return "API request failed (response body redacted for security)"
}

// APIError represents an error response from the API
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Body)
}
