package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/dryrun"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

// CommentResult is the response for the comment/note commands.
type CommentResult struct {
	Action         string       `json:"action"` // "commented" | "noted"
	ConversationID int          `json:"conversation_id"`
	MessageID      int          `json:"message_id,omitempty"`
	Message        *api.Message `json:"message,omitempty"`
	Private        bool         `json:"private"`
	Resolved       bool         `json:"resolved,omitempty"`
	URL            string       `json:"url,omitempty"`
}

func newCommentCmd() *cobra.Command {
	var (
		content   string
		resolve   bool
		labels    []string
		priority  string
		snoozeFor string
	)

	cmd := &cobra.Command{
		Use:     "comment <conversation-id|url> [text...]",
		Aliases: []string{"cmt"},
		Short:   "Send a public reply to a conversation",
		Long: `Send a public (customer-visible) message to a conversation.

This is a convenience shortcut for:
  cw messages create <conversation-id> --content "..."`,
		Example: strings.TrimSpace(`
  # Comment by conversation ID
  cw comment 123 "Hello! How can I help?"

  # Comment by URL from browser
  cw comment https://app.chatwoot.com/app/accounts/1/conversations/123 "On it."

  # Resolve after commenting
  cw comment 123 "Done" --resolve

  # Use --content instead of positional text
  cw comment 123 --content "Hello!"

  # Agent-friendly envelope
  cw comment 123 "Hello" --output agent
`),
		Args: cobra.MinimumNArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			conversationID, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			// Allow text either via positional args or --content.
			positional := strings.TrimSpace(strings.Join(args[1:], " "))
			if cmd.Flags().Changed("content") && positional != "" {
				return fmt.Errorf("provide message text either as args or with --content, not both")
			}
			if !cmd.Flags().Changed("content") {
				content = positional
			}
			if content == "" {
				return fmt.Errorf("message text is required (use --content or provide trailing args)")
			}

			if err := validation.ValidateMessageContent(content); err != nil {
				return err
			}

			// Validate side-effect flags before sending so we fail fast.
			if priority != "" {
				if priority, err = validatePriority(priority); err != nil {
					return err
				}
			}
			if snoozeFor != "" {
				if _, err := parseSnoozeFor(snoozeFor, time.Now()); err != nil {
					return err
				}
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "create",
				Resource:  "message",
				Details: map[string]any{
					"conversation_id": conversationID,
					"content":         content,
					"private":         false,
					"type":            "outgoing",
					"resolve":         resolve,
				},
			}); ok {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			message, err := client.Messages().Create(ctx, conversationID, content, false, "outgoing")
			if err != nil {
				return fmt.Errorf("failed to send message to conversation %d: %w", conversationID, err)
			}

			resolved := false
			if resolve {
				_, err := client.Conversations().ToggleStatus(ctx, conversationID, "resolved", 0)
				if err != nil {
					return fmt.Errorf("message sent (ID: %d) but failed to resolve conversation: %w", message.ID, err)
				}
				resolved = true
			}

			if len(labels) > 0 {
				existing, _ := client.Conversations().Labels(ctx, conversationID)
				merged := dedupeStrings(append(existing, labels...))
				if _, err := client.Conversations().AddLabels(ctx, conversationID, merged); err != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: message sent but failed to add labels: %v\n", err)
				}
			}
			if priority != "" {
				if err := client.Conversations().TogglePriority(ctx, conversationID, priority); err != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: message sent but failed to set priority: %v\n", err)
				}
			}
			if snoozeFor != "" {
				snoozedUntil, err := parseSnoozeFor(snoozeFor, time.Now())
				if err != nil {
					return err // should not happen; already validated above
				}
				_, err = client.Conversations().ToggleStatus(ctx, conversationID, "snoozed", snoozedUntil.Unix())
				if err != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: message sent but failed to snooze: %v\n", err)
				}
			}

			u, _ := resourceURL("conversations", conversationID)
			result := CommentResult{
				Action:         "commented",
				ConversationID: conversationID,
				MessageID:      message.ID,
				Message:        message,
				Private:        false,
				Resolved:       resolved,
				URL:            u,
			}

			if isAgent(cmd) {
				return printJSON(cmd, agentfmt.ItemEnvelope{
					Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Item: result,
				})
			}
			if isJSON(cmd) {
				return printJSON(cmd, result)
			}

			printAction(cmd, "Sent", "message", message.ID, "")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversation: %d\n", conversationID)
			if resolved {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Status: Resolved")
			}
			return nil
		}),
	}

	cmd.Flags().StringVarP(&content, "content", "c", "", "Message content (alternative to positional text)")
	cmd.Flags().BoolVarP(&resolve, "resolve", "R", false, "Resolve the conversation after sending")
	cmd.Flags().StringSliceVar(&labels, "label", nil, "Add labels after sending (repeatable)")
	flagAlias(cmd.Flags(), "label", "lb")
	cmd.Flags().StringVar(&priority, "priority", "", "Set priority after sending (urgent|high|medium|low|none)")
	flagAlias(cmd.Flags(), "priority", "pri")
	cmd.Flags().StringVar(&snoozeFor, "snooze-for", "", "Snooze after sending (e.g., 2h, 30m)")
	flagAlias(cmd.Flags(), "snooze-for", "for")

	return cmd
}
