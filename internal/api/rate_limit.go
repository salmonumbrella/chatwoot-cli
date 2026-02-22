package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// unixTimestampThreshold is used to distinguish Unix timestamps from relative seconds.
// Values above this (1 billion) are interpreted as Unix timestamps (dates after 2001-09-09).
// Values below are interpreted as seconds from now.
const unixTimestampThreshold = 1_000_000_000

// RateLimitInfo holds parsed rate limit header values.
type RateLimitInfo struct {
	Limit     *int
	Remaining *int
	ResetAt   *time.Time
	ResetRaw  string
}

// Meta returns a JSON-ready map for CLI output metadata.
func (r *RateLimitInfo) Meta() map[string]any {
	if r == nil {
		return nil
	}
	meta := map[string]any{}
	if r.Limit != nil {
		meta["limit"] = *r.Limit
	}
	if r.Remaining != nil {
		meta["remaining"] = *r.Remaining
	}
	if r.ResetAt != nil {
		meta["reset_at"] = r.ResetAt.UTC().Format(time.RFC3339)
	} else if r.ResetRaw != "" {
		meta["reset"] = r.ResetRaw
	}
	if len(meta) == 0 {
		return nil
	}
	return meta
}

// LastRateLimit returns the most recent rate limit info seen by the client.
func (c *Client) LastRateLimit() *RateLimitInfo {
	c.rateLimitMu.Lock()
	defer c.rateLimitMu.Unlock()
	if c.lastRateLimit == nil {
		return nil
	}
	copyInfo := *c.lastRateLimit
	if c.lastRateLimit.Limit != nil {
		v := *c.lastRateLimit.Limit
		copyInfo.Limit = &v
	}
	if c.lastRateLimit.Remaining != nil {
		v := *c.lastRateLimit.Remaining
		copyInfo.Remaining = &v
	}
	if c.lastRateLimit.ResetAt != nil {
		t := *c.lastRateLimit.ResetAt
		copyInfo.ResetAt = &t
	}
	return &copyInfo
}

// SetRateLimitInfo sets rate limit info (primarily for tests).
func (c *Client) SetRateLimitInfo(info *RateLimitInfo) {
	c.rateLimitMu.Lock()
	defer c.rateLimitMu.Unlock()
	c.lastRateLimit = info
}

func (c *Client) recordRateLimit(h http.Header) {
	info := parseRateLimitInfo(h, time.Now())
	c.rateLimitMu.Lock()
	defer c.rateLimitMu.Unlock()
	c.lastRateLimit = info
}

func parseRateLimitInfo(h http.Header, now time.Time) *RateLimitInfo {
	if h == nil {
		return nil
	}
	limitVal := firstHeader(h, "X-RateLimit-Limit", "RateLimit-Limit")
	remainingVal := firstHeader(h, "X-RateLimit-Remaining", "RateLimit-Remaining")
	resetVal := firstHeader(h, "X-RateLimit-Reset", "RateLimit-Reset")

	if limitVal == "" && remainingVal == "" && resetVal == "" {
		return nil
	}

	info := &RateLimitInfo{}
	if limitVal != "" {
		if v, err := strconv.Atoi(limitVal); err == nil {
			info.Limit = &v
		}
	}
	if remainingVal != "" {
		if v, err := strconv.Atoi(remainingVal); err == nil {
			info.Remaining = &v
		}
	}
	if resetVal != "" {
		info.ResetRaw = resetVal
		if t, ok := parseRateLimitReset(resetVal, now); ok {
			info.ResetAt = &t
		}
	}

	if info.Limit == nil && info.Remaining == nil && info.ResetAt == nil && info.ResetRaw == "" {
		return nil
	}
	return info
}

func firstHeader(h http.Header, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(h.Get(key))
		if value != "" {
			return value
		}
	}
	return ""
}

func parseRateLimitReset(value string, now time.Time) (time.Time, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, false
	}
	if secs, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		switch {
		case secs > unixTimestampThreshold:
			return time.Unix(secs, 0).UTC(), true
		case secs >= 0:
			return now.Add(time.Duration(secs) * time.Second).UTC(), true
		}
	}
	if t, err := http.ParseTime(trimmed); err == nil {
		return t.UTC(), true
	}
	return time.Time{}, false
}
