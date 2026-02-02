package cmd

import (
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

// newResolveCmd creates the top-level resolve command for quick conversation resolution
func newResolveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve <conversation-id> [conversation-id...]",
		Short: "Resolve one or more conversations",
		Long: `Resolve one or more conversations by setting their status to "resolved".

This is a convenience shortcut for 'chatwoot conversations status <id> resolved'.
Accepts multiple conversation IDs to resolve in a single command.`,
		Example: strings.TrimSpace(`
  # Resolve a single conversation
  chatwoot resolve 123

  # Resolve multiple conversations
  chatwoot resolve 123 456 789

  # JSON output (summary: resolved/total counts)
  chatwoot resolve 123 456 --output json
`),
		Args: cobra.MinimumNArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			// Parse and validate all conversation IDs upfront
			ids := make([]int, 0, len(args))
			for _, arg := range args {
				id, err := validation.ParsePositiveInt(arg, "conversation ID")
				if err != nil {
					return err
				}
				ids = append(ids, id)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			var resolved int
			var errors []string

			for _, id := range ids {
				resp, err := client.Conversations().ToggleStatus(ctx, id, "resolved", 0)
				if err != nil {
					errors = append(errors, fmt.Sprintf("conversation %d: %v", id, err))
					if !isJSON(cmd) {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Failed to resolve conversation %d: %v\n", id, err)
					}
					continue
				}

				if resp.Payload.Success {
					resolved++
					if !isJSON(cmd) {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversation #%d resolved\n", id)
					}
				} else {
					errors = append(errors, fmt.Sprintf("conversation %d: status change unsuccessful", id))
					if !isJSON(cmd) {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Failed to resolve conversation %d: status change unsuccessful\n", id)
					}
				}
			}

			if isJSON(cmd) {
				summary := map[string]int{
					"resolved": resolved,
					"total":    len(ids),
				}
				if err := printJSON(cmd, summary); err != nil {
					return err
				}
			}

			if len(errors) > 0 {
				return fmt.Errorf("failed to resolve %d of %d conversations", len(errors), len(ids))
			}

			return nil
		}),
	}

	return cmd
}
