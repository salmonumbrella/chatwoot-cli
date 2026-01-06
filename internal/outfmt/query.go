// internal/outfmt/query.go
package outfmt

import (
	"context"
	"encoding/json"
	"io"

	"github.com/chatwoot/chatwoot-cli/internal/filter"
)

type queryKey struct{}

// WithQuery adds a JQ query to the context
func WithQuery(ctx context.Context, query string) context.Context {
	return context.WithValue(ctx, queryKey{}, query)
}

// GetQuery retrieves the JQ query from context
func GetQuery(ctx context.Context) string {
	if q, ok := ctx.Value(queryKey{}).(string); ok {
		return q
	}
	return ""
}

// WriteJSONFiltered writes JSON with optional JQ filtering
func WriteJSONFiltered(w io.Writer, v any, query string) error {
	v = normalizeJSONOutput(v)
	if query == "" {
		return WriteJSON(w, v)
	}

	// Marshal to JSON first
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	// Apply filter
	filtered, err := filter.ApplyToJSON(data, query)
	if err != nil {
		return err
	}

	_, err = w.Write(filtered)
	if err == nil {
		_, err = w.Write([]byte("\n"))
	}
	return err
}

// ApplyQuery applies a JQ query to structured data and returns the filtered value.
func ApplyQuery(v any, query string) (any, error) {
	v = normalizeJSONOutput(v)
	if query == "" {
		return v, nil
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	filtered, err := filter.ApplyToJSON(data, query)
	if err != nil {
		return nil, err
	}

	var out any
	if err := json.Unmarshal(filtered, &out); err != nil {
		return nil, err
	}
	return out, nil
}
