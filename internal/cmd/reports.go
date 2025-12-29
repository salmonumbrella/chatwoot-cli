package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newReportsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reports",
		Aliases: []string{"report"},
		Short:   "View reports and analytics",
		Long: `View reports and analytics from the Chatwoot API.

Available report types:
  summary      - Get summary statistics (conversations, messages, response times)
  data         - Get time-series report data for a specific metric
  live         - Get real-time conversation metrics (open/unattended/unassigned)
  agents       - Get agent conversation metrics

Date parameters use Unix timestamps. Use --from and --to flags with dates like
"2024-01-01" which will be converted to timestamps automatically.`,
	}

	cmd.AddCommand(newReportsSummaryCmd())
	cmd.AddCommand(newReportsDataCmd())
	cmd.AddCommand(newReportsLiveCmd())
	cmd.AddCommand(newReportsAgentsCmd())

	return cmd
}

// parseDate converts a date string (YYYY-MM-DD) to Unix timestamp string
func parseDate(date string) (string, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", fmt.Errorf("invalid date format %q (expected YYYY-MM-DD): %w", date, err)
	}
	return fmt.Sprintf("%d", t.Unix()), nil
}

func newReportsSummaryCmd() *cobra.Command {
	var reportType string
	var from string
	var to string
	var id string

	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Get summary report",
		Long: `Get summary report with aggregate statistics.

Report types:
  account - Account-wide summary
  agent   - Specific agent summary (requires --id)
  inbox   - Specific inbox summary (requires --id)
  label   - Specific label summary (requires --id)
  team    - Specific team summary (requires --id)`,
		Example: `  chatwoot reports summary --type account --from 2024-01-01 --to 2024-01-31
  chatwoot reports summary --type agent --id 123 --from 2024-01-01 --to 2024-01-31`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if reportType == "" {
				return fmt.Errorf("--type is required (account, agent, inbox, label, or team)")
			}
			if from == "" {
				return fmt.Errorf("--from is required (format: YYYY-MM-DD)")
			}
			if to == "" {
				return fmt.Errorf("--to is required (format: YYYY-MM-DD)")
			}

			sinceTS, err := parseDate(from)
			if err != nil {
				return err
			}
			untilTS, err := parseDate(to)
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			report, err := client.GetReportSummary(cmdContext(cmd), reportType, sinceTS, untilTS, id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(report)
			}

			fmt.Println("Report Summary:")
			fmt.Printf("Conversations: %d\n", report.ConversationsCount)
			fmt.Printf("Resolutions: %d\n", report.ResolutionsCount)
			fmt.Printf("Incoming Messages: %d\n", report.IncomingMessagesCount)
			fmt.Printf("Outgoing Messages: %d\n", report.OutgoingMessagesCount)
			fmt.Printf("Avg First Response Time: %s\n", report.AvgFirstResponseTime)
			fmt.Printf("Avg Resolution Time: %s\n", report.AvgResolutionTime)

			if report.Previous != nil {
				fmt.Println("\nPrevious Period:")
				fmt.Printf("  Conversations: %d\n", report.Previous.ConversationsCount)
				fmt.Printf("  Resolutions: %d\n", report.Previous.ResolutionsCount)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&reportType, "type", "", "Report type: account, agent, inbox, label, or team (required)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD) (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD) (required)")
	cmd.Flags().StringVar(&id, "id", "", "ID of agent/inbox/label/team (required for non-account types)")

	return cmd
}

func newReportsDataCmd() *cobra.Command {
	var metric string
	var reportType string
	var from string
	var to string
	var id string

	cmd := &cobra.Command{
		Use:   "data",
		Short: "Get time-series report data",
		Long: `Get time-series report data for a specific metric.

Metrics:
  conversations_count      - Number of conversations
  incoming_messages_count  - Number of incoming messages
  outgoing_messages_count  - Number of outgoing messages
  avg_first_response_time  - Average first response time
  avg_resolution_time      - Average resolution time
  resolutions_count        - Number of resolutions

Report types:
  account - Account-wide data
  agent   - Specific agent data (requires --id)
  inbox   - Specific inbox data (requires --id)
  label   - Specific label data (requires --id)
  team    - Specific team data (requires --id)`,
		Example: `  chatwoot reports data --metric conversations_count --type account --from 2024-01-01 --to 2024-01-31
  chatwoot reports data --metric avg_first_response_time --type inbox --id 5 --from 2024-01-01 --to 2024-01-31`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if metric == "" {
				return fmt.Errorf("--metric is required (conversations_count, incoming_messages_count, outgoing_messages_count, avg_first_response_time, avg_resolution_time, resolutions_count)")
			}
			if reportType == "" {
				return fmt.Errorf("--type is required (account, agent, inbox, label, or team)")
			}
			if from == "" {
				return fmt.Errorf("--from is required (format: YYYY-MM-DD)")
			}
			if to == "" {
				return fmt.Errorf("--to is required (format: YYYY-MM-DD)")
			}

			sinceTS, err := parseDate(from)
			if err != nil {
				return err
			}
			untilTS, err := parseDate(to)
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			report, err := client.GetReportTimeSeries(cmdContext(cmd), metric, reportType, sinceTS, untilTS, id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(report)
			}

			if len(report) == 0 {
				fmt.Println("No data points found for the specified period")
				return nil
			}

			fmt.Printf("Time-Series Report: %s (%s)\n\n", metric, reportType)
			w := newTabWriter()
			_, _ = fmt.Fprintln(w, "TIMESTAMP\tVALUE")
			for _, dp := range report {
				t := time.Unix(dp.Timestamp, 0)
				_, _ = fmt.Fprintf(w, "%s\t%s\n", t.Format("2006-01-02 15:04"), dp.Value)
			}
			_ = w.Flush()
			return nil
		},
	}

	cmd.Flags().StringVar(&metric, "metric", "", "Metric to retrieve (required)")
	cmd.Flags().StringVar(&reportType, "type", "", "Report type: account, agent, inbox, label, or team (required)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD) (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD) (required)")
	cmd.Flags().StringVar(&id, "id", "", "ID of agent/inbox/label/team (required for non-account types)")

	return cmd
}

func newReportsLiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "live",
		Short:   "Get real-time conversation metrics",
		Long:    "Get current counts of open, unattended, and unassigned conversations.",
		Example: "chatwoot reports live",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			metrics, err := client.GetConversationMetrics(cmdContext(cmd))
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(metrics)
			}

			fmt.Println("Live Conversation Metrics:")
			fmt.Printf("  Open:       %d\n", metrics.Open)
			fmt.Printf("  Unattended: %d\n", metrics.Unattended)
			fmt.Printf("  Unassigned: %d\n", metrics.Unassigned)
			return nil
		},
	}

	return cmd
}

func newReportsAgentsCmd() *cobra.Command {
	var userID string

	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Get agent conversation metrics",
		Long:  "Get current conversation metrics for all agents or a specific agent.",
		Example: `  chatwoot reports agents
  chatwoot reports agents --user-id 123`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			agents, err := client.GetAgentMetrics(cmdContext(cmd), userID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(agents)
			}

			if len(agents) == 0 {
				fmt.Println("No agent data found")
				return nil
			}

			w := newTabWriter()
			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tOPEN\tUNATTENDED\tAVAILABILITY")
			for _, agent := range agents {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%d\t%s\n",
					agent.ID, agent.Name, agent.Email,
					agent.Metric.Open, agent.Metric.Unattended,
					agent.Availability)
			}
			_ = w.Flush()
			return nil
		},
	}

	cmd.Flags().StringVar(&userID, "user-id", "", "Filter by specific user ID")
	return cmd
}
