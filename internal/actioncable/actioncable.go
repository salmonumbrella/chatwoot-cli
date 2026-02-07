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

// Client is an ActionCable WebSocket client.
type Client struct {
	conn *websocket.Conn
	url  string
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
