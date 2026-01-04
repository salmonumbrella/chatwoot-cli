package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newCampaignsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "campaigns",
		Short: "Manage campaigns",
		Long:  "Create, list, update, and delete campaigns for SMS and other channels.",
	}

	cmd.AddCommand(newCampaignsListCmd())
	cmd.AddCommand(newCampaignsGetCmd())
	cmd.AddCommand(newCampaignsCreateCmd())
	cmd.AddCommand(newCampaignsUpdateCmd())
	cmd.AddCommand(newCampaignsDeleteCmd())

	return cmd
}

func newCampaignsListCmd() *cobra.Command {
	var page int

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all campaigns",
		Example: "chatwoot campaigns list --page 2",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			campaigns, err := client.ListCampaigns(cmdContext(cmd), page)
			if err != nil {
				return fmt.Errorf("failed to list campaigns: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, campaigns)
			}

			if len(campaigns) == 0 {
				fmt.Println("No campaigns found.")
				return nil
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tTITLE\tTYPE\tSTATUS\tSCHEDULED\tENABLED")
			for _, c := range campaigns {
				scheduled := "-"
				if c.ScheduledAt > 0 {
					scheduled = c.ScheduledAtTime().Format("2006-01-02 15:04")
				}
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%t\n",
					c.ID, c.Title, c.CampaignType, c.CampaignStatus, scheduled, c.Enabled)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&page, "page", 0, "Page number for pagination")

	return cmd
}

func newCampaignsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <id>",
		Short:   "Get a campaign by ID",
		Example: "chatwoot campaigns get 123",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			campaign, err := client.GetCampaign(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get campaign: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, campaign)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintf(w, "ID:\t%d\n", campaign.ID)
			_, _ = fmt.Fprintf(w, "Title:\t%s\n", campaign.Title)
			_, _ = fmt.Fprintf(w, "Description:\t%s\n", campaign.Description)
			_, _ = fmt.Fprintf(w, "Message:\t%s\n", campaign.Message)
			_, _ = fmt.Fprintf(w, "Type:\t%s\n", campaign.CampaignType)
			_, _ = fmt.Fprintf(w, "Status:\t%s\n", campaign.CampaignStatus)
			_, _ = fmt.Fprintf(w, "Inbox ID:\t%d\n", campaign.InboxID)
			_, _ = fmt.Fprintf(w, "Sender ID:\t%d\n", campaign.SenderID)
			_, _ = fmt.Fprintf(w, "Enabled:\t%t\n", campaign.Enabled)
			_, _ = fmt.Fprintf(w, "Business Hours Only:\t%t\n", campaign.TriggerOnlyDuringBusinessHours)
			if campaign.ScheduledAt > 0 {
				_, _ = fmt.Fprintf(w, "Scheduled At:\t%s\n", campaign.ScheduledAtTime().Format("2006-01-02 15:04:05"))
			}
			_, _ = fmt.Fprintf(w, "Created:\t%s\n", campaign.CreatedAtTime().Format("2006-01-02 15:04:05"))

			return nil
		},
	}

	return cmd
}

func newCampaignsCreateCmd() *cobra.Command {
	var (
		title         string
		description   string
		message       string
		inboxID       int
		senderID      int
		scheduledAt   string
		audience      string
		labels        string
		enabled       bool
		businessHours bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new campaign",
		Long: `Create a new campaign. For SMS campaigns, provide inbox-id of your Twilio/SMS inbox.

The --labels flag accepts comma-separated label IDs for simpler targeting:
  --labels 1,2,3

The --audience flag accepts JSON array of audience targets (mutually exclusive with --labels):
  --audience '[{"type":"Label","id":1}]'

The --scheduled-at flag accepts RFC3339 format, e.g.:
  --scheduled-at '2025-01-15T10:00:00Z'`,
		Example: `  # Create an SMS campaign with label targeting (simple)
  chatwoot campaigns create --title "Promo" --message "50% off today!" --inbox-id 5 --labels 1,2,3 --scheduled-at '2025-01-15T10:00:00Z'

  # Create an SMS campaign with JSON audience (advanced)
  chatwoot campaigns create --title "Promo" --message "50% off today!" --inbox-id 5 --audience '[{"type":"Label","id":1}]' --scheduled-at '2025-01-15T10:00:00Z'`,
		RunE: func(cmd *cobra.Command, args []string) error {
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

			req := api.CreateCampaignRequest{
				Title:                          title,
				Description:                    description,
				Message:                        message,
				InboxID:                        inboxID,
				SenderID:                       senderID,
				Enabled:                        enabled,
				TriggerOnlyDuringBusinessHours: businessHours,
			}

			if scheduledAt != "" {
				t, err := time.Parse(time.RFC3339, scheduledAt)
				if err != nil {
					return fmt.Errorf("invalid scheduled-at format (use RFC3339): %w", err)
				}
				req.ScheduledAt = t.Unix()
			}

			if labels != "" && audience != "" {
				return fmt.Errorf("--labels and --audience are mutually exclusive")
			}

			if labels != "" {
				for _, idStr := range strings.Split(labels, ",") {
					id, err := validation.ParsePositiveInt(strings.TrimSpace(idStr), "label ID")
					if err != nil {
						return err
					}
					req.Audience = append(req.Audience, api.CampaignAudience{Type: "Label", ID: id})
				}
			}

			if audience != "" {
				var aud []api.CampaignAudience
				if err := json.Unmarshal([]byte(audience), &aud); err != nil {
					return fmt.Errorf("invalid audience JSON: %w", err)
				}
				req.Audience = aud
			}

			campaign, err := client.CreateCampaign(cmdContext(cmd), req)
			if err != nil {
				return fmt.Errorf("failed to create campaign: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, campaign)
			}

			fmt.Printf("Campaign created successfully (ID: %d)\n", campaign.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Campaign title (required)")
	cmd.Flags().StringVar(&description, "description", "", "Campaign description")
	cmd.Flags().StringVar(&message, "message", "", "Campaign message (required)")
	cmd.Flags().IntVar(&inboxID, "inbox-id", 0, "Inbox ID for the campaign (required)")
	cmd.Flags().IntVar(&senderID, "sender-id", 0, "Sender ID (agent)")
	cmd.Flags().StringVar(&scheduledAt, "scheduled-at", "", "Scheduled time (RFC3339 format)")
	cmd.Flags().StringVar(&labels, "labels", "", "Comma-separated label IDs for targeting (e.g., 1,2,3)")
	cmd.Flags().StringVar(&audience, "audience", "", "Audience targeting as JSON array (mutually exclusive with --labels)")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "Enable the campaign")
	cmd.Flags().BoolVar(&businessHours, "business-hours", false, "Trigger only during business hours")

	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("message")
	_ = cmd.MarkFlagRequired("inbox-id")

	return cmd
}

func newCampaignsUpdateCmd() *cobra.Command {
	var (
		title         string
		description   string
		message       string
		senderID      int
		scheduledAt   string
		audience      string
		labels        string
		enabled       bool
		businessHours bool
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an existing campaign",
		Long: `Update an existing campaign.

The --labels flag accepts comma-separated label IDs for simpler targeting:
  --labels 1,2,3

The --audience flag accepts JSON array of audience targets (mutually exclusive with --labels):
  --audience '[{"type":"Label","id":1}]'`,
		Example: `  # Update campaign with label targeting (simple)
  chatwoot campaigns update 123 --title 'New Title' --labels 1,2,3 --enabled true

  # Update campaign with JSON audience (advanced)
  chatwoot campaigns update 123 --audience '[{"type":"Label","id":1}]' --enabled true`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			req := api.UpdateCampaignRequest{
				Title:       title,
				Description: description,
				Message:     message,
				SenderID:    senderID,
			}

			if scheduledAt != "" {
				t, err := time.Parse(time.RFC3339, scheduledAt)
				if err != nil {
					return fmt.Errorf("invalid scheduled-at format (use RFC3339): %w", err)
				}
				req.ScheduledAt = t.Unix()
			}

			if labels != "" && audience != "" {
				return fmt.Errorf("--labels and --audience are mutually exclusive")
			}

			if labels != "" {
				for _, idStr := range strings.Split(labels, ",") {
					id, err := validation.ParsePositiveInt(strings.TrimSpace(idStr), "label ID")
					if err != nil {
						return err
					}
					req.Audience = append(req.Audience, api.CampaignAudience{Type: "Label", ID: id})
				}
			}

			if audience != "" {
				var aud []api.CampaignAudience
				if err := json.Unmarshal([]byte(audience), &aud); err != nil {
					return fmt.Errorf("invalid audience JSON: %w", err)
				}
				req.Audience = aud
			}

			if cmd.Flags().Changed("enabled") {
				req.Enabled = &enabled
			}

			if cmd.Flags().Changed("business-hours") {
				req.TriggerOnlyDuringBusinessHours = &businessHours
			}

			campaign, err := client.UpdateCampaign(cmdContext(cmd), id, req)
			if err != nil {
				return fmt.Errorf("failed to update campaign: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, campaign)
			}

			fmt.Printf("Campaign updated successfully (ID: %d)\n", campaign.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Campaign title")
	cmd.Flags().StringVar(&description, "description", "", "Campaign description")
	cmd.Flags().StringVar(&message, "message", "", "Campaign message")
	cmd.Flags().IntVar(&senderID, "sender-id", 0, "Sender ID (agent)")
	cmd.Flags().StringVar(&scheduledAt, "scheduled-at", "", "Scheduled time (RFC3339 format)")
	cmd.Flags().StringVar(&labels, "labels", "", "Comma-separated label IDs for targeting (e.g., 1,2,3)")
	cmd.Flags().StringVar(&audience, "audience", "", "Audience targeting as JSON array (mutually exclusive with --labels)")
	cmd.Flags().BoolVar(&enabled, "enabled", false, "Enable/disable campaign")
	cmd.Flags().BoolVar(&businessHours, "business-hours", false, "Trigger only during business hours")

	return cmd
}

func newCampaignsDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <id>",
		Short:   "Delete a campaign",
		Example: "chatwoot campaigns delete 123 --force",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			// In JSON mode, --force is required (can't prompt interactively)
			if isJSON(cmd) && !force {
				return fmt.Errorf("--force flag is required when using --output json")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			// If not forced and not in JSON mode, fetch campaign and prompt for confirmation
			if !force && !isJSON(cmd) {
				// Try to fetch campaign to show title in confirmation
				campaign, err := client.GetCampaign(cmdContext(cmd), id)
				if err != nil {
					// Fall back to just showing ID if fetch fails
					fmt.Printf("Delete campaign %d? (y/N): ", id)
				} else {
					fmt.Printf("Delete campaign %q (ID: %d)? (y/N): ", campaign.Title, id)
				}

				var response string
				_, _ = fmt.Scanln(&response)
				response = strings.TrimSpace(strings.ToLower(response))
				if response != "y" {
					fmt.Println("Deletion cancelled.")
					return nil
				}
			}

			if err := client.DeleteCampaign(cmdContext(cmd), id); err != nil {
				return fmt.Errorf("failed to delete campaign: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": id})
			}
			fmt.Printf("Campaign %d deleted successfully\n", id)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}
