package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

// newMessagesCmd creates the messages command
func newMessagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "messages",
		Short: "Manage conversation messages",
		Long:  "List, create, and delete messages in conversations",
	}

	cmd.AddCommand(newMessagesListCmd())
	cmd.AddCommand(newMessagesCreateCmd())
	cmd.AddCommand(newMessagesDeleteCmd())
	cmd.AddCommand(newMessagesUpdateCmd())
	cmd.AddCommand(newMessagesBatchSendCmd())

	return cmd
}

// newMessagesListCmd creates the list subcommand
func newMessagesListCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "list <conversation-id>",
		Short: "List messages in a conversation",
		Example: `  # List recent messages
  chatwoot messages list 123

  # List all messages (paginated)
  chatwoot messages list 123 --all

  # JSON output - returns array directly
  chatwoot messages list 123 --all --output json | jq '[.[] | select(.private)]'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			conversationID, err := validation.ParsePositiveInt(args[0], "conversation ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			var messages []api.Message
			if all {
				messages, err = client.ListAllMessages(cmdContext(cmd), conversationID)
			} else {
				messages, err = client.ListMessages(cmdContext(cmd), conversationID)
			}
			if err != nil {
				return fmt.Errorf("failed to list messages for conversation %d: %w", conversationID, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, messages)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tTYPE\tPRIVATE\tCONTENT\tCREATED_AT")
			for _, msg := range messages {
				content := msg.Content
				if len(content) > 50 {
					content = content[:47] + "..."
				}
				// Replace newlines with spaces for display
				content = strings.ReplaceAll(content, "\n", " ")
				content = strings.ReplaceAll(content, "\r", " ")

				_, _ = fmt.Fprintf(w, "%d\t%s\t%t\t%s\t%s\n",
					msg.ID,
					msg.MessageTypeName(),
					msg.Private,
					content,
					msg.CreatedAtTime().Format("2006-01-02 15:04:05"),
				)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Fetch all messages (paginated)")

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
		Use:   "create <conversation-id>",
		Short: "Create a message in a conversation",
		Example: strings.TrimSpace(`
  # Send a text message
  chatwoot messages create 123 --content "Hello!"

  # Send a private note (internal, not visible to customer)
  chatwoot messages create 123 --private --content "Internal note for team"

  # Tag/mention an agent in a private note (agent will be notified)
  chatwoot messages create 123 --private --mention lily --content "Can you follow up on this?"

  # Mention multiple agents
  chatwoot messages create 123 --private --mention lily --mention jack --content "Please review"

  # Send a message with attachment
  chatwoot messages create 123 --content "See attached" --attachment /path/to/file.pdf

  # Send multiple attachments
  chatwoot messages create 123 --content "Documents" --attachment doc1.pdf --attachment doc2.png

  # Send attachment only (no text)
  chatwoot messages create 123 --attachment screenshot.png
`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			conversationID, err := validation.ParsePositiveInt(args[0], "conversation ID")
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
					agent, err := client.FindAgentByNameOrEmail(ctx, m)
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

				message, err = client.CreateMessageWithAttachments(
					cmdContext(cmd),
					conversationID,
					content,
					private,
					messageType,
					fileData,
				)
			} else {
				message, err = client.CreateMessage(cmdContext(cmd), conversationID, content, private, messageType)
			}

			if err != nil {
				return fmt.Errorf("failed to create message in conversation %d: %w", conversationID, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, message)
			}

			fmt.Printf("Message created successfully (ID: %d)\n", message.ID)
			fmt.Printf("Type: %s\n", message.MessageTypeName())
			fmt.Printf("Private: %t\n", message.Private)
			if message.Content != "" {
				fmt.Printf("Content: %s\n", message.Content)
			}
			if len(message.Attachments) > 0 {
				fmt.Printf("Attachments: %d file(s)\n", len(message.Attachments))
				for _, att := range message.Attachments {
					fmt.Printf("  - %s (%s)\n", att.DataURL, att.FileType)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&content, "content", "", "Message content")
	cmd.Flags().BoolVar(&private, "private", false, "Mark message as private (internal note)")
	cmd.Flags().StringVar(&messageType, "type", "outgoing", "Message type: outgoing|incoming")
	cmd.Flags().StringArrayVar(&attachments, "attachment", nil, "File path to attach (can be repeated)")
	cmd.Flags().StringArrayVar(&mentions, "mention", nil, "Agent to mention/tag (name or email, can be repeated). Requires --private")

	return cmd
}

// newMessagesDeleteCmd creates the delete subcommand
func newMessagesDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <conversation-id> <message-id>",
		Short: "Delete a message from a conversation",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			conversationID, err := validation.ParsePositiveInt(args[0], "conversation ID")
			if err != nil {
				return err
			}

			messageID, err := validation.ParsePositiveInt(args[1], "message ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.DeleteMessage(cmdContext(cmd), conversationID, messageID); err != nil {
				return fmt.Errorf("failed to delete message %d from conversation %d: %w", messageID, conversationID, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]interface{}{
					"message_id":      messageID,
					"conversation_id": conversationID,
					"deleted":         true,
				})
			}

			fmt.Printf("Message %d deleted successfully\n", messageID)

			return nil
		},
	}

	return cmd
}

// newMessagesUpdateCmd creates the update subcommand
func newMessagesUpdateCmd() *cobra.Command {
	var content string

	cmd := &cobra.Command{
		Use:   "update <conversation-id> <message-id>",
		Short: "Update a message's content",
		Example: `  # Update a message
  chatwoot messages update 123 456 --content "Updated text"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			conversationID, err := validation.ParsePositiveInt(args[0], "conversation ID")
			if err != nil {
				return err
			}

			messageID, err := validation.ParsePositiveInt(args[1], "message ID")
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

			message, err := client.UpdateMessage(cmdContext(cmd), conversationID, messageID, content)
			if err != nil {
				return fmt.Errorf("failed to update message %d: %w", messageID, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, message)
			}

			fmt.Printf("Message %d updated successfully\n", message.ID)
			fmt.Printf("Content: %s\n", message.Content)

			return nil
		},
	}

	cmd.Flags().StringVar(&content, "content", "", "New message content (required)")
	_ = cmd.MarkFlagRequired("content")

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
		Use:   "batch-send",
		Short: "Send messages to multiple conversations",
		Long: `Send messages to multiple conversations in parallel.

Reads JSON input from stdin with an array of messages to send.
Messages are sent concurrently for efficiency.`,
		Example: strings.TrimSpace(`
  # Send messages to multiple conversations
  echo '[
    {"conversation_id": 123, "content": "Thanks for your patience!"},
    {"conversation_id": 456, "content": "Your order has shipped."}
  ]' | chatwoot messages batch-send

  # Send from a file
  cat messages.json | chatwoot messages batch-send

  # With custom concurrency
  cat messages.json | chatwoot messages batch-send --concurrency 10

  # Send private notes
  echo '[{"conversation_id": 123, "content": "Internal note", "private": true}]' | chatwoot messages batch-send
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Read input from stdin
			var items []BatchSendItem
			decoder := json.NewDecoder(os.Stdin)
			if err := decoder.Decode(&items); err != nil {
				return fmt.Errorf("failed to parse JSON input: %w", err)
			}

			if len(items) == 0 {
				return fmt.Errorf("no messages to send")
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

					msg, err := client.CreateMessage(ctx, item.ConversationID, item.Content, item.Private, "outgoing")
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
			fmt.Printf("Batch send complete: %d sent, %d failed (total: %d)\n", succeeded, failed, len(items))
			if failed > 0 {
				fmt.Println("\nFailed messages:")
				for _, r := range results {
					if r.Status == "error" {
						fmt.Printf("  Conversation %d: %s\n", r.ConversationID, r.Error)
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&concurrency, "concurrency", 5, "Maximum concurrent requests")

	return cmd
}
