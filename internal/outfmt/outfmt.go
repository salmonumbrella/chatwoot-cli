package outfmt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// Mode represents the output format mode
type Mode int

const (
	// Text is the default human-readable output
	Text Mode = iota
	// JSON outputs structured JSON
	JSON
)

type contextKey struct{}

// Parse parses an output mode string
func Parse(s string) (Mode, error) {
	switch s {
	case "text", "":
		return Text, nil
	case "json":
		return JSON, nil
	default:
		return Text, fmt.Errorf("invalid output format: %q (use 'text' or 'json')", s)
	}
}

// WithMode adds the output mode to the context
func WithMode(ctx context.Context, mode Mode) context.Context {
	return context.WithValue(ctx, contextKey{}, mode)
}

// ModeFromContext retrieves the output mode from context
func ModeFromContext(ctx context.Context) Mode {
	if mode, ok := ctx.Value(contextKey{}).(Mode); ok {
		return mode
	}
	return Text
}

// IsJSON returns true if the context is set to JSON output
func IsJSON(ctx context.Context) bool {
	return ModeFromContext(ctx) == JSON
}

// WriteJSON writes a value as pretty-printed JSON
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// String returns the string representation of the mode
func (m Mode) String() string {
	switch m {
	case JSON:
		return "json"
	default:
		return "text"
	}
}
