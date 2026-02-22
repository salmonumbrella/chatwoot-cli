package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/dryrun"
	"github.com/spf13/cobra"
)

func newSnoozeCmd() *cobra.Command {
	var (
		forDuration string
		note        string
	)

	cmd := &cobra.Command{
		Use:     "snooze <conversation-id|url>",
		Aliases: []string{"pause", "defer", "sn"},
		Short:   "Snooze a conversation",
		Long: `Snooze a conversation for a specified duration.

This is a convenience shortcut for:
  cw conversations toggle-status <id> --status snoozed`,
		Example: strings.TrimSpace(`
  # Snooze for 2 hours
  cw snooze 123 --for 2h

  # Snooze for 1 day
  cw snooze 123 --for 24h

  # Snooze with a note
  cw snooze 123 --for 2h --note "Waiting for customer to check email"
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			conversationID, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			if forDuration == "" {
				return fmt.Errorf("--for is required (e.g., --for 2h)")
			}

			snoozedUntil, err := parseSnoozeFor(forDuration, time.Now())
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "snooze",
				Resource:  "conversation",
				Details: map[string]any{
					"conversation_id": conversationID,
					"snoozed_until":   snoozedUntil.Format(time.RFC3339),
					"note":            note,
				},
			}); ok {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			// Send note first if provided.
			if strings.TrimSpace(note) != "" {
				_, err := client.Messages().Create(ctx, conversationID, note, true, "outgoing")
				if err != nil {
					return fmt.Errorf("failed to add snooze note: %w", err)
				}
			}

			result, err := client.Conversations().ToggleStatus(ctx, conversationID, "snoozed", snoozedUntil.Unix())
			if err != nil {
				return fmt.Errorf("failed to snooze conversation %d: %w", conversationID, err)
			}

			snoozedStr := snoozedUntil.Format(time.RFC3339)
			if result.Payload.SnoozedUntil != nil && *result.Payload.SnoozedUntil > 0 {
				snoozedStr = time.Unix(*result.Payload.SnoozedUntil, 0).Format(time.RFC3339)
			}

			out := map[string]any{
				"action":          "snoozed",
				"conversation_id": conversationID,
				"snoozed_until":   snoozedStr,
			}
			if note != "" {
				out["note"] = note
			}

			if isAgent(cmd) {
				return printJSON(cmd, agentfmt.ItemEnvelope{
					Kind: "snooze",
					Item: out,
				})
			}
			if isJSON(cmd) {
				return printJSON(cmd, out)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Snoozed conversation %d until %s\n", conversationID, formatTimestampWithZone(snoozedUntil))
			return nil
		}),
	}

	cmd.Flags().StringVar(&forDuration, "for", "", "Snooze duration (e.g., 2h, 30m, 24h)")
	flagAlias(cmd.Flags(), "for", "fr")
	cmd.Flags().StringVar(&note, "note", "", "Add a private note before snoozing")
	_ = cmd.MarkFlagRequired("for")
	flagAlias(cmd.Flags(), "note", "nt")

	return cmd
}

func parseSnoozeFor(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("duration cannot be empty")
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid duration %q: %w (use Go duration syntax like 2h, 30m, 24h)", s, err)
	}
	if d <= 0 {
		return time.Time{}, fmt.Errorf("duration must be positive")
	}
	return now.Add(d), nil
}
