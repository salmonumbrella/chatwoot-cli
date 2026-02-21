// internal/filter/filter.go
package filter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/queryalias"
	"github.com/itchyny/gojq"
)

// NormalizeExpression fixes shell-escaped operators in jq expressions.
// Zsh escapes ! to \! even in single quotes, breaking operators like !=.
func NormalizeExpression(expr string) string {
	// Replace \! with ! (zsh escapes ! due to history expansion)
	expr = strings.ReplaceAll(expr, `\!`, `!`)
	return queryalias.Normalize(expr, queryalias.ContextQuery)
}

// applyWith is the shared core: normalize the expression, parse, and run jq.
func applyWith(data interface{}, expression string, normalize func(string) string) (interface{}, error) {
	if expression == "" {
		return data, nil
	}

	expression = normalize(expression)
	query, err := gojq.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("invalid filter expression: %w", err)
	}

	iter := query.Run(data)

	var results []interface{}
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, fmt.Errorf("filter error: %w", err)
		}
		results = append(results, v)
	}

	if len(results) == 1 {
		return results[0], nil
	}
	return results, nil
}

// Apply applies a JQ filter expression to the input data
func Apply(data interface{}, expression string) (interface{}, error) {
	return applyWith(data, expression, NormalizeExpression)
}

// applyToJSONWith is the shared JSON wrapper: unmarshal, apply, marshal.
func applyToJSONWith(jsonData []byte, expression string, apply func(interface{}, string) (interface{}, error)) ([]byte, error) {
	if expression == "" {
		return jsonData, nil
	}

	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	result, err := apply(data, expression)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(result, "", "  ")
}

// ApplyToJSON applies filter to JSON bytes and returns filtered JSON bytes (pretty-printed).
func ApplyToJSON(jsonData []byte, expression string) ([]byte, error) {
	return applyToJSONWith(jsonData, expression, Apply)
}

// ApplyFromJSON applies a JQ filter to JSON bytes and returns the result as a Go value.
// Unlike ApplyToJSON, this returns the unmarshaled value for the caller to format.
func ApplyFromJSON(jsonData []byte, expression string) (interface{}, error) {
	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return Apply(data, expression)
}

// fixShellEscapes fixes shell-escaped operators without alias normalization.
func fixShellEscapes(expr string) string {
	return strings.ReplaceAll(expr, `\!`, `!`)
}

// ApplyLiteral applies a JQ filter without query alias normalization.
// Only fixes shell escapes (\! → !). Use for light mode where JSON keys
// are intentionally short and collide with query aliases.
func ApplyLiteral(data interface{}, expression string) (interface{}, error) {
	return applyWith(data, expression, fixShellEscapes)
}

// ApplyFromJSONLiteral applies a JQ filter to JSON bytes without alias normalization.
func ApplyFromJSONLiteral(jsonData []byte, expression string) (interface{}, error) {
	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return ApplyLiteral(data, expression)
}

// ApplyToJSONLiteral applies a literal filter to JSON bytes and returns filtered JSON bytes.
func ApplyToJSONLiteral(jsonData []byte, expression string) ([]byte, error) {
	return applyToJSONWith(jsonData, expression, ApplyLiteral)
}
