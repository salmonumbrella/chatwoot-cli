package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

// HandleError processes an error and returns a user-friendly message with suggestions
func HandleError(err error) string {
	if err == nil {
		return ""
	}

	var msg strings.Builder

	// Check for specific error types
	var apiErr *api.APIError
	var rateLimitErr *api.RateLimitError
	var circuitBreakerErr *api.CircuitBreakerError
	var authErr *api.AuthError

	switch {
	case errors.As(err, &rateLimitErr):
		msg.WriteString("Rate limit exceeded.\n\n")
		msg.WriteString("Suggestions:\n")
		msg.WriteString("  - Wait a few seconds and retry\n")
		msg.WriteString("  - Reduce request frequency\n")
		msg.WriteString("  - Use --dry-run to preview operations\n")

	case errors.As(err, &circuitBreakerErr):
		msg.WriteString("Service temporarily unavailable (circuit breaker open).\n\n")
		msg.WriteString("Suggestions:\n")
		msg.WriteString("  - The API has had multiple failures recently\n")
		msg.WriteString("  - Wait 30 seconds and retry\n")
		msg.WriteString("  - Check if the Chatwoot server is healthy\n")

	case errors.As(err, &authErr):
		fmt.Fprintf(&msg, "Authentication failed: %s\n\n", authErr.Reason)
		msg.WriteString("Suggestions:\n")
		msg.WriteString("  - Run: cw auth login\n")
		msg.WriteString("  - Verify your API token is valid\n")
		msg.WriteString("  - Check if your account has the required permissions\n")

	case errors.As(err, &apiErr):
		fmt.Fprintf(&msg, "API error (HTTP %d): %s\n\n", apiErr.StatusCode, apiErr.Body)
		msg.WriteString(suggestionsForStatusCode(apiErr.StatusCode, apiErr.Body))
		if apiErr.RequestID != "" {
			fmt.Fprintf(&msg, "\nRequest ID: %s\n", apiErr.RequestID)
		}

	case strings.Contains(err.Error(), "connection refused"):
		msg.WriteString("Connection refused.\n\n")
		msg.WriteString("Suggestions:\n")
		msg.WriteString("  - Check if the Chatwoot server is running\n")
		msg.WriteString("  - Verify the URL: cw auth status\n")
		msg.WriteString("  - Check your network connection\n")

	case strings.Contains(err.Error(), "no such host"):
		msg.WriteString("DNS resolution failed.\n\n")
		msg.WriteString("Suggestions:\n")
		msg.WriteString("  - Check the Chatwoot URL spelling\n")
		msg.WriteString("  - Verify your DNS settings\n")
		msg.WriteString("  - Try using the IP address directly\n")

	case strings.Contains(err.Error(), "certificate"):
		msg.WriteString("TLS certificate error.\n\n")
		msg.WriteString("Suggestions:\n")
		msg.WriteString("  - Verify the server's SSL certificate\n")
		msg.WriteString("  - Check if the certificate is expired\n")
		msg.WriteString("  - Ensure you're using https:// correctly\n")

	default:
		fmt.Fprintf(&msg, "Error: %s\n", err.Error())
	}

	return msg.String()
}

func suggestionsForStatusCode(code int, body string) string {
	var suggestions strings.Builder
	suggestions.WriteString("Suggestions:\n")

	switch code {
	case 400:
		suggestions.WriteString("  - Check your request parameters\n")
		suggestions.WriteString("  - Use --debug to see the full request\n")
		if strings.Contains(body, "required") {
			suggestions.WriteString("  - A required field may be missing\n")
		}

	case 401:
		suggestions.WriteString("  - Your API token may be invalid or expired\n")
		suggestions.WriteString("  - Run: cw auth login\n")

	case 403:
		suggestions.WriteString("  - You don't have permission for this action\n")
		suggestions.WriteString("  - Check your account role and permissions\n")
		suggestions.WriteString("  - Contact your Chatwoot admin\n")

	case 404:
		suggestions.WriteString("  - The resource doesn't exist\n")
		suggestions.WriteString("  - Check the ID is correct\n")
		suggestions.WriteString("  - The resource may have been deleted\n")

	case 422:
		suggestions.WriteString("  - Validation failed\n")
		suggestions.WriteString("  - Check your input values\n")
		suggestions.WriteString("  - Some fields may have invalid formats\n")

	case 429:
		suggestions.WriteString("  - Too many requests\n")
		suggestions.WriteString("  - Wait and retry in a few seconds\n")

	case 500, 502, 503, 504:
		suggestions.WriteString("  - Server error - not your fault\n")
		suggestions.WriteString("  - Wait and retry\n")
		suggestions.WriteString("  - Check Chatwoot server status\n")

	default:
		suggestions.WriteString("  - Use --debug for more details\n")
		suggestions.WriteString("  - Check the Chatwoot API documentation\n")
	}

	return suggestions.String()
}

// ExitWithError prints error with suggestions and exits
func ExitWithError(err error) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprint(os.Stderr, HandleError(err))
	os.Exit(ExitCode(err))
}
