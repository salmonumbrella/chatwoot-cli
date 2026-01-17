package cmd

import (
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/urlparse"
	"github.com/spf13/cobra"
)

func newOpenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open <url>",
		Short: "Open a Chatwoot URL and display resource details",
		Long: `Parse a Chatwoot URL and display the corresponding resource details.

This command accepts Chatwoot URLs and extracts the resource information,
then fetches and displays the resource just as if you had run the appropriate
get command directly.

Supported URL formats:
  https://app.chatwoot.com/app/accounts/{account_id}/conversations/{id}
  https://app.chatwoot.com/app/accounts/{account_id}/contacts/{id}
  https://app.chatwoot.com/app/accounts/{account_id}/inboxes/{id}
  https://app.chatwoot.com/app/accounts/{account_id}/teams/{id}
  https://app.chatwoot.com/app/accounts/{account_id}/agents/{id}
  https://app.chatwoot.com/app/accounts/{account_id}/campaigns/{id}`,
		Example: strings.TrimSpace(`
  # Open a conversation URL
  chatwoot open https://app.chatwoot.com/app/accounts/1/conversations/123

  # Open a contact URL
  chatwoot open https://app.chatwoot.com/app/accounts/1/contacts/456

  # Open with JSON output
  chatwoot open https://app.chatwoot.com/app/accounts/1/conversations/123 --output json
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			rawURL := args[0]

			// Parse the URL
			parsed, err := urlparse.Parse(rawURL)
			if err != nil {
				return fmt.Errorf("failed to parse URL: %w", err)
			}

			// Get the client
			client, err := getClient()
			if err != nil {
				return err
			}

			// Verify account ID matches
			if client.AccountID != parsed.AccountID {
				return fmt.Errorf("URL account ID (%d) does not match authenticated account ID (%d); use 'chatwoot auth login' to switch accounts", parsed.AccountID, client.AccountID)
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

	return cmd
}
