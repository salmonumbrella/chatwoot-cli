package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newAPICmd() *cobra.Command {
	var method string
	var fields []string
	var rawFields []string
	var inputFile string
	var jsonBody string
	var silent bool
	var includeHeaders bool

	cmd := &cobra.Command{
		Use:     "api <endpoint>",
		Aliases: []string{"ap"},
		Short:   "Make raw API requests to any Chatwoot endpoint",
		Long: `Make raw API requests to any Chatwoot endpoint.

This command provides direct access to any Chatwoot API endpoint, giving agents
and scripts full flexibility to call APIs that may not have dedicated CLI commands.

The endpoint path is relative to the account API base path:
  /api/v1/accounts/{account_id}/<endpoint>

For example, "/conversations/123" becomes:
  /api/v1/accounts/1/conversations/123`,
		Example: `  # GET request (default)
  cw api /conversations/123

  # POST request with fields
  cw api /conversations -X POST -f inbox_id=1 -f contact_id=5

  # PATCH with JSON array using raw field
  cw api /conversations/123 -X PATCH -F 'labels=["bug", "urgent"]'

  # Inline JSON body
  cw api /automation_rules/14 -X PATCH -d '{"automation_rule":{"active":true}}'

  # Read body from file
  cw api /contacts -X POST -i body.json

  # Read body from stdin
  echo '{"name": "Test"}' | cw api /contacts -X POST -i -

  # Filter response with jq (JSON output required)
  cw api /contacts --output json --jq '.payload[0].name'

  # Silent mode (no output, useful for mutations)
  cw api /conversations/123 -X DELETE --silent

  # Show response headers
  cw api /conversations/123 --include`,
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

			// Validate that --body and --input are not both set
			if jsonBody != "" && inputFile != "" {
				return fmt.Errorf("cannot use both --body and --input flags")
			}

			// Build request body from fields and input
			body, err := buildRequestBody(fields, rawFields, inputFile, jsonBody)
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

			// JSON output uses global outfmt pipeline (--output json/--json)
			if isJSON(cmd) {
				payload := apiJSONPayload(respBody, headers, statusCode, includeHeaders)
				return printJSON(cmd, payload)
			}

			// Text output (legacy behavior)
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
	cmd.Flags().StringVarP(&jsonBody, "body", "d", "", "Request body as inline JSON string")
	cmd.Flags().BoolVar(&silent, "silent", false, "Suppress output")
	cmd.Flags().BoolVar(&includeHeaders, "include", false, "Include response headers in output")
	flagAlias(cmd.Flags(), "include", "inc")

	return cmd
}

func apiJSONPayload(respBody []byte, headers map[string][]string, statusCode int, includeHeaders bool) any {
	body := apiJSONBody(respBody)
	if !includeHeaders {
		return body
	}
	return map[string]any{
		"status":  statusCode,
		"headers": headers,
		"body":    body,
	}
}

func apiJSONBody(respBody []byte) any {
	if len(respBody) == 0 {
		return nil
	}
	if !json.Valid(respBody) {
		return string(respBody)
	}
	return json.RawMessage(respBody)
}

// buildRequestBody constructs the request body from fields and/or input file/inline JSON
func buildRequestBody(fields, rawFields []string, inputFile, jsonBody string) (any, error) {
	mergeFields := len(fields) > 0 || len(rawFields) > 0

	var (
		bodyValue any
		hasBody   bool
	)

	// Parse inline JSON body first (can be overridden by fields when the body is an object).
	if jsonBody != "" {
		value, err := parseJSONValue([]byte(jsonBody), true, jsonBody)
		if err != nil {
			return nil, err
		}
		bodyValue = value
		hasBody = true
	}

	// Read from input file (can be overridden by fields when the body is an object).
	if inputFile != "" {
		inputData, err := readInputData(inputFile)
		if err != nil {
			return nil, err
		}
		value, err := parseJSONValue(inputData, false, inputFile)
		if err != nil {
			return nil, err
		}
		bodyValue = value
		hasBody = true
	}

	if !mergeFields {
		if !hasBody {
			return nil, nil
		}
		if err := validateRequestBodySize(bodyValue); err != nil {
			return nil, err
		}
		return bodyValue, nil
	}

	body := make(map[string]any)
	if hasBody {
		obj, ok := bodyValue.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot use --field or --raw-field with a non-object JSON body")
		}
		for k, v := range obj {
			body[k] = v
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

	if len(body) == 0 {
		return nil, nil
	}
	if err := validateRequestBodySize(body); err != nil {
		return nil, err
	}

	return body, nil
}

func readInputData(inputFile string) ([]byte, error) {
	if inputFile == "-" {
		limited := io.LimitReader(os.Stdin, validation.MaxJSONPayload+1)
		inputData, err := io.ReadAll(limited)
		if err != nil {
			return nil, fmt.Errorf("failed to read input: %w", err)
		}
		if len(inputData) > validation.MaxJSONPayload {
			return nil, fmt.Errorf("JSON payload exceeds maximum size of %d bytes (got more than %d)", validation.MaxJSONPayload, validation.MaxJSONPayload)
		}
		return inputData, nil
	}

	info, err := os.Stat(inputFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read input: %w", err)
		}
		return nil, fmt.Errorf("failed to stat input: %w", err)
	}
	if info.Size() > validation.MaxJSONPayload {
		return nil, fmt.Errorf("JSON payload exceeds maximum size of %d bytes (got %d)", validation.MaxJSONPayload, info.Size())
	}

	inputData, err := os.ReadFile(inputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	return inputData, nil
}

func parseJSONValue(data []byte, inline bool, source string) (any, error) {
	if err := validation.ValidateJSONPayload(string(data)); err != nil {
		return nil, err
	}

	trimmed := bytes.TrimSpace(data)
	body, err := decodeJSONValue(data)
	if err != nil {
		if inline {
			return nil, formatBodyJSONParseError(source, err)
		}
		return nil, fmt.Errorf("failed to parse input JSON: %w", err)
	}
	if body == nil && bytes.Equal(trimmed, []byte("null")) {
		return json.RawMessage("null"), nil
	}
	return body, nil
}

func decodeJSONValue(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("invalid JSON: multiple top-level values")
		}
		return nil, err
	}

	return value, nil
}

func validateRequestBodySize(body any) error {
	if body == nil {
		return nil
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body for validation: %w", err)
	}
	return validation.ValidateJSONPayload(string(payload))
}

func formatBodyJSONParseError(raw string, err error) error {
	msg := fmt.Sprintf("failed to parse --body JSON: %v", err)

	hints := []string{
		"Tip: --body expects strict JSON.",
	}

	parseErr := err.Error()
	if strings.Contains(parseErr, "string escape code") {
		// Detect common shell-mangled characters in the raw input.
		for _, esc := range []string{`\!`, `\?`, `\'`, `\(`, `\)`, `\$`} {
			if strings.Contains(raw, esc) {
				char := esc[1:]
				hints = append(hints, fmt.Sprintf(`Found invalid escape "%s". Use "%s" (no backslash) in JSON strings.`, esc, char))
			}
		}
		if len(hints) == 1 { // only the "Tip:" prefix, no specific match
			hints = append(hints, `Only JSON escapes are valid: \", \\, \/, \b, \f, \n, \r, \t, \uXXXX.`)
		}
	}

	hints = append(hints, "For long payloads, prefer --input/-i (file/stdin) or --field/-f and --raw-field/-F.")

	return fmt.Errorf("%s\n%s", msg, strings.Join(hints, "\n"))
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

	if err := validation.ValidateJSONPayload(valueStr); err != nil {
		return "", nil, fmt.Errorf("invalid JSON in raw field %q: %w", key, err)
	}

	// Try to parse as JSON
	value, err := decodeJSONValue([]byte(valueStr))
	if err != nil {
		return "", nil, fmt.Errorf("invalid JSON in raw field %q: %w", key, err)
	}

	return key, value, nil
}
