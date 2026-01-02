package cmd

import (
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
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
		RunE: func(cmd *cobra.Command, args []string) error {
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
				conv, err := client.GetConversation(ctx, parsed.ResourceID)
				if err != nil {
					return fmt.Errorf("failed to get conversation %d: %w", parsed.ResourceID, err)
				}
				if isJSON(cmd) {
					return printJSON(cmd, conv)
				}
				return printConversationDetails(conv)

			case "contact":
				contact, err := client.GetContact(ctx, parsed.ResourceID)
				if err != nil {
					return fmt.Errorf("failed to get contact %d: %w", parsed.ResourceID, err)
				}
				if isJSON(cmd) {
					return printJSON(cmd, contact)
				}
				return printContactDetails(contact)

			case "inbox":
				inbox, err := client.GetInbox(ctx, parsed.ResourceID)
				if err != nil {
					return fmt.Errorf("failed to get inbox %d: %w", parsed.ResourceID, err)
				}
				if isJSON(cmd) {
					return printJSON(cmd, inbox)
				}
				return printInboxDetails(inbox)

			case "team":
				team, err := client.GetTeam(ctx, parsed.ResourceID)
				if err != nil {
					return fmt.Errorf("failed to get team %d: %w", parsed.ResourceID, err)
				}
				if isJSON(cmd) {
					return printJSON(cmd, team)
				}
				return printTeamDetails(team)

			case "agent":
				agent, err := client.GetAgent(ctx, parsed.ResourceID)
				if err != nil {
					return fmt.Errorf("failed to get agent %d: %w", parsed.ResourceID, err)
				}
				if isJSON(cmd) {
					return printJSON(cmd, agent)
				}
				return printAgentDetails(agent)

			case "campaign":
				campaign, err := client.GetCampaign(ctx, parsed.ResourceID)
				if err != nil {
					return fmt.Errorf("failed to get campaign %d: %w", parsed.ResourceID, err)
				}
				if isJSON(cmd) {
					return printJSON(cmd, campaign)
				}
				return printCampaignDetails(campaign)

			default:
				return fmt.Errorf("unsupported resource type: %s", parsed.ResourceType)
			}
		},
	}

	return cmd
}

// printConversationDetails outputs conversation details in text format
func printConversationDetails(conv *api.Conversation) error {
	displayID := conv.ID
	if conv.DisplayID != nil {
		displayID = *conv.DisplayID
	}
	fmt.Printf("Conversation #%d\n", displayID)
	fmt.Printf("  ID:         %d\n", conv.ID)
	fmt.Printf("  Inbox ID:   %d\n", conv.InboxID)
	fmt.Printf("  Contact ID: %d\n", conv.ContactID)
	fmt.Printf("  Status:     %s\n", conv.Status)
	if conv.Priority != nil {
		fmt.Printf("  Priority:   %s\n", *conv.Priority)
	}
	if conv.AssigneeID != nil {
		fmt.Printf("  Assignee:   %d\n", *conv.AssigneeID)
	}
	if conv.TeamID != nil {
		fmt.Printf("  Team:       %d\n", *conv.TeamID)
	}
	fmt.Printf("  Unread:     %d\n", conv.Unread)
	fmt.Printf("  Muted:      %t\n", conv.Muted)
	fmt.Printf("  Created:    %s\n", conv.CreatedAtTime().Format("2006-01-02 15:04:05"))
	if len(conv.Labels) > 0 {
		fmt.Printf("  Labels:     %s\n", strings.Join(conv.Labels, ", "))
	}
	return nil
}

// printContactDetails outputs contact details in text format
func printContactDetails(contact *api.Contact) error {
	fmt.Printf("Contact #%d\n", contact.ID)
	fmt.Printf("  Name:  %s\n", contact.Name)
	if contact.Email != "" {
		fmt.Printf("  Email: %s\n", contact.Email)
	}
	if contact.PhoneNumber != "" {
		fmt.Printf("  Phone: %s\n", contact.PhoneNumber)
	}
	if contact.Identifier != "" {
		fmt.Printf("  Identifier: %s\n", contact.Identifier)
	}
	return nil
}

// printInboxDetails outputs inbox details in text format
func printInboxDetails(inbox *api.Inbox) error {
	fmt.Printf("Inbox #%d\n", inbox.ID)
	fmt.Printf("  Name:             %s\n", inbox.Name)
	fmt.Printf("  Channel Type:     %s\n", inbox.ChannelType)
	fmt.Printf("  Auto Assignment:  %t\n", inbox.EnableAutoAssignment)
	fmt.Printf("  Greeting Enabled: %t\n", inbox.GreetingEnabled)
	if inbox.GreetingMessage != "" {
		fmt.Printf("  Greeting Message: %s\n", inbox.GreetingMessage)
	}
	return nil
}

// printTeamDetails outputs team details in text format
func printTeamDetails(team *api.Team) error {
	fmt.Printf("Team #%d\n", team.ID)
	fmt.Printf("  Name:        %s\n", team.Name)
	fmt.Printf("  Description: %s\n", team.Description)
	return nil
}

// printAgentDetails outputs agent details in text format
func printAgentDetails(agent *api.Agent) error {
	fmt.Printf("Agent #%d\n", agent.ID)
	fmt.Printf("  Name:  %s\n", agent.Name)
	fmt.Printf("  Email: %s\n", agent.Email)
	if agent.Role != "" {
		fmt.Printf("  Role:  %s\n", agent.Role)
	}
	return nil
}

// printCampaignDetails outputs campaign details in text format
func printCampaignDetails(campaign *api.Campaign) error {
	fmt.Printf("Campaign #%d\n", campaign.ID)
	fmt.Printf("  Title:    %s\n", campaign.Title)
	fmt.Printf("  Message:  %s\n", campaign.Message)
	fmt.Printf("  Inbox ID: %d\n", campaign.InboxID)
	fmt.Printf("  Enabled:  %t\n", campaign.Enabled)
	return nil
}
