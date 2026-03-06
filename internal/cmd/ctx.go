package cmd

import (
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newCtxCmd() *cobra.Command {
	var embedImages bool
	var excludeAttachments bool
	var light bool
	var publicOnly bool
	var tail int

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
			if cmd.Flags().Changed("tail") && tail < 1 {
				return fmt.Errorf("--tail must be at least 1")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			requestEmbeddedImages := embedImages && !light && !excludeAttachments
			ctx, err := client.Context().GetConversationWithOptions(cmdContext(cmd), id, api.ConversationContextOptions{
				EmbedImages:        requestEmbeddedImages,
				Tail:               tail,
				PublicOnly:         publicOnly,
				ExcludeAttachments: excludeAttachments,
			})
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
					"embed_images":     requestEmbeddedImages,
				}
				if ctx.Meta != nil {
					meta["total_messages"] = ctx.Meta.TotalMessages
					meta["returned_messages"] = ctx.Meta.ReturnedMessages
					if ctx.Meta.Tail > 0 {
						meta["tail"] = ctx.Meta.Tail
					}
					if ctx.Meta.Truncated {
						meta["truncated"] = true
					}
					if ctx.Meta.PublicOnly {
						meta["public_only"] = true
					}
					if ctx.Meta.ExcludeAttachments {
						meta["exclude_attachments"] = true
					}
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
	cmd.Flags().BoolVar(&excludeAttachments, "exclude-attachments", false, "Omit attachment metadata from returned context")
	cmd.Flags().BoolVar(&light, "light", false, "Return minimal context payload (id, st, inbox, contact, msgs)")
	cmd.Flags().BoolVar(&publicOnly, "public-only", false, "Exclude private notes from returned context")
	cmd.Flags().IntVar(&tail, "tail", 0, "Limit returned context to the last N messages after filtering")
	flagAlias(cmd.Flags(), "embed-images", "embed")
	flagAlias(cmd.Flags(), "embed-images", "em")
	flagAlias(cmd.Flags(), "exclude-attachments", "xa")
	flagAlias(cmd.Flags(), "light", "li")
	flagAlias(cmd.Flags(), "public-only", "pub")

	return cmd
}
