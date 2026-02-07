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

	"github.com/chatwoot/chatwoot-cli/internal/actioncable"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/coder/websocket"
	"github.com/spf13/cobra"
)

func mockActionCableServer(t *testing.T, handler func(ctx context.Context, conn *websocket.Conn)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			Subprotocols: []string{"actioncable-v1-json"},
		})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer func() { _ = conn.CloseNow() }()
		handler(r.Context(), conn)
	}))
}

func TestFollowViaWebSocket_RoutesDebounces_JSONL_WithRaw(t *testing.T) {
	done := make(chan struct{})
	sent := make(chan struct{})

	srv := mockActionCableServer(t, func(ctx context.Context, conn *websocket.Conn) {
		write := func(v any) {
			b, _ := json.Marshal(v)
			_ = conn.Write(ctx, websocket.MessageText, b)
		}

		// Welcome.
		write(map[string]any{"type": "welcome"})

		// Subscribe.
		_, _, _ = conn.Read(ctx)
		write(map[string]any{
			"type":       "confirm_subscription",
			"identifier": `{"channel":"RoomChannel"}`,
		})

		// Give the client a moment to process the confirm frame and enter the listen loop
		// before we send events.
		time.Sleep(50 * time.Millisecond)

		identifier := `{"channel":"RoomChannel"}`
		sendEvent := func(event string, data any) {
			payload := map[string]any{
				"event": event,
				"data":  data,
			}
			write(map[string]any{
				"identifier": identifier,
				"message":    payload,
			})
		}

		// Two rapid messages for convo 100 (should batch).
		sendEvent("message.created", map[string]any{
			"id":              1,
			"conversation_id": 100,
			"content":         "hi",
			"message_type":    0,
			"private":         false,
			"created_at":      time.Now().Unix(),
		})
		sendEvent("message.created", map[string]any{
			"id":              2,
			"conversation_id": 100,
			"content":         "there",
			"message_type":    0,
			"private":         false,
			"created_at":      time.Now().Unix(),
		})

		// Different conversation (should be ignored).
		sendEvent("message.created", map[string]any{
			"id":              3,
			"conversation_id": 200,
			"content":         "ignore",
			"message_type":    0,
			"private":         false,
			"created_at":      time.Now().Unix(),
		})

		// Allowed non-message event.
		sendEvent("conversation.status_changed", map[string]any{
			"id":     100,
			"status": "resolved",
		})

		// Disallowed event (should be filtered).
		sendEvent("assignee.changed", map[string]any{
			"id": 100,
			"assignee": map[string]any{
				"id":   55,
				"name": "Agent",
			},
		})

		close(sent)
		<-done
	})
	defer srv.Close()

	cableURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = outfmt.WithMode(ctx, outfmt.JSONL)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	allowed := map[string]struct{}{
		"message.created":             {},
		"conversation.status_changed": {},
	}

	channelID := actioncable.ChannelID{
		Channel:     "RoomChannel",
		PubsubToken: "tok",
		AccountID:   1,
		UserID:      1,
	}

	var lastSeen int
	errCh := make(chan error, 1)
	go func() {
		errCh <- followViaWebSocket(ctx, cmd, followWebSocketConfig{
			CableURL:      cableURL,
			ChannelID:     channelID,
			ConvID:        100,
			IncomingOnly:  true,
			LastSeenID:    &lastSeen,
			AllowedEvents: allowed,
			Debounce:      250 * time.Millisecond,
			IncludeRaw:    true,
		})
	}()

	select {
	case <-sent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server to send events")
	}

	// Give the client time to read and print the non-debounced status event.
	time.Sleep(150 * time.Millisecond)

	cancel()
	close(done)

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("followViaWebSocket returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for followViaWebSocket to return")
	}

	// Parse JSONL output.
	var events []map[string]any
	sc := bufio.NewScanner(bytes.NewReader(buf.Bytes()))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("invalid jsonl line %q: %v", line, err)
		}
		events = append(events, m)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan output: %v", err)
	}

	var (
		gotBatch       bool
		gotStatus      bool
		gotCreated     bool
		gotAssignee    bool
		batchRawItemsN int
	)

	for _, e := range events {
		ev, _ := e["event"].(string)
		switch ev {
		case "message.batch":
			gotBatch = true
			if _, ok := e["raw_items"]; !ok {
				t.Fatalf("expected raw_items in batch event, got: %v", e)
			}
			rawItems, ok := e["raw_items"].([]any)
			if !ok {
				t.Fatalf("expected raw_items to be an array, got: %T", e["raw_items"])
			}
			batchRawItemsN = len(rawItems)
		case "conversation.status_changed":
			gotStatus = true
			if _, ok := e["raw"]; !ok {
				t.Fatalf("expected raw in status event, got: %v", e)
			}
		case "message.created":
			gotCreated = true
		case "assignee.changed":
			gotAssignee = true
		}
	}

	if !gotBatch {
		t.Fatalf("expected message.batch event, got %d events: %v", len(events), events)
	}
	if batchRawItemsN != 2 {
		t.Fatalf("expected 2 raw_items in batch, got %d", batchRawItemsN)
	}
	if !gotStatus {
		t.Fatalf("expected conversation.status_changed event, got %d events: %v", len(events), events)
	}
	if gotCreated {
		t.Fatalf("did not expect any message.created events when debounce is enabled, got: %v", events)
	}
	if gotAssignee {
		t.Fatalf("did not expect assignee.changed (should be filtered), got: %v", events)
	}
}
