package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/dryrun"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newContactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "contacts",
		Aliases: []string{"contact", "co"},
		Short:   "Manage contacts",
		Long:    "List, create, update, delete, and search contacts in your Chatwoot account",
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
	cmd.AddCommand(newContactsBulkCmd())
	cmd.AddCommand(newContactsMergeCmd())

	return cmd
}

func newContactsListCmd() *cobra.Command {
	var page int
	var sort string
	var order string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all contacts",
		Long: `List all contacts in your Chatwoot account.

JSON output returns an object with an "items" array for easy jq processing.`,
		Example: `  # List contacts in table format
  cw contacts list

  # List with pagination
  cw contacts list --page 2

  # JSON output - returns an object with an "items" array
  cw contacts list --output json | jq '.items[0]'
  cw contacts list --output json | jq '.items[] | {id, name, email}'`,
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			sortField, sortOrder, err := parseSortOrder(sort, order)
			if err != nil {
				return err
			}

			contacts, err := client.Contacts().List(cmdContext(cmd), api.ListContactsParams{
				Page:  page,
				Sort:  sortField,
				Order: sortOrder,
			})
			if err != nil {
				return fmt.Errorf("failed to list contacts: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, contacts.Payload)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tPHONE\tCREATED")
			for _, contact := range contacts.Payload {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
					contact.ID,
					displayContactName(contact.Name),
					strings.TrimSpace(contact.Email),
					strings.TrimSpace(contact.PhoneNumber),
					formatTimestampShort(contact.CreatedAtTime()),
				)
			}

			return nil
		}),
	}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name", "email"},
		"default": {"id", "name", "email", "phone_number", "identifier", "created_at"},
		"debug":   {"id", "name", "email", "phone_number", "identifier", "thumbnail", "custom_attributes", "created_at", "last_activity_at"},
	})
	registerFieldSchema(cmd, "contact")

	cmd.Flags().IntVarP(&page, "page", "p", 0, "Page number for pagination")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort by field (name|email|phone_number|last_activity_at); prefix with '-' for desc")
	cmd.Flags().StringVar(&order, "order", "", "Sort order (asc|desc); overrides '-' prefix")

	return cmd
}

// contactGetRunE is the shared implementation for get/show commands
func contactGetRunE(cmd *cobra.Command, args []string) error {
	identifier := args[0]
	emit, _ := cmd.Flags().GetString("emit")

	// Check if identifier is numeric - if so, handle --url flag before any API call
	// This is consistent with other commands (agents, campaigns, etc.)
	if numericID, err := strconv.Atoi(identifier); err == nil && numericID > 0 {
		mode, err := normalizeEmitFlag(emit)
		if err != nil {
			return err
		}
		if mode == "id" || mode == "url" {
			_, err := maybeEmit(cmd, mode, "contact", numericID, nil)
			return err
		}

		if handled, err := handleURLFlag(cmd, "contacts", numericID); handled {
			return err
		}
	}

	client, err := getClient()
	if err != nil {
		return err
	}

	ctx := cmdContext(cmd)

	id, err := resolveContactID(ctx, client, identifier)
	if err != nil {
		return err
	}

	mode, err := normalizeEmitFlag(emit)
	if err != nil {
		return err
	}
	if mode == "id" || mode == "url" {
		_, err := maybeEmit(cmd, mode, "contact", id, nil)
		return err
	}

	// For non-numeric identifiers, check --url flag after resolution
	if handled, err := handleURLFlag(cmd, "contacts", id); handled {
		return err
	}

	contact, err := client.Contacts().Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get contact %d: %w", id, err)
	}

	if mode == "json" && !isAgent(cmd) {
		return printJSON(cmd, contact)
	}

	if isAgent(cmd) {
		detail := agentfmt.ContactDetailFromContact(*contact)

		// Fetch conversations for relationship summary
		convs, err := client.Contacts().Conversations(ctx, id)
		if err == nil && convs != nil {
			relationship := agentfmt.ComputeRelationshipSummary(convs)

			result := agentfmt.ContactDetailWithRelationship{
				ContactDetail: detail,
				Relationship:  relationship,
			}

			// Check if --with-open-conversations flag is set
			withOpenConversations, _ := cmd.Flags().GetBool("with-open-conversations")
			if withOpenConversations {
				var openConvs []api.Conversation
				for _, conv := range convs {
					if conv.Status == "open" || conv.Status == "pending" {
						openConvs = append(openConvs, conv)
					}
				}
				result.OpenConversations = agentfmt.ConversationSummaries(openConvs)
			}

			return printJSON(cmd, agentfmt.ItemEnvelope{
				Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
				Item: result,
			})
		}

		return printJSON(cmd, agentfmt.ItemEnvelope{
			Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
			Item: detail,
		})
	}

	if isJSON(cmd) {
		return printJSON(cmd, contact)
	}
	return printContactDetails(cmd.OutOrStdout(), contact)
}

func newContactsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <identifier>",
		Aliases: []string{"g"},
		Short:   "Get contact by ID, email, or name",
		Long: `Get a specific contact by their ID, email address, phone number, or name.

Accepts numeric ID, Chatwoot URL, email address, phone number, or name to search.
Use 'cw contacts show <id>' as an alias for this command.`,
		Example: `  # Get contact by ID
  cw contacts get 123

  # Get contact by email
  cw contacts get john@example.com

  # Get contact by phone number
  cw contacts get +16042091231

  # Get contact by name
  cw contacts get "John Smith"

  # Get contact as JSON
  cw contacts get 123 --output json

  # Agent mode with open conversations
  cw contacts get 123 --output agent --with-open-conversations

  # Get contact using URL from browser
  cw contacts get https://app.chatwoot.com/app/accounts/1/contacts/123`,
		Args: cobra.ExactArgs(1),
		RunE: RunE(contactGetRunE),
	}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name", "email"},
		"default": {"id", "name", "email", "phone_number", "identifier", "created_at"},
		"debug":   {"id", "name", "email", "phone_number", "identifier", "thumbnail", "custom_attributes", "created_at", "last_activity_at"},
	})
	registerFieldSchema(cmd, "contact")

	cmd.Flags().Bool("with-open-conversations", false, "Include open/pending conversations in agent output")
	cmd.Flags().Bool("url", false, "Print the Chatwoot web UI URL for this resource and exit")
	cmd.Flags().StringP("emit", "E", "", "Emit: json|id|url (overrides normal text output)")

	return cmd
}

// newContactsShowCmd creates a 'show' command as an alias for 'get'
func newContactsShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show contact by ID (alias for 'get')",
		Long: `Show a specific contact by their ID.

This is an alias for 'cw contacts get <id>'.`,
		Example: `  # Show contact by ID
  cw contacts show 123

  # Show contact as JSON
  cw contacts show 123 --output json

  # Agent mode with open conversations
  cw contacts show 123 --output agent --with-open-conversations`,
		Args: cobra.ExactArgs(1),
		RunE: RunE(contactGetRunE),
	}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name", "email"},
		"default": {"id", "name", "email", "phone_number", "identifier", "created_at"},
		"debug":   {"id", "name", "email", "phone_number", "identifier", "thumbnail", "custom_attributes", "created_at", "last_activity_at"},
	})
	registerFieldSchema(cmd, "contact")

	cmd.Flags().Bool("with-open-conversations", false, "Include open/pending conversations in agent output")
	cmd.Flags().Bool("url", false, "Print the Chatwoot web UI URL for this resource and exit")
	cmd.Flags().StringP("emit", "E", "", "Emit: json|id|url (overrides normal text output)")

	return cmd
}

func newContactsCreateCmd() *cobra.Command {
	var (
		name      string
		email     string
		phone     string
		fromStdin bool
		emit      string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"mk"},
		Short:   "Create a new contact",
		Long: `Create a new contact with the specified name, email, and/or phone number.

When using --json flag, reads JSON from stdin. CLI flags override JSON values.`,
		Example: `  # Create contact with flags
  cw contacts create --name "John Doe" --email "john@example.com"

  # Create contact from JSON stdin
  echo '{"name":"John","email":"john@test.com"}' | cw contacts create --json

  # JSON stdin with flag override (flag takes precedence)
  echo '{"name":"JSON Name","email":"json@test.com"}' | cw contacts create --json --name "Override Name"

  # Create contact with additional fields via JSON
  cat <<EOF | cw contacts create --json
  {
    "name": "Full Contact",
    "email": "full@test.com",
    "identifier": "ext-123",
    "custom_attributes": {"plan": "enterprise"}
  }
  EOF`,
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			var body map[string]any

			// If --json flag is set, read from stdin
			if fromStdin {
				stdinData, err := readJSONFromStdin()
				if err != nil {
					return err
				}
				body = stdinData
			} else {
				body = make(map[string]any)
			}

			// CLI flags override stdin JSON values
			if name != "" {
				body["name"] = name
			}
			if email != "" {
				body["email"] = email
			}
			if phone != "" {
				body["phone_number"] = phone
			}

			// Get the final name value for validation
			finalName, _ := body["name"].(string)
			if finalName == "" {
				return fmt.Errorf("--name is required")
			}

			// Get values for validation
			finalEmail, _ := body["email"].(string)
			finalPhone, _ := body["phone_number"].(string)

			// Validate input lengths
			if err := validation.ValidateName(finalName); err != nil {
				return err
			}
			if err := validation.ValidateEmail(finalEmail); err != nil {
				return err
			}
			if err := validation.ValidatePhone(finalPhone); err != nil {
				return err
			}

			// Validate input formats
			if err := validation.ValidateEmailFormat(finalEmail); err != nil {
				return err
			}
			if err := validation.ValidatePhoneFormat(finalPhone); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			contact, err := client.Contacts().CreateFromMap(cmdContext(cmd), body)
			if err != nil {
				return fmt.Errorf("failed to create contact: %w", err)
			}

			if emitted, err := maybeEmit(cmd, emit, "contact", contact.ID, contact); emitted {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, contact)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tPHONE")
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
				contact.ID,
				displayContactName(contact.Name),
				strings.TrimSpace(contact.Email),
				strings.TrimSpace(contact.PhoneNumber),
			)

			return nil
		}),
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Contact name (required unless provided via --json)")
	cmd.Flags().StringVarP(&email, "email", "e", "", "Contact email address")
	cmd.Flags().StringVar(&phone, "phone", "", "Contact phone number")
	cmd.Flags().BoolVar(&fromStdin, "json", false, "Read contact data from stdin as JSON")
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")

	return cmd
}

func newContactsUpdateCmd() *cobra.Command {
	var (
		name  string
		email string
		phone string
		emit  string
	)

	cmd := &cobra.Command{
		Use:     "update <identifier>",
		Aliases: []string{"up"},
		Short:   "Update a contact",
		Long: `Update a contact's name, email, and/or phone number.

Accepts numeric ID, Chatwoot URL, email address, name, or phone number to resolve the contact.`,
		Example: `  # Update by ID
  cw contacts update 123 --name "Updated Name"

  # Update by email
  cw contacts update john@example.com --name "John Smith"

  # Update by phone number (if searchable)
  cw contacts update +16042091231 --name "Wenqi Qu" --email "quwenqi@example.com"`,
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {

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

			ctx := cmdContext(cmd)
			id, err := resolveContactID(ctx, client, args[0])
			if err != nil {
				return err
			}

			contact, err := client.Contacts().Update(ctx, id, name, email, phone)
			if err != nil {
				return fmt.Errorf("failed to update contact %d: %w", id, err)
			}

			if emitted, err := maybeEmit(cmd, emit, "contact", contact.ID, contact); emitted {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, contact)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tPHONE")
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
				contact.ID,
				displayContactName(contact.Name),
				strings.TrimSpace(contact.Email),
				strings.TrimSpace(contact.PhoneNumber),
			)

			return nil
		}),
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "New contact name")
	cmd.Flags().StringVarP(&email, "email", "e", "", "New contact email address")
	cmd.Flags().StringVar(&phone, "phone", "", "New contact phone number")
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")

	return cmd
}

func newContactsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm"},
		Short:   "Delete a contact",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "contact")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.Contacts().Delete(cmdContext(cmd), id); err != nil {
				return fmt.Errorf("failed to delete contact %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": id})
			}
			printAction(cmd, "Deleted", "contact", id, "")
			return nil
		}),
	}
}

func newContactsSearchCmd() *cobra.Command {
	var query string

	cmd := &cobra.Command{
		Use:     "search",
		Aliases: []string{"q"},
		Short:   "Search contacts",
		Long: `Search for contacts by query string.

The query matches against contact name, email, phone number, and identifier.
JSON output returns an object with an "items" array for easy jq processing.`,
		Example: `  # Search for contacts by name
  cw contacts search --query "John"

  # Search and output as JSON
  cw contacts search --query "acme" --output json | jq '.items[] | {id, name}'`,
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if query == "" {
				return fmt.Errorf("--query is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			contacts, err := client.Contacts().Search(cmdContext(cmd), query, 1)
			if err != nil {
				return fmt.Errorf("failed to search contacts: %w", err)
			}

			if isAgent(cmd) {
				payload := agentfmt.SearchEnvelope{
					Kind:    agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Query:   query,
					Results: agentfmt.ContactSummaries(contacts.Payload),
					Summary: map[string]int{"contacts": len(contacts.Payload)},
				}
				return printJSON(cmd, payload)
			}
			if isJSON(cmd) {
				return printJSON(cmd, contacts.Payload)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tPHONE\tCREATED")
			for _, contact := range contacts.Payload {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
					contact.ID,
					displayContactName(contact.Name),
					strings.TrimSpace(contact.Email),
					strings.TrimSpace(contact.PhoneNumber),
					formatTimestampShort(contact.CreatedAtTime()),
				)
			}

			return nil
		}),
	}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name", "email"},
		"default": {"id", "name", "email", "phone_number", "identifier", "created_at"},
		"debug":   {"id", "name", "email", "phone_number", "identifier", "thumbnail", "custom_attributes", "created_at", "last_activity_at"},
	})
	registerFieldSchema(cmd, "contact")

	cmd.Flags().StringVar(&query, "query", "", "Search query string")

	return cmd
}

func newContactsFilterCmd() *cobra.Command {
	var payload string

	cmd := &cobra.Command{
		Use:     "filter",
		Aliases: []string{"f"},
		Short:   "Filter contacts",
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
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
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

			contacts, err := client.Contacts().Filter(cmdContext(cmd), filterPayload)
			if err != nil {
				return fmt.Errorf("failed to filter contacts: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, contacts.Payload)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tPHONE\tCREATED")
			for _, contact := range contacts.Payload {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
					contact.ID,
					contact.Name,
					contact.Email,
					contact.PhoneNumber,
					formatTimestampShort(contact.CreatedAtTime()),
				)
			}

			return nil
		}),
	}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name", "email"},
		"default": {"id", "name", "email", "phone_number", "identifier", "created_at"},
		"debug":   {"id", "name", "email", "phone_number", "identifier", "thumbnail", "custom_attributes", "created_at", "last_activity_at"},
	})
	registerFieldSchema(cmd, "contact")

	cmd.Flags().StringVar(&payload, "payload", "", "JSON array of filter conditions")

	return cmd
}

func newContactsConversationsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "conversations <identifier>",
		Short: "Get contact conversations",
		Long: `Get all conversations for a specific contact.

Accepts contact ID, email address, phone number, or name to search for the contact.`,
		Example: `  # Get conversations by contact ID
  cw contacts conversations 123

  # Get conversations by email
  cw contacts conversations john@example.com

  # Get conversations by name
  cw contacts conversations "John Smith"`,
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)
			id, err := resolveContactID(ctx, client, args[0])
			if err != nil {
				return err
			}

			conversations, err := client.Contacts().Conversations(ctx, id)
			if err != nil {
				return fmt.Errorf("failed to get conversations for contact %d: %w", id, err)
			}

			if isAgent(cmd) {
				summaries := agentfmt.ConversationSummaries(conversations)
				summaries = resolveConversationSummaries(cmdContext(cmd), client, summaries)
				payload := agentfmt.ListEnvelope{
					Kind:  agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Items: summaries,
					Meta: map[string]any{
						"contact_id":  id,
						"total_items": len(summaries),
					},
				}
				return printJSON(cmd, payload)
			}
			if isJSON(cmd) {
				return printJSON(cmd, conversations)
			}

			w := newTabWriterFromCmd(cmd)
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
		}),
	}
}

func newContactsLabelsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "labels <id>",
		Short: "Get contact labels",
		Long:  "Get all labels for a specific contact",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "contact")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			labels, err := client.Contacts().Labels(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get labels for contact %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, labels)
			}

			if len(labels) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No labels found")
				return nil
			}

			for _, label := range labels {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), label)
			}

			return nil
		}),
	}
}

func newContactsLabelsAddCmd() *cobra.Command {
	var labels string

	cmd := &cobra.Command{
		Use:     "labels-add <id>",
		Aliases: []string{"la"},
		Short:   "Add labels to contact",
		Long:    "Add one or more labels to a contact (comma-separated)",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "contact")
			if err != nil {
				return err
			}

			if labels == "" {
				return fmt.Errorf("--labels is required")
			}

			labelList, err := ParseStringListFlag(labels)
			if err != nil {
				return fmt.Errorf("invalid labels: %w", err)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			updatedLabels, err := client.Contacts().AddLabels(cmdContext(cmd), id, labelList)
			if err != nil {
				return fmt.Errorf("failed to add labels to contact %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, updatedLabels)
			}

			printAction(cmd, "Added labels to", "contact", id, "")
			for _, label := range updatedLabels {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), label)
			}

			return nil
		}),
	}

	cmd.Flags().StringVar(&labels, "labels", "", "Labels (CSV, whitespace, JSON array; or @- / @path)")

	return cmd
}

func newContactsContactableInboxesCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "contactable-inboxes <id>",
		Aliases: []string{"ci"},
		Short:   "Get contactable inboxes",
		Long:    "Get all contactable inboxes for a contact",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "contact")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			inboxes, err := client.Contacts().ContactableInboxes(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get contactable inboxes for contact %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, inboxes)
			}

			if len(inboxes) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No contactable inboxes found")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
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
		}),
	}
}

func newContactsCreateInboxCmd() *cobra.Command {
	var (
		inboxID  int
		sourceID string
	)

	cmd := &cobra.Command{
		Use:     "create-inbox <contact-id>",
		Aliases: []string{"cri"},
		Short:   "Associate contact with an inbox",
		Long:    "Create a contact inbox association, allowing the contact to be reached via that inbox",
		Example: `  # Associate contact 123 with inbox 1
  cw contacts create-inbox 123 --inbox-id 1

  # With custom source ID (for channel-specific identifiers)
  cw contacts create-inbox 123 --inbox-id 1 --source-id "+15551234567"`,
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			contactID, err := parseIDOrURL(args[0], "contact")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}
			if inboxID == 0 {
				if isInteractive() {
					selected, err := promptInboxID(cmdContext(cmd), client)
					if err != nil {
						return err
					}
					inboxID = selected
				} else {
					return fmt.Errorf("--inbox-id is required")
				}
			}

			result, err := client.Contacts().CreateInbox(cmdContext(cmd), contactID, inboxID, sourceID)
			if err != nil {
				return fmt.Errorf("failed to create contact inbox: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, result)
			}

			if result.Inbox.ID == 0 {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Contact %d associated with inbox (no details returned)\n", contactID)
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Contact %d associated with inbox %d\n", contactID, result.Inbox.ID)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Inbox: %s (%s)\n", result.Inbox.Name, result.Inbox.ChannelType)
			}
			if result.SourceID != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Source ID: %s\n", result.SourceID)
			}

			return nil
		}),
	}

	cmd.Flags().IntVar(&inboxID, "inbox-id", 0, "Inbox ID (required)")
	cmd.Flags().StringVar(&sourceID, "source-id", "", "Channel-specific source identifier")
	flagAlias(cmd.Flags(), "inbox-id", "iid")

	return cmd
}

func newContactsNotesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "notes <contact-id>",
		Short: "List contact notes",
		Long:  "List all notes for a specific contact",
		Example: strings.TrimSpace(`
  # List notes for a contact
  cw contacts notes 123

  # JSON output
  cw contacts notes 123 -o json
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "contact")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			notes, err := client.Contacts().Notes(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get notes for contact %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, notes)
			}

			if len(notes) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No notes found")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
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
		}),
	}
}

func newContactsNotesAddCmd() *cobra.Command {
	var content string

	cmd := &cobra.Command{
		Use:     "notes-add <contact-id>",
		Aliases: []string{"na"},
		Short:   "Add note to contact",
		Long:    "Add a new note to a contact",
		Example: strings.TrimSpace(`
  # Add a note to a contact
  cw contacts notes-add 123 --content "VIP customer, handle with care"
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "contact")
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

			note, err := client.Contacts().CreateNote(cmdContext(cmd), id, content)
			if err != nil {
				return fmt.Errorf("failed to add note to contact %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, note)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added note #%d to contact %d\n", note.ID, id)
			return nil
		}),
	}

	cmd.Flags().StringVar(&content, "content", "", "Note content (required)")

	return cmd
}

func newContactsNotesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "notes-delete <contact-id> <note-id>",
		Aliases: []string{"nd"},
		Short:   "Delete contact note",
		Long:    "Delete a note from a contact",
		Example: strings.TrimSpace(`
  # Delete a note from a contact
  cw contacts notes-delete 123 456
`),
		Args: cobra.ExactArgs(2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			contactID, err := parseIDOrURL(args[0], "contact")
			if err != nil {
				return err
			}

			noteID, err := parsePositiveIntArg(args[1], "note ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.Contacts().DeleteNote(cmdContext(cmd), contactID, noteID); err != nil {
				return fmt.Errorf("failed to delete note %d from contact %d: %w", noteID, contactID, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": noteID, "contact_id": contactID})
			}
			printAction(cmd, "Deleted", "contact note", noteID, fmt.Sprintf("contact %d", contactID))
			return nil
		}),
	}
}

func newContactsBulkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bulk",
		Short: "Bulk operations on contacts",
		Long:  "Perform bulk operations on multiple contacts at once",
	}

	cmd.AddCommand(newContactsBulkAddLabelCmd())
	cmd.AddCommand(newContactsBulkRemoveLabelCmd())

	return cmd
}

func newContactsBulkAddLabelCmd() *cobra.Command {
	var (
		contactIDs  string
		labels      string
		concurrency int
		progress    bool
		noProgress  bool
	)

	cmd := &cobra.Command{
		Use:     "add-label",
		Aliases: []string{"al"},
		Short:   "Add labels to multiple contacts",
		Long:    "Add one or more labels to multiple contacts at once",
		Example: strings.TrimSpace(`
  # Add a single label to multiple contacts
  cw contacts bulk add-label --ids 1,2,3 --labels important

  # Add multiple labels to multiple contacts
  cw contacts bulk add-label --ids 1,2,3 --labels important,vip

  # Control concurrency (default: 5)
  cw contacts bulk add-label --ids 1,2,3 --labels vip --concurrency 10
`),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			ids, err := ParseResourceIDListFlag(contactIDs, "contact")
			if err != nil {
				return fmt.Errorf("invalid contact IDs: %w", err)
			}

			labelList, err := ParseStringListFlag(labels)
			if err != nil {
				return fmt.Errorf("invalid labels: %w", err)
			}
			if len(labelList) == 0 {
				return fmt.Errorf("no valid labels provided")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			results := runBulkOperation(
				ctx,
				ids,
				int64(concurrency),
				bulkProgressEnabled(cmd, progress, noProgress),
				cmd.ErrOrStderr(),
				func(ctx context.Context, id int) (any, error) {
					_, err := client.Contacts().AddLabels(ctx, id, labelList)
					if err != nil {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Failed to add labels to contact %d: %v\n", id, err)
						return nil, err
					}
					return nil, nil
				},
			)

			successCount, failCount := countResults(results)

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{
					"success_count": successCount,
					"fail_count":    failCount,
				})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added labels to %d contacts (%d failed)\n", successCount, failCount)
			return nil
		}),
	}

	cmd.Flags().StringVar(&contactIDs, "ids", "", "Contact IDs (CSV, whitespace, JSON array; or @- / @path) (required)")
	cmd.Flags().StringVar(&labels, "labels", "", "Labels to add (CSV, whitespace, JSON array; or @- / @path) (required)")
	cmd.Flags().IntVar(&concurrency, "concurrency", DefaultConcurrency, "Max concurrent operations")
	cmd.Flags().BoolVar(&progress, "progress", true, "Show progress while running")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "Disable progress output")
	flagAlias(cmd.Flags(), "concurrency", "cc")
	_ = cmd.MarkFlagRequired("ids")
	_ = cmd.MarkFlagRequired("labels")

	return cmd
}

func newContactsBulkRemoveLabelCmd() *cobra.Command {
	var (
		contactIDs  string
		labels      string
		concurrency int
		progress    bool
		noProgress  bool
	)

	cmd := &cobra.Command{
		Use:     "remove-label",
		Aliases: []string{"rl"},
		Short:   "Remove labels from multiple contacts",
		Long: `Remove one or more labels from multiple contacts at once.

For each contact, this command fetches current labels, removes the specified
labels, and updates the contact with the remaining labels.`,
		Example: strings.TrimSpace(`
  # Remove a single label from multiple contacts
  cw contacts bulk remove-label --ids 1,2,3 --labels spam

  # Remove multiple labels from multiple contacts
  cw contacts bulk remove-label --ids 1,2,3 --labels spam,inactive

  # Control concurrency (default: 5)
  cw contacts bulk remove-label --ids 1,2,3 --labels spam --concurrency 10
`),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			ids, err := ParseResourceIDListFlag(contactIDs, "contact")
			if err != nil {
				return fmt.Errorf("invalid contact IDs: %w", err)
			}

			labelsToRemove := make(map[string]bool)
			labelList, err := ParseStringListFlag(labels)
			if err != nil {
				return fmt.Errorf("invalid labels: %w", err)
			}
			for _, l := range labelList {
				labelsToRemove[l] = true
			}
			if len(labelsToRemove) == 0 {
				return fmt.Errorf("no valid labels provided")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			results := runBulkOperation(
				ctx,
				ids,
				int64(concurrency),
				bulkProgressEnabled(cmd, progress, noProgress),
				cmd.ErrOrStderr(),
				func(ctx context.Context, id int) (any, error) {
					// Get current labels
					currentLabels, err := client.Contacts().Labels(ctx, id)
					if err != nil {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Failed to get labels for contact %d: %v\n", id, err)
						return nil, err
					}

					// Filter out labels to remove
					var remainingLabels []string
					for _, label := range currentLabels {
						if !labelsToRemove[label] {
							remainingLabels = append(remainingLabels, label)
						}
					}

					// Update with remaining labels (API replaces all labels)
					_, err = client.Contacts().AddLabels(ctx, id, remainingLabels)
					if err != nil {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Failed to update labels for contact %d: %v\n", id, err)
						return nil, err
					}
					return nil, nil
				},
			)

			successCount, failCount := countResults(results)

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{
					"success_count": successCount,
					"fail_count":    failCount,
				})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed labels from %d contacts (%d failed)\n", successCount, failCount)
			return nil
		}),
	}

	cmd.Flags().StringVar(&contactIDs, "ids", "", "Contact IDs (CSV, whitespace, JSON array; or @- / @path) (required)")
	cmd.Flags().StringVar(&labels, "labels", "", "Labels to remove (CSV, whitespace, JSON array; or @- / @path) (required)")
	cmd.Flags().IntVar(&concurrency, "concurrency", DefaultConcurrency, "Max concurrent operations")
	cmd.Flags().BoolVar(&progress, "progress", true, "Show progress while running")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "Disable progress output")
	flagAlias(cmd.Flags(), "concurrency", "cc")
	_ = cmd.MarkFlagRequired("ids")
	_ = cmd.MarkFlagRequired("labels")

	return cmd
}

func newContactsMergeCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "merge <keep-id> <delete-id>",
		Short: "Merge two contacts",
		Long: `Merge two contacts into one.

The first argument (keep-id) is the contact that SURVIVES and receives all data.
The second argument (delete-id) is the contact that gets PERMANENTLY DELETED.

All conversations, messages, notes, and other data from the deleted contact
will be transferred to the surviving contact.

This operation is IRREVERSIBLE. The deleted contact cannot be recovered.`,
		Example: `  # Merge contact 456 INTO contact 123 (456 gets deleted, 123 keeps all data)
  cw contacts merge 123 456

  # Preview the merge without executing
  cw contacts merge 123 456 --dry-run

  # Skip confirmation (for scripting)
  cw contacts merge 123 456 --force

  # JSON output (requires --force)
  cw contacts merge 123 456 --force --output json`,
		Args: cobra.ExactArgs(2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			keepID, err := parsePositiveIntArg(args[0], "keep-id")
			if err != nil {
				return err
			}

			deleteID, err := parsePositiveIntArg(args[1], "delete-id")
			if err != nil {
				return err
			}

			if keepID == deleteID {
				return fmt.Errorf("cannot merge contact with itself: both IDs are %d", keepID)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			// Handle dry-run mode - preview merge without executing
			if dryrun.IsEnabled(ctx) {
				return printMergeDryRun(cmd, client, keepID, deleteID)
			}

			if err := requireForceForJSON(cmd, force); err != nil {
				return err
			}

			// If not forced, fetch both contacts and prompt for confirmation
			if !force {
				// Fetch keep contact
				keepContact, err := client.Contacts().Get(ctx, keepID)
				if err != nil {
					return fmt.Errorf("failed to get keep contact %d: %w", keepID, err)
				}

				// Fetch delete contact
				deleteContact, err := client.Contacts().Get(ctx, deleteID)
				if err != nil {
					return fmt.Errorf("failed to get delete contact %d: %w", deleteID, err)
				}

				// Display merge preview
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), bold("MERGE CONTACTS"))
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s #%d %s\n", green("KEEP (base):    "), keepContact.ID, formatContactSummary(keepContact))
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s #%d %s\n", red("DELETE (mergee):"), deleteContact.ID, formatContactSummary(deleteContact))
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "The contact #%d will be %s.\n", deleteID, red("PERMANENTLY DELETED"))
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "All conversations, messages, and notes will be transferred to #%d.\n", keepID)
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
				ok, err := confirmAction(cmd, confirmOptions{
					Prompt:              yellow("Type 'merge' to confirm: "),
					Expected:            "merge",
					CancelMessage:       "Merge cancelled.",
					Force:               force,
					RequireForceForJSON: true,
				})
				if err != nil {
					return err
				}
				if !ok {
					return nil
				}
			}

			// Perform the merge
			mergedContact, err := client.Contacts().Merge(ctx, keepID, deleteID)
			if err != nil {
				return fmt.Errorf("failed to merge contacts: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, mergedContact)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Successfully merged contact #%d into #%d\n", deleteID, keepID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Contact #%d has been deleted. Contact #%d now contains all data.\n", deleteID, keepID)

			return nil
		}),
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt (required for --output json)")

	return cmd
}

// formatContactSummary formats a contact for display in the merge confirmation
func formatContactSummary(c *api.Contact) string {
	var parts []string

	if c.Name != "" {
		parts = append(parts, fmt.Sprintf("%q", c.Name))
	}
	if c.Email != "" {
		parts = append(parts, fmt.Sprintf("<%s>", c.Email))
	}
	if c.PhoneNumber != "" {
		parts = append(parts, c.PhoneNumber)
	}
	if c.Identifier != "" {
		parts = append(parts, fmt.Sprintf("[id:%s]", c.Identifier))
	}

	if len(parts) == 0 {
		return "(no details)"
	}

	summary := strings.Join(parts, " ")

	// Add last activity if available
	if c.LastActivityAt != nil && *c.LastActivityAt > 0 {
		lastActivity := time.Unix(*c.LastActivityAt, 0)
		summary += fmt.Sprintf(" (last active: %s)", formatDate(lastActivity))
	}

	return summary
}

// printMergeDryRun displays a preview of the merge operation without executing it
func printMergeDryRun(cmd *cobra.Command, client *api.Client, keepID, deleteID int) error {
	ctx := cmdContext(cmd)
	out := cmd.OutOrStdout()

	// Fetch both contacts
	targetContact, err := client.Contacts().Get(ctx, keepID)
	if err != nil {
		return fmt.Errorf("failed to get target contact %d: %w", keepID, err)
	}

	sourceContact, err := client.Contacts().Get(ctx, deleteID)
	if err != nil {
		return fmt.Errorf("failed to get source contact %d: %w", deleteID, err)
	}

	// Fetch conversations for both contacts
	targetConvs, err := client.Contacts().Conversations(ctx, keepID)
	if err != nil {
		return fmt.Errorf("failed to get conversations for target contact %d: %w", keepID, err)
	}

	sourceConvs, err := client.Contacts().Conversations(ctx, deleteID)
	if err != nil {
		return fmt.Errorf("failed to get conversations for source contact %d: %w", deleteID, err)
	}

	// Calculate conversation counts
	targetOpen := countByStatus(targetConvs, "open")
	targetResolved := countByStatus(targetConvs, "resolved")
	sourceOpen := countByStatus(sourceConvs, "open")
	sourceResolved := countByStatus(sourceConvs, "resolved")

	// Determine merge result
	mergedName := targetContact.Name
	if mergedName == "" {
		mergedName = sourceContact.Name
	}

	mergedEmail := targetContact.Email
	emailFromSource := false
	if mergedEmail == "" && sourceContact.Email != "" {
		mergedEmail = sourceContact.Email
		emailFromSource = true
	}

	mergedPhone := targetContact.PhoneNumber
	phoneFromSource := false
	if mergedPhone == "" && sourceContact.PhoneNumber != "" {
		mergedPhone = sourceContact.PhoneNumber
		phoneFromSource = true
	}

	// Check for conflicts (data will be lost)
	var conflicts []string
	if targetContact.Email != "" && sourceContact.Email != "" && targetContact.Email != sourceContact.Email {
		conflicts = append(conflicts, fmt.Sprintf("Email: target has %q, source has %q (source email will be lost)",
			targetContact.Email, sourceContact.Email))
	}
	if targetContact.PhoneNumber != "" && sourceContact.PhoneNumber != "" && targetContact.PhoneNumber != sourceContact.PhoneNumber {
		conflicts = append(conflicts, fmt.Sprintf("Phone: target has %q, source has %q (source phone will be lost)",
			targetContact.PhoneNumber, sourceContact.PhoneNumber))
	}

	// JSON output
	if isJSON(cmd) {
		result := map[string]any{
			"dry_run": true,
			"source": map[string]any{
				"id":                   sourceContact.ID,
				"name":                 sourceContact.Name,
				"email":                sourceContact.Email,
				"phone_number":         sourceContact.PhoneNumber,
				"conversations_open":   sourceOpen,
				"conversations_closed": sourceResolved,
			},
			"target": map[string]any{
				"id":                   targetContact.ID,
				"name":                 targetContact.Name,
				"email":                targetContact.Email,
				"phone_number":         targetContact.PhoneNumber,
				"conversations_open":   targetOpen,
				"conversations_closed": targetResolved,
			},
			"after_merge": map[string]any{
				"name":                mergedName,
				"email":               mergedEmail,
				"phone_number":        mergedPhone,
				"total_conversations": len(targetConvs) + len(sourceConvs),
			},
			"conflicts": conflicts,
		}
		return printJSON(cmd, result)
	}

	// Text output
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, yellow("DRY RUN - Contacts will NOT be merged"))
	_, _ = fmt.Fprintln(out)

	// SOURCE section
	_, _ = fmt.Fprintln(out, bold("SOURCE (will be deleted):"))
	_, _ = fmt.Fprintf(out, "  ID:            #%d\n", sourceContact.ID)
	_, _ = fmt.Fprintf(out, "  Name:          %s\n", valueOrNone(sourceContact.Name))
	_, _ = fmt.Fprintf(out, "  Email:         %s\n", valueOrNone(sourceContact.Email))
	_, _ = fmt.Fprintf(out, "  Phone:         %s\n", valueOrNone(sourceContact.PhoneNumber))
	_, _ = fmt.Fprintf(out, "  Conversations: %d open, %d resolved\n", sourceOpen, sourceResolved)
	_, _ = fmt.Fprintln(out)

	// TARGET section
	_, _ = fmt.Fprintln(out, bold("TARGET (will be kept):"))
	_, _ = fmt.Fprintf(out, "  ID:            #%d\n", targetContact.ID)
	_, _ = fmt.Fprintf(out, "  Name:          %s\n", valueOrNone(targetContact.Name))
	_, _ = fmt.Fprintf(out, "  Email:         %s\n", valueOrNone(targetContact.Email))
	_, _ = fmt.Fprintf(out, "  Phone:         %s\n", valueOrNone(targetContact.PhoneNumber))
	_, _ = fmt.Fprintf(out, "  Conversations: %d open, %d resolved\n", targetOpen, targetResolved)
	_, _ = fmt.Fprintln(out)

	// AFTER MERGE section
	_, _ = fmt.Fprintln(out, bold("AFTER MERGE:"))
	_, _ = fmt.Fprintf(out, "  Name:          %s\n", valueOrNone(mergedName))
	emailNote := ""
	if emailFromSource {
		emailNote = " (from source)"
	}
	_, _ = fmt.Fprintf(out, "  Email:         %s%s\n", valueOrNone(mergedEmail), emailNote)
	phoneNote := ""
	if phoneFromSource {
		phoneNote = " (from source)"
	}
	_, _ = fmt.Fprintf(out, "  Phone:         %s%s\n", valueOrNone(mergedPhone), phoneNote)
	_, _ = fmt.Fprintf(out, "  Conversations: %d total\n", len(targetConvs)+len(sourceConvs))
	_, _ = fmt.Fprintln(out)

	// CONFLICTS section
	if len(conflicts) > 0 {
		_, _ = fmt.Fprintln(out, red("CONFLICTS (data will be lost):"))
		for _, c := range conflicts {
			_, _ = fmt.Fprintf(out, "  - %s\n", c)
		}
		_, _ = fmt.Fprintln(out)
	}

	_, _ = fmt.Fprintln(out, "To merge these contacts, run without --dry-run")

	return nil
}

// countByStatus counts conversations with the given status
func countByStatus(convs []api.Conversation, status string) int {
	count := 0
	for _, c := range convs {
		if c.Status == status {
			count++
		}
	}
	return count
}

// valueOrNone returns the value if non-empty, otherwise "(none)"
func valueOrNone(s string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return "(none)"
	}
	return trimmed
}

func displayContactName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "(none)"
	}
	return trimmed
}
