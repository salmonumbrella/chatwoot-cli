package cmd

import (
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newConversationsAttachmentsExtractCmd() *cobra.Command {
	var (
		indexes           []int
		limit             int
		light             bool
		maxBytes          int64
		maxChars          int
		maxTotalBytes     int64
		unsafeNoSizeLimit bool
	)

	cmd := &cobra.Command{
		Use:   "extract <conversation-id>",
		Short: "Extract text from document attachments in a conversation",
		Long: `Download supported document attachments and extract bounded text for agent analysis.

This command is separate from 'ctx' so document extraction stays explicit and token-bounded.`,
		Example: strings.TrimSpace(`
  # Extract up to 3 document attachments with safe defaults
  cw conversations attachments extract 123 -o agent

  # Extract specific attachments by list index from 'conversations attachments'
  cw conversations attachments extract 123 --index 3 --index 4 --compact-json

  # Override per-file and total download caps
  cw conversations attachments extract 123 --max-bytes 15728640 --max-total-bytes 31457280
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			if limit < 0 {
				return fmt.Errorf("--limit must be zero or greater")
			}
			if maxBytes < 0 {
				return fmt.Errorf("--max-bytes must be zero or greater")
			}
			if maxTotalBytes < 0 {
				return fmt.Errorf("--max-total-bytes must be zero or greater")
			}
			if maxChars < 0 {
				return fmt.Errorf("--max-chars must be zero or greater")
			}
			for _, index := range indexes {
				if index < 1 {
					return fmt.Errorf("--index values must be at least 1")
				}
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			result, err := client.ExtractConversationAttachments(cmdContext(cmd), id, api.ConversationAttachmentExtractOptions{
				Indexes:           indexes,
				Limit:             limit,
				MaxBytes:          maxBytes,
				MaxTotalBytes:     maxTotalBytes,
				MaxChars:          maxChars,
				UnsafeNoSizeLimit: unsafeNoSizeLimit,
			})
			if err != nil {
				return fmt.Errorf("failed to extract attachment text for conversation %d: %w", id, err)
			}

			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, buildLightConversationAttachmentExtraction(result))
			}

			if isAgent(cmd) {
				return printJSON(cmd, agentfmt.ItemEnvelope{
					Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Item: map[string]any{
						"conversation_id": result.ConversationID,
						"attachments":     result.Items,
						"meta":            result.Meta,
					},
				})
			}

			if isJSON(cmd) {
				return printJSON(cmd, result)
			}

			if len(result.Items) == 0 {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No document attachments extracted from conversation #%d\n", result.ConversationID)
				return nil
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversation #%d document extraction\n\n", result.ConversationID)
			for _, item := range result.Items {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[%d] %s (%s, %s, %d chars)\n", item.Index, item.Name, item.MIMEType, formatFileSize(int(item.DownloadedBytes)), item.TextChars)
				if item.Truncated {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "  truncated: true")
				}
				if item.Extractor != "" {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  extractor: %s\n", item.Extractor)
				}
				preview := summarizeAttachmentText(item.Text, 240)
				if preview != "" {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  preview: %s\n", preview)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
			}

			return nil
		}),
	}

	cmd.Flags().IntSliceVar(&indexes, "index", nil, "Attachment list indexes to extract (1-based, from 'conversations attachments')")
	cmd.Flags().IntVar(&limit, "limit", api.DefaultDocumentExtractLimit, "Maximum number of document attachments to extract when --index is not provided (0 means all)")
	cmd.Flags().Int64Var(&maxBytes, "max-bytes", api.DefaultDocumentExtractMaxBytes, "Maximum bytes to download per attachment (0 means unlimited)")
	cmd.Flags().Int64Var(&maxTotalBytes, "max-total-bytes", api.DefaultDocumentExtractMaxTotalBytes, "Maximum total bytes to download across attachments (0 means unlimited)")
	cmd.Flags().IntVar(&maxChars, "max-chars", api.DefaultDocumentExtractMaxChars, "Maximum extracted characters per attachment (0 means unlimited)")
	cmd.Flags().BoolVar(&unsafeNoSizeLimit, "unsafe-no-size-limit", false, "Disable per-file and total download limits")
	cmd.Flags().BoolVar(&light, "light", false, "Return minimal extraction payload with compact keys")
	flagAlias(cmd.Flags(), "light", "li")

	return cmd
}

func summarizeAttachmentText(text string, maxChars int) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.Join(strings.Fields(text), " ")
	if maxChars <= 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) <= maxChars {
		return text
	}
	return string(runes[:maxChars]) + "..."
}
