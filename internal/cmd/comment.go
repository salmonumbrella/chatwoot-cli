package cmd

import (
	"fmt"
	"strings"

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
		content string
		resolve bool
	)

	cmd := &cobra.Command{
		Use:     "comment <conversation-id|url> [text...]",
		Aliases: []string{"cmt"},
		Short:   "Send a public reply to a conversation",
		Long: `Send a public (customer-visible) message to a conversation.

This is a convenience shortcut for:
  chatwoot messages create <conversation-id> --content "..."`,
		Example: strings.TrimSpace(`
  # Comment by conversation ID
  chatwoot comment 123 "Hello! How can I help?"

  # Comment by URL from browser
  chatwoot comment https://app.chatwoot.com/app/accounts/1/conversations/123 "On it."

  # Resolve after commenting
  chatwoot comment 123 "Done" --resolve

  # Use --content instead of positional text
  chatwoot comment 123 --content "Hello!"

  # Agent-friendly envelope
  chatwoot comment 123 "Hello" --output agent
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

	cmd.Flags().StringVar(&content, "content", "", "Message content (alternative to positional text)")
	cmd.Flags().BoolVar(&resolve, "resolve", false, "Resolve the conversation after sending")

	return cmd
}
