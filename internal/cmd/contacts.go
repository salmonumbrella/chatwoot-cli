package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newContactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contacts",
		Short: "Manage contacts",
		Long:  "List, create, update, delete, and search contacts in your Chatwoot account",
	}

	cmd.AddCommand(newContactsListCmd())
	cmd.AddCommand(newContactsGetCmd())
	cmd.AddCommand(newContactsShowCmd()) // alias for get
	cmd.AddCommand(newContactsCreateCmd())
	cmd.AddCommand(newContactsUpdateCmd())
	cmd.AddCommand(newContactsDeleteCmd())
	cmd.AddCommand(newContactsSearchCmd())
	cmd.AddCommand(newContactsFilterCmd())
	cmd.AddCommand(newContactsConversationsCmd())
	cmd.AddCommand(newContactsLabelsCmd())
	cmd.AddCommand(newContactsLabelsAddCmd())
	cmd.AddCommand(newContactsContactableInboxesCmd())
	cmd.AddCommand(newContactsCreateInboxCmd())
	cmd.AddCommand(newContactsNotesCmd())
	cmd.AddCommand(newContactsNotesAddCmd())
	cmd.AddCommand(newContactsNotesDeleteCmd())

	return cmd
}

func newContactsListCmd() *cobra.Command {
	var page int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all contacts",
		Long: `List all contacts in your Chatwoot account.

JSON output returns an array of contacts directly for easy jq processing.`,
		Example: `  # List contacts in table format
  chatwoot contacts list

  # List with pagination
  chatwoot contacts list --page 2

  # JSON output - returns array directly
  chatwoot contacts list --output json | jq '.[0]'
  chatwoot contacts list --output json | jq '.[] | {id, name, email}'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			contacts, err := client.ListContacts(cmdContext(cmd), page)
			if err != nil {
				return fmt.Errorf("failed to list contacts: %w", err)
			}

			if isJSON(cmd) {
				// Return array directly for easier jq processing
				return printJSON(contacts.Payload)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tPHONE")
			for _, contact := range contacts.Payload {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
					contact.ID,
					contact.Name,
					contact.Email,
					contact.PhoneNumber,
				)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&page, "page", 0, "Page number for pagination")

	return cmd
}

func newContactsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get contact by ID",
		Long: `Get a specific contact by their ID.

Use 'chatwoot contacts show <id>' as an alias for this command.`,
		Example: `  # Get contact by ID
  chatwoot contacts get 123

  # Get contact as JSON
  chatwoot contacts get 123 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "contact ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			contact, err := client.GetContact(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get contact %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(contact)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tPHONE")
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
				contact.ID,
				contact.Name,
				contact.Email,
				contact.PhoneNumber,
			)

			return nil
		},
	}
}

// newContactsShowCmd creates a 'show' command as an alias for 'get'
func newContactsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show contact by ID (alias for 'get')",
		Long: `Show a specific contact by their ID.

This is an alias for 'chatwoot contacts get <id>'.`,
		Example: `  # Show contact by ID
  chatwoot contacts show 123

  # Show contact as JSON
  chatwoot contacts show 123 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "contact ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			contact, err := client.GetContact(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get contact %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(contact)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tPHONE")
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
				contact.ID,
				contact.Name,
				contact.Email,
				contact.PhoneNumber,
			)

			return nil
		},
	}
}

func newContactsCreateCmd() *cobra.Command {
	var (
		name  string
		email string
		phone string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new contact",
		Long:  "Create a new contact with the specified name, email, and/or phone number",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			// Validate input lengths
			if err := validation.ValidateName(name); err != nil {
				return err
			}
			if err := validation.ValidateEmail(email); err != nil {
				return err
			}
			if err := validation.ValidatePhone(phone); err != nil {
				return err
			}

			// Validate input formats
			if err := validation.ValidateEmailFormat(email); err != nil {
				return err
			}
			if err := validation.ValidatePhoneFormat(phone); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			contact, err := client.CreateContact(cmdContext(cmd), name, email, phone)
			if err != nil {
				return fmt.Errorf("failed to create contact: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(contact)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tPHONE")
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
				contact.ID,
				contact.Name,
				contact.Email,
				contact.PhoneNumber,
			)

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Contact name (required)")
	cmd.Flags().StringVar(&email, "email", "", "Contact email address")
	cmd.Flags().StringVar(&phone, "phone", "", "Contact phone number")

	return cmd
}

func newContactsUpdateCmd() *cobra.Command {
	var (
		name  string
		email string
		phone string
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a contact",
		Long:  "Update a contact's name, email, and/or phone number",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "contact ID")
			if err != nil {
				return err
			}

			if name == "" && email == "" && phone == "" {
				return fmt.Errorf("at least one of --name, --email, or --phone must be provided")
			}

			// Validate input lengths
			if err := validation.ValidateName(name); err != nil {
				return err
			}
			if err := validation.ValidateEmail(email); err != nil {
				return err
			}
			if err := validation.ValidatePhone(phone); err != nil {
				return err
			}

			// Validate input formats
			if err := validation.ValidateEmailFormat(email); err != nil {
				return err
			}
			if err := validation.ValidatePhoneFormat(phone); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			contact, err := client.UpdateContact(cmdContext(cmd), id, name, email, phone)
			if err != nil {
				return fmt.Errorf("failed to update contact %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(contact)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tPHONE")
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
				contact.ID,
				contact.Name,
				contact.Email,
				contact.PhoneNumber,
			)

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New contact name")
	cmd.Flags().StringVar(&email, "email", "", "New contact email address")
	cmd.Flags().StringVar(&phone, "phone", "", "New contact phone number")

	return cmd
}

func newContactsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a contact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "contact ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.DeleteContact(cmdContext(cmd), id); err != nil {
				return fmt.Errorf("failed to delete contact %d: %w", id, err)
			}

			if !isJSON(cmd) {
				fmt.Printf("Contact %d deleted successfully\n", id)
			}

			return nil
		},
	}
}

func newContactsSearchCmd() *cobra.Command {
	var query string

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search contacts",
		Long: `Search for contacts by query string.

The query matches against contact name, email, phone number, and identifier.
JSON output returns an array of contacts directly for easy jq processing.`,
		Example: `  # Search for contacts by name
  chatwoot contacts search --query "John"

  # Search and output as JSON
  chatwoot contacts search --query "acme" --output json | jq '.[] | {id, name}'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if query == "" {
				return fmt.Errorf("--query is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			contacts, err := client.SearchContacts(cmdContext(cmd), query)
			if err != nil {
				return fmt.Errorf("failed to search contacts: %w", err)
			}

			if isJSON(cmd) {
				// Return array directly for easier jq processing
				return printJSON(contacts.Payload)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tPHONE")
			for _, contact := range contacts.Payload {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
					contact.ID,
					contact.Name,
					contact.Email,
					contact.PhoneNumber,
				)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&query, "query", "", "Search query string")

	return cmd
}

func newContactsFilterCmd() *cobra.Command {
	var payload string

	cmd := &cobra.Command{
		Use:   "filter",
		Short: "Filter contacts",
		Long: `Filter contacts using a JSON array of filter conditions.

Example payload format:
[
  {
    "attribute_key": "name",
    "filter_operator": "contains",
    "values": ["test"],
    "query_operator": "and"
  }
]

Available filter operators: equal_to, not_equal_to, contains, does_not_contain, is_present, is_not_present
Available query operators: and, or`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if payload == "" {
				return fmt.Errorf("--payload is required")
			}

			// Validate JSON payload size
			if err := validation.ValidateJSONPayload(payload); err != nil {
				return err
			}

			var filterConditions []map[string]any
			if err := json.Unmarshal([]byte(payload), &filterConditions); err != nil {
				return fmt.Errorf("invalid JSON payload (must be an array of filter conditions): %w", err)
			}

			filterPayload := map[string]any{
				"payload": filterConditions,
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			contacts, err := client.FilterContacts(cmdContext(cmd), filterPayload)
			if err != nil {
				return fmt.Errorf("failed to filter contacts: %w", err)
			}

			if isJSON(cmd) {
				// Return array directly for easier jq processing
				return printJSON(contacts.Payload)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tPHONE")
			for _, contact := range contacts.Payload {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
					contact.ID,
					contact.Name,
					contact.Email,
					contact.PhoneNumber,
				)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&payload, "payload", "", "JSON array of filter conditions")

	return cmd
}

func newContactsConversationsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "conversations <id>",
		Short: "Get contact conversations",
		Long:  "Get all conversations for a specific contact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "contact ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			conversations, err := client.GetContactConversations(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get conversations for contact %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(conversations)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tSTATUS\tINBOX_ID\tUNREAD")
			for _, conv := range conversations {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%d\t%d\n",
					conv.ID,
					conv.Status,
					conv.InboxID,
					conv.Unread,
				)
			}

			return nil
		},
	}
}

func newContactsLabelsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "labels <id>",
		Short: "Get contact labels",
		Long:  "Get all labels for a specific contact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "contact ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			labels, err := client.GetContactLabels(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get labels for contact %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(labels)
			}

			if len(labels) == 0 {
				fmt.Println("No labels found")
				return nil
			}

			for _, label := range labels {
				fmt.Println(label)
			}

			return nil
		},
	}
}

func newContactsLabelsAddCmd() *cobra.Command {
	var labels string

	cmd := &cobra.Command{
		Use:   "labels-add <id>",
		Short: "Add labels to contact",
		Long:  "Add one or more labels to a contact (comma-separated)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "contact ID")
			if err != nil {
				return err
			}

			if labels == "" {
				return fmt.Errorf("--labels is required")
			}

			labelList := strings.Split(labels, ",")
			for i := range labelList {
				labelList[i] = strings.TrimSpace(labelList[i])
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			updatedLabels, err := client.AddContactLabels(cmdContext(cmd), id, labelList)
			if err != nil {
				return fmt.Errorf("failed to add labels to contact %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(updatedLabels)
			}

			fmt.Println("Labels added successfully:")
			for _, label := range updatedLabels {
				fmt.Println(label)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&labels, "labels", "", "Comma-separated list of labels")

	return cmd
}

func newContactsContactableInboxesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "contactable-inboxes <id>",
		Short: "Get contactable inboxes",
		Long:  "Get all contactable inboxes for a contact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "contact ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			inboxes, err := client.GetContactableInboxes(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get contactable inboxes for contact %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(inboxes)
			}

			if len(inboxes) == 0 {
				fmt.Println("No contactable inboxes found")
				return nil
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tCHANNEL_TYPE")
			for _, inbox := range inboxes {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\n",
					inbox.ID,
					inbox.Name,
					inbox.ChannelType,
				)
			}

			return nil
		},
	}
}

func newContactsCreateInboxCmd() *cobra.Command {
	var (
		inboxID  int
		sourceID string
	)

	cmd := &cobra.Command{
		Use:   "create-inbox <contact-id>",
		Short: "Associate contact with an inbox",
		Long:  "Create a contact inbox association, allowing the contact to be reached via that inbox",
		Example: `  # Associate contact 123 with inbox 1
  chatwoot contacts create-inbox 123 --inbox-id 1

  # With custom source ID (for channel-specific identifiers)
  chatwoot contacts create-inbox 123 --inbox-id 1 --source-id "+15551234567"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			contactID, err := validation.ParsePositiveInt(args[0], "contact ID")
			if err != nil {
				return err
			}

			if inboxID == 0 {
				return fmt.Errorf("--inbox-id is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			result, err := client.CreateContactInbox(cmdContext(cmd), contactID, inboxID, sourceID)
			if err != nil {
				return fmt.Errorf("failed to create contact inbox: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(result)
			}

			if result.Inbox.ID == 0 {
				fmt.Printf("Contact %d associated with inbox (no details returned)\n", contactID)
			} else {
				fmt.Printf("Contact %d associated with inbox %d\n", contactID, result.Inbox.ID)
				fmt.Printf("Inbox: %s (%s)\n", result.Inbox.Name, result.Inbox.ChannelType)
			}
			if result.SourceID != "" {
				fmt.Printf("Source ID: %s\n", result.SourceID)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&inboxID, "inbox-id", 0, "Inbox ID (required)")
	cmd.Flags().StringVar(&sourceID, "source-id", "", "Channel-specific source identifier")
	_ = cmd.MarkFlagRequired("inbox-id")

	return cmd
}

func newContactsNotesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "notes <contact-id>",
		Short: "List contact notes",
		Long:  "List all notes for a specific contact",
		Example: strings.TrimSpace(`
  # List notes for a contact
  chatwoot contacts notes 123

  # JSON output
  chatwoot contacts notes 123 -o json
`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "contact ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			notes, err := client.GetContactNotes(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get notes for contact %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(notes)
			}

			if len(notes) == 0 {
				fmt.Println("No notes found")
				return nil
			}

			w := newTabWriter()
			_, _ = fmt.Fprintln(w, "ID\tCREATED\tAUTHOR\tCONTENT")
			for _, note := range notes {
				author := ""
				if note.User != nil {
					author = note.User.Email
				}
				content := note.Content
				if len(content) > 50 {
					content = content[:47] + "..."
				}
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", note.ID, note.CreatedAt, author, content)
			}
			_ = w.Flush()

			return nil
		},
	}
}

func newContactsNotesAddCmd() *cobra.Command {
	var content string

	cmd := &cobra.Command{
		Use:   "notes-add <contact-id>",
		Short: "Add note to contact",
		Long:  "Add a new note to a contact",
		Example: strings.TrimSpace(`
  # Add a note to a contact
  chatwoot contacts notes-add 123 --content "VIP customer, handle with care"
`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "contact ID")
			if err != nil {
				return err
			}

			if content == "" {
				return fmt.Errorf("--content is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			note, err := client.CreateContactNote(cmdContext(cmd), id, content)
			if err != nil {
				return fmt.Errorf("failed to add note to contact %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(note)
			}

			fmt.Printf("Added note #%d to contact %d\n", note.ID, id)
			return nil
		},
	}

	cmd.Flags().StringVar(&content, "content", "", "Note content (required)")

	return cmd
}

func newContactsNotesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "notes-delete <contact-id> <note-id>",
		Short: "Delete contact note",
		Long:  "Delete a note from a contact",
		Example: strings.TrimSpace(`
  # Delete a note from a contact
  chatwoot contacts notes-delete 123 456
`),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			contactID, err := validation.ParsePositiveInt(args[0], "contact ID")
			if err != nil {
				return err
			}

			noteID, err := validation.ParsePositiveInt(args[1], "note ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.DeleteContactNote(cmdContext(cmd), contactID, noteID); err != nil {
				return fmt.Errorf("failed to delete note %d from contact %d: %w", noteID, contactID, err)
			}

			fmt.Printf("Deleted note #%d from contact %d\n", noteID, contactID)
			return nil
		},
	}
}
