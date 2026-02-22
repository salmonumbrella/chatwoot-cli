package cmd

import (
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newCloseCmd() *cobra.Command {
	var light bool

	cmd := &cobra.Command{
		Use:     "close <conversation-id> [conversation-id...]",
		Aliases: []string{"close-conversation", "resolve-conversation", "resolve", "x"},
		Short:   "Resolve one or more conversations",
		Long: `Resolve one or more conversations by setting their status to "resolved".

Aliases: close, resolve, close-conversation, resolve-conversation.

This is a convenience shortcut for:
  cw conversations toggle-status <id> --status resolved`,
		Example: strings.TrimSpace(`
  # Close a single conversation
  cw close 123

  # Close multiple conversations
  cw close 123 456 789

  # JSON output summary
  cw close 123 456 --output json

  # Light token-optimized output
  cw close 123 --li
`),
		Args: cobra.MinimumNArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			ids, err := parseIDArgs(args, "conversation")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)
			var okCount int
			var failures []string

			for _, id := range ids {
				resp, err := client.Conversations().ToggleStatus(ctx, id, "resolved", 0)
				if err != nil {
					failures = append(failures, fmt.Sprintf("conversation %d: %v", id, err))
					if !isJSON(cmd) {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Failed to close conversation %d: %v\n", id, err)
					}
					continue
				}
				if resp.Payload.Success {
					okCount++
					if !isJSON(cmd) {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversation #%d closed\n", id)
					}
				} else {
					failures = append(failures, fmt.Sprintf("conversation %d: status change unsuccessful", id))
					if !isJSON(cmd) {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Failed to close conversation %d: status change unsuccessful\n", id)
					}
				}
			}

			summary := map[string]any{
				"closed": okCount,
				"total":  len(ids),
			}

			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, buildLightBulkMutationSummary(okCount, len(ids)))
			}
			if isJSON(cmd) {
				// Keep close/reopen summaries flat in both json and agent modes.
				return printRawJSON(cmd, summary)
			}
			if len(failures) > 0 {
				return fmt.Errorf("failed to close %d of %d conversations", len(failures), len(ids))
			}
			return nil
		}),
	}

	cmd.Flags().BoolVar(&light, "light", false, "Return minimal mutation payload")
	flagAlias(cmd.Flags(), "light", "li")

	return cmd
}

func newReopenCmd() *cobra.Command {
	var light bool

	cmd := &cobra.Command{
		Use:     "reopen <conversation-id> [conversation-id...]",
		Aliases: []string{"open-conversation", "ro"},
		Short:   "Reopen one or more conversations",
		Long: `Reopen one or more conversations by setting their status to "open".

This is a convenience shortcut for:
  cw conversations toggle-status <id> --status open`,
		Example: strings.TrimSpace(`
  # Reopen a single conversation
  cw reopen 123

  # Reopen multiple conversations
  cw reopen 123 456 789

  # JSON output summary
  cw reopen 123 456 --output json

  # Light token-optimized output
  cw reopen 123 --li
`),
		Args: cobra.MinimumNArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			ids, err := parseIDArgs(args, "conversation")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)
			var okCount int
			var failures []string

			for _, id := range ids {
				resp, err := client.Conversations().ToggleStatus(ctx, id, "open", 0)
				if err != nil {
					failures = append(failures, fmt.Sprintf("conversation %d: %v", id, err))
					if !isJSON(cmd) {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Failed to reopen conversation %d: %v\n", id, err)
					}
					continue
				}
				if resp.Payload.Success {
					okCount++
					if !isJSON(cmd) {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversation #%d reopened\n", id)
					}
				} else {
					failures = append(failures, fmt.Sprintf("conversation %d: status change unsuccessful", id))
					if !isJSON(cmd) {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Failed to reopen conversation %d: status change unsuccessful\n", id)
					}
				}
			}

			summary := map[string]any{
				"reopened": okCount,
				"total":    len(ids),
			}

			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, buildLightBulkMutationSummary(okCount, len(ids)))
			}
			if isJSON(cmd) {
				// Keep close/reopen summaries flat in both json and agent modes.
				return printRawJSON(cmd, summary)
			}
			if len(failures) > 0 {
				return fmt.Errorf("failed to reopen %d of %d conversations", len(failures), len(ids))
			}
			return nil
		}),
	}

	cmd.Flags().BoolVar(&light, "light", false, "Return minimal mutation payload")
	flagAlias(cmd.Flags(), "light", "li")

	return cmd
}
