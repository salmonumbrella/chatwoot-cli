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

// Apply applies a JQ filter expression to the input data
func Apply(data interface{}, expression string) (interface{}, error) {
	if expression == "" {
		return data, nil
	}

	expression = NormalizeExpression(expression)
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

// ApplyToJSON applies filter to JSON bytes and returns filtered JSON bytes
func ApplyToJSON(jsonData []byte, expression string) ([]byte, error) {
	if expression == "" {
		return jsonData, nil
	}

	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	result, err := Apply(data, expression)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(result, "", "  ")
}
