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

// WriteJSONFiltered writes JSON with optional JQ filtering.
// Uses pretty-printed output by default; pass compact=true for single-line output.
func WriteJSONFiltered(w io.Writer, v any, query string, compact bool) error {
	v = normalizeJSONOutput(v)
	if query == "" {
		return WriteJSONMaybeCompact(w, v, compact)
	}

	// Marshal to JSON, apply filter, then re-marshal with desired formatting.
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	result, err := filter.ApplyFromJSON(data, query)
	if err != nil {
		return err
	}

	return WriteJSONMaybeCompact(w, result, compact)
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

// WriteJSONFilteredLiteral writes JSON with JQ filtering but without alias normalization.
// Use for light mode output where JSON keys are intentionally short.
func WriteJSONFilteredLiteral(w io.Writer, v any, query string, compact bool) error {
	v = normalizeJSONOutput(v)
	if query == "" {
		return WriteJSONMaybeCompact(w, v, compact)
	}

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	result, err := filter.ApplyFromJSONLiteral(data, query)
	if err != nil {
		return err
	}

	return WriteJSONMaybeCompact(w, result, compact)
}

// ApplyQueryLiteral applies a JQ query without alias normalization.
func ApplyQueryLiteral(v any, query string) (any, error) {
	v = normalizeJSONOutput(v)
	if query == "" {
		return v, nil
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	filtered, err := filter.ApplyToJSONLiteral(data, query)
	if err != nil {
		return nil, err
	}

	var out any
	if err := json.Unmarshal(filtered, &out); err != nil {
		return nil, err
	}
	return out, nil
}
