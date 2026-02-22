package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newContactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "contacts",
		Aliases: []string{"contact", "customers", "co"},
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
	var sort string
	var order string
	var light bool

	cfg := ListConfig[api.Contact]{
		Use:             "list",
		Short:           "List all contacts",
		StripPagination: true,
		Long: `List all contacts in your Chatwoot account.

JSON output returns an object with an "items" array for easy jq processing.`,
		Example: `  # List contacts in table format
  cw contacts list

  # List with pagination
  cw contacts list --page 2

  # JSON output - returns an object with an "items" array
  cw contacts list --output json | jq '.items[0]'
  cw contacts list --output json | jq '.items[] | {id, name, email}'`,
		DisableLimit: true,
		Fetch: func(ctx context.Context, client *api.Client, page, _ int) (ListResult[api.Contact], error) {
			sortField, sortOrder, err := parseSortOrder(sort, order)
			if err != nil {
				return ListResult[api.Contact]{}, err
			}

			contacts, err := client.Contacts().List(ctx, api.ListContactsParams{
				Page:  page,
				Sort:  sortField,
				Order: sortOrder,
			})
			if err != nil {
				return ListResult[api.Contact]{}, fmt.Errorf("failed to list contacts: %w", err)
			}

			return ListResult[api.Contact]{
				Items:   contacts.Payload,
				HasMore: contactsMetaHasMore(contacts.Meta),
			}, nil
		},
		Headers: []string{"ID", "NAME", "EMAIL", "PHONE", "CREATED"},
		RowFunc: func(contact api.Contact) []string {
			return []string{
				fmt.Sprintf("%d", contact.ID),
				displayContactName(contact.Name),
				strings.TrimSpace(contact.Email),
				strings.TrimSpace(contact.PhoneNumber),
				formatTimestampShort(contact.CreatedAtTime()),
			}
		},
		JSONTransform: func(ctx context.Context, _ *api.Client, items []api.Contact) (any, error) {
			if !outfmt.IsLight(ctx) {
				return items, nil
			}
			return buildLightContacts(items), nil
		},
		ForceJSON: func(_ *cobra.Command) bool {
			return light
		},
		ForceJSONUnwrapItems: true,
	}

	cmd := NewListCommand(cfg, func(ctx context.Context) (*api.Client, error) {
		return getClient()
	})
	cmd.Aliases = []string{"ls"}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name", "email"},
		"default": {"id", "name", "email", "phone_number", "identifier", "created_at"},
		"debug":   {"id", "name", "email", "phone_number", "identifier", "thumbnail", "custom_attributes", "created_at", "last_activity_at"},
	})
	registerFieldSchema(cmd, "contact")

	cmd.Flags().StringVar(&sort, "sort", "", "Sort by field (name|email|phone_number|last_activity_at; aliases: n|e|pn|la); prefix with '-' for desc")
	cmd.Flags().StringVar(&order, "order", "", "Sort order (asc|desc); overrides '-' prefix")
	cmd.Flags().BoolVar(&light, "light", false, "Return minimal contact payload")
	flagAlias(cmd.Flags(), "light", "li")
	flagAlias(cmd.Flags(), "sort", "so")
	flagAlias(cmd.Flags(), "order", "ord")
	_ = cmd.Flags().MarkHidden("limit")
	_ = cmd.Flags().MarkHidden("all")
	_ = cmd.Flags().MarkHidden("max-pages")

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

	// Light mode: minimal payload with open/pending conversations
	light, _ := cmd.Flags().GetBool("light")
	if light {
		cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
		lc := buildLightContact(contact)

		// Fetch open/pending conversations with last message
		convs, err := client.Contacts().Conversations(ctx, id)
		if err == nil {
			for _, conv := range convs {
				if conv.Status != "open" && conv.Status != "pending" {
					continue
				}
				lastMsg := extractLastNonActivityMessage(conv)
				lc.Convs = append(lc.Convs, buildLightContactConversation(conv, lastMsg))
			}
		}
		if lc.Convs == nil {
			lc.Convs = []lightContactConv{}
		}

		return printRawJSON(cmd, lc)
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
	flagAlias(cmd.Flags(), "with-open-conversations", "woc")
	cmd.Flags().Bool("light", false, "Return minimal contact payload with active conversations")
	flagAlias(cmd.Flags(), "light", "li")

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
	flagAlias(cmd.Flags(), "with-open-conversations", "woc")
	cmd.Flags().Bool("light", false, "Return minimal contact payload with active conversations")
	flagAlias(cmd.Flags(), "light", "li")

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
	flagAlias(cmd.Flags(), "phone", "ph")

	return cmd
}

func newContactsUpdateCmd() *cobra.Command {
	var (
		name        string
		email       string
		phone       string
		company     string
		country     string
		customAttrs []string
		social      []string
		emit        string
	)

	cmd := &cobra.Command{
		Use:     "update <identifier>",
		Aliases: []string{"up"},
		Short:   "Update a contact",
		Long: `Update a contact's name, email, phone, company, country, custom attributes, and/or social profiles.

Accepts numeric ID, Chatwoot URL, email address, name, or phone number to resolve the contact.

Custom attributes are set via repeatable --custom-attr/-A flags with key=value format.
Social profiles are set via repeatable --social/-S flags with platform=url format.`,
		Example: `  # Update by ID
  cw contacts update 123 --name "Updated Name"

  # Update by email
  cw contacts update john@example.com --name "John Smith"

  # Update by phone number (if searchable)
  cw contacts update +16042091231 --name "Wenqi Qu" --email "quwenqi@example.com"

  # Set company and country
  cw contacts update 123 --company "Acme Corp" --country "Canada"

  # Set custom attributes
  cw contacts update 123 -A plan=enterprise -A region=APAC

  # Set social profiles
  cw contacts update 123 -S twitter=https://twitter.com/acme -S linkedin=https://linkedin.com/company/acme`,
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if name == "" && email == "" && phone == "" && company == "" && country == "" && len(customAttrs) == 0 && len(social) == 0 {
				return fmt.Errorf("at least one of --name, --email, --phone, --company, --country, --custom-attr, or --social must be provided")
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

			// Parse custom attributes
			customAttrMap := make(map[string]any)
			for _, attr := range customAttrs {
				key, value, found := strings.Cut(attr, "=")
				if !found {
					return fmt.Errorf("invalid custom-attr format %q, expected key=value", attr)
				}
				customAttrMap[key] = value
			}

			// Parse social profiles
			socialMap := make(map[string]string)
			for _, s := range social {
				platform, url, found := strings.Cut(s, "=")
				if !found {
					return fmt.Errorf("invalid social format %q, expected platform=url", s)
				}
				socialMap[platform] = url
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

			opts := api.UpdateContactOpts{
				Name:    name,
				Email:   email,
				Phone:   phone,
				Company: company,
				Country: country,
			}
			if len(customAttrMap) > 0 {
				opts.CustomAttributes = customAttrMap
			}
			if len(socialMap) > 0 {
				opts.SocialProfiles = socialMap
			}

			contact, err := client.Contacts().UpdateWithOpts(ctx, id, opts)
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
	cmd.Flags().StringVarP(&company, "company", "C", "", "Company name")
	cmd.Flags().StringVarP(&country, "country", "K", "", "Country name (e.g. Taiwan, Canada)")
	cmd.Flags().StringSliceVarP(&customAttrs, "custom-attr", "A", nil, "Custom attribute key=value (repeatable)")
	cmd.Flags().StringSliceVarP(&social, "social", "S", nil, "Social profile platform=url (repeatable)")
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")
	flagAlias(cmd.Flags(), "phone", "ph")

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
	flagAlias(cmd.Flags(), "query", "q")

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
	flagAlias(cmd.Flags(), "payload", "pl")

	return cmd
}

func newContactsConversationsCmd() *cobra.Command {
	var light bool

	cmd := &cobra.Command{
		Use:     "conversations <identifier>",
		Aliases: []string{"cv"},
		Short:   "Get contact conversations",
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
			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, buildLightConversationLookups(conversations))
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

	cmd.Flags().BoolVar(&light, "light", false, "Return minimal conversation payload for lookup")
	flagAlias(cmd.Flags(), "light", "li")

	return cmd
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
	flagAlias(cmd.Flags(), "labels", "lb")

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

			contactInboxes, err := client.Contacts().ContactableInboxes(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get contactable inboxes for contact %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, contactInboxes)
			}

			if len(contactInboxes) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No contactable inboxes found")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tCHANNEL_TYPE\tSOURCE_ID")
			for _, contactInbox := range contactInboxes {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
					contactInbox.Inbox.ID,
					contactInbox.Inbox.Name,
					contactInbox.Inbox.ChannelType,
					contactInbox.SourceID,
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

	cmd.Flags().IntVarP(&inboxID, "inbox-id", "I", 0, "Inbox ID (required)")
	cmd.Flags().StringVar(&sourceID, "source-id", "", "Channel-specific source identifier")
	flagAlias(cmd.Flags(), "inbox-id", "iid")
	flagAlias(cmd.Flags(), "source-id", "sid")

	return cmd
}

func contactsMetaHasMore(meta api.PaginationMeta) bool {
	if meta.HasMore != nil {
		return *meta.HasMore
	}
	if int(meta.TotalPages) > 0 && int(meta.CurrentPage) > 0 {
		return int(meta.CurrentPage) < int(meta.TotalPages)
	}
	return false
}

func displayContactName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "(none)"
	}
	return trimmed
}
