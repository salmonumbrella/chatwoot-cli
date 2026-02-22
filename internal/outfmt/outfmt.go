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
	// JSONL outputs newline-delimited JSON
	JSONL
	// Agent outputs agent-friendly structured JSON
	Agent
)

type (
	contextKey struct{}
	compactKey struct{}
	lightKey   struct{}
)

// Parse parses an output mode string
func Parse(s string) (Mode, error) {
	switch s {
	case "text", "":
		return Text, nil
	case "json":
		return JSON, nil
	case "jsonl", "ndjson":
		return JSONL, nil
	case "agent":
		return Agent, nil
	default:
		return Text, fmt.Errorf("invalid output format: %q (use 'text', 'json', 'jsonl', 'ndjson', or 'agent')", s)
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
	mode := ModeFromContext(ctx)
	return mode == JSON || mode == JSONL || mode == Agent
}

// IsJSONL returns true if the context is set to JSONL output
func IsJSONL(ctx context.Context) bool {
	return ModeFromContext(ctx) == JSONL
}

// IsAgent returns true if the context is set to agent output.
func IsAgent(ctx context.Context) bool {
	return ModeFromContext(ctx) == Agent
}

// WithCompact adds the compact flag to the context
func WithCompact(ctx context.Context, compact bool) context.Context {
	return context.WithValue(ctx, compactKey{}, compact)
}

// IsCompact returns true if compact output mode is set in the context
func IsCompact(ctx context.Context) bool {
	if c, ok := ctx.Value(compactKey{}).(bool); ok {
		return c
	}
	return false
}

// WithLight marks the context as producing light mode output.
// When set, jq query alias expansion is skipped to preserve short JSON keys.
func WithLight(ctx context.Context, light bool) context.Context {
	return context.WithValue(ctx, lightKey{}, light)
}

// IsLight returns true if light mode output is active in the context.
func IsLight(ctx context.Context) bool {
	if l, ok := ctx.Value(lightKey{}).(bool); ok {
		return l
	}
	return false
}

// WriteJSON writes a value as pretty-printed JSON
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// WriteJSONMaybeCompact writes JSON, using compact format if compact is true.
func WriteJSONMaybeCompact(w io.Writer, v any, compact bool) error {
	enc := json.NewEncoder(w)
	if !compact {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(v)
}

// String returns the string representation of the mode
func (m Mode) String() string {
	switch m {
	case JSON:
		return "json"
	case JSONL:
		return "jsonl"
	case Agent:
		return "agent"
	default:
		return "text"
	}
}
