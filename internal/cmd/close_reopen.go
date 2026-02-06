package cmd

import (
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/spf13/cobra"
)

func newCloseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "close <conversation-id> [conversation-id...]",
		Aliases: []string{"close-conversation", "resolve-conversation"},
		Short:   "Resolve one or more conversations",
		Long: `Resolve one or more conversations by setting their status to "resolved".

This is a convenience shortcut for:
  chatwoot conversations toggle-status <id> --status resolved
and overlaps with:
  chatwoot resolve <id> [id...]`,
		Example: strings.TrimSpace(`
  # Close a single conversation
  chatwoot close 123

  # Close multiple conversations
  chatwoot close 123 456 789

  # JSON output summary
  chatwoot close 123 456 --output json
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

			if isAgent(cmd) {
				return printJSON(cmd, agentfmt.ItemEnvelope{
					Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Item: summary,
				})
			}
			if isJSON(cmd) {
				return printJSON(cmd, summary)
			}

			if len(failures) > 0 {
				return fmt.Errorf("failed to close %d of %d conversations", len(failures), len(ids))
			}
			return nil
		}),
	}

	return cmd
}

func newReopenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reopen <conversation-id> [conversation-id...]",
		Aliases: []string{"open-conversation"},
		Short:   "Reopen one or more conversations",
		Long: `Reopen one or more conversations by setting their status to "open".

This is a convenience shortcut for:
  chatwoot conversations toggle-status <id> --status open`,
		Example: strings.TrimSpace(`
  # Reopen a single conversation
  chatwoot reopen 123

  # Reopen multiple conversations
  chatwoot reopen 123 456 789

  # JSON output summary
  chatwoot reopen 123 456 --output json
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

			if isAgent(cmd) {
				return printJSON(cmd, agentfmt.ItemEnvelope{
					Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Item: summary,
				})
			}
			if isJSON(cmd) {
				return printJSON(cmd, summary)
			}

			if len(failures) > 0 {
				return fmt.Errorf("failed to reopen %d of %d conversations", len(failures), len(ids))
			}
			return nil
		}),
	}

	return cmd
}
