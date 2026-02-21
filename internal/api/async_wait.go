package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// maxAsyncWaitIterations is a safety limit to prevent infinite loops in the async wait loop.
// At the default 1s interval, this allows ~16 minutes of waiting before giving up.
const maxAsyncWaitIterations = 1000

func (c *Client) waitForAsync(ctx context.Context, location string, headers http.Header) ([]byte, http.Header, int, error) {
	asyncURL, err := c.resolveAsyncURL(location)
	if err != nil {
		return nil, nil, 0, err
	}
	waitCtx, cancel := withOptionalTimeout(ctx, c.WaitTimeout)
	defer cancel()

	delay := c.waitDelay(headers)

	for iteration := 0; iteration < maxAsyncWaitIterations; iteration++ {
		if err := sleepWithContext(waitCtx, delay); err != nil {
			return nil, nil, 0, err
		}

		respBody, respHeader, status, err := c.executeRequestWithBodyInternal(waitCtx, http.MethodGet, asyncURL, nil, "", false)
		if err != nil {
			// Normalize timeout/cancellation to the canonical context errors so callers
			// (and tests) can reliably check equality without relying on error wrapping.
			if errors.Is(err, context.DeadlineExceeded) {
				return nil, respHeader, status, context.DeadlineExceeded
			}
			if errors.Is(err, context.Canceled) {
				return nil, respHeader, status, context.Canceled
			}
			return nil, respHeader, status, err
		}
		if status == http.StatusAccepted {
			delay = c.waitDelay(respHeader)
			continue
		}
		return respBody, respHeader, status, nil
	}

	return nil, nil, 0, fmt.Errorf("async wait exceeded maximum iterations (%d); operation may still be in progress", maxAsyncWaitIterations)
}

func (c *Client) waitDelay(headers http.Header) time.Duration {
	if delay, ok := retryAfterDuration(headers); ok {
		return delay
	}
	if c.WaitInterval > 0 {
		return c.WaitInterval
	}
	return DefaultWaitInterval
}

func (c *Client) resolveAsyncURL(location string) (string, error) {
	trimmed := strings.TrimSpace(location)
	if trimmed == "" {
		return "", fmt.Errorf("async wait location is empty")
	}

	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}
	loc, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid async wait location %q: %w", trimmed, err)
	}
	if loc.IsAbs() {
		if !sameHost(base, loc) {
			return "", fmt.Errorf("async wait location host mismatch: %s", loc.Host)
		}
		return loc.String(), nil
	}
	return base.ResolveReference(loc).String(), nil
}

func sameHost(a, b *url.URL) bool {
	if !strings.EqualFold(a.Scheme, b.Scheme) {
		return false
	}
	if !strings.EqualFold(a.Hostname(), b.Hostname()) {
		return false
	}
	return effectivePort(a) == effectivePort(b)
}

func effectivePort(u *url.URL) string {
	if u == nil {
		return ""
	}
	if port := u.Port(); port != "" {
		return port
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return "443"
	case "http":
		return "80"
	default:
		return ""
	}
}

func withOptionalTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 || remaining <= timeout {
			return ctx, func() {}
		}
	}
	return context.WithTimeout(ctx, timeout)
}
