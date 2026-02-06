package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"
)

func TestWatchSetup_CreatesWebhook(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/webhooks", jsonResponse(200, `{"payload":{"webhooks":[]}}`)).
		On("POST", "/api/v1/accounts/1/webhooks", jsonResponse(200, `{"payload":{"webhook":{"id":9,"url":"https://chatwoot.example/hooks/chatwoot?token=tok","subscriptions":["message_created"],"account_id":1}}}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"watch", "setup", "--url", "https://chatwoot.example/hooks/chatwoot", "--token", "tok", "--subscriptions", "message_created", "--output", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	out := buf.String()

	if err != nil {
		t.Fatalf("watch setup failed: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid json output: %v (%s)", err, out)
	}
	if payload["action"] != "created" {
		t.Fatalf("expected action created, got %v", payload["action"])
	}
}

func TestWatchSetup_UpdatesWebhookWhenTokenRotates(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/webhooks", jsonResponse(200, `{"payload":{"webhooks":[{"id":9,"url":"https://chatwoot.example/hooks/chatwoot?token=old","subscriptions":["message_created"],"account_id":1}]}}`)).
		On("PATCH", "/api/v1/accounts/1/webhooks/9", jsonResponse(200, `{"payload":{"webhook":{"id":9,"url":"https://chatwoot.example/hooks/chatwoot?token=new","subscriptions":["message_created"],"account_id":1}}}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"watch", "setup", "--url", "https://chatwoot.example/hooks/chatwoot", "--token", "new", "--subscriptions", "message_created", "--output", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	out := buf.String()

	if err != nil {
		t.Fatalf("watch setup failed: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("invalid json output: %v (%s)", err, out)
	}
	if payload["action"] != "updated" {
		t.Fatalf("expected action updated, got %v", payload["action"])
	}
}
