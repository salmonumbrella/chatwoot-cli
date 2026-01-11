package cmd

import (
	"fmt"
	"io"
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
				conv, err := client.GetConversation(ctx, parsed.ResourceID)
				if err != nil {
					return fmt.Errorf("failed to get conversation %d: %w", parsed.ResourceID, err)
				}
				if isJSON(cmd) {
					return printJSON(cmd, conv)
				}
				return printConversationDetails(cmd.OutOrStdout(), conv)

			case "contact":
				contact, err := client.GetContact(ctx, parsed.ResourceID)
				if err != nil {
					return fmt.Errorf("failed to get contact %d: %w", parsed.ResourceID, err)
				}
				if isJSON(cmd) {
					return printJSON(cmd, contact)
				}
				return printContactDetails(cmd.OutOrStdout(), contact)

			case "inbox":
				inbox, err := client.GetInbox(ctx, parsed.ResourceID)
				if err != nil {
					return fmt.Errorf("failed to get inbox %d: %w", parsed.ResourceID, err)
				}
				if isJSON(cmd) {
					return printJSON(cmd, inbox)
				}
				return printInboxDetails(cmd.OutOrStdout(), inbox)

			case "team":
				team, err := client.GetTeam(ctx, parsed.ResourceID)
				if err != nil {
					return fmt.Errorf("failed to get team %d: %w", parsed.ResourceID, err)
				}
				if isJSON(cmd) {
					return printJSON(cmd, team)
				}
				return printTeamDetails(cmd.OutOrStdout(), team)

			case "agent":
				agent, err := client.GetAgent(ctx, parsed.ResourceID)
				if err != nil {
					return fmt.Errorf("failed to get agent %d: %w", parsed.ResourceID, err)
				}
				if isJSON(cmd) {
					return printJSON(cmd, agent)
				}
				return printAgentDetails(cmd.OutOrStdout(), agent)

			case "campaign":
				campaign, err := client.GetCampaign(ctx, parsed.ResourceID)
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

// printConversationDetails outputs conversation details in text format
func printConversationDetails(out io.Writer, conv *api.Conversation) error {
	displayID := conv.ID
	if conv.DisplayID != nil {
		displayID = *conv.DisplayID
	}
	_, _ = fmt.Fprintf(out, "Conversation #%d\n", displayID)
	_, _ = fmt.Fprintf(out, "  ID:         %d\n", conv.ID)
	_, _ = fmt.Fprintf(out, "  Inbox ID:   %d\n", conv.InboxID)
	_, _ = fmt.Fprintf(out, "  Contact ID: %d\n", conv.ContactID)
	_, _ = fmt.Fprintf(out, "  Status:     %s\n", conv.Status)
	if conv.Priority != nil {
		_, _ = fmt.Fprintf(out, "  Priority:   %s\n", *conv.Priority)
	}
	if conv.AssigneeID != nil {
		_, _ = fmt.Fprintf(out, "  Assignee:   %d\n", *conv.AssigneeID)
	}
	if conv.TeamID != nil {
		_, _ = fmt.Fprintf(out, "  Team:       %d\n", *conv.TeamID)
	}
	_, _ = fmt.Fprintf(out, "  Unread:     %d\n", conv.Unread)
	_, _ = fmt.Fprintf(out, "  Muted:      %t\n", conv.Muted)
	_, _ = fmt.Fprintf(out, "  Created:    %s\n", conv.CreatedAtTime().Format("2006-01-02 15:04:05"))
	if len(conv.Labels) > 0 {
		_, _ = fmt.Fprintf(out, "  Labels:     %s\n", strings.Join(conv.Labels, ", "))
	}
	return nil
}

// printContactDetails outputs contact details in text format
func printContactDetails(out io.Writer, contact *api.Contact) error {
	_, _ = fmt.Fprintf(out, "Contact #%d\n", contact.ID)
	_, _ = fmt.Fprintf(out, "  Name:  %s\n", contact.Name)
	if contact.Email != "" {
		_, _ = fmt.Fprintf(out, "  Email: %s\n", contact.Email)
	}
	if contact.PhoneNumber != "" {
		_, _ = fmt.Fprintf(out, "  Phone: %s\n", contact.PhoneNumber)
	}
	if contact.Identifier != "" {
		_, _ = fmt.Fprintf(out, "  Identifier: %s\n", contact.Identifier)
	}
	return nil
}

// printInboxDetails outputs inbox details in text format
func printInboxDetails(out io.Writer, inbox *api.Inbox) error {
	_, _ = fmt.Fprintf(out, "Inbox #%d\n", inbox.ID)
	_, _ = fmt.Fprintf(out, "  Name:             %s\n", inbox.Name)
	_, _ = fmt.Fprintf(out, "  Channel Type:     %s\n", inbox.ChannelType)
	_, _ = fmt.Fprintf(out, "  Auto Assignment:  %t\n", inbox.EnableAutoAssignment)
	_, _ = fmt.Fprintf(out, "  Greeting Enabled: %t\n", inbox.GreetingEnabled)
	if inbox.GreetingMessage != "" {
		_, _ = fmt.Fprintf(out, "  Greeting Message: %s\n", inbox.GreetingMessage)
	}
	return nil
}

// printTeamDetails outputs team details in text format
func printTeamDetails(out io.Writer, team *api.Team) error {
	_, _ = fmt.Fprintf(out, "Team #%d\n", team.ID)
	_, _ = fmt.Fprintf(out, "  Name:        %s\n", team.Name)
	_, _ = fmt.Fprintf(out, "  Description: %s\n", team.Description)
	return nil
}

// printAgentDetails outputs agent details in text format
func printAgentDetails(out io.Writer, agent *api.Agent) error {
	_, _ = fmt.Fprintf(out, "Agent #%d\n", agent.ID)
	_, _ = fmt.Fprintf(out, "  Name:  %s\n", agent.Name)
	_, _ = fmt.Fprintf(out, "  Email: %s\n", agent.Email)
	if agent.Role != "" {
		_, _ = fmt.Fprintf(out, "  Role:  %s\n", agent.Role)
	}
	return nil
}

// printCampaignDetails outputs campaign details in text format
func printCampaignDetails(out io.Writer, campaign *api.Campaign) error {
	_, _ = fmt.Fprintf(out, "Campaign #%d\n", campaign.ID)
	_, _ = fmt.Fprintf(out, "  Title:    %s\n", campaign.Title)
	_, _ = fmt.Fprintf(out, "  Message:  %s\n", campaign.Message)
	_, _ = fmt.Fprintf(out, "  Inbox ID: %d\n", campaign.InboxID)
	_, _ = fmt.Fprintf(out, "  Enabled:  %t\n", campaign.Enabled)
	return nil
}
