package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// newMentionsCmd creates the mentions command
func newMentionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "mentions",
		Aliases: []string{"m"},
		Short:   "View mentions in private notes",
		Long:    "List mentions of the current user in private notes across conversations",
	}

	cmd.AddCommand(newMentionsListCmd())

	return cmd
}

// newMentionsListCmd creates the list subcommand
func newMentionsListCmd() *cobra.Command {
	var (
		conversationID int
		since          string
		limit          int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List mentions of the current user",
		Long: `List mentions of the current user in private notes across conversations.

Mentions are created when an agent tags another agent in a private note using
the @mention syntax. This command helps you find all places where you've been
mentioned so you can follow up on requests from teammates.`,
		Example: strings.TrimSpace(`
  # List all recent mentions
  chatwoot mentions list

  # List mentions from the last 24 hours
  chatwoot mentions list --since 24h

  # List mentions from the last 7 days
  chatwoot mentions list --since 7d

  # List mentions from the last week
  chatwoot mentions list --since 1w

  # List mentions in a specific conversation
  chatwoot mentions list --conversation-id 123

  # Limit results
  chatwoot mentions list --limit 10

  # JSON output
  chatwoot mentions list --output json

  # Combine filters
  chatwoot mentions list --since 7d --limit 20 --output json
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse --since duration if provided
			var sinceTime *time.Time
			if since != "" {
				duration, err := parseDuration(since)
				if err != nil {
					return fmt.Errorf("invalid --since value %q: %w", since, err)
				}
				t := time.Now().Add(-duration)
				sinceTime = &t
			}

			// Validate conversation ID if provided
			if conversationID < 0 {
				return fmt.Errorf("--conversation-id must be a positive integer")
			}

			// Validate limit
			if limit < 1 {
				return fmt.Errorf("--limit must be at least 1")
			}

			// Placeholder output for now - API logic will be added in Task 2
			if isJSON(cmd) {
				return printJSON(cmd, []map[string]any{})
			}

			fmt.Println("Mentions list (placeholder)")
			if conversationID > 0 {
				fmt.Printf("  Filtering by conversation: %d\n", conversationID)
			}
			if sinceTime != nil {
				fmt.Printf("  Since: %s\n", sinceTime.Format("2006-01-02 15:04:05"))
			}
			fmt.Printf("  Limit: %d\n", limit)
			fmt.Println("\nNo mentions found (API not yet implemented)")

			return nil
		},
	}

	cmd.Flags().IntVar(&conversationID, "conversation-id", 0, "Filter mentions to a specific conversation")
	cmd.Flags().StringVar(&since, "since", "", "Filter mentions by time (e.g., 24h, 7d, 1w)")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of mentions to return")

	return cmd
}

// parseDuration parses a duration string like "24h", "7d", "1w"
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	// Check for week suffix (not supported by time.ParseDuration)
	if strings.HasSuffix(s, "w") {
		numStr := strings.TrimSuffix(s, "w")
		var weeks int
		if _, err := fmt.Sscanf(numStr, "%d", &weeks); err != nil {
			return 0, fmt.Errorf("invalid week duration: %s", s)
		}
		if weeks < 1 {
			return 0, fmt.Errorf("weeks must be at least 1")
		}
		return time.Duration(weeks) * 7 * 24 * time.Hour, nil
	}

	// Check for day suffix (not supported by time.ParseDuration)
	if strings.HasSuffix(s, "d") {
		numStr := strings.TrimSuffix(s, "d")
		var days int
		if _, err := fmt.Sscanf(numStr, "%d", &days); err != nil {
			return 0, fmt.Errorf("invalid day duration: %s", s)
		}
		if days < 1 {
			return 0, fmt.Errorf("days must be at least 1")
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	// Try standard time.ParseDuration for hours, minutes, seconds
	return time.ParseDuration(s)
}
