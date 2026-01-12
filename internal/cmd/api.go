package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/filter"
	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"
)

func newAPICmd() *cobra.Command {
	var method string
	var fields []string
	var rawFields []string
	var inputFile string
	var jqQuery string
	var silent bool
	var includeHeaders bool

	cmd := &cobra.Command{
		Use:   "api <endpoint>",
		Short: "Make raw API requests to any Chatwoot endpoint",
		Long: `Make raw API requests to any Chatwoot endpoint.

This command provides direct access to any Chatwoot API endpoint, giving agents
and scripts full flexibility to call APIs that may not have dedicated CLI commands.

The endpoint path is relative to the account API base path:
  /api/v1/accounts/{account_id}/<endpoint>

For example, "/conversations/123" becomes:
  /api/v1/accounts/1/conversations/123`,
		Example: `  # GET request (default)
  chatwoot api /conversations/123

  # POST request with fields
  chatwoot api /conversations -X POST -f inbox_id=1 -f contact_id=5

  # PATCH with JSON array using raw field
  chatwoot api /conversations/123 -X PATCH -F 'labels=["bug", "urgent"]'

  # Read body from file
  chatwoot api /contacts -X POST -i body.json

  # Read body from stdin
  echo '{"name": "Test"}' | chatwoot api /contacts -X POST -i -

  # Filter response with jq
  chatwoot api /contacts --jq '.payload[0].name'

  # Silent mode (no output, useful for mutations)
  chatwoot api /conversations/123 -X DELETE --silent

  # Show response headers
  chatwoot api /conversations/123 --include`,
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			endpoint := args[0]
			out := cmd.OutOrStdout()

			// Validate method
			validMethods := map[string]bool{
				"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
			}
			method = strings.ToUpper(method)
			if !validMethods[method] {
				return fmt.Errorf("invalid HTTP method %q: must be one of GET, POST, PUT, PATCH, DELETE", method)
			}

			// Build request body from fields and input
			body, err := buildRequestBody(fields, rawFields, inputFile)
			if err != nil {
				return err
			}

			// Get client
			client, err := getClient()
			if err != nil {
				return err
			}

			// Make request
			respBody, headers, statusCode, err := client.DoRaw(cmdContext(cmd), method, endpoint, body)
			if err != nil {
				return err
			}

			// Silent mode - no output
			if silent {
				return nil
			}

			// Include headers
			if includeHeaders {
				_, _ = fmt.Fprintf(out, "HTTP %d\n", statusCode)
				// Sort headers for consistent output
				keys := make([]string, 0, len(headers))
				for k := range headers {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					for _, v := range headers[k] {
						_, _ = fmt.Fprintf(out, "%s: %s\n", k, v)
					}
				}
				_, _ = fmt.Fprintln(out)
			}

			// Apply jq filter if specified
			if jqQuery != "" {
				filtered, err := applyJqFilter(respBody, jqQuery)
				if err != nil {
					return fmt.Errorf("jq filter error: %w", err)
				}
				_, _ = fmt.Fprint(out, filtered)
				return nil
			}

			// Output raw response body
			if len(respBody) > 0 {
				// Pretty print JSON if possible
				var jsonData any
				if err := json.Unmarshal(respBody, &jsonData); err == nil {
					prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
					if err == nil {
						_, _ = fmt.Fprintln(out, string(prettyJSON))
						return nil
					}
				}
				// Fall back to raw output
				_, _ = fmt.Fprintln(out, string(respBody))
			}

			return nil
		}),
	}

	cmd.Flags().StringVarP(&method, "method", "X", "GET", "HTTP method (GET, POST, PUT, PATCH, DELETE)")
	cmd.Flags().StringArrayVarP(&fields, "field", "f", nil, "Request body field as key=value (string)")
	cmd.Flags().StringArrayVarP(&rawFields, "raw-field", "F", nil, "Request body field as key=value (JSON parsed)")
	cmd.Flags().StringVarP(&inputFile, "input", "i", "", "Read request body from file (use - for stdin)")
	cmd.Flags().StringVar(&jqQuery, "jq", "", "jq expression to filter JSON response")
	cmd.Flags().BoolVarP(&silent, "silent", "s", false, "Suppress output")
	cmd.Flags().BoolVar(&includeHeaders, "include", false, "Include response headers in output")

	return cmd
}

// buildRequestBody constructs the request body from fields and/or input file
func buildRequestBody(fields, rawFields []string, inputFile string) (map[string]any, error) {
	body := make(map[string]any)

	// Read from input file first (can be overridden by fields)
	if inputFile != "" {
		var inputData []byte
		var err error

		if inputFile == "-" {
			inputData, err = io.ReadAll(os.Stdin)
		} else {
			inputData, err = os.ReadFile(inputFile)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read input: %w", err)
		}

		if err := json.Unmarshal(inputData, &body); err != nil {
			return nil, fmt.Errorf("failed to parse input JSON: %w", err)
		}
	}

	// Parse regular fields (string values)
	for _, field := range fields {
		key, value, err := parseField(field)
		if err != nil {
			return nil, err
		}
		body[key] = value
	}

	// Parse raw fields (JSON values)
	for _, field := range rawFields {
		key, value, err := parseRawField(field)
		if err != nil {
			return nil, err
		}
		body[key] = value
	}

	// Return nil if no body content
	if len(body) == 0 {
		return nil, nil
	}

	return body, nil
}

// parseField parses a key=value field where value is a string
func parseField(field string) (string, string, error) {
	parts := strings.SplitN(field, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid field format %q: must be key=value", field)
	}
	return parts[0], parts[1], nil
}

// parseRawField parses a key=value field where value is JSON
func parseRawField(field string) (string, any, error) {
	parts := strings.SplitN(field, "=", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("invalid raw field format %q: must be key=value", field)
	}

	key := parts[0]
	valueStr := parts[1]

	// Try to parse as JSON
	var value any
	if err := json.Unmarshal([]byte(valueStr), &value); err != nil {
		return "", nil, fmt.Errorf("invalid JSON in raw field %q: %w", key, err)
	}

	return key, value, nil
}

// applyJqFilter applies a jq expression to JSON data and returns the result
func applyJqFilter(data []byte, queryStr string) (string, error) {
	// Parse JSON data
	var input any
	if err := json.Unmarshal(data, &input); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Normalize shell-escaped operators (e.g., \!= from zsh)
	queryStr = filter.NormalizeExpression(queryStr)

	// Parse jq query
	query, err := gojq.Parse(queryStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse jq query: %w", err)
	}

	// Run query
	iter := query.Run(input)
	var results []string

	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return "", fmt.Errorf("jq execution error: %w", err)
		}

		// Format output
		output, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal jq result: %w", err)
		}
		results = append(results, string(output))
	}

	return strings.Join(results, "\n") + "\n", nil
}
