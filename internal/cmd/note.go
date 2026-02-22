package cmd

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/dryrun"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newNoteCmd() *cobra.Command {
	var (
		content   string
		mentions  []string
		resolve   bool
		pending   bool
		labels    []string
		priority  string
		snoozeFor string
		light     bool
	)

	cmd := &cobra.Command{
		Use:     "note <conversation-id|url> [text...]",
		Aliases: []string{"internal-note", "n"},
		Short:   "Add a private note to a conversation",
		Long: `Send a private (internal-only) note to a conversation.

This is a convenience shortcut for:
  cw messages create <conversation-id> --private --content "..."`,
		Example: strings.TrimSpace(`
  # Private note by conversation ID
  cw note 123 "Internal note for the team"

  # Mention/tag agents (they'll be notified)
  cw note 123 --mention lily --mention jack "Please review"

  # Resolve after noting
  cw note 123 "Resolved and documented" --resolve

  # Use --content instead of positional text
  cw note 123 --content "Internal note"

  # Agent-friendly envelope
  cw note 123 "FYI" --output agent
`),
		Args: cobra.MinimumNArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			conversationID, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			positional := strings.TrimSpace(strings.Join(args[1:], " "))
			if cmd.Flags().Changed("content") && positional != "" {
				return fmt.Errorf("provide note text either as args or with --content, not both")
			}
			if !cmd.Flags().Changed("content") {
				content = positional
			}
			if content == "" {
				return fmt.Errorf("note text is required (use --content or provide trailing args)")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			// Resolve mentions to user IDs and build mention prefix.
			if len(mentions) > 0 {
				var mentionParts []string
				for _, m := range mentions {
					agent, err := client.Agents().Find(ctx, m)
					if err != nil {
						return fmt.Errorf("failed to resolve mention %q: %w", m, err)
					}
					encodedName := url.PathEscape(agent.Name)
					mentionParts = append(mentionParts, fmt.Sprintf("[@%s](mention://user/%d/%s)", agent.Name, agent.ID, encodedName))
				}
				content = strings.Join(mentionParts, " ") + " " + content
			}

			if err := validation.ValidateMessageContent(content); err != nil {
				return err
			}

			// Validate side-effect flags before sending so we fail fast.
			if err := validateExclusiveStatus(resolve, pending, snoozeFor); err != nil {
				return err
			}
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
					"private":         true,
					"type":            "outgoing",
					"resolve":         resolve,
					"pending":         pending,
					"mentions":        mentions,
				},
			}); ok {
				return err
			}

			message, err := client.Messages().Create(ctx, conversationID, content, true, "outgoing")
			if err != nil {
				return fmt.Errorf("failed to send private note to conversation %d: %w", conversationID, err)
			}

			resolved := false
			if resolve {
				_, err := client.Conversations().ToggleStatus(ctx, conversationID, "resolved", 0)
				if err != nil {
					return fmt.Errorf("note sent (ID: %d) but failed to resolve conversation: %w", message.ID, err)
				}
				resolved = true
			}

			pendingSet := false
			if pending {
				_, err := client.Conversations().ToggleStatus(ctx, conversationID, "pending", 0)
				if err != nil {
					return fmt.Errorf("note sent (ID: %d) but failed to set conversation to pending: %w", message.ID, err)
				}
				pendingSet = true
			}

			if len(labels) > 0 {
				existing, _ := client.Conversations().Labels(ctx, conversationID)
				merged := dedupeStrings(append(existing, labels...))
				if _, err := client.Conversations().AddLabels(ctx, conversationID, merged); err != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: note sent but failed to add labels: %v\n", err)
				}
			}
			if priority != "" {
				if err := client.Conversations().TogglePriority(ctx, conversationID, priority); err != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: note sent but failed to set priority: %v\n", err)
				}
			}
			if snoozeFor != "" {
				snoozedUntil, err := parseSnoozeFor(snoozeFor, time.Now())
				if err != nil {
					return err
				}
				_, err = client.Conversations().ToggleStatus(ctx, conversationID, "snoozed", snoozedUntil.Unix())
				if err != nil {
					return fmt.Errorf("note sent (ID: %d) but failed to snooze conversation: %w", message.ID, err)
				}
			}

			u, _ := resourceURL("conversations", conversationID)
			result := CommentResult{
				Action:         "noted",
				ConversationID: conversationID,
				MessageID:      message.ID,
				Message:        message,
				Private:        true,
				Resolved:       resolved,
				Pending:        pendingSet,
				URL:            u,
			}
			status := ""
			if resolved {
				status = "resolved"
			} else if pendingSet {
				status = "pending"
			} else if snoozeFor != "" {
				status = "snoozed"
			}

			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, buildLightMessageMutationResult(conversationID, message.ID, status))
			}

			if isAgent(cmd) {
				if !flagOrAliasChanged(cmd, "compact-json") {
					cmd.SetContext(outfmt.WithCompact(cmd.Context(), true))
				}
				item := map[string]any{
					"id":  result.ConversationID,
					"mid": result.MessageID,
					"prv": true,
				}
				if status != "" {
					item["st"] = shortStatus(status)
				}
				return printRawJSON(cmd, item)
			}
			if isJSON(cmd) {
				return printJSON(cmd, result)
			}

			printAction(cmd, "Sent", "note", message.ID, "")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversation: %d\n", conversationID)
			if resolved {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Status: Resolved")
			}
			if pendingSet {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Status: Pending")
			}
			return nil
		}),
	}

	cmd.Flags().StringVarP(&content, "content", "c", "", "Note content (alternative to positional text)")
	cmd.Flags().StringArrayVar(&mentions, "mention", nil, "Agent to mention/tag (name or email, can be repeated)")
	flagAlias(cmd.Flags(), "mention", "mn")
	flagAlias(cmd.Flags(), "mention", "mt")
	cmd.Flags().BoolVarP(&resolve, "resolve", "R", false, "Resolve the conversation after sending")
	cmd.Flags().BoolVarP(&pending, "pending", "p", false, "Set conversation to pending after sending")
	cmd.Flags().StringSliceVar(&labels, "label", nil, "Add labels after sending (repeatable)")
	flagAlias(cmd.Flags(), "label", "lb")
	cmd.Flags().StringVar(&priority, "priority", "", "Set priority after sending (urgent|high|medium|low|none)")
	flagAlias(cmd.Flags(), "priority", "pri")
	cmd.Flags().StringVar(&snoozeFor, "snooze-for", "", "Snooze after sending (e.g., 2h, 30m)")
	flagAlias(cmd.Flags(), "snooze-for", "for")
	cmd.Flags().BoolVar(&light, "light", false, "Return minimal mutation payload")
	flagAlias(cmd.Flags(), "light", "li")

	return cmd
}
