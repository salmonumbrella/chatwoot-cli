package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newReportsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reports",
		Aliases: []string{"report", "rpt", "rp"},
		Short:   "View reports and analytics",
		Long: `View reports and analytics from the Chatwoot API.

Available report types:
  summary      - Get summary statistics (conversations, messages, response times)
  data         - Get time-series report data for a specific metric
  live         - Get real-time conversation metrics (open/unattended/unassigned)
  agents       - Get live agent conversation metrics
  agent-summary - Get summary report grouped by agent
  inboxes      - Get summary report grouped by inbox
  teams        - Get summary report grouped by team
  channels     - Get conversation statistics grouped by channel type

Date parameters use Unix timestamps. Use --from and --to flags with dates like
"2024-01-01" or relative expressions like "yesterday"; values are converted
to timestamps automatically.`,
	}

	cmd.AddCommand(newReportsSummaryCmd())
	cmd.AddCommand(newReportsDataCmd())
	cmd.AddCommand(newReportsLiveCmd())
	cmd.AddCommand(newReportsAgentsCmd())
	cmd.AddCommand(newReportsAgentSummaryCmd())
	cmd.AddCommand(newReportsInboxesCmd())
	cmd.AddCommand(newReportsTeamsCmd())
	cmd.AddCommand(newReportsChannelsCmd())
	cmd.AddCommand(newReportingEventsCmd())

	return cmd
}

// parseDate converts a date string to Unix timestamp string.
func parseDate(date string) (string, error) {
	t, err := parseDateTime(date)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", t.Unix()), nil
}

func parseDateTime(date string) (time.Time, error) {
	t, err := cli.ParseRelativeTime(date, time.Now().UTC())
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format %q (expected YYYY-MM-DD or relative): %w", date, err)
	}
	return t, nil
}

func parseOptionalDateRange(from, to string) (string, string, error) {
	var sinceTS string
	var untilTS string
	var fromTime, toTime time.Time

	if from != "" {
		parsed, err := parseDateTime(from)
		if err != nil {
			return "", "", err
		}
		sinceTS = fmt.Sprintf("%d", parsed.Unix())
		fromTime = parsed
	}
	if to != "" {
		parsed, err := parseDateTime(to)
		if err != nil {
			return "", "", err
		}
		untilTS = fmt.Sprintf("%d", parsed.Unix())
		toTime = parsed
	}

	if !fromTime.IsZero() && !toTime.IsZero() && toTime.Before(fromTime) {
		return "", "", fmt.Errorf("--to date (%s) must be on or after --from date (%s)", to, from)
	}

	return sinceTS, untilTS, nil
}

func formatOptionalFloat(value *api.FlexFloat) string {
	if value == nil {
		return "-"
	}
	return formatValue(float64(*value))
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
		Example: `  cw reports summary --type account --from 2024-01-01 --to 2024-01-31
  cw reports summary --type agent --id 123 --from 2024-01-01 --to 2024-01-31`,
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
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

			report, err := client.Reports().Summary(cmdContext(cmd), reportType, sinceTS, untilTS, id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, report)
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Report Summary:")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversations: %d\n", report.ConversationsCount)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Resolutions: %d\n", report.ResolutionsCount)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Incoming Messages: %d\n", report.IncomingMessagesCount)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Outgoing Messages: %d\n", report.OutgoingMessagesCount)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Avg First Response Time: %s\n", report.AvgFirstResponseTime)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Avg Resolution Time: %s\n", report.AvgResolutionTime)

			if report.Previous != nil {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nPrevious Period:")
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Conversations: %d\n", report.Previous.ConversationsCount)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Resolutions: %d\n", report.Previous.ResolutionsCount)
			}
			return nil
		}),
	}

	cmd.Flags().StringVar(&reportType, "type", "", "Report type: account, agent, inbox, label, or team (required)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD or relative) (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD or relative) (required)")
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
		Example: `  cw reports data --metric conversations_count --type account --from 2024-01-01 --to 2024-01-31
  cw reports data --metric avg_first_response_time --type inbox --id 5 --from 2024-01-01 --to 2024-01-31`,
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
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

			report, err := client.Reports().TimeSeries(cmdContext(cmd), metric, reportType, sinceTS, untilTS, id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, report)
			}

			if len(report) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No data points found for the specified period")
				return nil
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Time-Series Report: %s (%s)\n\n", metric, reportType)
			w := newTabWriterFromCmd(cmd)
			_, _ = fmt.Fprintln(w, "TIMESTAMP\tVALUE")
			for _, dp := range report {
				t := time.Unix(dp.Timestamp, 0)
				_, _ = fmt.Fprintf(w, "%s\t%s\n", formatTimestampShort(t), dp.Value)
			}
			_ = w.Flush()
			return nil
		}),
	}

	cmd.Flags().StringVar(&metric, "metric", "", "Metric to retrieve (required)")
	cmd.Flags().StringVar(&reportType, "type", "", "Report type: account, agent, inbox, label, or team (required)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD or relative) (required)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD or relative) (required)")
	cmd.Flags().StringVar(&id, "id", "", "ID of agent/inbox/label/team (required for non-account types)")

	return cmd
}

func newReportsLiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "live",
		Short:   "Get real-time conversation metrics",
		Long:    "Get current counts of open, unattended, and unassigned conversations.",
		Example: "cw reports live",
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			metrics, err := client.Reports().ConversationMetrics(cmdContext(cmd))
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, metrics)
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Live Conversation Metrics:")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Open:       %d\n", metrics.Open)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Unattended: %d\n", metrics.Unattended)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Unassigned: %d\n", metrics.Unassigned)
			return nil
		}),
	}

	return cmd
}

func newReportsAgentsCmd() *cobra.Command {
	var userID string

	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Get agent conversation metrics",
		Long:  "Get current conversation metrics for all agents or a specific agent.",
		Example: `  cw reports agents
  cw reports agents --user-id 123`,
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			agents, err := client.Reports().AgentMetrics(cmdContext(cmd), userID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, agents)
			}

			if len(agents) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No agent data found")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tOPEN\tUNATTENDED\tAVAILABILITY")
			for _, agent := range agents {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%d\t%s\n",
					agent.ID, agent.Name, agent.Email,
					agent.Metric.Open, agent.Metric.Unattended,
					agent.Availability)
			}
			_ = w.Flush()
			return nil
		}),
	}

	cmd.Flags().StringVar(&userID, "user-id", "", "Filter by specific user ID")
	return cmd
}

func newReportsChannelsCmd() *cobra.Command {
	var from string
	var to string
	var businessHours bool

	cmd := &cobra.Command{
		Use:   "channels",
		Short: "Get conversation statistics grouped by channel type",
		Long: `Get conversation statistics grouped by channel type.

Date parameters use YYYY-MM-DD or relative expressions and are converted to Unix timestamps.`,
		Example: `  cw reports channels --from 2024-01-01 --to 2024-01-31
  cw reports channels --business-hours
  cw reports channels -o json`,
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			sinceTS, untilTS, err := parseOptionalDateRange(from, to)
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			channelSummary, err := client.Reports().ChannelSummary(cmdContext(cmd), sinceTS, untilTS, boolPtrIfChanged(cmd, "business-hours", businessHours))
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, channelSummary)
			}

			if len(channelSummary) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No channel summary data found")
				return nil
			}

			keys := make([]string, 0, len(channelSummary))
			for k := range channelSummary {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			w := newTabWriterFromCmd(cmd)
			_, _ = fmt.Fprintln(w, "CHANNEL\tOPEN\tRESOLVED\tPENDING\tSNOOZED\tTOTAL")
			for _, k := range keys {
				summary := channelSummary[k]
				_, _ = fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\t%d\n",
					k, summary.Open, summary.Resolved, summary.Pending, summary.Snoozed, summary.Total)
			}
			_ = w.Flush()
			return nil
		}),
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD or relative)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD or relative)")
	cmd.Flags().BoolVar(&businessHours, "business-hours", false, "Restrict to business hours")

	return cmd
}

func renderSummaryReportEntries(cmd *cobra.Command, entries []api.SummaryReportEntry, emptyMessage string) error {
	if isJSON(cmd) {
		return printJSON(cmd, entries)
	}

	if len(entries) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), emptyMessage)
		return nil
	}

	w := newTabWriterFromCmd(cmd)
	_, _ = fmt.Fprintln(w, "ID\tCONVERSATIONS\tRESOLVED\tAVG_FIRST_RESPONSE\tAVG_RESOLUTION\tAVG_REPLY")
	for _, entry := range entries {
		_, _ = fmt.Fprintf(w, "%d\t%d\t%d\t%s\t%s\t%s\n",
			entry.ID,
			entry.ConversationsCount,
			entry.ResolvedConversationsCount,
			formatOptionalFloat(entry.AvgFirstResponseTime),
			formatOptionalFloat(entry.AvgResolutionTime),
			formatOptionalFloat(entry.AvgReplyTime),
		)
	}
	_ = w.Flush()
	return nil
}

func newReportsAgentSummaryCmd() *cobra.Command {
	var from string
	var to string
	var businessHours bool

	cmd := &cobra.Command{
		Use:     "agent-summary",
		Aliases: []string{"agents-summary"},
		Short:   "Get summary report grouped by agent",
		Long: `Get summary report grouped by agent.

Date parameters use YYYY-MM-DD or relative expressions and are converted to Unix timestamps.`,
		Example: `  cw reports agent-summary --from 2024-01-01 --to 2024-01-31
  cw reports agent-summary --business-hours`,
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			sinceTS, untilTS, err := parseOptionalDateRange(from, to)
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			entries, err := client.Reports().SummaryByAgent(cmdContext(cmd), sinceTS, untilTS, boolPtrIfChanged(cmd, "business-hours", businessHours))
			if err != nil {
				return err
			}

			return renderSummaryReportEntries(cmd, entries, "No agent summary data found")
		}),
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD or relative)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD or relative)")
	cmd.Flags().BoolVar(&businessHours, "business-hours", false, "Restrict to business hours")

	return cmd
}

func newReportsInboxesCmd() *cobra.Command {
	var from string
	var to string
	var businessHours bool

	cmd := &cobra.Command{
		Use:   "inboxes",
		Short: "Get summary report grouped by inbox",
		Long: `Get summary report grouped by inbox.

Date parameters use YYYY-MM-DD or relative expressions and are converted to Unix timestamps.`,
		Example: `  cw reports inboxes --from 2024-01-01 --to 2024-01-31
  cw reports inboxes --business-hours`,
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			sinceTS, untilTS, err := parseOptionalDateRange(from, to)
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			entries, err := client.Reports().SummaryByInbox(cmdContext(cmd), sinceTS, untilTS, boolPtrIfChanged(cmd, "business-hours", businessHours))
			if err != nil {
				return err
			}

			return renderSummaryReportEntries(cmd, entries, "No inbox summary data found")
		}),
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD or relative)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD or relative)")
	cmd.Flags().BoolVar(&businessHours, "business-hours", false, "Restrict to business hours")

	return cmd
}

func newReportsTeamsCmd() *cobra.Command {
	var from string
	var to string
	var businessHours bool

	cmd := &cobra.Command{
		Use:   "teams",
		Short: "Get summary report grouped by team",
		Long: `Get summary report grouped by team.

Date parameters use YYYY-MM-DD or relative expressions and are converted to Unix timestamps.`,
		Example: `  cw reports teams --from 2024-01-01 --to 2024-01-31
  cw reports teams --business-hours`,
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			sinceTS, untilTS, err := parseOptionalDateRange(from, to)
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			entries, err := client.Reports().SummaryByTeam(cmdContext(cmd), sinceTS, untilTS, boolPtrIfChanged(cmd, "business-hours", businessHours))
			if err != nil {
				return err
			}

			return renderSummaryReportEntries(cmd, entries, "No team summary data found")
		}),
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD or relative)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD or relative)")
	cmd.Flags().BoolVar(&businessHours, "business-hours", false, "Restrict to business hours")

	return cmd
}

func newReportingEventsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Manage reporting events",
	}

	// List account events
	var since, until, eventType string
	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List account reporting events",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			parseEventTimestamp := func(value string) (string, time.Time, error) {
				trimmed := strings.TrimSpace(value)
				if trimmed == "" {
					return "", time.Time{}, nil
				}
				if ts, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
					return trimmed, time.Unix(ts, 0), nil
				}
				parsed, err := cli.ParseRelativeTime(trimmed, time.Now().UTC())
				if err != nil {
					return "", time.Time{}, fmt.Errorf("invalid time %q (use Unix, RFC3339, or relative): %w", value, err)
				}
				return fmt.Sprintf("%d", parsed.Unix()), parsed, nil
			}

			var sinceTime, untilTime time.Time
			if since != "" {
				parsed, parsedTime, err := parseEventTimestamp(since)
				if err != nil {
					return err
				}
				since = parsed
				sinceTime = parsedTime
			}
			if until != "" {
				parsed, parsedTime, err := parseEventTimestamp(until)
				if err != nil {
					return err
				}
				until = parsed
				untilTime = parsedTime
			}
			if !sinceTime.IsZero() && !untilTime.IsZero() && untilTime.Before(sinceTime) {
				return fmt.Errorf("--until must be on or after --since")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			events, err := client.Reports().ListEvents(cmdContext(cmd), since, until, eventType)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, events)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tNAME\tVALUE\tCREATED")
			for _, e := range events {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%v\t%s\n", e.ID, e.Name, e.Value, e.CreatedAt)
			}
			return nil
		}),
	}
	listCmd.Flags().StringVarP(&since, "since", "S", "", "Start time (Unix, RFC3339, YYYY-MM-DD, or relative)")
	listCmd.Flags().StringVar(&until, "until", "", "End time (Unix, RFC3339, YYYY-MM-DD, or relative)")
	listCmd.Flags().StringVar(&eventType, "type", "", "Event type filter")
	cmd.AddCommand(listCmd)

	// Conversation events
	cmd.AddCommand(&cobra.Command{
		Use:   "conversation <conversation-id>",
		Short: "List reporting events for a conversation",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			conversationID, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			events, err := client.Reports().ConversationEvents(cmdContext(cmd), conversationID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, events)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tNAME\tVALUE\tCREATED")
			for _, e := range events {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%v\t%s\n", e.ID, e.Name, e.Value, e.CreatedAt)
			}
			return nil
		}),
	})

	return cmd
}
