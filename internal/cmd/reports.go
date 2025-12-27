package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newReportsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reports",
		Aliases: []string{"report"},
		Short:   "View reports and analytics",
	}

	cmd.AddCommand(newReportsSummaryCmd())
	cmd.AddCommand(newReportsConversationsCmd())
	cmd.AddCommand(newReportsMetricsCmd())
	cmd.AddCommand(newReportsAgentsCmd())
	cmd.AddCommand(newReportsInboxesCmd())
	cmd.AddCommand(newReportsTeamsCmd())
	cmd.AddCommand(newReportsLabelsCmd())
	cmd.AddCommand(newReportsBotSummaryCmd())
	cmd.AddCommand(newReportsTrafficCmd())
	cmd.AddCommand(newReportsLiveCmd())

	return cmd
}

func newReportsSummaryCmd() *cobra.Command {
	var reportType string
	var from string
	var to string

	cmd := &cobra.Command{
		Use:     "summary",
		Short:   "Get summary report",
		Example: "chatwoot reports summary --type account --from 2024-01-01 --to 2024-01-31",
		RunE: func(cmd *cobra.Command, args []string) error {
			if reportType == "" {
				return fmt.Errorf("--type is required (account, agent, inbox, or team)")
			}
			if from == "" {
				return fmt.Errorf("--from is required (format: YYYY-MM-DD)")
			}
			if to == "" {
				return fmt.Errorf("--to is required (format: YYYY-MM-DD)")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			report, err := client.GetReportSummary(cmdContext(cmd), reportType, from, to)
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
			return nil
		},
	}

	cmd.Flags().StringVar(&reportType, "type", "", "Report type: account, agent, inbox, or team (required)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD) (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD) (required)")

	return cmd
}

func newReportsConversationsCmd() *cobra.Command {
	var reportType string
	var from string
	var to string

	cmd := &cobra.Command{
		Use:     "conversations",
		Short:   "Get conversations report",
		Example: "chatwoot reports conversations --type account --from 2024-01-01 --to 2024-01-31",
		RunE: func(cmd *cobra.Command, args []string) error {
			if reportType == "" {
				return fmt.Errorf("--type is required (account, agent, inbox, or team)")
			}
			if from == "" {
				return fmt.Errorf("--from is required (format: YYYY-MM-DD)")
			}
			if to == "" {
				return fmt.Errorf("--to is required (format: YYYY-MM-DD)")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			report, err := client.GetConversationsReport(cmdContext(cmd), reportType, from, to)
			if err != nil {
				return err
			}

			return printJSON(report)
		},
	}

	cmd.Flags().StringVar(&reportType, "type", "", "Report type: account, agent, inbox, or team (required)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD) (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD) (required)")

	return cmd
}

func newReportsMetricsCmd() *cobra.Command {
	var from string
	var to string

	cmd := &cobra.Command{
		Use:     "metrics",
		Short:   "Get metrics report",
		Example: "chatwoot reports metrics --from 2024-01-01 --to 2024-01-31",
		RunE: func(cmd *cobra.Command, args []string) error {
			if from == "" {
				return fmt.Errorf("--from is required (format: YYYY-MM-DD)")
			}
			if to == "" {
				return fmt.Errorf("--to is required (format: YYYY-MM-DD)")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			report, err := client.GetMetricsReport(cmdContext(cmd), from, to)
			if err != nil {
				return err
			}

			return printJSON(report)
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD) (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD) (required)")

	return cmd
}

func newReportsAgentsCmd() *cobra.Command {
	var from, to string

	cmd := &cobra.Command{
		Use:     "agents",
		Short:   "Get agent performance report",
		Example: "chatwoot reports agents --from 2024-01-01 --to 2024-01-31",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if from == "" || to == "" {
				return fmt.Errorf("--from and --to are required (format: YYYY-MM-DD)")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			report, err := client.GetAgentsReport(cmdContext(cmd), from, to)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(report)
			}

			if len(report) == 0 {
				fmt.Println("No agent data found")
				return nil
			}

			w := newTabWriter()
			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL")
			for _, agent := range report {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\n", agent.ID, agent.Name, agent.Email)
			}
			_ = w.Flush()
			return nil
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD) (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD) (required)")
	return cmd
}

func newReportsInboxesCmd() *cobra.Command {
	var from, to string

	cmd := &cobra.Command{
		Use:     "inboxes",
		Short:   "Get inbox metrics report",
		Example: "chatwoot reports inboxes --from 2024-01-01 --to 2024-01-31",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if from == "" || to == "" {
				return fmt.Errorf("--from and --to are required (format: YYYY-MM-DD)")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			report, err := client.GetInboxesReport(cmdContext(cmd), from, to)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(report)
			}

			if len(report) == 0 {
				fmt.Println("No inbox data found")
				return nil
			}

			w := newTabWriter()
			_, _ = fmt.Fprintln(w, "ID\tNAME")
			for _, inbox := range report {
				_, _ = fmt.Fprintf(w, "%d\t%s\n", inbox.ID, inbox.Name)
			}
			_ = w.Flush()
			return nil
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD) (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD) (required)")
	return cmd
}

func newReportsTeamsCmd() *cobra.Command {
	var from, to string

	cmd := &cobra.Command{
		Use:     "teams",
		Short:   "Get team metrics report",
		Example: "chatwoot reports teams --from 2024-01-01 --to 2024-01-31",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if from == "" || to == "" {
				return fmt.Errorf("--from and --to are required (format: YYYY-MM-DD)")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			report, err := client.GetTeamsReport(cmdContext(cmd), from, to)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(report)
			}

			if len(report) == 0 {
				fmt.Println("No team data found")
				return nil
			}

			w := newTabWriter()
			_, _ = fmt.Fprintln(w, "ID\tNAME")
			for _, team := range report {
				_, _ = fmt.Fprintf(w, "%d\t%s\n", team.ID, team.Name)
			}
			_ = w.Flush()
			return nil
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD) (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD) (required)")
	return cmd
}

func newReportsLabelsCmd() *cobra.Command {
	var from, to string

	cmd := &cobra.Command{
		Use:     "labels",
		Short:   "Get label analytics report",
		Example: "chatwoot reports labels --from 2024-01-01 --to 2024-01-31",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if from == "" || to == "" {
				return fmt.Errorf("--from and --to are required (format: YYYY-MM-DD)")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			report, err := client.GetLabelsReport(cmdContext(cmd), from, to)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(report)
			}

			if len(report) == 0 {
				fmt.Println("No label data found")
				return nil
			}

			w := newTabWriter()
			_, _ = fmt.Fprintln(w, "ID\tTITLE")
			for _, label := range report {
				_, _ = fmt.Fprintf(w, "%d\t%s\n", label.ID, label.Title)
			}
			_ = w.Flush()
			return nil
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD) (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD) (required)")
	return cmd
}

func newReportsBotSummaryCmd() *cobra.Command {
	var from, to string

	cmd := &cobra.Command{
		Use:     "bot-summary",
		Short:   "Get bot performance summary",
		Example: "chatwoot reports bot-summary --from 2024-01-01 --to 2024-01-31",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if from == "" || to == "" {
				return fmt.Errorf("--from and --to are required (format: YYYY-MM-DD)")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			report, err := client.GetBotSummaryReport(cmdContext(cmd), from, to)
			if err != nil {
				return err
			}

			return printJSON(report)
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD) (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD) (required)")
	return cmd
}

func newReportsTrafficCmd() *cobra.Command {
	var from, to string

	cmd := &cobra.Command{
		Use:     "traffic",
		Short:   "Get conversation traffic time-series data",
		Example: "chatwoot reports traffic --from 2024-01-01 --to 2024-01-31",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if from == "" || to == "" {
				return fmt.Errorf("--from and --to are required (format: YYYY-MM-DD)")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			report, err := client.GetConversationTrafficReport(cmdContext(cmd), from, to)
			if err != nil {
				return err
			}

			return printJSON(report)
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD) (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD) (required)")
	return cmd
}

func newReportsLiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "live",
		Short:   "Get real-time conversation metrics",
		Example: "chatwoot reports live",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			metrics, err := client.GetLiveMetrics(cmdContext(cmd))
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(metrics)
			}

			fmt.Println("Live Metrics:")
			fmt.Printf("  Open:       %d\n", metrics.Open)
			fmt.Printf("  Pending:    %d\n", metrics.Pending)
			fmt.Printf("  Unassigned: %d\n", metrics.Unassigned)
			return nil
		},
	}

	return cmd
}
