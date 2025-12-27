package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/config"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

// getClient creates an API client from stored credentials
func getClient() (*api.Client, error) {
	account, err := config.LoadAccount()
	if err != nil {
		return nil, err
	}
	return api.New(account.BaseURL, account.APIToken, account.AccountID), nil
}

// newTabWriter creates a tabwriter for text output
func newTabWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
}

// printJSON outputs data as JSON
func printJSON(v any) error {
	return outfmt.WriteJSON(os.Stdout, v)
}

// isJSON checks if the command context wants JSON output
func isJSON(cmd *cobra.Command) bool {
	return outfmt.IsJSON(cmd.Context())
}

// cmdContext returns the command context
func cmdContext(cmd *cobra.Command) context.Context {
	return cmd.Context()
}

// validatePriority validates a conversation priority value
func validatePriority(priority string) error {
	valid := []string{"urgent", "high", "medium", "low", "none"}
	for _, v := range valid {
		if priority == v {
			return nil
		}
	}
	return fmt.Errorf("invalid priority %q: must be one of %s", priority, strings.Join(valid, ", "))
}

// validateStatus validates a conversation status value
func validateStatus(status string) error {
	valid := []string{"open", "resolved", "pending", "snoozed"}
	for _, v := range valid {
		if status == v {
			return nil
		}
	}
	return fmt.Errorf("invalid status %q: must be one of %s", status, strings.Join(valid, ", "))
}

// validateSlug validates a portal/article/category slug
func validateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("slug cannot be empty")
	}
	if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(slug) {
		return fmt.Errorf("invalid slug %q: must contain only lowercase letters, numbers, and hyphens", slug)
	}
	return nil
}
