package cmd

import (
	"fmt"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/spf13/cobra"
)

func newPublicCmd() *cobra.Command {
	var baseURL string

	cmd := &cobra.Command{
		Use:     "public",
		Aliases: []string{"pub"},
		Short:   "Public API commands (unauthenticated)",
		Long: `Access Chatwoot's public API for widget/client-side operations.

The public API does not require authentication, but requires --base-url to be set.
It uses inbox identifiers instead of account IDs.`,
		Example: `  # Get inbox info
  cw public inboxes get abc123 --base-url https://chatwoot.example.com

  # Create a contact
  cw public contacts create abc123 --name "John Doe" --email "john@example.com" --base-url https://chatwoot.example.com

  # List conversations for a contact
  cw public conversations list abc123 contact456 --base-url https://chatwoot.example.com`,
	}

	cmd.PersistentFlags().StringVar(&baseURL, "base-url", "", "Chatwoot instance base URL (required, or set CHATWOOT_BASE_URL)")

	// Add subcommands
	cmd.AddCommand(newPublicInboxesCmd(&baseURL))
	cmd.AddCommand(newPublicContactsCmd(&baseURL))
	cmd.AddCommand(newPublicConversationsCmd(&baseURL))
	cmd.AddCommand(newPublicMessagesCmd(&baseURL))

	return cmd
}

// newPublicInboxesCmd creates the public inboxes subcommand
func newPublicInboxesCmd(baseURL *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inboxes",
		Short: "Public inbox operations",
	}

	cmd.AddCommand(newPublicInboxesGetCmd(baseURL))

	return cmd
}

func newPublicInboxesGetCmd(baseURL *string) *cobra.Command {
	return &cobra.Command{
		Use:     "get <inbox-identifier>",
		Aliases: []string{"g"},
		Short:   "Get inbox info",
		Long:    "Get information about an inbox via the public API",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			inbox, err := client.Public().GetInbox(cmdContext(cmd), args[0])
			if err != nil {
				return fmt.Errorf("failed to get inbox: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, inbox)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "NAME\tWORKING_HOURS\tTIMEZONE\tCSAT_ENABLED")
			_, _ = fmt.Fprintf(w, "%s\t%v\t%s\t%v\n",
				inbox.Name,
				inbox.WorkingHoursEnabled,
				inbox.Timezone,
				inbox.CsatSurveyEnabled,
			)

			return nil
		}),
	}
}

// newPublicContactsCmd creates the public contacts subcommand
func newPublicContactsCmd(baseURL *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contacts",
		Short: "Public contact operations",
	}

	cmd.AddCommand(newPublicContactsCreateCmd(baseURL))
	cmd.AddCommand(newPublicContactsGetCmd(baseURL))
	cmd.AddCommand(newPublicContactsUpdateCmd(baseURL))

	return cmd
}

func newPublicContactsCreateCmd(baseURL *string) *cobra.Command {
	var (
		name       string
		email      string
		phone      string
		identifier string
	)

	cmd := &cobra.Command{
		Use:     "create <inbox-identifier>",
		Aliases: []string{"mk"},
		Short:   "Create a contact",
		Long:    "Create a contact via the public API",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			req := api.PublicCreateContactRequest{
				Name:        name,
				Email:       email,
				PhoneNumber: phone,
				Identifier:  identifier,
			}

			contact, err := client.Public().CreateContact(cmdContext(cmd), args[0], req)
			if err != nil {
				return fmt.Errorf("failed to create contact: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, contact)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tSOURCE_ID\tNAME\tEMAIL")
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
				contact.ID,
				contact.SourceID,
				contact.Name,
				contact.Email,
			)

			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Contact name")
	cmd.Flags().StringVar(&email, "email", "", "Contact email")
	cmd.Flags().StringVar(&phone, "phone", "", "Contact phone number")
	cmd.Flags().StringVar(&identifier, "identifier", "", "Contact identifier")
	flagAlias(cmd.Flags(), "name", "nm")
	flagAlias(cmd.Flags(), "email", "em")
	flagAlias(cmd.Flags(), "phone", "ph")
	flagAlias(cmd.Flags(), "identifier", "idn")

	return cmd
}

func newPublicContactsGetCmd(baseURL *string) *cobra.Command {
	return &cobra.Command{
		Use:     "get <inbox-identifier> <contact-identifier>",
		Aliases: []string{"g"},
		Short:   "Get a contact",
		Long:    "Get contact information via the public API",
		Args:    cobra.ExactArgs(2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			contact, err := client.Public().GetContact(cmdContext(cmd), args[0], args[1])
			if err != nil {
				return fmt.Errorf("failed to get contact: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, contact)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tSOURCE_ID\tNAME\tEMAIL")
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
				contact.ID,
				contact.SourceID,
				contact.Name,
				contact.Email,
			)

			return nil
		}),
	}
}

func newPublicContactsUpdateCmd(baseURL *string) *cobra.Command {
	var (
		name  string
		email string
		phone string
	)

	cmd := &cobra.Command{
		Use:     "update <inbox-identifier> <contact-identifier>",
		Aliases: []string{"up"},
		Short:   "Update a contact",
		Long:    "Update contact information via the public API",
		Args:    cobra.ExactArgs(2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if name == "" && email == "" && phone == "" {
				return fmt.Errorf("at least one of --name, --email, or --phone must be provided")
			}

			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			req := api.PublicUpdateContactRequest{
				Name:        name,
				Email:       email,
				PhoneNumber: phone,
			}

			contact, err := client.Public().UpdateContact(cmdContext(cmd), args[0], args[1], req)
			if err != nil {
				return fmt.Errorf("failed to update contact: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, contact)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tSOURCE_ID\tNAME\tEMAIL")
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
				contact.ID,
				contact.SourceID,
				contact.Name,
				contact.Email,
			)

			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "New contact name")
	cmd.Flags().StringVar(&email, "email", "", "New contact email")
	cmd.Flags().StringVar(&phone, "phone", "", "New contact phone number")
	flagAlias(cmd.Flags(), "name", "nm")
	flagAlias(cmd.Flags(), "email", "em")
	flagAlias(cmd.Flags(), "phone", "ph")

	return cmd
}

// newPublicConversationsCmd creates the public conversations subcommand
func newPublicConversationsCmd(baseURL *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "conversations",
		Short: "Public conversation operations",
	}

	cmd.AddCommand(newPublicConversationsListCmd(baseURL))
	cmd.AddCommand(newPublicConversationsGetCmd(baseURL))
	cmd.AddCommand(newPublicConversationsCreateCmd(baseURL))
	cmd.AddCommand(newPublicConversationsResolveCmd(baseURL))

	return cmd
}

func newPublicConversationsListCmd(baseURL *string) *cobra.Command {
	return &cobra.Command{
		Use:     "list <inbox-identifier> <contact-identifier>",
		Aliases: []string{"ls"},
		Short:   "List conversations",
		Long:    "List conversations for a contact via the public API",
		Args:    cobra.ExactArgs(2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			conversations, err := client.Public().ListConversations(cmdContext(cmd), args[0], args[1])
			if err != nil {
				return fmt.Errorf("failed to list conversations: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, conversations)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tSTATUS")
			for _, conv := range conversations {
				id := ""
				status := ""
				if v, ok := conv["id"]; ok {
					id = fmt.Sprintf("%v", v)
				}
				if v, ok := conv["status"]; ok {
					status = fmt.Sprintf("%v", v)
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\n", id, status)
			}

			return nil
		}),
	}
}

func newPublicConversationsGetCmd(baseURL *string) *cobra.Command {
	return &cobra.Command{
		Use:     "get <inbox-identifier> <contact-identifier> <conversation-id>",
		Aliases: []string{"g"},
		Short:   "Get a conversation",
		Long:    "Get conversation details via the public API",
		Args:    cobra.ExactArgs(3),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			conversationID, err := parsePositiveIntArg(args[2], "conversation ID")
			if err != nil {
				return err
			}

			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			conversation, err := client.Public().GetConversation(cmdContext(cmd), args[0], args[1], conversationID)
			if err != nil {
				return fmt.Errorf("failed to get conversation: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, conversation)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tSTATUS")
			id := ""
			status := ""
			if v, ok := conversation["id"]; ok {
				id = fmt.Sprintf("%v", v)
			}
			if v, ok := conversation["status"]; ok {
				status = fmt.Sprintf("%v", v)
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\n", id, status)

			return nil
		}),
	}
}

func newPublicConversationsCreateCmd(baseURL *string) *cobra.Command {
	return &cobra.Command{
		Use:     "create <inbox-identifier> <contact-identifier>",
		Aliases: []string{"mk"},
		Short:   "Create a conversation",
		Long:    "Create a new conversation via the public API",
		Args:    cobra.ExactArgs(2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			conversation, err := client.Public().CreateConversation(cmdContext(cmd), args[0], args[1], nil)
			if err != nil {
				return fmt.Errorf("failed to create conversation: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, conversation)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tSTATUS")
			id := ""
			status := ""
			if v, ok := conversation["id"]; ok {
				id = fmt.Sprintf("%v", v)
			}
			if v, ok := conversation["status"]; ok {
				status = fmt.Sprintf("%v", v)
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\n", id, status)

			return nil
		}),
	}
}

func newPublicConversationsResolveCmd(baseURL *string) *cobra.Command {
	return &cobra.Command{
		Use:   "resolve <inbox-identifier> <contact-identifier> <conversation-id>",
		Short: "Resolve a conversation",
		Long:  "Resolve/toggle status of a conversation via the public API",
		Args:  cobra.ExactArgs(3),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			conversationID, err := parsePositiveIntArg(args[2], "conversation ID")
			if err != nil {
				return err
			}

			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			result, err := client.Public().ResolveConversation(cmdContext(cmd), args[0], args[1], conversationID)
			if err != nil {
				return fmt.Errorf("failed to resolve conversation: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, result)
			}

			status := ""
			if v, ok := result["status"]; ok {
				status = fmt.Sprintf("%v", v)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversation %d status: %s\n", conversationID, status)

			return nil
		}),
	}
}

// newPublicMessagesCmd creates the public messages subcommand
func newPublicMessagesCmd(baseURL *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "messages",
		Short: "Public message operations",
	}

	cmd.AddCommand(newPublicMessagesListCmd(baseURL))
	cmd.AddCommand(newPublicMessagesCreateCmd(baseURL))
	cmd.AddCommand(newPublicMessagesUpdateCmd(baseURL))

	return cmd
}

func newPublicMessagesListCmd(baseURL *string) *cobra.Command {
	return &cobra.Command{
		Use:     "list <inbox-identifier> <contact-identifier> <conversation-id>",
		Aliases: []string{"ls"},
		Short:   "List messages",
		Long:    "List messages in a conversation via the public API",
		Args:    cobra.ExactArgs(3),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			conversationID, err := parsePositiveIntArg(args[2], "conversation ID")
			if err != nil {
				return err
			}

			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			messages, err := client.Public().ListMessages(cmdContext(cmd), args[0], args[1], conversationID)
			if err != nil {
				return fmt.Errorf("failed to list messages: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, messages)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tTYPE\tCONTENT")
			for _, msg := range messages {
				id := ""
				msgType := ""
				content := ""
				if v, ok := msg["id"]; ok {
					id = fmt.Sprintf("%v", v)
				}
				if v, ok := msg["message_type"]; ok {
					msgType = fmt.Sprintf("%v", v)
				}
				if v, ok := msg["content"]; ok {
					content = fmt.Sprintf("%v", v)
					if len(content) > 50 {
						content = content[:47] + "..."
					}
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", id, msgType, content)
			}

			return nil
		}),
	}
}

func newPublicMessagesCreateCmd(baseURL *string) *cobra.Command {
	var (
		content string
		echoID  string
	)

	cmd := &cobra.Command{
		Use:     "create <inbox-identifier> <contact-identifier> <conversation-id>",
		Aliases: []string{"mk"},
		Short:   "Create a message",
		Long:    "Create a new message in a conversation via the public API",
		Args:    cobra.ExactArgs(3),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if content == "" {
				return fmt.Errorf("--content is required")
			}

			conversationID, err := parsePositiveIntArg(args[2], "conversation ID")
			if err != nil {
				return err
			}

			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			message, err := client.Public().CreateMessage(cmdContext(cmd), args[0], args[1], conversationID, content, echoID)
			if err != nil {
				return fmt.Errorf("failed to create message: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, message)
			}

			id := ""
			if v, ok := message["id"]; ok {
				id = fmt.Sprintf("%v", v)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Message %s created\n", id)

			return nil
		}),
	}

	cmd.Flags().StringVar(&content, "content", "", "Message content (required)")
	cmd.Flags().StringVar(&echoID, "echo-id", "", "Echo ID for message deduplication")
	flagAlias(cmd.Flags(), "content", "ct")
	flagAlias(cmd.Flags(), "echo-id", "eid")

	return cmd
}

func newPublicMessagesUpdateCmd(baseURL *string) *cobra.Command {
	var content string

	cmd := &cobra.Command{
		Use:     "update <inbox-identifier> <contact-identifier> <conversation-id> <message-id>",
		Aliases: []string{"up"},
		Short:   "Update a message",
		Long:    "Update a message in a conversation via the public API",
		Args:    cobra.ExactArgs(4),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if content == "" {
				return fmt.Errorf("--content is required")
			}

			conversationID, err := parsePositiveIntArg(args[2], "conversation ID")
			if err != nil {
				return err
			}

			messageID, err := parsePositiveIntArg(args[3], "message ID")
			if err != nil {
				return err
			}

			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			message, err := client.Public().UpdateMessage(cmdContext(cmd), args[0], args[1], conversationID, messageID, content)
			if err != nil {
				return fmt.Errorf("failed to update message: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, message)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Message %d updated\n", messageID)

			return nil
		}),
	}

	cmd.Flags().StringVar(&content, "content", "", "New message content (required)")
	flagAlias(cmd.Flags(), "content", "ct")

	return cmd
}
