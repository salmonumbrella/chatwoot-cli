package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/dryrun"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

// newMessagesCmd creates the messages command
func newMessagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "messages",
		Aliases: []string{"message", "msg", "m"},
		Short:   "Manage conversation messages",
		Long:    "List, create, and delete messages in conversations",
	}

	cmd.AddCommand(newMessagesListCmd())
	cmd.AddCommand(newMessagesCreateCmd())
	cmd.AddCommand(newMessagesDeleteCmd())
	cmd.AddCommand(newMessagesUpdateCmd())
	cmd.AddCommand(newMessagesTranslateCmd())
	cmd.AddCommand(newMessagesRetryCmd())
	cmd.AddCommand(newMessagesBatchSendCmd())

	return cmd
}

// newMessagesListCmd creates the list subcommand
func newMessagesListCmd() *cobra.Command {
	var all bool
	var maxPages int
	var limit int
	var transcript bool
	var sinceLastAgent bool
	var keyword string
	var light bool

	cmd := &cobra.Command{
		Use:     "list <conversation-id>",
		Aliases: []string{"ls"},
		Short:   "List messages in a conversation",
		Long: `List messages in a conversation.

Messages are returned in chronological order: oldest first, most recent at the
end of the array. To get the last N messages, use jq '.items[-N:]'.`,
		Example: `  # List recent messages
  cw messages list 123

  # List all messages (paginated)
  cw messages list 123 --all

  # Limit messages (paginates as needed)
  cw messages list 123 --limit 500

  # JSON output - returns an object with an "items" array
  cw messages list 123 --all --output json | jq '[.items[] | select(.private)]'

  # Get last 6 messages (most recent) - messages are oldest-first in the array
  cw messages list 123 --json | jq '.items[-6:]'

  # Filter messages by keyword (case-insensitive)
  cw messages list 123 --keyword refund

  # Use conversation URL from browser
  cw messages list https://app.chatwoot.com/app/accounts/1/conversations/123`,
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			conversationID, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			if cmd.Flags().Changed("limit") && limit < 1 {
				return fmt.Errorf("--limit must be at least 1")
			}
			keyword = strings.TrimSpace(keyword)
			if cmd.Flags().Changed("keyword") && keyword == "" {
				return fmt.Errorf("--keyword must be non-empty")
			}
			if light && transcript {
				return fmt.Errorf("--light cannot be combined with --transcript")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			var messages []api.Message
			if limit > 0 {
				messages, err = client.Messages().ListWithLimit(cmdContext(cmd), conversationID, limit, maxPages)
			} else if all {
				messages, err = client.Messages().ListAllWithMaxPages(cmdContext(cmd), conversationID, maxPages)
			} else {
				messages, err = client.Messages().List(cmdContext(cmd), conversationID)
			}
			if err != nil {
				return fmt.Errorf("failed to list messages for conversation %d: %w", conversationID, err)
			}

			// Filter to only messages since the last agent reply
			if sinceLastAgent {
				lastAgentIdx := -1
				for i := len(messages) - 1; i >= 0; i-- {
					if messages[i].MessageType == api.MessageTypeOutgoing {
						lastAgentIdx = i
						break
					}
				}

				if lastAgentIdx >= 0 && lastAgentIdx < len(messages)-1 {
					// Keep only messages after the last agent message
					messages = messages[lastAgentIdx+1:]
				} else if lastAgentIdx == len(messages)-1 {
					// Agent message is the last one, no new customer messages
					messages = nil
				}
				// If no agent message found (lastAgentIdx == -1), keep all messages
			}
			if keyword != "" {
				needle := strings.ToLower(keyword)
				filtered := make([]api.Message, 0, len(messages))
				for _, msg := range messages {
					if strings.Contains(strings.ToLower(msg.Content), needle) {
						filtered = append(filtered, msg)
					}
				}
				messages = filtered
			}

			totalMessages := len(messages)
			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				lookups := buildLightMessageLookups(messages)
				// Preserve historic .items query/template behavior when users
				// provide --jq/--query/--template, but keep the default payload
				// flattened for token-efficient automation.
				if outfmt.GetQuery(cmd.Context()) != "" || outfmt.GetTemplate(cmd.Context()) != "" {
					return printRawJSON(cmd, map[string]any{"items": lookups})
				}
				// Keep light messages output as a raw array for token-efficient
				// agent workflows (no implicit {"items": ...} envelope).
				raw, err := json.Marshal(lookups)
				if err != nil {
					return err
				}
				return printRawJSON(cmd, json.RawMessage(raw))
			}

			if transcript {
				conv, err := client.Conversations().Get(cmdContext(cmd), conversationID)
				if err != nil {
					slog.Debug("Failed to fetch conversation metadata for transcript", "error", err)
				}
				formatTranscript(cmd.OutOrStdout(), messages, conv)
				return nil
			}

			if isAgent(cmd) {
				var conversationDetail *agentfmt.ConversationDetail
				if flags.ResolveNames {
					conv, err := client.Conversations().Get(cmdContext(cmd), conversationID)
					if err != nil {
						return fmt.Errorf("failed to resolve conversation %d: %w", conversationID, err)
					}
					detail := agentfmt.ConversationDetailFromConversation(*conv)
					detail = resolveConversationDetail(cmdContext(cmd), client, detail)
					conversationDetail = &detail
				}

				wrapped := make([]agentfmt.MessageSummaryWithPosition, len(messages))
				for i, msg := range messages {
					summary := agentfmt.MessageSummaryFromMessage(msg)
					wrapped[i] = agentfmt.MessageSummaryWithPosition{
						MessageSummary: summary,
						Position:       i + 1,
						TotalMessages:  totalMessages,
					}
				}
				meta := map[string]any{
					"conversation_id": conversationID,
					"total_messages":  totalMessages,
				}
				if conversationDetail != nil {
					meta["conversation"] = conversationDetail
				}
				payload := agentfmt.ListEnvelope{
					Kind:  agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Items: wrapped,
					Meta:  meta,
				}
				return printJSON(cmd, payload)
			}
			if isJSON(cmd) {
				// Wrap messages with position metadata
				type messageWithPosition struct {
					*api.Message
					Position      int `json:"position"`
					TotalMessages int `json:"total_messages"`
				}
				wrapped := make([]messageWithPosition, len(messages))
				for i, msg := range messages {
					msgCopy := msg
					wrapped[i] = messageWithPosition{
						Message:       &msgCopy,
						Position:      i + 1,
						TotalMessages: totalMessages,
					}
				}
				return printJSON(cmd, wrapped)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "POS\tID\tTYPE\tPRIVATE\tCONTENT\tCREATED_AT")
			for i, msg := range messages {
				content := msg.Content
				if len(content) > 50 {
					content = content[:47] + "..."
				}
				// Replace newlines with spaces for display
				content = strings.ReplaceAll(content, "\n", " ")
				content = strings.ReplaceAll(content, "\r", " ")

				_, _ = fmt.Fprintf(w, "%s\t%d\t%s\t%t\t%s\t%s\n",
					formatPosition(i+1, totalMessages),
					msg.ID,
					msg.MessageTypeName(),
					msg.Private,
					content,
					formatTimestamp(msg.CreatedAtTime()),
				)
			}

			return nil
		}),
	}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "content", "message_type", "created_at"},
		"default": {"id", "content", "message_type", "private", "sender_type", "created_at"},
		"debug":   {"id", "conversation_id", "content", "content_type", "message_type", "private", "sender_id", "sender_type", "attachments", "created_at"},
	})
	registerFieldSchema(cmd, "message")

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Fetch all messages (paginated)")
	cmd.Flags().IntVarP(&maxPages, "max-pages", "M", 100, "Maximum pages to fetch when using --all or --limit")
	flagAlias(cmd.Flags(), "max-pages", "mp")
	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "Maximum messages to return (paginates as needed; 0 means no limit)")
	cmd.Flags().BoolVar(&transcript, "transcript", false, "Output as readable conversation transcript")
	flagAlias(cmd.Flags(), "transcript", "tr")
	cmd.Flags().BoolVar(&sinceLastAgent, "since-last-agent", false, "Only show messages since the last agent reply")
	flagAlias(cmd.Flags(), "since-last-agent", "sla")
	cmd.Flags().StringVar(&keyword, "keyword", "", "Filter messages by keyword in content (case-insensitive)")
	flagAlias(cmd.Flags(), "keyword", "kw")
	cmd.Flags().BoolVar(&light, "light", false, "Return minimal message payload for lookup")
	flagAlias(cmd.Flags(), "light", "li")

	return cmd
}

// newMessagesCreateCmd creates the create subcommand
func newMessagesCreateCmd() *cobra.Command {
	var (
		content     string
		private     bool
		messageType string
		attachments []string
		mentions    []string
	)

	cmd := &cobra.Command{
		Use:     "create <conversation-id>",
		Aliases: []string{"mk"},
		Short:   "Create a message in a conversation",
		Example: strings.TrimSpace(`
  # Send a text message
  cw messages create 123 --content "Hello!"

  # Send a private note (internal, not visible to customer)
  cw messages create 123 --private --content "Internal note for team"

  # Tag/mention an agent in a private note (agent will be notified)
  cw messages create 123 --private --mention lily --content "Can you follow up on this?"

  # Mention multiple agents
  cw messages create 123 --private --mention lily --mention jack --content "Please review"

  # Send a message with attachment
  cw messages create 123 --content "See attached" --attachment /path/to/file.pdf

  # Send multiple attachments
  cw messages create 123 --content "Documents" --attachment doc1.pdf --attachment doc2.png

  # Send attachment only (no text)
  cw messages create 123 --attachment screenshot.png
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			conversationID, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			if content == "" && len(attachments) == 0 {
				return fmt.Errorf("either --content or --attachment is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			// Resolve mentions to user IDs and build mention prefix
			if len(mentions) > 0 {
				if !private {
					return fmt.Errorf("--mention requires --private flag (mentions only work in private notes)")
				}

				var mentionParts []string
				ctx := cmdContext(cmd)
				for _, m := range mentions {
					agent, err := client.Agents().Find(ctx, m)
					if err != nil {
						return fmt.Errorf("failed to resolve mention '%s': %w", m, err)
					}
					// Format: [@DisplayName](mention://user/{id}/{url-encoded-name})
					// URL-encode the name in the URL part to handle spaces correctly
					encodedName := url.PathEscape(agent.Name)
					mentionParts = append(mentionParts, fmt.Sprintf("[@%s](mention://user/%d/%s)", agent.Name, agent.ID, encodedName))
				}
				// Prepend mentions to content
				content = strings.Join(mentionParts, " ") + " " + content
			}

			// Validate message content length if provided
			if content != "" {
				if err := validation.ValidateMessageContent(content); err != nil {
					return err
				}
			}

			var message *api.Message

			if len(attachments) > 0 {
				// Validate attachment count
				if len(attachments) > api.MaxAttachments {
					return fmt.Errorf("too many attachments: maximum %d allowed (got %d)", api.MaxAttachments, len(attachments))
				}

				// Read attachment files
				fileData := make(map[string][]byte)
				for _, path := range attachments {
					// Validate file size before reading
					fileInfo, err := os.Stat(path)
					if err != nil {
						return fmt.Errorf("failed to access attachment %s: %w", path, err)
					}
					if fileInfo.Size() > api.MaxAttachmentSize {
						return fmt.Errorf("attachment %s exceeds %dMB limit (%d bytes)", path, api.MaxAttachmentSize/(1024*1024), fileInfo.Size())
					}

					data, err := os.ReadFile(path)
					if err != nil {
						return fmt.Errorf("failed to read attachment %s: %w", path, err)
					}
					filename := filepath.Base(path)
					if _, exists := fileData[filename]; exists {
						return fmt.Errorf("duplicate filename detected: %s. Please rename one of the files before uploading", filename)
					}
					fileData[filename] = data
				}

				attachmentNames := make([]string, 0, len(fileData))
				for name := range fileData {
					attachmentNames = append(attachmentNames, name)
				}
				if ok, err := maybeDryRun(cmd, &dryrun.Preview{
					Operation: "create",
					Resource:  "message",
					Details: map[string]any{
						"conversation_id": conversationID,
						"content":         content,
						"private":         private,
						"type":            messageType,
						"attachments":     attachmentNames,
					},
				}); ok {
					return err
				}

				message, err = client.Messages().CreateWithAttachments(
					cmdContext(cmd),
					conversationID,
					content,
					private,
					messageType,
					fileData,
				)
			} else {
				if ok, err := maybeDryRun(cmd, &dryrun.Preview{
					Operation: "create",
					Resource:  "message",
					Details: map[string]any{
						"conversation_id": conversationID,
						"content":         content,
						"private":         private,
						"type":            messageType,
					},
				}); ok {
					return err
				}

				message, err = client.Messages().Create(cmdContext(cmd), conversationID, content, private, messageType)
			}

			if err != nil {
				return fmt.Errorf("failed to create message in conversation %d: %w", conversationID, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, message)
			}

			printAction(cmd, "Created", "message", message.ID, "")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Type: %s\n", message.MessageTypeName())
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Private: %t\n", message.Private)
			if message.Content != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Content: %s\n", message.Content)
			}
			if len(message.Attachments) > 0 {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Attachments: %d file(s)\n", len(message.Attachments))
				for _, att := range message.Attachments {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s (%s)\n", att.DataURL, att.FileType)
				}
			}

			return nil
		}),
	}

	cmd.Flags().StringVarP(&content, "content", "c", "", "Message content")
	cmd.Flags().BoolVarP(&private, "private", "P", false, "Mark message as private (internal note)")
	cmd.Flags().StringVar(&messageType, "type", "outgoing", "Message type: outgoing|incoming")
	cmd.Flags().StringArrayVar(&attachments, "attachment", nil, "File path to attach (can be repeated)")
	cmd.Flags().StringArrayVar(&mentions, "mention", nil, "Agent to mention/tag (name or email, can be repeated). Requires --private")
	flagAlias(cmd.Flags(), "type", "ty")
	flagAlias(cmd.Flags(), "attachment", "att")
	flagAlias(cmd.Flags(), "mention", "mt")

	return cmd
}

// newMessagesDeleteCmd creates the delete subcommand
func newMessagesDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <conversation-id> <message-id>",
		Aliases: []string{"rm"},
		Short:   "Delete a message from a conversation",
		Args:    cobra.ExactArgs(2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			conversationID, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			messageID, err := parsePositiveIntArg(args[1], "message ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "delete",
				Resource:  "message",
				Details: map[string]any{
					"conversation_id": conversationID,
					"message_id":      messageID,
				},
			}); ok {
				return err
			}

			if err := client.Messages().Delete(cmdContext(cmd), conversationID, messageID); err != nil {
				return fmt.Errorf("failed to delete message %d from conversation %d: %w", messageID, conversationID, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]interface{}{
					"message_id":      messageID,
					"conversation_id": conversationID,
					"deleted":         true,
				})
			}

			printAction(cmd, "Deleted", "message", messageID, "")

			return nil
		}),
	}

	return cmd
}

// newMessagesUpdateCmd creates the update subcommand
func newMessagesUpdateCmd() *cobra.Command {
	var content string

	cmd := &cobra.Command{
		Use:     "update <conversation-id> <message-id>",
		Aliases: []string{"up"},
		Short:   "Update a message's content",
		Example: `  # Update a message
  cw messages update 123 456 --content "Updated text"`,
		Args: cobra.ExactArgs(2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			conversationID, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			messageID, err := parsePositiveIntArg(args[1], "message ID")
			if err != nil {
				return err
			}

			if content == "" {
				return fmt.Errorf("--content is required")
			}

			// Validate message content length
			if err := validation.ValidateMessageContent(content); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "update",
				Resource:  "message",
				Details: map[string]any{
					"conversation_id": conversationID,
					"message_id":      messageID,
					"content":         content,
				},
			}); ok {
				return err
			}

			message, err := client.Messages().Update(cmdContext(cmd), conversationID, messageID, content)
			if err != nil {
				return fmt.Errorf("failed to update message %d: %w", messageID, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, message)
			}

			printAction(cmd, "Updated", "message", message.ID, "")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Content: %s\n", message.Content)

			return nil
		}),
	}

	cmd.Flags().StringVarP(&content, "content", "c", "", "New message content (required)")
	_ = cmd.MarkFlagRequired("content")

	return cmd
}

// newMessagesTranslateCmd creates the translate subcommand
func newMessagesTranslateCmd() *cobra.Command {
	var lang string

	cmd := &cobra.Command{
		Use:   "translate <conversation-id> <message-id>",
		Short: "Translate a message to another language",
		Long: `Translate a message's content to a specified target language.

Uses the Chatwoot translation service to translate message content.
Requires an AI integration to be configured in your Chatwoot instance.`,
		Example: strings.TrimSpace(`
  # Translate a message to Spanish
  cw messages translate 123 456 --lang es

  # Translate to French
  cw messages translate 123 456 --lang fr

  # Get translation as JSON
  cw messages translate 123 456 --lang de --output json
`),
		Args: cobra.ExactArgs(2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			conversationID, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			messageID, err := parsePositiveIntArg(args[1], "message ID")
			if err != nil {
				return err
			}

			if lang == "" {
				return fmt.Errorf("--lang is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			content, err := client.Messages().Translate(cmdContext(cmd), conversationID, messageID, lang)
			if err != nil {
				return fmt.Errorf("failed to translate message %d: %w", messageID, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{
					"conversation_id": conversationID,
					"message_id":      messageID,
					"target_language": lang,
					"translated_text": content,
				})
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), content)
			return nil
		}),
	}

	cmd.Flags().StringVar(&lang, "lang", "", "Target language code (e.g., es, fr, de, ja)")
	flagAlias(cmd.Flags(), "lang", "lg")

	return cmd
}

// newMessagesRetryCmd creates the retry subcommand
func newMessagesRetryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retry <conversation-id> <message-id>",
		Short: "Retry sending a failed message",
		Long: `Retry sending a message that previously failed to send.

Use this to reattempt delivery of messages that encountered
temporary failures (e.g., network issues, rate limiting).`,
		Example: strings.TrimSpace(`
  # Retry a failed message
  cw messages retry 123 456

  # Retry and get result as JSON
  cw messages retry 123 456 --output json
`),
		Args: cobra.ExactArgs(2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			conversationID, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			messageID, err := parsePositiveIntArg(args[1], "message ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			message, err := client.Messages().Retry(cmdContext(cmd), conversationID, messageID)
			if err != nil {
				return fmt.Errorf("failed to retry message %d: %w", messageID, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, message)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Message %d retry successful\n", message.ID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Content: %s\n", message.Content)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Type: %s\n", message.MessageTypeName())
			return nil
		}),
	}

	return cmd
}

// BatchSendItem represents a single message to send in a batch operation
type BatchSendItem struct {
	ConversationID int    `json:"conversation_id"`
	Content        string `json:"content"`
	Private        bool   `json:"private,omitempty"`
}

// BatchSendResult represents the result of a single batch send operation
type BatchSendResult struct {
	ConversationID int    `json:"conversation_id"`
	MessageID      int    `json:"message_id,omitempty"`
	Status         string `json:"status"` // "sent" | "error"
	Error          string `json:"error,omitempty"`
}

// BatchSendResponse is the response for the batch-send command
type BatchSendResponse struct {
	Total     int               `json:"total"`
	Succeeded int               `json:"succeeded"`
	Failed    int               `json:"failed"`
	Results   []BatchSendResult `json:"results"`
}

// newMessagesBatchSendCmd creates the batch-send subcommand
func newMessagesBatchSendCmd() *cobra.Command {
	var concurrency int

	cmd := &cobra.Command{
		Use:     "batch-send",
		Aliases: []string{"bs"},
		Short:   "Send messages to multiple conversations",
		Long: `Send messages to multiple conversations in parallel.

Reads JSON input from stdin with an array of messages to send.
Messages are sent concurrently for efficiency.`,
		Example: strings.TrimSpace(`
  # Send messages to multiple conversations
  echo '[
    {"conversation_id": 123, "content": "Thanks for your patience!"},
    {"conversation_id": 456, "content": "Your order has shipped."}
  ]' | cw messages batch-send

  # Send from a file
  cat messages.json | cw messages batch-send

  # With custom concurrency
  cat messages.json | cw messages batch-send --concurrency 10

  # Send private notes
  echo '[{"conversation_id": 123, "content": "Internal note", "private": true}]' | cw messages batch-send
`),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			// Read input from stdin
			var items []BatchSendItem
			decoder := json.NewDecoder(os.Stdin)
			if err := decoder.Decode(&items); err != nil {
				return fmt.Errorf("failed to parse JSON input: %w", err)
			}

			if len(items) == 0 {
				return fmt.Errorf("no messages to send")
			}
			if concurrency <= 0 {
				return fmt.Errorf("--concurrency must be greater than 0")
			}

			// Validate items
			for i, item := range items {
				if item.ConversationID <= 0 {
					return fmt.Errorf("item %d: conversation_id must be positive", i)
				}
				if item.Content == "" {
					return fmt.Errorf("item %d: content is required", i)
				}
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			// Process messages in parallel with bounded concurrency
			results := make([]BatchSendResult, len(items))
			sem := make(chan struct{}, concurrency)
			var wg sync.WaitGroup

			for i, item := range items {
				wg.Add(1)
				go func(idx int, item BatchSendItem) {
					defer wg.Done()
					sem <- struct{}{}        // Acquire semaphore
					defer func() { <-sem }() // Release semaphore

					result := BatchSendResult{
						ConversationID: item.ConversationID,
					}

					msg, err := client.Messages().Create(ctx, item.ConversationID, item.Content, item.Private, "outgoing")
					if err != nil {
						result.Status = "error"
						result.Error = err.Error()
					} else {
						result.Status = "sent"
						result.MessageID = msg.ID
					}

					results[idx] = result
				}(i, item)
			}

			wg.Wait()

			// Count successes and failures
			var succeeded, failed int
			for _, r := range results {
				if r.Status == "sent" {
					succeeded++
				} else {
					failed++
				}
			}

			response := BatchSendResponse{
				Total:     len(items),
				Succeeded: succeeded,
				Failed:    failed,
				Results:   results,
			}

			if isJSON(cmd) {
				return printJSON(cmd, response)
			}

			// Text output
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Batch send complete: %d sent, %d failed (total: %d)\n", succeeded, failed, len(items))
			if failed > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nFailed messages:")
				for _, r := range results {
					if r.Status == "error" {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Conversation %d: %s\n", r.ConversationID, r.Error)
					}
				}
			}

			return nil
		}),
	}

	cmd.Flags().IntVar(&concurrency, "concurrency", 5, "Maximum concurrent requests")
	flagAlias(cmd.Flags(), "concurrency", "cc")

	return cmd
}
