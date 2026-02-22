package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

// printConversationDetails outputs conversation details in text format.
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
	_, _ = fmt.Fprintf(out, "  Created:    %s\n", formatTimestamp(conv.CreatedAtTime()))
	if len(conv.Labels) > 0 {
		_, _ = fmt.Fprintf(out, "  Labels:     %s\n", strings.Join(conv.Labels, ", "))
	}
	return nil
}

// printContactDetails outputs contact details in text format.
func printContactDetails(out io.Writer, contact *api.Contact) error {
	_, _ = fmt.Fprintf(out, "Contact #%d\n", contact.ID)
	_, _ = fmt.Fprintf(out, "  Name:  %s\n", displayContactName(contact.Name))
	if email := strings.TrimSpace(contact.Email); email != "" {
		_, _ = fmt.Fprintf(out, "  Email: %s\n", email)
	}
	if phone := strings.TrimSpace(contact.PhoneNumber); phone != "" {
		_, _ = fmt.Fprintf(out, "  Phone: %s\n", phone)
	}
	if identifier := strings.TrimSpace(contact.Identifier); identifier != "" {
		_, _ = fmt.Fprintf(out, "  Identifier: %s\n", identifier)
	}
	return nil
}

// printInboxDetails outputs inbox details in text format.
func printInboxDetails(out io.Writer, inbox *api.Inbox) error {
	_, _ = fmt.Fprintf(out, "Inbox #%d\n", inbox.ID)
	_, _ = fmt.Fprintf(out, "  Name:             %s\n", inbox.Name)
	_, _ = fmt.Fprintf(out, "  Channel Type:     %s\n", inbox.ChannelType)
	_, _ = fmt.Fprintf(out, "  Auto Assignment:  %t\n", inbox.EnableAutoAssignment)
	_, _ = fmt.Fprintf(out, "  Greeting Enabled: %t\n", inbox.GreetingEnabled)
	if inbox.GreetingMessage != "" {
		_, _ = fmt.Fprintf(out, "  Greeting Message: %s\n", inbox.GreetingMessage)
	}
	if inbox.WebsiteURL != "" {
		_, _ = fmt.Fprintf(out, "  Website URL:      %s\n", inbox.WebsiteURL)
	}
	return nil
}

// printTeamDetails outputs team details in text format.
func printTeamDetails(out io.Writer, team *api.Team) error {
	_, _ = fmt.Fprintf(out, "Team #%d\n", team.ID)
	_, _ = fmt.Fprintf(out, "  Name:        %s\n", team.Name)
	if team.Description != "" {
		_, _ = fmt.Fprintf(out, "  Description: %s\n", team.Description)
	}
	_, _ = fmt.Fprintf(out, "  Auto Assign: %t\n", team.AllowAutoAssign)
	if team.AccountID != 0 {
		_, _ = fmt.Fprintf(out, "  Account ID:  %d\n", team.AccountID)
	}
	return nil
}

// printAgentDetails outputs agent details in text format.
func printAgentDetails(out io.Writer, agent *api.Agent) error {
	_, _ = fmt.Fprintf(out, "Agent #%d\n", agent.ID)
	_, _ = fmt.Fprintf(out, "  Name:  %s\n", agent.Name)
	_, _ = fmt.Fprintf(out, "  Email: %s\n", agent.Email)
	if agent.Role != "" {
		_, _ = fmt.Fprintf(out, "  Role:  %s\n", agent.Role)
	}
	if agent.AvailabilityStatus != "" {
		_, _ = fmt.Fprintf(out, "  Availability: %s\n", agent.AvailabilityStatus)
	}
	if !agent.ConfirmedAt.IsZero() {
		_, _ = fmt.Fprintf(out, "  Confirmed:    %s\n", formatTimestamp(agent.ConfirmedAt))
	}
	return nil
}

// printCampaignDetails outputs campaign details in text format.
func printCampaignDetails(out io.Writer, campaign *api.Campaign) error {
	_, _ = fmt.Fprintf(out, "Campaign #%d\n", campaign.ID)
	_, _ = fmt.Fprintf(out, "  Title:    %s\n", campaign.Title)
	if campaign.Description != "" {
		_, _ = fmt.Fprintf(out, "  Description: %s\n", campaign.Description)
	}
	_, _ = fmt.Fprintf(out, "  Message:  %s\n", campaign.Message)
	if campaign.CampaignType != "" {
		_, _ = fmt.Fprintf(out, "  Type:     %s\n", campaign.CampaignType)
	}
	if campaign.CampaignStatus != "" {
		_, _ = fmt.Fprintf(out, "  Status:   %s\n", campaign.CampaignStatus)
	}
	_, _ = fmt.Fprintf(out, "  Inbox ID: %d\n", campaign.InboxID)
	if campaign.SenderID != 0 {
		_, _ = fmt.Fprintf(out, "  Sender ID: %d\n", campaign.SenderID)
	}
	_, _ = fmt.Fprintf(out, "  Enabled:  %t\n", campaign.Enabled)
	_, _ = fmt.Fprintf(out, "  Business Hours Only: %t\n", campaign.TriggerOnlyDuringBusinessHours)
	if scheduledAt := campaign.ScheduledAtTime(); !scheduledAt.IsZero() {
		_, _ = fmt.Fprintf(out, "  Scheduled At: %s\n", formatTimestamp(scheduledAt))
	}
	_, _ = fmt.Fprintf(out, "  Created:  %s\n", formatTimestamp(campaign.CreatedAtTime()))
	return nil
}
