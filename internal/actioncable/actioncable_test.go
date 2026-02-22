package actioncable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// mockCable is a minimal ActionCable server for testing.
func mockCable(t *testing.T, handler func(ctx context.Context, conn *websocket.Conn)) *httptest.Server {
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

func TestConnectReceivesWelcome(t *testing.T) {
	srv := mockCable(t, func(ctx context.Context, conn *websocket.Conn) {
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"welcome"}`))
		time.Sleep(100 * time.Millisecond)
	})
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c, err := Connect(ctx, url)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer func() { _ = c.Close() }()
}

func TestConnectRejectsNoWelcome(t *testing.T) {
	srv := mockCable(t, func(ctx context.Context, conn *websocket.Conn) {
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"disconnect","reason":"unauthorized"}`))
	})
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	_, err := Connect(ctx, url)
	if err == nil {
		t.Fatal("expected error for non-welcome frame")
	}
}

func TestSubscribeConfirm(t *testing.T) {
	srv := mockCable(t, func(ctx context.Context, conn *websocket.Conn) {
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"welcome"}`))
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Errorf("read subscribe: %v", err)
			return
		}
		var f frame
		_ = json.Unmarshal(data, &f)
		if f.Command != "subscribe" {
			t.Errorf("expected subscribe, got %q", f.Command)
		}
		idQuoted, _ := json.Marshal(f.Identifier)
		_ = conn.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf(
			`{"type":"confirm_subscription","identifier":%s}`, idQuoted,
		)))
		time.Sleep(100 * time.Millisecond)
	})
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c, err := Connect(ctx, url)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer func() { _ = c.Close() }()

	err = c.Subscribe(ctx, ChannelID{
		Channel:     "RoomChannel",
		PubsubToken: "tok123",
		AccountID:   1,
		UserID:      2,
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
}

func TestSubscribeSkipsPingBeforeConfirm(t *testing.T) {
	srv := mockCable(t, func(ctx context.Context, conn *websocket.Conn) {
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"welcome"}`))
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Errorf("read subscribe: %v", err)
			return
		}
		var f frame
		_ = json.Unmarshal(data, &f)
		// Send pings before confirm (real servers do this)
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"ping","message":1234}`))
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"ping","message":1235}`))
		// Now confirm
		idQuoted, _ := json.Marshal(f.Identifier)
		_ = conn.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf(
			`{"type":"confirm_subscription","identifier":%s}`, idQuoted,
		)))
		time.Sleep(100 * time.Millisecond)
	})
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c, err := Connect(ctx, url)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer func() { _ = c.Close() }()

	err = c.Subscribe(ctx, ChannelID{
		Channel:     "RoomChannel",
		PubsubToken: "tok123",
		AccountID:   1,
		UserID:      2,
	})
	if err != nil {
		t.Fatalf("Subscribe should succeed despite pings: %v", err)
	}
}

func TestSubscribeReject(t *testing.T) {
	srv := mockCable(t, func(ctx context.Context, conn *websocket.Conn) {
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"welcome"}`))
		_, data, _ := conn.Read(ctx)
		var f frame
		_ = json.Unmarshal(data, &f)
		idQuoted, _ := json.Marshal(f.Identifier)
		_ = conn.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf(
			`{"type":"reject_subscription","identifier":%s}`, idQuoted,
		)))
	})
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c, _ := Connect(ctx, url)
	defer func() { _ = c.Close() }()

	err := c.Subscribe(ctx, ChannelID{
		Channel:     "RoomChannel",
		PubsubToken: "bad_token",
		AccountID:   1,
		UserID:      2,
	})
	if err == nil {
		t.Fatal("expected rejection error")
	}
}

func TestListenDeliversEvents(t *testing.T) {
	srv := mockCable(t, func(ctx context.Context, conn *websocket.Conn) {
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"welcome"}`))
		_, _, _ = conn.Read(ctx) // subscribe
		id, _ := json.Marshal(`{"channel":"RoomChannel"}`)
		_ = conn.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf(`{"type":"confirm_subscription","identifier":%s}`, string(id))))

		// send a ping (should be filtered)
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"ping","message":1234}`))

		// send a data message
		_ = conn.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf(`{"identifier":%s,"message":{"event":"message.created","data":{"id":99}}}`, string(id))))

		time.Sleep(200 * time.Millisecond)
	})
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c, _ := Connect(ctx, url)
	defer func() { _ = c.Close() }()
	_ = c.Subscribe(ctx, ChannelID{Channel: "RoomChannel", PubsubToken: "t", AccountID: 1, UserID: 1})

	events := c.Listen(ctx)
	select {
	case ev := <-events:
		if ev.Err != nil {
			t.Fatalf("event error: %v", ev.Err)
		}
		if len(ev.Data) == 0 {
			t.Fatal("empty event data")
		}
		// Verify the data contains our message
		var payload map[string]any
		if err := json.Unmarshal(ev.Data, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if payload["event"] != "message.created" {
			t.Errorf("event = %v, want message.created", payload["event"])
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for event")
	}
}

func TestListenHandlesDisconnect(t *testing.T) {
	srv := mockCable(t, func(ctx context.Context, conn *websocket.Conn) {
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"welcome"}`))
		_, _, _ = conn.Read(ctx) // subscribe
		id, _ := json.Marshal(`{"channel":"RoomChannel"}`)
		_ = conn.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf(`{"type":"confirm_subscription","identifier":%s}`, string(id))))

		// send disconnect
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"disconnect","reason":"server_restart","reconnect":true}`))
	})
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c, _ := Connect(ctx, url)
	defer func() { _ = c.Close() }()
	_ = c.Subscribe(ctx, ChannelID{Channel: "RoomChannel", PubsubToken: "t", AccountID: 1, UserID: 1})

	events := c.Listen(ctx)
	select {
	case ev := <-events:
		if ev.Err == nil {
			t.Fatal("expected error for disconnect")
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for disconnect event")
	}
}

func TestListenPingTimeoutOnSilence(t *testing.T) {
	srv := mockCable(t, func(ctx context.Context, conn *websocket.Conn) {
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"welcome"}`))
		_, _, _ = conn.Read(ctx) // subscribe
		id, _ := json.Marshal(`{"channel":"RoomChannel"}`)
		_ = conn.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf(`{"type":"confirm_subscription","identifier":%s}`, string(id))))
		// Send nothing after confirm — simulate dead connection.
		time.Sleep(2 * time.Second)
	})
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c, _ := Connect(ctx, url)
	defer func() { _ = c.Close() }()
	_ = c.Subscribe(ctx, ChannelID{Channel: "RoomChannel", PubsubToken: "t", AccountID: 1, UserID: 1})

	// Use a short timeout so the test doesn't take 15 seconds.
	events := c.ListenWithTimeout(ctx, 200*time.Millisecond)
	select {
	case ev := <-events:
		if ev.Err == nil {
			t.Fatal("expected error from ping timeout")
		}
		if !errors.Is(ev.Err, ErrPingTimeout) {
			t.Fatalf("expected ErrPingTimeout, got: %v", ev.Err)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for ping timeout event")
	}
}

func TestListenPingsKeepConnectionAlive(t *testing.T) {
	srv := mockCable(t, func(ctx context.Context, conn *websocket.Conn) {
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"welcome"}`))
		_, _, _ = conn.Read(ctx) // subscribe
		id, _ := json.Marshal(`{"channel":"RoomChannel"}`)
		_ = conn.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf(`{"type":"confirm_subscription","identifier":%s}`, string(id))))

		// Send pings faster than the timeout to keep connection alive.
		// First ping arrives immediately to avoid a race between
		// ListenWithTimeout starting and the timeout expiring.
		for i := 0; i < 5; i++ {
			if err := conn.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf(`{"type":"ping","message":%d}`, i))); err != nil {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		// Now send a real message.
		_ = conn.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf(`{"identifier":%s,"message":{"event":"message.created","data":{"id":1}}}`, string(id))))
		time.Sleep(200 * time.Millisecond)
	})
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c, _ := Connect(ctx, url)
	defer func() { _ = c.Close() }()
	_ = c.Subscribe(ctx, ChannelID{Channel: "RoomChannel", PubsubToken: "t", AccountID: 1, UserID: 1})

	// Timeout is 500ms, but pings arrive every 100ms — should stay alive.
	events := c.ListenWithTimeout(ctx, 500*time.Millisecond)
	select {
	case ev := <-events:
		if ev.Err != nil {
			t.Fatalf("unexpected error (pings should have kept connection alive): %v", ev.Err)
		}
		var payload map[string]any
		if err := json.Unmarshal(ev.Data, &payload); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if payload["event"] != "message.created" {
			t.Errorf("event = %v, want message.created", payload["event"])
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for event")
	}
}

func TestPresenceKeepalive(t *testing.T) {
	var presenceCount int32
	srv := mockCable(t, func(ctx context.Context, conn *websocket.Conn) {
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"welcome"}`))
		_, _, _ = conn.Read(ctx) // subscribe
		id, _ := json.Marshal(`{"channel":"RoomChannel"}`)
		_ = conn.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf(`{"type":"confirm_subscription","identifier":%s}`, string(id))))

		// read presence messages
		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			var f frame
			_ = json.Unmarshal(data, &f)
			if f.Command == "message" {
				var d struct {
					Action string `json:"action"`
				}
				_ = json.Unmarshal([]byte(f.Data), &d)
				if d.Action == "update_presence" {
					atomic.AddInt32(&presenceCount, 1)
				}
			}
		}
	})
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
	defer cancel()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c, _ := Connect(ctx, url)
	defer func() { _ = c.Close() }()
	_ = c.Subscribe(ctx, ChannelID{Channel: "RoomChannel", PubsubToken: "t", AccountID: 1, UserID: 1})

	// Start presence with short interval for testing
	c.StartPresence(ctx, 100*time.Millisecond, nil)

	<-ctx.Done()
	time.Sleep(50 * time.Millisecond) // let goroutine finish

	count := atomic.LoadInt32(&presenceCount)
	if count < 2 {
		t.Errorf("expected at least 2 presence pings, got %d", count)
	}
}
