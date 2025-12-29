package cmd

import (
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newCSATCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "csat",
		Aliases: []string{"satisfaction"},
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
		Use:   "list",
		Short: "List CSAT responses",
		Example: strings.TrimSpace(`
  # List all CSAT responses
  chatwoot csat list

  # List responses in date range
  chatwoot csat list --from 2024-01-01 --to 2024-01-31

  # Filter by low ratings
  chatwoot csat list --rating 1,2

  # Filter by inbox
  chatwoot csat list --inbox-id 5
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			params := api.CSATListParams{
				Page:    page,
				Since:   from,
				Until:   to,
				InboxID: inboxID,
				Rating:  rating,
			}

			responses, err := client.ListCSATResponses(cmdContext(cmd), params)
			if err != nil {
				return fmt.Errorf("failed to list CSAT responses: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, responses)
			}

			if len(responses) == 0 {
				fmt.Println("No CSAT responses found")
				return nil
			}

			w := newTabWriter()
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
					csat.CreatedAtTime().Format("2006-01-02 15:04"),
				)
			}
			_ = w.Flush()

			return nil
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD)")
	cmd.Flags().IntVar(&inboxID, "inbox-id", 0, "Filter by inbox ID")
	cmd.Flags().StringVar(&rating, "rating", "", "Filter by ratings (comma-separated, e.g., 1,2)")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")

	return cmd
}

func newCSATGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <conversation-id>",
		Short: "Get CSAT for a conversation",
		Example: strings.TrimSpace(`
  # Get CSAT for conversation
  chatwoot csat get 123

  # JSON output
  chatwoot csat get 123 -o json
`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			conversationID, err := validation.ParsePositiveInt(args[0], "conversation ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			csat, err := client.GetConversationCSAT(cmdContext(cmd), conversationID)
			if err != nil {
				return fmt.Errorf("failed to get CSAT for conversation %d: %w", conversationID, err)
			}

			if csat == nil {
				if isJSON(cmd) {
					return printJSON(cmd, nil)
				}
				fmt.Printf("No CSAT response for conversation %d\n", conversationID)
				return nil
			}

			if isJSON(cmd) {
				return printJSON(cmd, csat)
			}

			fmt.Printf("CSAT for Conversation #%d\n", conversationID)
			fmt.Printf("  Rating:   %d/5 %s\n", csat.Rating, ratingStars(csat.Rating))
			if csat.FeedbackMessage != "" {
				fmt.Printf("  Feedback: %s\n", csat.FeedbackMessage)
			}
			fmt.Printf("  Created:  %s\n", csat.CreatedAtTime().Format("2006-01-02 15:04:05"))

			return nil
		},
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
  chatwoot csat summary --from 2024-01-01 --to 2024-01-31

  # Get summary for inbox
  chatwoot csat summary --inbox-id 5 --from 2024-01-01 --to 2024-12-31
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
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

				responses, err := client.ListCSATResponses(cmdContext(cmd), params)
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
				fmt.Println("No CSAT responses found for the specified period")
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

			fmt.Printf("CSAT Summary%s:\n", dateRange)
			fmt.Printf("  Total Responses:    %d\n", total)
			fmt.Printf("  Average Rating:     %.1f / 5\n", avg)
			fmt.Printf("  Satisfaction Rate:  %.0f%%\n\n", satisfactionRate)
			fmt.Println("  Distribution:")
			for i := 5; i >= 1; i-- {
				count := distribution[i]
				pct := float64(count) / float64(total) * 100
				fmt.Printf("    %s (%d): %d (%.0f%%)\n", ratingStars(i), i, count, pct)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD)")
	cmd.Flags().IntVar(&inboxID, "inbox-id", 0, "Filter by inbox ID")

	return cmd
}

func ratingStars(rating int) string {
	filled := strings.Repeat("*", rating)
	empty := strings.Repeat("-", 5-rating)
	return filled + empty
}
