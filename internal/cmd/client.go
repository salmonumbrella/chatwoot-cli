package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/spf13/cobra"
)

func newClientCmd() *cobra.Command {
	var baseURL string
	var inboxIdentifier string
	var contactIdentifier string

	cmd := &cobra.Command{
		Use:     "client",
		Aliases: []string{"cl"},
		Short:   "Access Chatwoot public client APIs",
		Long:    "Interact with Chatwoot public client APIs using inbox and contact identifiers.",
	}

	cmd.PersistentFlags().StringVar(&baseURL, "base-url", "", "Override Chatwoot base URL")
	cmd.PersistentFlags().StringVar(&inboxIdentifier, "inbox", "", "Inbox identifier (required for client APIs)")
	cmd.PersistentFlags().StringVar(&contactIdentifier, "contact", "", "Contact identifier (required for contact-scoped operations)")

	cmd.AddCommand(newClientContactsCmd(&baseURL, &inboxIdentifier, &contactIdentifier))
	cmd.AddCommand(newClientConversationsCmd(&baseURL, &inboxIdentifier, &contactIdentifier))
	cmd.AddCommand(newClientMessagesCmd(&baseURL, &inboxIdentifier, &contactIdentifier))
	cmd.AddCommand(newClientTypingCmd(&baseURL, &inboxIdentifier, &contactIdentifier))
	cmd.AddCommand(newClientLastSeenCmd(&baseURL, &inboxIdentifier, &contactIdentifier))

	return cmd
}

func requirePublicIdentifiers(inboxIdentifier, contactIdentifier *string, requireContact bool) error {
	if inboxIdentifier == nil || strings.TrimSpace(*inboxIdentifier) == "" {
		return fmt.Errorf("--inbox is required")
	}
	if requireContact && (contactIdentifier == nil || strings.TrimSpace(*contactIdentifier) == "") {
		return fmt.Errorf("--contact is required")
	}
	return nil
}

func newClientContactsCmd(baseURL, inboxIdentifier, contactIdentifier *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contacts",
		Short: "Manage contacts via public API",
	}

	cmd.AddCommand(newClientContactsCreateCmd(baseURL, inboxIdentifier))
	cmd.AddCommand(newClientContactsGetCmd(baseURL, inboxIdentifier, contactIdentifier))

	return cmd
}

func newClientContactsCreateCmd(baseURL, inboxIdentifier *string) *cobra.Command {
	var (
		name             string
		email            string
		phone            string
		identifier       string
		identifierHash   string
		avatarURL        string
		customAttributes string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"mk"},
		Short:   "Create a contact via public API",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if err := requirePublicIdentifiers(inboxIdentifier, nil, false); err != nil {
				return err
			}

			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			var attrs map[string]any
			if customAttributes != "" {
				if err := json.Unmarshal([]byte(customAttributes), &attrs); err != nil {
					return fmt.Errorf("invalid custom-attributes JSON: %w", err)
				}
			}

			contact, err := client.Public().CreateContact(cmdContext(cmd), *inboxIdentifier, api.PublicCreateContactRequest{
				Identifier:       identifier,
				IdentifierHash:   identifierHash,
				Email:            email,
				Name:             name,
				PhoneNumber:      phone,
				AvatarURL:        avatarURL,
				CustomAttributes: attrs,
			})
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, contact)
			}

			printAction(cmd, "Created", "contact", contact.ID, contact.Name)
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Contact name")
	cmd.Flags().StringVar(&email, "email", "", "Contact email")
	cmd.Flags().StringVar(&phone, "phone", "", "Contact phone number")
	cmd.Flags().StringVar(&identifier, "identifier", "", "Contact identifier")
	cmd.Flags().StringVar(&identifierHash, "identifier-hash", "", "Contact identifier hash")
	cmd.Flags().StringVar(&avatarURL, "avatar-url", "", "Contact avatar URL")
	cmd.Flags().StringVar(&customAttributes, "custom-attributes", "", "Custom attributes JSON")
	flagAlias(cmd.Flags(), "identifier", "idn")
	flagAlias(cmd.Flags(), "email", "em")
	flagAlias(cmd.Flags(), "name", "nm")
	flagAlias(cmd.Flags(), "phone", "ph")
	flagAlias(cmd.Flags(), "avatar-url", "av")

	return cmd
}

func newClientContactsGetCmd(baseURL, inboxIdentifier, contactIdentifier *string) *cobra.Command {
	return &cobra.Command{
		Use:     "get",
		Aliases: []string{"g"},
		Short:   "Get a contact via public API",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if err := requirePublicIdentifiers(inboxIdentifier, contactIdentifier, true); err != nil {
				return err
			}

			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			contact, err := client.Public().GetContact(cmdContext(cmd), *inboxIdentifier, *contactIdentifier)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, contact)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Contact %d\n", contact.ID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Name: %s\n", contact.Name)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Email: %s\n", contact.Email)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Source ID: %s\n", contact.SourceID)
			return nil
		}),
	}
}

func newClientConversationsCmd(baseURL, inboxIdentifier, contactIdentifier *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "conversations",
		Short: "Manage conversations via public API",
	}

	cmd.AddCommand(newClientConversationsListCmd(baseURL, inboxIdentifier, contactIdentifier))
	cmd.AddCommand(newClientConversationsCreateCmd(baseURL, inboxIdentifier, contactIdentifier))
	cmd.AddCommand(newClientConversationsGetCmd(baseURL, inboxIdentifier, contactIdentifier))
	cmd.AddCommand(newClientConversationsResolveCmd(baseURL, inboxIdentifier, contactIdentifier))

	return cmd
}

func newClientConversationsListCmd(baseURL, inboxIdentifier, contactIdentifier *string) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List conversations for a contact",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if err := requirePublicIdentifiers(inboxIdentifier, contactIdentifier, true); err != nil {
				return err
			}

			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			conversations, err := client.Public().ListConversations(cmdContext(cmd), *inboxIdentifier, *contactIdentifier)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, conversations)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tSTATUS\tINBOX")
			for _, conv := range conversations {
				id := conv["id"]
				status := conv["status"]
				inbox := conv["inbox_id"]
				_, _ = fmt.Fprintf(w, "%v\t%v\t%v\n", id, status, inbox)
			}
			return nil
		}),
	}
}

func newClientConversationsCreateCmd(baseURL, inboxIdentifier, contactIdentifier *string) *cobra.Command {
	var customAttributes string

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"mk"},
		Short:   "Create a conversation for a contact",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if err := requirePublicIdentifiers(inboxIdentifier, contactIdentifier, true); err != nil {
				return err
			}

			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			var attrs map[string]any
			if customAttributes != "" {
				if err := json.Unmarshal([]byte(customAttributes), &attrs); err != nil {
					return fmt.Errorf("invalid custom-attributes JSON: %w", err)
				}
			}

			conversation, err := client.Public().CreateConversation(cmdContext(cmd), *inboxIdentifier, *contactIdentifier, attrs)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, conversation)
			}

			printAction(cmd, "Created", "conversation", conversation["id"], "")
			return nil
		}),
	}

	cmd.Flags().StringVar(&customAttributes, "custom-attributes", "", "Custom attributes JSON")

	return cmd
}

func newClientConversationsGetCmd(baseURL, inboxIdentifier, contactIdentifier *string) *cobra.Command {
	return &cobra.Command{
		Use:     "get <conversation-id>",
		Aliases: []string{"g"},
		Short:   "Get a conversation",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if err := requirePublicIdentifiers(inboxIdentifier, contactIdentifier, true); err != nil {
				return err
			}

			conversationID, err := parsePositiveIntArg(args[0], "conversation ID")
			if err != nil {
				return err
			}

			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			conversation, err := client.Public().GetConversation(cmdContext(cmd), *inboxIdentifier, *contactIdentifier, conversationID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, conversation)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversation %v (%v)\n", conversation["id"], conversation["status"])
			return nil
		}),
	}
}

func newClientConversationsResolveCmd(baseURL, inboxIdentifier, contactIdentifier *string) *cobra.Command {
	return &cobra.Command{
		Use:   "resolve <conversation-id>",
		Short: "Resolve a conversation",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if err := requirePublicIdentifiers(inboxIdentifier, contactIdentifier, true); err != nil {
				return err
			}

			conversationID, err := parsePositiveIntArg(args[0], "conversation ID")
			if err != nil {
				return err
			}

			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			conversation, err := client.Public().ResolveConversation(cmdContext(cmd), *inboxIdentifier, *contactIdentifier, conversationID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, conversation)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Resolved conversation %v\n", conversation["id"])
			return nil
		}),
	}
}

func newClientMessagesCmd(baseURL, inboxIdentifier, contactIdentifier *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "messages",
		Short: "Manage messages via public API",
	}

	cmd.AddCommand(newClientMessagesCreateCmd(baseURL, inboxIdentifier, contactIdentifier))

	return cmd
}

func newClientMessagesCreateCmd(baseURL, inboxIdentifier, contactIdentifier *string) *cobra.Command {
	var content string
	var echoID string

	cmd := &cobra.Command{
		Use:     "create <conversation-id>",
		Aliases: []string{"mk"},
		Short:   "Create a message in a conversation",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if err := requirePublicIdentifiers(inboxIdentifier, contactIdentifier, true); err != nil {
				return err
			}
			if content == "" {
				return fmt.Errorf("--content is required")
			}

			conversationID, err := parsePositiveIntArg(args[0], "conversation ID")
			if err != nil {
				return err
			}

			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			message, err := client.Public().CreateMessage(cmdContext(cmd), *inboxIdentifier, *contactIdentifier, conversationID, content, echoID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, message)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Sent message %v\n", message["id"])
			return nil
		}),
	}

	cmd.Flags().StringVar(&content, "content", "", "Message content")
	cmd.Flags().StringVar(&echoID, "echo-id", "", "Optional echo ID")

	return cmd
}

func newClientTypingCmd(baseURL, inboxIdentifier, contactIdentifier *string) *cobra.Command {
	var status string

	cmd := &cobra.Command{
		Use:   "typing <conversation-id>",
		Short: "Toggle typing status",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if err := requirePublicIdentifiers(inboxIdentifier, contactIdentifier, true); err != nil {
				return err
			}

			conversationID, err := parsePositiveIntArg(args[0], "conversation ID")
			if err != nil {
				return err
			}

			if status != "on" && status != "off" {
				return fmt.Errorf("--status must be 'on' or 'off'")
			}

			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			if err := client.Public().ToggleTyping(cmdContext(cmd), *inboxIdentifier, *contactIdentifier, conversationID, status); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"status": status})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Typing status set to %s\n", status)
			return nil
		}),
	}

	cmd.Flags().StringVar(&status, "status", "on", "Typing status (on|off)")

	return cmd
}

func newClientLastSeenCmd(baseURL, inboxIdentifier, contactIdentifier *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "last-seen",
		Aliases: []string{"lsn"},
		Short:   "Update last seen status",
	}

	cmd.AddCommand(newClientLastSeenUpdateCmd(baseURL, inboxIdentifier, contactIdentifier))

	return cmd
}

func newClientLastSeenUpdateCmd(baseURL, inboxIdentifier, contactIdentifier *string) *cobra.Command {
	return &cobra.Command{
		Use:     "update <conversation-id>",
		Aliases: []string{"up"},
		Short:   "Update last seen for a conversation",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if err := requirePublicIdentifiers(inboxIdentifier, contactIdentifier, true); err != nil {
				return err
			}

			conversationID, err := parsePositiveIntArg(args[0], "conversation ID")
			if err != nil {
				return err
			}

			client, err := getPublicClient(*baseURL)
			if err != nil {
				return err
			}

			if err := client.Public().UpdateLastSeen(cmdContext(cmd), *inboxIdentifier, *contactIdentifier, conversationID); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"updated": true})
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Last seen updated")
			return nil
		}),
	}
}
