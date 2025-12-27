package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
				return printJSON(messages)
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
	)

	cmd := &cobra.Command{
		Use:   "create <conversation-id>",
		Short: "Create a message in a conversation",
		Example: strings.TrimSpace(`
  # Send a text message
  chatwoot messages create 123 --content "Hello!"

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

			// Validate message content length if provided
			if content != "" {
				if err := validation.ValidateMessageContent(content); err != nil {
					return err
				}
			}

			client, err := getClient()
			if err != nil {
				return err
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
				return printJSON(message)
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
				return printJSON(map[string]interface{}{
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
				return printJSON(message)
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
