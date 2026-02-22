package cmd

import (
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/urlparse"
	"github.com/spf13/cobra"
)

var openResourceAliases = map[string]string{
	"conversation":  "conversation",
	"conversations": "conversation",
	"conv":          "conversation",
	"cv":            "conversation",
	"contact":       "contact",
	"contacts":      "contact",
	"ct":            "contact",
	"inbox":         "inbox",
	"inboxes":       "inbox",
	"ib":            "inbox",
	"team":          "team",
	"teams":         "team",
	"tm":            "team",
	"agent":         "agent",
	"agents":        "agent",
	"ag":            "agent",
	"campaign":      "campaign",
	"campaigns":     "campaign",
	"cp":            "campaign",
}

func normalizeOpenResourceType(input string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(input))
	if normalized == "" {
		return "", fmt.Errorf("resource type cannot be empty")
	}
	if resourceType, ok := openResourceAliases[normalized]; ok {
		return resourceType, nil
	}
	valid := []string{"conversation (cv)", "contact (ct)", "inbox (ib)", "team (tm)", "agent (ag)", "campaign (cp)"}
	return "", fmt.Errorf("invalid resource type %q: must be one of %s", input, strings.Join(valid, ", "))
}

// resolveOpenTarget parses the arguments to the open command and returns the
// resolved resource type, resource ID, and (for URL inputs) the parsed URL.
// It handles five cases:
//  1. open <resource> <id>         — two args, normalize resource type, parse ID
//  2. open <id> --type <resource>  — one arg with --type flag
//  3. open <url>                   — one arg starting with http(s)://
//  4. open <typed-id>              — one arg like "contact:456", infer type from prefix
//  5. open <id>                    — bare ID, default to conversation
func resolveOpenTarget(args []string, resourceTypeFlag string) (resourceType string, resourceID int, parsedFromURL bool, parsed *urlparse.ParsedURL, err error) {
	if len(args) == 2 {
		// open <resource> <id>
		if resourceTypeFlag != "" {
			return "", 0, false, nil, fmt.Errorf("--type cannot be used with <resource> <id> arguments")
		}
		rt, err := normalizeOpenResourceType(args[0])
		if err != nil {
			return "", 0, false, nil, err
		}
		id, err := parseIDOrURL(args[1], rt)
		if err != nil {
			return "", 0, false, nil, err
		}
		return rt, id, false, nil, nil
	}

	if strings.TrimSpace(resourceTypeFlag) != "" {
		// open <id> --type <resource>
		rt, err := normalizeOpenResourceType(resourceTypeFlag)
		if err != nil {
			return "", 0, false, nil, err
		}
		id, err := parseIDOrURL(args[0], rt)
		if err != nil {
			return "", 0, false, nil, err
		}
		return rt, id, false, nil, nil
	}

	if strings.HasPrefix(strings.TrimSpace(args[0]), "http://") || strings.HasPrefix(strings.TrimSpace(args[0]), "https://") {
		// open <url>
		rawURL := strings.TrimSpace(args[0])
		parsedURL, err := urlparse.Parse(rawURL)
		if err != nil {
			return "", 0, false, nil, fmt.Errorf("failed to parse URL: %w", err)
		}
		return "", 0, true, parsedURL, nil
	}

	// open <id> (default to conversation) OR open <typed-id> like "contact:456"
	raw := strings.TrimSpace(args[0])

	// If the input looks like a typed ID (e.g. "contact:456"), infer the resource type
	// and open that resource without requiring --type.
	if !strings.Contains(raw, "://") {
		if prefix, _, ok := strings.Cut(raw, ":"); ok {
			if rt, normErr := normalizeOpenResourceType(prefix); normErr == nil {
				id, err := parseIDOrURL(raw, rt)
				if err != nil {
					return "", 0, false, nil, err
				}
				return rt, id, false, nil, nil
			}
		}
	}

	id, err := parseIDOrURL(raw, "conversation")
	if err != nil {
		// If the input looks like a URL missing a scheme, surface the URL parser error.
		if strings.Contains(raw, "/") {
			if _, urlErr := urlparse.Parse(raw); urlErr != nil {
				return "", 0, false, nil, fmt.Errorf("failed to parse URL: %w", urlErr)
			}
		}
		return "", 0, false, nil, err
	}
	return "conversation", id, false, nil, nil
}

func newOpenCmd() *cobra.Command {
	var resourceTypeFlag string

	cmd := &cobra.Command{
		Use:     "open <url> | open <resource> <id> | open <id> [--type <resource>]",
		Aliases: []string{"get", "show", "o"},
		Short:   "Open a Chatwoot URL or resource ID and display details",
		Long: `Parse a Chatwoot URL (or resource + ID) and display the corresponding resource details.

This command accepts Chatwoot URLs and extracts the resource information,
then fetches and displays the resource just as if you had run the appropriate
get command directly.

If you provide a bare ID (or ID shorthand like "#123" / "conv:123") without a
resource type, it defaults to opening a conversation.

Supported URL formats:
  https://app.chatwoot.com/app/accounts/{account_id}/conversations/{id}
  https://app.chatwoot.com/app/accounts/{account_id}/contacts/{id}
  https://app.chatwoot.com/app/accounts/{account_id}/inboxes/{id}
  https://app.chatwoot.com/app/accounts/{account_id}/teams/{id}
  https://app.chatwoot.com/app/accounts/{account_id}/agents/{id}
  https://app.chatwoot.com/app/accounts/{account_id}/campaigns/{id}

You can also provide a resource type and ID directly:
  cw open contact 456
  cw open 456 --type contact

Or provide a bare ID (defaults to conversation):
  cw open 456`,
		Example: strings.TrimSpace(`
  # Open a conversation URL
  cw open https://app.chatwoot.com/app/accounts/1/conversations/123

  # Open a contact URL
  cw open https://app.chatwoot.com/app/accounts/1/contacts/456

  # Open by resource type + ID
  cw open contact 456

  # Open by bare ID (defaults to conversation)
  cw open 123

  # Open by bare ID with explicit type
  cw open 456 --type contact

  # Open with JSON output
  cw open https://app.chatwoot.com/app/accounts/1/conversations/123 --output json
`),
		Args: cobra.RangeArgs(1, 2),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			resourceType, resourceID, parsedFromURL, parsed, err := resolveOpenTarget(args, resourceTypeFlag)
			if err != nil {
				return err
			}

			// Get the client
			client, err := getClient()
			if err != nil {
				return err
			}

			if !parsedFromURL {
				parsed = &urlparse.ParsedURL{
					AccountID:    client.AccountID,
					ResourceType: resourceType,
					ResourceID:   resourceID,
				}
			}

			// Verify account ID matches (URLs only)
			if parsedFromURL && client.AccountID != parsed.AccountID {
				return fmt.Errorf("URL account ID (%d) does not match authenticated account ID (%d); use 'cw auth login' to switch accounts", parsed.AccountID, client.AccountID)
			}

			// Require resource ID for all resource types
			if !parsed.HasResourceID() {
				return fmt.Errorf("URL must include a resource ID (e.g., /conversations/123)")
			}

			// Dispatch to appropriate resource handler
			ctx := cmdContext(cmd)
			switch parsed.ResourceType {
			case "conversation":
				conv, err := client.Conversations().Get(ctx, parsed.ResourceID)
				if err != nil {
					return fmt.Errorf("failed to get conversation %d: %w", parsed.ResourceID, err)
				}
				if isAgent(cmd) {
					detail := agentfmt.ConversationDetailFromConversation(*conv)
					detail = resolveConversationDetail(ctx, client, detail)
					payload := agentfmt.ItemEnvelope{
						Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
						Item: detail,
					}
					return printJSON(cmd, payload)
				}
				if isJSON(cmd) {
					return printJSON(cmd, conv)
				}
				return printConversationDetails(cmd.OutOrStdout(), conv)

			case "contact":
				contact, err := client.Contacts().Get(ctx, parsed.ResourceID)
				if err != nil {
					return fmt.Errorf("failed to get contact %d: %w", parsed.ResourceID, err)
				}
				if isJSON(cmd) {
					return printJSON(cmd, contact)
				}
				return printContactDetails(cmd.OutOrStdout(), contact)

			case "inbox":
				inbox, err := client.Inboxes().Get(ctx, parsed.ResourceID)
				if err != nil {
					return fmt.Errorf("failed to get inbox %d: %w", parsed.ResourceID, err)
				}
				if isJSON(cmd) {
					return printJSON(cmd, inbox)
				}
				return printInboxDetails(cmd.OutOrStdout(), inbox)

			case "team":
				team, err := client.Teams().Get(ctx, parsed.ResourceID)
				if err != nil {
					return fmt.Errorf("failed to get team %d: %w", parsed.ResourceID, err)
				}
				if isJSON(cmd) {
					return printJSON(cmd, team)
				}
				return printTeamDetails(cmd.OutOrStdout(), team)

			case "agent":
				agent, err := client.Agents().Get(ctx, parsed.ResourceID)
				if err != nil {
					return fmt.Errorf("failed to get agent %d: %w", parsed.ResourceID, err)
				}
				if isJSON(cmd) {
					return printJSON(cmd, agent)
				}
				return printAgentDetails(cmd.OutOrStdout(), agent)

			case "campaign":
				campaign, err := client.Campaigns().Get(ctx, parsed.ResourceID)
				if err != nil {
					return fmt.Errorf("failed to get campaign %d: %w", parsed.ResourceID, err)
				}
				if isJSON(cmd) {
					return printJSON(cmd, campaign)
				}
				return printCampaignDetails(cmd.OutOrStdout(), campaign)

			default:
				return fmt.Errorf("unsupported resource type: %s", parsed.ResourceType)
			}
		}),
	}

	cmd.Flags().StringVarP(&resourceTypeFlag, "type", "T", "", "Resource type when opening by ID (contact, conversation, inbox, team, agent, campaign)")
	registerStaticCompletions(cmd, "type", []string{"contact", "conversation", "inbox", "team", "agent", "campaign"})

	return cmd
}
