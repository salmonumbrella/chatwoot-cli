package cmd

import (
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newCtxCmd() *cobra.Command {
	var embedImages bool
	var light bool

	cmd := &cobra.Command{
		Use:     "ctx <conversation-id|url>",
		Aliases: []string{"context", "ct"},
		Short:   "Get full conversation context for AI",
		Long: `Convenience shortcut for 'cw conversations context'.

Accepts a conversation ID or a pasted Chatwoot URL.`,
		Example: strings.TrimSpace(`
  # Context by conversation ID
  cw ctx 123 --output agent

  # Context by URL from browser
  cw ctx https://app.chatwoot.com/app/accounts/1/conversations/123 --output agent

  # Embed images for vision models
  cw ctx 123 --embed-images --output json

  # Lightweight context (minimal JSON for triage)
  cw ctx 123 --li --cj
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			requestEmbeddedImages := embedImages && !light
			ctx, err := client.Context().GetConversation(cmdContext(cmd), id, requestEmbeddedImages)
			if err != nil {
				return fmt.Errorf("failed to get conversation context: %w", err)
			}

			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, buildLightConversationContext(id, ctx))
			}

			if isAgent(cmd) {
				var detail any
				if ctx.Conversation != nil {
					convDetail := agentfmt.ConversationDetailFromConversation(*ctx.Conversation)
					convDetail = resolveConversationDetail(cmdContext(cmd), client, convDetail)
					detail = convDetail
				}

				var contactDetail any
				if ctx.Contact != nil {
					contactDetail = agentfmt.ContactDetailFromContact(*ctx.Contact)
				}

				var contactLabels []string
				var contactInboxes []contextInboxSummary
				if ctx.Contact != nil && ctx.Contact.ID > 0 {
					labels, err := client.Contacts().Labels(cmdContext(cmd), ctx.Contact.ID)
					if err == nil {
						contactLabels = labels
					}
					inboxes, err := client.Contacts().ContactableInboxes(cmdContext(cmd), ctx.Contact.ID)
					if err == nil {
						contactInboxes = contextInboxSummaries(inboxes)
					}
				}

				messages, publicCount, privateCount, embeddedCount := contextMessageSummaries(ctx.Messages)
				meta := map[string]any{
					"conversation_id":  id,
					"message_count":    len(ctx.Messages),
					"public_messages":  publicCount,
					"private_messages": privateCount,
					"embed_images":     embedImages,
				}
				if embeddedCount > 0 {
					meta["embedded_attachments"] = embeddedCount
				}

				item := map[string]any{
					"conversation": detail,
					"contact":      contactDetail,
					"messages":     messages,
					"summary":      ctx.Summary,
					"meta":         meta,
				}
				if len(contactLabels) > 0 {
					item["contact_labels"] = contactLabels
				}
				if len(contactInboxes) > 0 {
					item["contact_inboxes"] = contactInboxes
				}

				payload := agentfmt.ItemEnvelope{
					Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Item: item,
				}
				return printJSON(cmd, payload)
			}
			if isJSON(cmd) {
				return printJSON(cmd, ctx)
			}

			// Keep output minimal; humans can use `conversations context` for the richer text view.
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversation #%d: %s\n", id, strings.TrimSpace(ctx.Summary))
			return nil
		}),
	}

	cmd.Flags().BoolVar(&embedImages, "embed-images", false, "Embed images as base64 data URIs for AI vision")
	cmd.Flags().BoolVar(&light, "light", false, "Return minimal context payload (id, st, inbox, contact, msgs)")
	flagAlias(cmd.Flags(), "embed-images", "embed")
	flagAlias(cmd.Flags(), "embed-images", "em")
	flagAlias(cmd.Flags(), "light", "li")

	return cmd
}
