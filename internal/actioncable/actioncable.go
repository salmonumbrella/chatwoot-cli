package actioncable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/coder/websocket"
)

// DefaultPingTimeout is how long we wait without receiving any frame
// (including server pings) before treating the connection as dead.
// ActionCable servers ping every ~3s, so 15s means ~5 missed pings.
var DefaultPingTimeout = 15 * time.Second

// ErrPingTimeout is returned when no frames are received within the ping timeout.
var ErrPingTimeout = errors.New("ping timeout: no frames received")

// frame is a raw ActionCable JSON frame.
type frame struct {
	Type       string          `json:"type,omitempty"`
	Identifier string          `json:"identifier,omitempty"`
	Message    json.RawMessage `json:"message,omitempty"`
	Command    string          `json:"command,omitempty"`
	Data       string          `json:"data,omitempty"`
	Reconnect  *bool           `json:"reconnect,omitempty"`
	Reason     string          `json:"reason,omitempty"`
}

// ChannelID identifies a channel subscription.
// Fields are serialized to JSON and double-encoded as the ActionCable identifier string.
type ChannelID struct {
	Channel     string `json:"channel"`
	PubsubToken string `json:"pubsub_token"`
	AccountID   int    `json:"account_id"`
	UserID      int    `json:"user_id,omitempty"`
}

// Event is a message received from the ActionCable server.
type Event struct {
	Data json.RawMessage // the "message" field payload
	Err  error           // non-nil on read error or disconnect
}

// Client is an ActionCable WebSocket client.
type Client struct {
	conn       *websocket.Conn
	url        string
	identifier string // set after Subscribe
}

// maxReadSize caps the maximum WebSocket frame size to 1 MB.
// ActionCable messages are small JSON; anything larger is likely malformed.
const maxReadSize = 1 << 20 // 1 MB

// Connect dials the ActionCable endpoint and waits for the welcome frame.
func Connect(ctx context.Context, url string) (*Client, error) {
	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		Subprotocols: []string{"actioncable-v1-json"},
	})
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	conn.SetReadLimit(maxReadSize)

	// Read the welcome frame.
	_, data, err := conn.Read(ctx)
	if err != nil {
		_ = conn.CloseNow()
		return nil, fmt.Errorf("read welcome: %w", err)
	}

	var f frame
	if err := json.Unmarshal(data, &f); err != nil {
		_ = conn.CloseNow()
		return nil, fmt.Errorf("parse welcome: %w", err)
	}
	if f.Type != "welcome" {
		_ = conn.CloseNow()
		return nil, fmt.Errorf("expected welcome, got %q (reason: %s)", f.Type, f.Reason)
	}

	return &Client{conn: conn, url: url}, nil
}

// Close gracefully closes the connection.
func (c *Client) Close() error {
	return c.conn.Close(websocket.StatusNormalClosure, "bye")
}

// Subscribe sends a subscribe command and waits for confirmation.
func (c *Client) Subscribe(ctx context.Context, id ChannelID) error {
	idJSON, err := json.Marshal(id)
	if err != nil {
		return fmt.Errorf("marshal identifier: %w", err)
	}
	idStr := string(idJSON)

	cmd := frame{
		Command:    "subscribe",
		Identifier: idStr,
	}
	data, _ := json.Marshal(cmd)
	if err := c.conn.Write(ctx, websocket.MessageText, data); err != nil {
		return fmt.Errorf("write subscribe: %w", err)
	}

	// Wait for confirm or reject, skipping pings that may arrive in between.
	for {
		_, resp, err := c.conn.Read(ctx)
		if err != nil {
			return fmt.Errorf("read subscription response: %w", err)
		}

		var f frame
		if err := json.Unmarshal(resp, &f); err != nil {
			return fmt.Errorf("parse response: %w", err)
		}

		switch f.Type {
		case "confirm_subscription":
			c.identifier = idStr
			return nil
		case "reject_subscription":
			return fmt.Errorf("subscription rejected (check pubsub_token)")
		case "ping":
			continue // server pings arrive every ~3s, skip them
		default:
			return fmt.Errorf("unexpected response type: %q", f.Type)
		}
	}
}

// StartPresence sends update_presence actions at the given interval.
// Stops when ctx is cancelled. For Chatwoot, use 30*time.Second.
// If onError is non-nil, it is called once on the first write failure
// before the goroutine exits (useful for logging).
func (c *Client) StartPresence(ctx context.Context, interval time.Duration, onError func(error)) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				msg := frame{
					Command:    "message",
					Identifier: c.identifier,
					Data:       `{"action":"update_presence"}`,
				}
				data, _ := json.Marshal(msg)
				if err := c.conn.Write(ctx, websocket.MessageText, data); err != nil {
					if onError != nil && ctx.Err() == nil {
						onError(fmt.Errorf("presence write: %w", err))
					}
					return
				}
			}
		}
	}()
}

// Listen starts the read loop and returns a channel of events.
// Pings and internal frames are handled silently.
// The channel closes when the connection drops or ctx is cancelled.
//
// A rolling ping timeout detects half-dead connections: if no frame
// (including server pings) arrives within DefaultPingTimeout, the
// connection is treated as dead and an ErrPingTimeout is emitted.
func (c *Client) Listen(ctx context.Context) <-chan Event {
	return c.ListenWithTimeout(ctx, DefaultPingTimeout)
}

// ListenWithTimeout is like Listen but with a configurable ping timeout.
// Use 0 to disable the timeout (not recommended in production).
func (c *Client) ListenWithTimeout(ctx context.Context, pingTimeout time.Duration) <-chan Event {
	ch := make(chan Event, 64)
	go func() {
		defer close(ch)
		for {
			// Create a per-read context with a deadline so that half-dead
			// connections (no FIN/RST, just silence) get detected.
			readCtx := ctx
			var readCancel context.CancelFunc
			if pingTimeout > 0 {
				readCtx, readCancel = context.WithTimeout(ctx, pingTimeout)
			}

			_, data, err := c.conn.Read(readCtx)

			if readCancel != nil {
				readCancel()
			}

			if err != nil {
				// Distinguish ping timeout from parent context cancellation.
				if pingTimeout > 0 && ctx.Err() == nil && readCtx.Err() != nil {
					err = ErrPingTimeout
				}
				select {
				case ch <- Event{Err: err}:
				case <-ctx.Done():
				}
				return
			}

			var f frame
			if err := json.Unmarshal(data, &f); err != nil {
				continue // skip malformed frames
			}

			switch {
			case f.Type == "ping":
				continue
			case f.Type == "disconnect":
				reconnect := f.Reconnect != nil && *f.Reconnect
				select {
				case ch <- Event{Err: fmt.Errorf("disconnect (reason=%s, reconnect=%v)", f.Reason, reconnect)}:
				case <-ctx.Done():
				}
				return
			case f.Type == "confirm_subscription", f.Type == "reject_subscription":
				continue
			case len(f.Message) > 0:
				select {
				case ch <- Event{Data: f.Message}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch
}
