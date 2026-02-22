package cmd

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/dryrun"
	"github.com/chatwoot/chatwoot-cli/internal/iocontext"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

// replySideEffects holds optional post-send side-effect parameters for reply.
type replySideEffects struct {
	labels    []string
	priority  string
	snoozeFor string
	pending   bool
}

// newReplyCmd creates the reply command for one-shot messaging by contact search
func newReplyCmd() *cobra.Command {
	var (
		content        string
		resolve        bool
		pending        bool
		contactID      int
		conversationID int
		private        bool
		labels         []string
		priority       string
		snoozeFor      string
	)

	cmd := &cobra.Command{
		Use:     "reply [contact-search]",
		Aliases: []string{"respond", "r"},
		Short:   "Send a reply to a contact's open conversation",
		Long: `Send a reply by searching for a contact by name or email.
If multiple contacts match, disambiguation is required.
If multiple open conversations exist for the contact, disambiguation is required.`,
		Example: strings.TrimSpace(`
  # Reply by contact name
  cw reply "welgrow" --content "Your shipment is ready!"

  # Reply by email
  cw reply "john@example.com" --content "Thanks for reaching out"

  # Reply and resolve the conversation
  cw reply "welgrow" --content "Done!" --resolve

  # Reply using a specific contact ID (skip search)
  cw reply --contact-id 789 --content "Hello!"

  # Reply to a specific conversation (skip all lookups)
  cw reply --conversation-id 123 --content "Confirmed"

  # Send a private note (internal, not visible to customer)
  cw reply "welgrow" --content "Internal note" --private
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

			// Validate side-effect flags before sending so we fail fast.
			if err := validateExclusiveStatus(resolve, pending, snoozeFor); err != nil {
				return err
			}
			var err error
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

			se := replySideEffects{labels: labels, priority: priority, snoozeFor: snoozeFor, pending: pending}

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
				return replyToConversation(cmd, client, conversationID, content, private, resolve, nil, se)
			}

			// Mode 2: Contact ID - skip search, lookup conversations
			if contactID > 0 {
				return replyByContactID(cmd, client, contactID, content, private, resolve, se)
			}

			// Mode 3: Search by contact name/email
			query := args[0]
			contacts, err := client.Contacts().Search(ctx, query, 1)
			if err != nil {
				return fmt.Errorf("failed to search contacts: %w", err)
			}

			if len(contacts.Payload) == 0 {
				if isDryRun(cmd) {
					return fmt.Errorf("no contacts found matching %q (dry-run still requires a real contact/conversation; use --contact-id or --conversation-id)", query)
				}
				return fmt.Errorf("no contacts found matching %q", query)
			}

			if len(contacts.Payload) > 1 {
				if !isJSON(cmd) && !flags.NoInput && isInteractive() {
					var options []selectOption
					for _, c := range contacts.Payload {
						label := c.Name
						if label == "" {
							label = fmt.Sprintf("Contact %d", c.ID)
						}
						if c.Email != "" {
							label = fmt.Sprintf("%s <%s>", label, c.Email)
						}
						options = append(options, selectOption{
							ID:    c.ID,
							Label: label,
						})
					}
					selectedID, ok, err := promptSelect(ctx, "Select contact", options, true)
					if err != nil {
						return err
					}
					if !ok {
						return nil
					}
					return replyByContactID(cmd, client, selectedID, content, private, resolve, se)
				}
				return outputDisambiguation(cmd, "multiple_contacts", contacts.Payload)
			}

			// Single contact found
			contact := contacts.Payload[0]
			return replyByContactID(cmd, client, contact.ID, content, private, resolve, se)
		}),
	}

	cmd.Flags().StringVarP(&content, "content", "c", "", "Message content (required)")
	cmd.Flags().BoolVarP(&resolve, "resolve", "R", false, "Resolve the conversation after replying")
	cmd.Flags().BoolVarP(&pending, "pending", "p", false, "Set conversation to pending after replying")
	cmd.Flags().IntVarP(&contactID, "contact-id", "C", 0, "Skip search, use specific contact ID")
	flagAlias(cmd.Flags(), "contact-id", "cid")
	cmd.Flags().IntVar(&conversationID, "conversation-id", 0, "Skip all lookups, reply to specific conversation")
	flagAlias(cmd.Flags(), "conversation-id", "cvid")
	cmd.Flags().BoolVarP(&private, "private", "P", false, "Send as private note (not visible to customer)")
	cmd.Flags().StringSliceVar(&labels, "label", nil, "Add labels after sending (repeatable)")
	flagAlias(cmd.Flags(), "label", "lb")
	cmd.Flags().StringVar(&priority, "priority", "", "Set priority after sending (urgent|high|medium|low|none)")
	flagAlias(cmd.Flags(), "priority", "pri")
	cmd.Flags().StringVar(&snoozeFor, "snooze-for", "", "Snooze after sending (e.g., 2h, 30m)")
	flagAlias(cmd.Flags(), "snooze-for", "for")

	return cmd
}

// replyByContactID finds open conversations for a contact and replies
func replyByContactID(cmd *cobra.Command, client *api.Client, contactID int, content string, private, resolve bool, se replySideEffects) error {
	ctx := cmdContext(cmd)

	// Get contact details for output
	contact, err := client.Contacts().Get(ctx, contactID)
	if err != nil {
		return fmt.Errorf("failed to get contact %d: %w", contactID, err)
	}

	// Get all conversations for this contact
	conversations, err := client.Contacts().Conversations(ctx, contactID)
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
		if !isJSON(cmd) && !flags.NoInput && isInteractive() {
			var options []selectOption
			for _, conv := range openConversations {
				label := fmt.Sprintf("Conversation %d (%s, inbox %d)", conv.ID, conv.Status, conv.InboxID)
				options = append(options, selectOption{
					ID:    conv.ID,
					Label: label,
				})
			}
			selectedID, ok, err := promptSelect(ctx, "Select conversation", options, true)
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
			triageContact := &api.TriageContact{
				ID:    contact.ID,
				Name:  contact.Name,
				Email: contact.Email,
			}
			return replyToConversation(cmd, client, selectedID, content, private, resolve, triageContact, se)
		}
		return outputConversationDisambiguation(cmd, openConversations, contactID)
	}

	// Single open conversation found
	triageContact := &api.TriageContact{
		ID:    contact.ID,
		Name:  contact.Name,
		Email: contact.Email,
	}
	return replyToConversation(cmd, client, openConversations[0].ID, content, private, resolve, triageContact, se)
}

// isDryRun checks if the command context has dry-run mode enabled
func isDryRun(cmd *cobra.Command) bool {
	return dryrun.IsEnabled(cmd.Context())
}

// replyToConversation sends a message to a specific conversation
func replyToConversation(cmd *cobra.Command, client *api.Client, conversationID int, content string, private, resolve bool, contact *api.TriageContact, se replySideEffects) error {
	ctx := cmdContext(cmd)

	// Check for dry-run mode BEFORE sending
	if isDryRun(cmd) {
		return printReplyDryRun(cmd, client, conversationID, content, private, contact)
	}

	// Send the message
	messageType := "outgoing"
	message, err := client.Messages().Create(ctx, conversationID, content, private, messageType)
	if err != nil {
		return fmt.Errorf("failed to send message to conversation %d: %w", conversationID, err)
	}

	resolved := false

	// Resolve if requested
	if resolve {
		_, err := client.Conversations().ToggleStatus(ctx, conversationID, "resolved", 0)
		if err != nil {
			return fmt.Errorf("message sent (ID: %d) but failed to resolve conversation: %w", message.ID, err)
		}
		resolved = true
	}

	pendingSet := false
	if se.pending {
		_, err := client.Conversations().ToggleStatus(ctx, conversationID, "pending", 0)
		if err != nil {
			return fmt.Errorf("message sent (ID: %d) but failed to set conversation to pending: %w", message.ID, err)
		}
		pendingSet = true
	}

	if len(se.labels) > 0 {
		existing, _ := client.Conversations().Labels(ctx, conversationID)
		merged := dedupeStrings(append(existing, se.labels...))
		if _, err := client.Conversations().AddLabels(ctx, conversationID, merged); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: message sent but failed to add labels: %v\n", err)
		}
	}
	if se.priority != "" {
		if err := client.Conversations().TogglePriority(ctx, conversationID, se.priority); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: message sent but failed to set priority: %v\n", err)
		}
	}
	if se.snoozeFor != "" {
		snoozedUntil, err := parseSnoozeFor(se.snoozeFor, time.Now())
		if err != nil {
			return err
		}
		_, err = client.Conversations().ToggleStatus(ctx, conversationID, "snoozed", snoozedUntil.Unix())
		if err != nil {
			return fmt.Errorf("message sent (ID: %d) but failed to snooze conversation: %w", message.ID, err)
		}
	}

	// If we don't have contact info, try to fetch conversation to get it
	if contact == nil {
		conv, err := client.Conversations().Get(ctx, conversationID)
		if err == nil && conv.ContactID > 0 {
			contactData, err := client.Contacts().Get(ctx, conv.ContactID)
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
		Pending:        pendingSet,
	}

	if isJSON(cmd) {
		return printJSON(cmd, result)
	}

	// Text output
	printAction(cmd, "Sent", "message", message.ID, "")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversation: %d\n", conversationID)
	if contact != nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Contact: %s", contact.Name)
		if contact.Email != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), " <%s>", contact.Email)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
	}
	if private {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Type: Private note")
	}
	if resolved {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Status: Resolved")
	}
	if pendingSet {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Status: Pending")
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
		Hint:    "Use contact ID: cw reply --contact-id <id> --content '...'",
	}

	if isJSON(cmd) {
		return printJSON(cmd, result)
	}

	// Text output
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Multiple contacts found. Please specify one:")
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	w := newTabWriterFromCmd(cmd)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL")
	for _, c := range contacts {
		email := c.Email
		if email == "" {
			email = "-"
		}
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\n", c.ID, c.Name, email)
	}
	_ = w.Flush()
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Hint:", result.Hint)

	return nil
}

// printReplyDryRun displays a preview of the message without sending it
func printReplyDryRun(cmd *cobra.Command, client *api.Client, conversationID int, content string, private bool, contact *api.TriageContact) error {
	ctx := cmdContext(cmd)

	// Fetch conversation to get inbox info
	conv, err := client.Conversations().Get(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation %d: %w", conversationID, err)
	}

	// Fetch inbox info
	var inbox *api.Inbox
	if conv.InboxID > 0 {
		inbox, err = client.Inboxes().Get(ctx, conv.InboxID)
		if err != nil {
			// Non-fatal, continue without inbox info
			inbox = nil
		}
	}

	// If we don't have contact info, try to fetch it
	if contact == nil && conv.ContactID > 0 {
		contactData, err := client.Contacts().Get(ctx, conv.ContactID)
		if err == nil {
			contact = &api.TriageContact{
				ID:    contactData.ID,
				Name:  contactData.Name,
				Email: contactData.Email,
			}
		}
	}

	// Calculate character count
	charCount := utf8.RuneCountInString(content)

	// Determine channel-specific warnings
	var warnings []string
	if inbox != nil {
		switch inbox.ChannelType {
		case "Channel::Line":
			if charCount > 2000 {
				warnings = append(warnings, fmt.Sprintf("LINE has a 2000 character limit; message is %d characters", charCount))
			}
		case "Channel::Whatsapp":
			if charCount > 4096 {
				warnings = append(warnings, fmt.Sprintf("WhatsApp has a 4096 character limit; message is %d characters", charCount))
			}
		case "Channel::Sms":
			if charCount > 160 {
				warnings = append(warnings, fmt.Sprintf("SMS may be split into multiple segments; message is %d characters", charCount))
			}
		}
	}

	// Build dry-run preview
	messageType := "Public reply (visible to contact)"
	if private {
		messageType = "Private note (internal only)"
	}

	// JSON output
	if isJSON(cmd) {
		payload := map[string]any{
			"dry_run":         true,
			"operation":       "send",
			"resource":        "message",
			"conversation_id": conversationID,
			"content":         content,
			"private":         private,
			"character_count": charCount,
		}
		if contact != nil {
			payload["contact"] = map[string]any{
				"id":    contact.ID,
				"name":  contact.Name,
				"email": contact.Email,
			}
		}
		if inbox != nil {
			payload["inbox"] = map[string]any{
				"id":           inbox.ID,
				"name":         inbox.Name,
				"channel_type": inbox.ChannelType,
			}
		}
		if len(warnings) > 0 {
			payload["warnings"] = warnings
		}
		return printJSON(cmd, payload)
	}

	// Text output
	ioStreams := iocontext.GetIO(ctx)
	out := ioStreams.Out

	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "[DRY-RUN] Message will NOT be sent")
	_, _ = fmt.Fprintln(out, strings.Repeat("─", 45))

	if inbox != nil {
		_, _ = fmt.Fprintf(out, "Channel: %s (%s)\n", inbox.Name, inbox.ChannelType)
	}
	if contact != nil {
		contactStr := contact.Name
		if contact.Email != "" {
			contactStr = fmt.Sprintf("%s <%s>", contact.Name, contact.Email)
		}
		_, _ = fmt.Fprintf(out, "Contact: %s\n", contactStr)
	}
	_, _ = fmt.Fprintf(out, "Conversation: %d\n", conversationID)
	_, _ = fmt.Fprintf(out, "Type: %s\n", messageType)

	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Message Preview:")
	_, _ = fmt.Fprintln(out, strings.Repeat("─", 45))
	_, _ = fmt.Fprintln(out, content)
	_, _ = fmt.Fprintln(out, strings.Repeat("─", 45))

	_, _ = fmt.Fprintf(out, "Characters: %d\n", charCount)

	if len(warnings) > 0 {
		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintln(out, "Warnings:")
		for _, warning := range warnings {
			_, _ = fmt.Fprintf(out, "  ! %s\n", warning)
		}
	}

	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "To send this message, run without --dry-run")

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
		Hint:    "Use conversation ID: cw reply --conversation-id <id> --content '...'",
	}

	if isJSON(cmd) {
		return printJSON(cmd, result)
	}

	// Text output
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Multiple open conversations found for contact (ID: %d). Please specify one:\n", contactID)
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	w := newTabWriterFromCmd(cmd)
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
			formatTimestamp(c.LastActivityAtTime()),
		)
	}
	_ = w.Flush()
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Hint:", result.Hint)

	return nil
}
