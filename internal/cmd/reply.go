package cmd

import (
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

// newReplyCmd creates the reply command for one-shot messaging by contact search
func newReplyCmd() *cobra.Command {
	var (
		content        string
		resolve        bool
		contactID      int
		conversationID int
		private        bool
	)

	cmd := &cobra.Command{
		Use:   "reply [contact-search]",
		Short: "Send a reply to a contact's open conversation",
		Long: `Send a reply by searching for a contact by name or email.
If multiple contacts match, disambiguation is required.
If multiple open conversations exist for the contact, disambiguation is required.`,
		Example: strings.TrimSpace(`
  # Reply by contact name
  chatwoot reply "welgrow" --content "Your shipment is ready!"

  # Reply by email
  chatwoot reply "john@example.com" --content "Thanks for reaching out"

  # Reply and resolve the conversation
  chatwoot reply "welgrow" --content "Done!" --resolve

  # Reply using a specific contact ID (skip search)
  chatwoot reply --contact-id 789 --content "Hello!"

  # Reply to a specific conversation (skip all lookups)
  chatwoot reply --conversation-id 123 --content "Confirmed"

  # Send a private note (internal, not visible to customer)
  chatwoot reply "welgrow" --content "Internal note" --private
`),
		Args: cobra.MaximumNArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			// Validate inputs
			if content == "" {
				return fmt.Errorf("--content is required")
			}

			if err := validation.ValidateMessageContent(content); err != nil {
				return err
			}

			// Determine the mode: direct conversation, contact ID, or search
			if conversationID == 0 && contactID == 0 && len(args) == 0 {
				return fmt.Errorf("either provide a contact search query, --contact-id, or --conversation-id")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			// Mode 1: Direct conversation ID - skip all lookups
			if conversationID > 0 {
				return replyToConversation(cmd, client, conversationID, content, private, resolve, nil)
			}

			// Mode 2: Contact ID - skip search, lookup conversations
			if contactID > 0 {
				return replyByContactID(cmd, client, contactID, content, private, resolve)
			}

			// Mode 3: Search by contact name/email
			query := args[0]
			contacts, err := client.SearchContacts(ctx, query, 1)
			if err != nil {
				return fmt.Errorf("failed to search contacts: %w", err)
			}

			if len(contacts.Payload) == 0 {
				return fmt.Errorf("no contacts found matching %q", query)
			}

			if len(contacts.Payload) > 1 {
				return outputDisambiguation(cmd, "multiple_contacts", contacts.Payload)
			}

			// Single contact found
			contact := contacts.Payload[0]
			return replyByContactID(cmd, client, contact.ID, content, private, resolve)
		}),
	}

	cmd.Flags().StringVar(&content, "content", "", "Message content (required)")
	cmd.Flags().BoolVar(&resolve, "resolve", false, "Resolve the conversation after replying")
	cmd.Flags().IntVar(&contactID, "contact-id", 0, "Skip search, use specific contact ID")
	cmd.Flags().IntVar(&conversationID, "conversation-id", 0, "Skip all lookups, reply to specific conversation")
	cmd.Flags().BoolVar(&private, "private", false, "Send as private note (not visible to customer)")

	return cmd
}

// replyByContactID finds open conversations for a contact and replies
func replyByContactID(cmd *cobra.Command, client *api.Client, contactID int, content string, private, resolve bool) error {
	ctx := cmdContext(cmd)

	// Get contact details for output
	contact, err := client.GetContact(ctx, contactID)
	if err != nil {
		return fmt.Errorf("failed to get contact %d: %w", contactID, err)
	}

	// Get all conversations for this contact
	conversations, err := client.GetContactConversations(ctx, contactID)
	if err != nil {
		return fmt.Errorf("failed to get conversations for contact %d: %w", contactID, err)
	}

	// Filter to open conversations only
	var openConversations []api.Conversation
	for _, conv := range conversations {
		if conv.Status == "open" {
			openConversations = append(openConversations, conv)
		}
	}

	if len(openConversations) == 0 {
		return fmt.Errorf("no open conversations found for contact %q (ID: %d)", contact.Name, contactID)
	}

	if len(openConversations) > 1 {
		return outputConversationDisambiguation(cmd, openConversations, contactID)
	}

	// Single open conversation found
	triageContact := &api.TriageContact{
		ID:    contact.ID,
		Name:  contact.Name,
		Email: contact.Email,
	}
	return replyToConversation(cmd, client, openConversations[0].ID, content, private, resolve, triageContact)
}

// replyToConversation sends a message to a specific conversation
func replyToConversation(cmd *cobra.Command, client *api.Client, conversationID int, content string, private, resolve bool, contact *api.TriageContact) error {
	ctx := cmdContext(cmd)

	// Send the message
	messageType := "outgoing"
	message, err := client.CreateMessage(ctx, conversationID, content, private, messageType)
	if err != nil {
		return fmt.Errorf("failed to send message to conversation %d: %w", conversationID, err)
	}

	resolved := false

	// Resolve if requested
	if resolve {
		_, err := client.ToggleConversationStatus(ctx, conversationID, "resolved", 0)
		if err != nil {
			return fmt.Errorf("message sent (ID: %d) but failed to resolve conversation: %w", message.ID, err)
		}
		resolved = true
	}

	// If we don't have contact info, try to fetch conversation to get it
	if contact == nil {
		conv, err := client.GetConversation(ctx, conversationID)
		if err == nil && conv.ContactID > 0 {
			contactData, err := client.GetContact(ctx, conv.ContactID)
			if err == nil {
				contact = &api.TriageContact{
					ID:    contactData.ID,
					Name:  contactData.Name,
					Email: contactData.Email,
				}
			}
		}
	}

	result := api.ReplyResult{
		Action:         "replied",
		ConversationID: conversationID,
		Contact:        contact,
		MessageID:      message.ID,
		Resolved:       resolved,
	}

	if isJSON(cmd) {
		return printJSON(cmd, result)
	}

	// Text output
	fmt.Printf("Message sent successfully (ID: %d)\n", message.ID)
	fmt.Printf("Conversation: %d\n", conversationID)
	if contact != nil {
		fmt.Printf("Contact: %s", contact.Name)
		if contact.Email != "" {
			fmt.Printf(" <%s>", contact.Email)
		}
		fmt.Println()
	}
	if private {
		fmt.Println("Type: Private note")
	}
	if resolved {
		fmt.Println("Status: Resolved")
	}

	return nil
}

// outputDisambiguation outputs disambiguation needed for multiple contacts
func outputDisambiguation(cmd *cobra.Command, disambiguationType string, contacts []api.Contact) error {
	// Build matches for output
	type contactMatch struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email,omitempty"`
	}

	matches := make([]contactMatch, 0, len(contacts))
	for _, c := range contacts {
		matches = append(matches, contactMatch{
			ID:    c.ID,
			Name:  c.Name,
			Email: c.Email,
		})
	}

	result := api.ReplyResult{
		Action:  "disambiguation_needed",
		Type:    disambiguationType,
		Matches: matches,
		Hint:    "Use contact ID: chatwoot reply --contact-id <id> --content '...'",
	}

	if isJSON(cmd) {
		return printJSON(cmd, result)
	}

	// Text output
	fmt.Println("Multiple contacts found. Please specify one:")
	fmt.Println()
	w := newTabWriter()
	_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL")
	for _, c := range contacts {
		email := c.Email
		if email == "" {
			email = "-"
		}
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\n", c.ID, c.Name, email)
	}
	_ = w.Flush()
	fmt.Println()
	fmt.Println("Hint:", result.Hint)

	return nil
}

// outputConversationDisambiguation outputs disambiguation needed for multiple conversations
func outputConversationDisambiguation(cmd *cobra.Command, conversations []api.Conversation, contactID int) error {
	type conversationMatch struct {
		ID             int    `json:"id"`
		DisplayID      *int   `json:"display_id,omitempty"`
		InboxID        int    `json:"inbox_id"`
		Status         string `json:"status"`
		LastActivityAt int64  `json:"last_activity_at"`
	}

	matches := make([]conversationMatch, 0, len(conversations))
	for _, c := range conversations {
		matches = append(matches, conversationMatch{
			ID:             c.ID,
			DisplayID:      c.DisplayID,
			InboxID:        c.InboxID,
			Status:         c.Status,
			LastActivityAt: c.LastActivityAt,
		})
	}

	result := api.ReplyResult{
		Action:  "disambiguation_needed",
		Type:    "multiple_conversations",
		Matches: matches,
		Hint:    "Use conversation ID: chatwoot reply --conversation-id <id> --content '...'",
	}

	if isJSON(cmd) {
		return printJSON(cmd, result)
	}

	// Text output
	fmt.Printf("Multiple open conversations found for contact (ID: %d). Please specify one:\n", contactID)
	fmt.Println()
	w := newTabWriter()
	_, _ = fmt.Fprintln(w, "ID\tDISPLAY_ID\tINBOX\tLAST_ACTIVITY")
	for _, c := range conversations {
		displayID := "-"
		if c.DisplayID != nil {
			displayID = fmt.Sprintf("%d", *c.DisplayID)
		}
		_, _ = fmt.Fprintf(w, "%d\t%s\t%d\t%s\n",
			c.ID,
			displayID,
			c.InboxID,
			c.LastActivityAtTime().Format("2006-01-02 15:04:05"),
		)
	}
	_ = w.Flush()
	fmt.Println()
	fmt.Println("Hint:", result.Hint)

	return nil
}
