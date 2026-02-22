package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newCSATCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "csat",
		Aliases: []string{"satisfaction", "cs"},
		Short:   "View customer satisfaction data",
		Long:    "List, view, and analyze CSAT (Customer Satisfaction) survey responses",
	}

	cmd.AddCommand(newCSATListCmd())
	cmd.AddCommand(newCSATGetCmd())
	cmd.AddCommand(newCSATSummaryCmd())

	return cmd
}

func newCSATListCmd() *cobra.Command {
	var (
		from    string
		to      string
		inboxID int
		rating  string
		page    int
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List CSAT responses",
		Example: strings.TrimSpace(`
  # List all CSAT responses
  cw csat list

  # List responses in date range
  cw csat list --from 2024-01-01 --to 2024-01-31

  # Filter by low ratings
  cw csat list --rating 1,2

  # Filter by inbox
  cw csat list --inbox-id 5
`),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			if from != "" {
				normalized, err := normalizeCSATDate(from)
				if err != nil {
					return err
				}
				from = normalized
			}
			if to != "" {
				normalized, err := normalizeCSATDate(to)
				if err != nil {
					return err
				}
				to = normalized
			}

			params := api.CSATListParams{
				Page:    page,
				Since:   from,
				Until:   to,
				InboxID: inboxID,
				Rating:  rating,
			}

			responses, err := client.CSAT().List(cmdContext(cmd), params)
			if err != nil {
				return fmt.Errorf("failed to list CSAT responses: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, responses)
			}

			if len(responses) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No CSAT responses found")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
			_, _ = fmt.Fprintln(w, "ID\tCONV\tRATING\tFEEDBACK\tCREATED")
			for _, csat := range responses {
				feedback := csat.FeedbackMessage
				if len(feedback) > 40 {
					feedback = feedback[:37] + "..."
				}
				_, _ = fmt.Fprintf(w, "%d\t%d\t%d\t%s\t%s\n",
					csat.ID,
					csat.ConversationID,
					csat.Rating,
					feedback,
					formatTimestampShort(csat.CreatedAtTime()),
				)
			}
			_ = w.Flush()

			return nil
		}),
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD or relative)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD or relative)")
	cmd.Flags().IntVarP(&inboxID, "inbox-id", "I", 0, "Filter by inbox ID")
	cmd.Flags().StringVar(&rating, "rating", "", "Filter by ratings (comma-separated, e.g., 1,2)")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	flagAlias(cmd.Flags(), "inbox-id", "iid")
	flagAlias(cmd.Flags(), "from", "fr")
	flagAlias(cmd.Flags(), "to", "t2")
	flagAlias(cmd.Flags(), "rating", "rt")
	flagAlias(cmd.Flags(), "page", "pg")

	return cmd
}

func newCSATGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <conversation-id>",
		Aliases: []string{"g"},
		Short:   "Get CSAT for a conversation",
		Example: strings.TrimSpace(`
  # Get CSAT for conversation
  cw csat get 123

  # JSON output
  cw csat get 123 -o json
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			conversationID, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			csat, err := client.CSAT().Conversation(cmdContext(cmd), conversationID)
			if err != nil {
				return fmt.Errorf("failed to get CSAT for conversation %d: %w", conversationID, err)
			}

			if csat == nil {
				if isJSON(cmd) {
					return printJSON(cmd, nil)
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No CSAT response for conversation %d\n", conversationID)
				return nil
			}

			if isJSON(cmd) {
				return printJSON(cmd, csat)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "CSAT for Conversation #%d\n", conversationID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Rating:   %d/5 %s\n", csat.Rating, ratingStars(csat.Rating))
			if csat.FeedbackMessage != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Feedback: %s\n", csat.FeedbackMessage)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Created:  %s\n", formatTimestamp(csat.CreatedAtTime()))

			return nil
		}),
	}

	return cmd
}

func newCSATSummaryCmd() *cobra.Command {
	var (
		from    string
		to      string
		inboxID int
	)

	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Get CSAT summary statistics",
		Example: strings.TrimSpace(`
  # Get summary for date range
  cw csat summary --from 2024-01-01 --to 2024-01-31

  # Get summary for inbox
  cw csat summary --inbox-id 5 --from 2024-01-01 --to 2024-12-31
`),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			if from != "" {
				normalized, err := normalizeCSATDate(from)
				if err != nil {
					return err
				}
				from = normalized
			}
			if to != "" {
				normalized, err := normalizeCSATDate(to)
				if err != nil {
					return err
				}
				to = normalized
			}

			// Fetch all pages of responses for accurate summary
			var allResponses []api.CSATResponse
			page := 1
			for {
				params := api.CSATListParams{
					Page:    page,
					Since:   from,
					Until:   to,
					InboxID: inboxID,
				}

				responses, err := client.CSAT().List(cmdContext(cmd), params)
				if err != nil {
					return fmt.Errorf("failed to get CSAT data: %w", err)
				}

				allResponses = append(allResponses, responses...)

				// Stop if we got less than a full page (no more pages)
				if len(responses) == 0 || len(responses) < 25 {
					break
				}
				page++
			}

			responses := allResponses
			if len(responses) == 0 {
				if isJSON(cmd) {
					return printJSON(cmd, map[string]any{
						"total_responses":   0,
						"average_rating":    0,
						"satisfaction_rate": 0,
						"distribution":      map[int]int{},
					})
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No CSAT responses found for the specified period")
				return nil
			}

			// Calculate statistics
			total := len(responses)
			sum := 0
			distribution := make(map[int]int)
			satisfied := 0 // ratings 4-5

			for _, r := range responses {
				sum += r.Rating
				distribution[r.Rating]++
				if r.Rating >= 4 {
					satisfied++
				}
			}

			avg := float64(sum) / float64(total)
			satisfactionRate := float64(satisfied) / float64(total) * 100

			if isJSON(cmd) {
				summary := map[string]any{
					"total_responses":   total,
					"average_rating":    avg,
					"satisfaction_rate": satisfactionRate,
					"distribution":      distribution,
				}
				return printJSON(cmd, summary)
			}

			dateRange := ""
			if from != "" || to != "" {
				dateRange = fmt.Sprintf(" (%s to %s)", from, to)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "CSAT Summary%s:\n", dateRange)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Total Responses:    %d\n", total)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Average Rating:     %.1f / 5\n", avg)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Satisfaction Rate:  %.0f%%\n\n", satisfactionRate)
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "  Distribution:")
			for i := 5; i >= 1; i-- {
				count := distribution[i]
				pct := float64(count) / float64(total) * 100
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    %s (%d): %d (%.0f%%)\n", ratingStars(i), i, count, pct)
			}

			return nil
		}),
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD or relative)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD or relative)")
	cmd.Flags().IntVarP(&inboxID, "inbox-id", "I", 0, "Filter by inbox ID")
	flagAlias(cmd.Flags(), "inbox-id", "iid")
	flagAlias(cmd.Flags(), "from", "fr")
	flagAlias(cmd.Flags(), "to", "t2")

	return cmd
}

func ratingStars(rating int) string {
	filled := strings.Repeat("*", rating)
	empty := strings.Repeat("-", 5-rating)
	return filled + empty
}

func normalizeCSATDate(input string) (string, error) {
	return normalizeCSATDateWithNow(input, time.Now().UTC())
}

// normalizeCSATDateWithNow is the testable version of normalizeCSATDate.
func normalizeCSATDateWithNow(input string, now time.Time) (string, error) {
	parsed, err := cli.ParseRelativeTime(input, now)
	if err != nil {
		return "", fmt.Errorf("invalid date format %q (expected YYYY-MM-DD or relative): %w", input, err)
	}
	return parsed.Format("2006-01-02"), nil
}
