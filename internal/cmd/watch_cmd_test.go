package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestWatchServe_SSEReceivesWebhookEvent(t *testing.T) {
	t.Setenv("CHATWOOT_TESTING", "1") // allow httptest backend URL

	backendToken := "agent-token"
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Authorization call done by watch server: GET /api/v1/accounts/1/conversations/123
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/accounts/1/conversations/123" {
			http.NotFound(w, r)
			return
		}
		if strings.TrimSpace(r.Header.Get("api_access_token")) != backendToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":123,"account_id":1,"inbox_id":1,"status":"open","created_at":1700000000,"last_activity_at":1700000000,"unread_count":0}`))
	}))
	defer backend.Close()

	cfg := watchServeConfig{
		HookPath:         "/hooks/chatwoot",
		WatchPath:        "/watch",
		HookToken:        "hooktok",
		BackendURL:       backend.URL,
		AccountID:        1,
		MaxHookBodyBytes: 1024 * 1024,
	}
	s := newWatchServer(cfg)

	srv := httptest.NewServer(http.HandlerFunc(s.serveHTTP))
	defer srv.Close()

	// Open SSE.
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/watch/conversations/123", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("api_access_token", backendToken)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %s", resp.Status)
	}

	// Send webhook.
	payload := map[string]any{
		"event":        "message_created",
		"id":           555,
		"content":      "hello",
		"content_type": "text",
		"message_type": "incoming",
		"private":      false,
		"created_at":   "2026-02-06T00:00:00Z",
		"sender":       map[string]any{"name": "Customer"},
		"contact":      map[string]any{"id": 9, "name": "Customer"},
		"conversation": map[string]any{"id": 123},
	}
	b, _ := json.Marshal(payload)
	hookReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/hooks/chatwoot?token=hooktok", bytes.NewReader(b))
	hookReq.Header.Set("Content-Type", "application/json")
	hookResp, err := srv.Client().Do(hookReq)
	if err != nil {
		t.Fatal(err)
	}
	_ = hookResp.Body.Close()
	if hookResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected hook status: %s", hookResp.Status)
	}

	// Read SSE until we see data.
	reader := bufio.NewReader(resp.Body)
	deadline := time.Now().Add(2 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for SSE event")
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(line, "data:") {
			raw := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			var ev watchEvent
			if err := json.Unmarshal([]byte(raw), &ev); err != nil {
				t.Fatalf("invalid json: %v (%q)", err, raw)
			}
			if ev.ConversationID != 123 || ev.MessageID != 555 || ev.MessageType != "incoming" {
				t.Fatalf("unexpected event: %+v", ev)
			}
			return
		}
	}
}

func TestWatchServe_SSEReplayWithLastEventID(t *testing.T) {
	t.Setenv("CHATWOOT_TESTING", "1") // allow httptest backend URL

	mr := miniredis.RunT(t)

	backendToken := "agent-token"
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/accounts/1/conversations/123" {
			http.NotFound(w, r)
			return
		}
		if strings.TrimSpace(r.Header.Get("api_access_token")) != backendToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":123,"account_id":1,"inbox_id":1,"status":"open","created_at":1700000000,"last_activity_at":1700000000,"unread_count":0}`))
	}))
	defer backend.Close()

	cfg := watchServeConfig{
		HookPath:         "/hooks/chatwoot",
		WatchPath:        "/watch",
		HookToken:        "hooktok",
		BackendURL:       backend.URL,
		AccountID:        1,
		MaxHookBodyBytes: 1024 * 1024,
		RedisURL:         "redis://" + mr.Addr(),
		RedisPrefix:      "test-watch",
		BufferSize:       50,
		BufferTTL:        10 * time.Minute,
	}

	// Publish two webhook events before SSE connects.
	sendHook := func(baseURL string, id int) {
		payload := map[string]any{
			"event":        "message_created",
			"id":           id,
			"content":      "hello",
			"content_type": "text",
			"message_type": "incoming",
			"private":      false,
			"created_at":   "2026-02-06T00:00:00Z",
			"sender":       map[string]any{"name": "Customer"},
			"contact":      map[string]any{"id": 9, "name": "Customer"},
			"conversation": map[string]any{"id": 123},
		}
		b, _ := json.Marshal(payload)
		hookReq, _ := http.NewRequest(http.MethodPost, baseURL+"/hooks/chatwoot?token=hooktok", bytes.NewReader(b))
		hookReq.Header.Set("Content-Type", "application/json")
		hookResp, err := http.DefaultClient.Do(hookReq)
		if err != nil {
			t.Fatal(err)
		}
		_ = hookResp.Body.Close()
		if hookResp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected hook status: %s", hookResp.Status)
		}
	}

	// Start server instance #1, send hooks, then restart server to ensure replay is durable.
	s1 := newWatchServer(cfg)
	srv1 := httptest.NewServer(http.HandlerFunc(s1.serveHTTP))
	sendHook(srv1.URL, 555)
	sendHook(srv1.URL, 556)
	srv1.Close()

	// Server instance #2 replays from Redis.
	s2 := newWatchServer(cfg)
	srv2 := httptest.NewServer(http.HandlerFunc(s2.serveHTTP))
	defer srv2.Close()

	req, err := http.NewRequest(http.MethodGet, srv2.URL+"/watch/conversations/123", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("api_access_token", backendToken)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Last-Event-ID", "555")

	resp, err := srv2.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %s", resp.Status)
	}

	reader := bufio.NewReader(resp.Body)
	deadline := time.Now().Add(2 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for replay event")
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(line, "data:") {
			raw := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			var ev watchEvent
			if err := json.Unmarshal([]byte(raw), &ev); err != nil {
				t.Fatalf("invalid json: %v (%q)", err, raw)
			}
			if ev.MessageID != 556 {
				t.Fatalf("expected replay message 556, got %+v", ev)
			}
			return
		}
	}
}

func TestWatchServe_HookTokenRequired(t *testing.T) {
	cfg := watchServeConfig{
		HookPath:         "/hooks/chatwoot",
		WatchPath:        "/watch",
		HookToken:        "hooktok",
		BackendURL:       "https://example.com",
		AccountID:        1,
		MaxHookBodyBytes: 1024 * 1024,
	}
	s := newWatchServer(cfg)
	srv := httptest.NewServer(http.HandlerFunc(s.serveHTTP))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/hooks/chatwoot", strings.NewReader(`{}`))
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %s", resp.Status)
	}
}

func TestWatchServe_SSEUnauthorizedWithoutAPIToken(t *testing.T) {
	cfg := watchServeConfig{
		HookPath:         "/hooks/chatwoot",
		WatchPath:        "/watch",
		HookToken:        "hooktok",
		BackendURL:       "https://example.com",
		AccountID:        1,
		MaxHookBodyBytes: 1024 * 1024,
	}
	s := newWatchServer(cfg)
	srv := httptest.NewServer(http.HandlerFunc(s.serveHTTP))
	defer srv.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/watch/conversations/123", nil)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %s", resp.Status)
	}
}

func TestWatchServe_RedisPubSubFanoutAcrossInstances(t *testing.T) {
	t.Setenv("CHATWOOT_TESTING", "1")

	mr := miniredis.RunT(t)
	redisURL := "redis://" + mr.Addr()

	backendToken := "agent-token"
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/accounts/1/conversations/123" {
			http.NotFound(w, r)
			return
		}
		if strings.TrimSpace(r.Header.Get("api_access_token")) != backendToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":123,"account_id":1,"inbox_id":1,"status":"open","created_at":1700000000,"last_activity_at":1700000000,"unread_count":0}`))
	}))
	defer backend.Close()

	cfg := watchServeConfig{
		HookPath:         "/hooks/chatwoot",
		WatchPath:        "/watch",
		HookToken:        "hooktok",
		BackendURL:       backend.URL,
		AccountID:        1,
		MaxHookBodyBytes: 1024 * 1024,
		RedisURL:         redisURL,
		RedisPrefix:      "test-watch",
		BufferSize:       50,
		BufferTTL:        10 * time.Minute,
	}

	// Instance A (receives webhook)
	a := newWatchServer(cfg)
	ctxA, cancelA := context.WithCancel(context.Background())
	defer cancelA()
	a.Start(ctxA)
	srvA := httptest.NewServer(http.HandlerFunc(a.serveHTTP))
	defer srvA.Close()

	// Instance B (SSE client connects here; should get event via Redis Pub/Sub)
	b := newWatchServer(cfg)
	ctxB, cancelB := context.WithCancel(context.Background())
	defer cancelB()
	b.Start(ctxB)
	srvB := httptest.NewServer(http.HandlerFunc(b.serveHTTP))
	defer srvB.Close()

	// Open SSE connection to B first.
	req, _ := http.NewRequest(http.MethodGet, srvB.URL+"/watch/conversations/123", nil)
	req.Header.Set("api_access_token", backendToken)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := srvB.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %s", resp.Status)
	}

	// Send webhook to A.
	payload := map[string]any{
		"event":        "message_created",
		"id":           777,
		"content":      "hello",
		"content_type": "text",
		"message_type": "incoming",
		"private":      false,
		"created_at":   "2026-02-06T00:00:00Z",
		"sender":       map[string]any{"name": "Customer"},
		"contact":      map[string]any{"id": 9, "name": "Customer"},
		"conversation": map[string]any{"id": 123},
	}
	bb, _ := json.Marshal(payload)
	hookReq, _ := http.NewRequest(http.MethodPost, srvA.URL+"/hooks/chatwoot?token=hooktok", bytes.NewReader(bb))
	hookReq.Header.Set("Content-Type", "application/json")
	hookResp, err := srvA.Client().Do(hookReq)
	if err != nil {
		t.Fatal(err)
	}
	_ = hookResp.Body.Close()
	if hookResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected hook status: %s", hookResp.Status)
	}

	// Read SSE from B until we see the event.
	reader := bufio.NewReader(resp.Body)
	deadline := time.Now().Add(2 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for pubsub fanout event")
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(line, "data:") {
			raw := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			var ev watchEvent
			if err := json.Unmarshal([]byte(raw), &ev); err != nil {
				t.Fatalf("invalid json: %v (%q)", err, raw)
			}
			if ev.MessageID != 777 || ev.ConversationID != 123 {
				t.Fatalf("unexpected event: %+v", ev)
			}
			return
		}
	}
}
