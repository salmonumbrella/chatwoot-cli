package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestContactsCreateFromMap(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/contacts" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"payload":{"contact":{"id":7,"name":"Map User","email":"map@example.com","created_at":1700000000},"contact_inbox":{}}}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "token", 1)
	contact, err := client.Contacts().CreateFromMap(context.Background(), map[string]any{
		"name":  "Map User",
		"email": "map@example.com",
		"custom_attributes": map[string]any{
			"tier": "vip",
		},
	})
	if err != nil {
		t.Fatalf("CreateFromMap error: %v", err)
	}
	if contact.ID != 7 || contact.Name != "Map User" {
		t.Fatalf("unexpected contact: %+v", contact)
	}
	if captured["name"] != "Map User" {
		t.Fatalf("expected name in request body, got %#v", captured)
	}
	attrs, ok := captured["custom_attributes"].(map[string]any)
	if !ok || attrs["tier"] != "vip" {
		t.Fatalf("expected custom_attributes.tier=vip, got %#v", captured["custom_attributes"])
	}
}

func TestContextServiceGetConversationWrapper(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/accounts/1/conversations/321":
			_, _ = w.Write([]byte(`{"id":321,"inbox_id":2,"status":"open","contact_id":44,"created_at":1700000000}`))
		case r.URL.Path == "/api/v1/accounts/1/conversations/321/messages" && r.URL.Query().Get("before") == "":
			_, _ = w.Write([]byte(`{"payload":[{"id":9,"conversation_id":321,"content":"hello","message_type":0,"private":false,"created_at":1700000001}]}`))
		case r.URL.Path == "/api/v1/accounts/1/conversations/321/messages" && r.URL.Query().Get("before") == "9":
			_, _ = w.Write([]byte(`{"payload":[]}`))
		case r.URL.Path == "/api/v1/accounts/1/contacts/44":
			_, _ = w.Write([]byte(`{"payload":{"id":44,"name":"Customer","email":"customer@example.com","created_at":1699999000}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, "token", 1)
	ctx, err := client.Context().GetConversation(context.Background(), 321, false)
	if err != nil {
		t.Fatalf("Context().GetConversation error: %v", err)
	}
	if ctx == nil || ctx.Conversation == nil || ctx.Conversation.ID != 321 {
		t.Fatalf("unexpected conversation context: %#v", ctx)
	}
	if ctx.Contact == nil || ctx.Contact.ID != 44 {
		t.Fatalf("expected contact in context, got %#v", ctx.Contact)
	}
	if len(ctx.Messages) != 1 || ctx.Messages[0].ID != 9 {
		t.Fatalf("unexpected messages: %#v", ctx.Messages)
	}
}

func TestConversationsMuteAndUnmute(t *testing.T) {
	tests := []struct {
		name        string
		run         func(*Client) error
		expectedURL string
	}{
		{
			name: "mute",
			run: func(c *Client) error {
				return c.Conversations().Mute(context.Background(), 88)
			},
			expectedURL: "/api/v1/accounts/1/conversations/88/mute",
		},
		{
			name: "unmute",
			run: func(c *Client) error {
				return c.Conversations().Unmute(context.Background(), 88)
			},
			expectedURL: "/api/v1/accounts/1/conversations/88/unmute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != tt.expectedURL {
					t.Fatalf("path = %s, want %s", r.URL.Path, tt.expectedURL)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := newTestClient(server.URL, "token", 1)
			if err := tt.run(client); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestMessagesListWithLimit(t *testing.T) {
	t.Run("invalid limit", func(t *testing.T) {
		client := newTestClient("https://example.com", "token", 1)
		_, err := client.Messages().ListWithLimit(context.Background(), 11, 0, 5)
		if err == nil {
			t.Fatal("expected error for invalid limit")
		}
		if !strings.Contains(err.Error(), "limit must be > 0") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns limited first page", func(t *testing.T) {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.URL.Query().Get("before") != "" {
				t.Fatalf("did not expect paginated call, got before=%s", r.URL.Query().Get("before"))
			}
			_, _ = w.Write([]byte(`{"payload":[{"id":5,"conversation_id":11,"content":"a","message_type":0,"private":false,"created_at":1700},{"id":4,"conversation_id":11,"content":"b","message_type":0,"private":false,"created_at":1701},{"id":3,"conversation_id":11,"content":"c","message_type":0,"private":false,"created_at":1702}]}`))
		}))
		defer server.Close()

		client := newTestClient(server.URL, "token", 1)
		msgs, err := client.Messages().ListWithLimit(context.Background(), 11, 2, 5)
		if err != nil {
			t.Fatalf("ListWithLimit error: %v", err)
		}
		if len(msgs) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(msgs))
		}
		if calls != 1 {
			t.Fatalf("expected 1 API call, got %d", calls)
		}
	})

	t.Run("pagination limit exceeded", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"payload":[{"id":10,"conversation_id":11,"content":"x","message_type":0,"private":false,"created_at":1700}]}`))
		}))
		defer server.Close()

		client := newTestClient(server.URL, "token", 1)
		_, err := client.Messages().ListWithLimit(context.Background(), 11, 5, 1)
		if err == nil {
			t.Fatal("expected pagination limit error")
		}
		if !strings.Contains(err.Error(), "pagination limit exceeded") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestRateLimitHelpers(t *testing.T) {
	t.Run("meta and client copy", func(t *testing.T) {
		var nilInfo *RateLimitInfo
		if nilInfo.Meta() != nil {
			t.Fatal("nil RateLimitInfo should return nil meta")
		}

		limit := 100
		remaining := 42
		reset := time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC)
		info := &RateLimitInfo{Limit: &limit, Remaining: &remaining, ResetAt: &reset}
		meta := info.Meta()
		if meta["limit"] != 100 || meta["remaining"] != 42 {
			t.Fatalf("unexpected meta: %#v", meta)
		}

		client := newTestClient("https://example.com", "token", 1)
		client.SetRateLimitInfo(info)
		copyInfo := client.LastRateLimit()
		if copyInfo == nil {
			t.Fatal("expected LastRateLimit copy")
		}
		*copyInfo.Limit = 1
		copyInfo.ResetAt = nil

		again := client.LastRateLimit()
		if again == nil || again.Limit == nil || *again.Limit != 100 {
			t.Fatalf("expected deep copy behavior, got %#v", again)
		}
		if again.ResetAt == nil {
			t.Fatal("expected original ResetAt unchanged")
		}

		client.SetRateLimitInfo(nil)
		if client.LastRateLimit() != nil {
			t.Fatal("expected nil last rate limit after reset")
		}
	})

	t.Run("parse rate limit info and reset", func(t *testing.T) {
		now := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)

		h := http.Header{}
		h.Set("X-RateLimit-Limit", "200")
		h.Set("X-RateLimit-Remaining", "55")
		h.Set("X-RateLimit-Reset", "60")
		info := parseRateLimitInfo(h, now)
		if info == nil || info.Limit == nil || *info.Limit != 200 {
			t.Fatalf("unexpected parsed limit info: %#v", info)
		}
		if info.Remaining == nil || *info.Remaining != 55 {
			t.Fatalf("unexpected remaining: %#v", info)
		}
		if info.ResetAt == nil || !info.ResetAt.Equal(now.Add(60*time.Second)) {
			t.Fatalf("unexpected reset time: %#v", info)
		}

		unix := now.Add(5 * time.Minute).Unix()
		parsedUnix, ok := parseRateLimitReset(fmt.Sprintf("%d", unix), now)
		if !ok || !parsedUnix.Equal(time.Unix(unix, 0).UTC()) {
			t.Fatalf("parseRateLimitReset unix failed: %v %v", parsedUnix, ok)
		}

		httpDate := now.Add(10 * time.Minute).Format(http.TimeFormat)
		parsedHTTP, ok := parseRateLimitReset(httpDate, now)
		if !ok || !parsedHTTP.Equal(now.Add(10*time.Minute).UTC()) {
			t.Fatalf("parseRateLimitReset http-date failed: %v %v", parsedHTTP, ok)
		}

		if _, ok := parseRateLimitReset("invalid", now); ok {
			t.Fatal("expected invalid reset parse to fail")
		}
		if parseRateLimitInfo(nil, now) != nil {
			t.Fatal("expected nil header parse to return nil")
		}
		if parseRateLimitInfo(http.Header{}, now) != nil {
			t.Fatal("expected empty header parse to return nil")
		}
	})
}

func TestFlexTypesUnmarshalJSON(t *testing.T) {
	var fi FlexInt
	if err := json.Unmarshal([]byte(`42`), &fi); err != nil || int(fi) != 42 {
		t.Fatalf("FlexInt int parse failed: %v, %d", err, fi)
	}
	if err := json.Unmarshal([]byte(`"7"`), &fi); err != nil || int(fi) != 7 {
		t.Fatalf("FlexInt string parse failed: %v, %d", err, fi)
	}
	if err := json.Unmarshal([]byte(`""`), &fi); err != nil || int(fi) != 0 {
		t.Fatalf("FlexInt empty string parse failed: %v, %d", err, fi)
	}
	if err := json.Unmarshal([]byte(`"bad"`), &fi); err == nil {
		t.Fatal("expected FlexInt invalid parse error")
	}

	var ff FlexFloat
	if err := json.Unmarshal([]byte(`1.5`), &ff); err != nil || float64(ff) != 1.5 {
		t.Fatalf("FlexFloat float parse failed: %v, %f", err, ff)
	}
	if err := json.Unmarshal([]byte(`"2.75"`), &ff); err != nil || float64(ff) != 2.75 {
		t.Fatalf("FlexFloat string parse failed: %v, %f", err, ff)
	}
	if err := json.Unmarshal([]byte(`""`), &ff); err != nil || float64(ff) != 0 {
		t.Fatalf("FlexFloat empty string parse failed: %v, %f", err, ff)
	}
	if err := json.Unmarshal([]byte(`null`), &ff); err != nil {
		t.Fatalf("FlexFloat null parse failed: %v", err)
	}
	if err := json.Unmarshal([]byte(`"bad"`), &ff); err == nil {
		t.Fatal("expected FlexFloat invalid parse error")
	}

	var fs FlexString
	if err := json.Unmarshal([]byte(`"hello"`), &fs); err != nil || fs.String() != "hello" {
		t.Fatalf("FlexString string parse failed: %v, %q", err, fs)
	}
	if err := json.Unmarshal([]byte(`42`), &fs); err != nil || fs.String() != "42" {
		t.Fatalf("FlexString numeric parse failed: %v, %q", err, fs)
	}
	if err := json.Unmarshal([]byte(`1.25`), &fs); err != nil || fs.String() != "1.25" {
		t.Fatalf("FlexString float parse failed: %v, %q", err, fs)
	}
	if err := json.Unmarshal([]byte(`{}`), &fs); err == nil {
		t.Fatal("expected FlexString invalid parse error")
	}
}
