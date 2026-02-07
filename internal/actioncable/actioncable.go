package actioncable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coder/websocket"
)

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

// Connect dials the ActionCable endpoint and waits for the welcome frame.
func Connect(ctx context.Context, url string) (*Client, error) {
	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		Subprotocols: []string{"actioncable-v1-json"},
	})
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

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

	// Wait for confirm or reject.
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
	default:
		return fmt.Errorf("unexpected response type: %q", f.Type)
	}
}

// Listen starts the read loop and returns a channel of events.
// Pings and internal frames are handled silently.
// The channel closes when the connection drops or ctx is cancelled.
func (c *Client) Listen(ctx context.Context) <-chan Event {
	ch := make(chan Event, 64)
	go func() {
		defer close(ch)
		for {
			_, data, err := c.conn.Read(ctx)
			if err != nil {
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
