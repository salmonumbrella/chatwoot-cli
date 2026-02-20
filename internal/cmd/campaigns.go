package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/cli"
	"github.com/chatwoot/chatwoot-cli/internal/dryrun"
	"github.com/spf13/cobra"
)

func newCampaignsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "campaigns",
		Aliases: []string{"campaign", "camp", "cm"},
		Short:   "Manage campaigns",
		Long:    "Create, list, update, and delete campaigns for SMS and other channels.",
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
		Aliases: []string{"ls"},
		Short:   "List all campaigns",
		Example: "cw campaigns list --page 2",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			campaigns, err := client.Campaigns().List(cmdContext(cmd), page)
			if err != nil {
				return fmt.Errorf("failed to list campaigns: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, campaigns)
			}

			if len(campaigns) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No campaigns found.")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tTITLE\tTYPE\tSTATUS\tSCHEDULED\tENABLED")
			for _, c := range campaigns {
				scheduled := "-"
				if c.ScheduledAt > 0 {
					scheduled = formatTimestampShort(c.ScheduledAtTime())
				}
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%t\n",
					c.ID, c.Title, c.CampaignType, c.CampaignStatus, scheduled, c.Enabled)
			}

			return nil
		}),
	}

	cmd.Flags().IntVar(&page, "page", 0, "Page number for pagination")
	flagAlias(cmd.Flags(), "page", "pg")

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "title", "campaign_status"},
		"default": {"id", "title", "campaign_status", "campaign_type", "enabled", "inbox_id", "created_at"},
		"debug": {
			"id",
			"title",
			"description",
			"message",
			"campaign_status",
			"campaign_type",
			"enabled",
			"inbox_id",
			"sender_id",
			"scheduled_at",
			"trigger_only_during_business_hours",
			"audience",
			"trigger_rules",
			"created_at",
			"updated_at",
			"account_id",
		},
	})

	return cmd
}

func newCampaignsGetCmd() *cobra.Command {
	var emit string

	cmd := &cobra.Command{
		Use:     "get <id>",
		Aliases: []string{"g"},
		Short:   "Get a campaign by ID",
		Example: "cw campaigns get 123",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "campaign")
			if err != nil {
				return err
			}

			mode, err := normalizeEmitFlag(emit)
			if err != nil {
				return err
			}
			if mode == "id" || mode == "url" {
				_, err := maybeEmit(cmd, mode, "campaign", id, nil)
				return err
			}

			if handled, err := handleURLFlag(cmd, "campaigns", id); handled {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			campaign, err := client.Campaigns().Get(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get campaign: %w", err)
			}

			if emitted, err := maybeEmit(cmd, emit, "campaign", campaign.ID, campaign); emitted {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, campaign)
			}
			return printCampaignDetails(cmd.OutOrStdout(), campaign)
		}),
	}

	cmd.Flags().Bool("url", false, "Print the Chatwoot web UI URL for this resource and exit")
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "title", "campaign_status"},
		"default": {"id", "title", "campaign_status", "campaign_type", "enabled", "inbox_id", "created_at"},
		"debug": {
			"id",
			"title",
			"description",
			"message",
			"campaign_status",
			"campaign_type",
			"enabled",
			"inbox_id",
			"sender_id",
			"scheduled_at",
			"trigger_only_during_business_hours",
			"audience",
			"trigger_rules",
			"created_at",
			"updated_at",
			"account_id",
		},
	})

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
		emit          string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"mk"},
		Short:   "Create a new campaign",
		Long: `Create a new campaign. For SMS campaigns, provide inbox-id of your Twilio/SMS inbox.

The --labels flag accepts comma-separated label IDs for simpler targeting:
  --labels 1,2,3

The --audience flag accepts JSON array of audience targets (mutually exclusive with --labels):
  --audience '[{"type":"Label","id":1}]'

The --scheduled-at flag accepts relative time or RFC3339 format, e.g.:
  --scheduled-at '30m'
  --scheduled-at '2025-01-15T10:00:00Z'`,
		Example: `  # Create an SMS campaign with label targeting (simple)
  cw campaigns create --title "Promo" --message "50% off today!" --inbox-id 5 --labels 1,2,3 --scheduled-at '2025-01-15T10:00:00Z'

  # Create an SMS campaign with JSON audience (advanced)
  cw campaigns create --title "Promo" --message "50% off today!" --inbox-id 5 --audience '[{"type":"Label","id":1}]' --scheduled-at '2025-01-15T10:00:00Z'`,
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
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
				t, err := cli.ParseRelativeTime(scheduledAt, time.Now().UTC())
				if err != nil {
					return fmt.Errorf("invalid scheduled-at format (use relative time, YYYY-MM-DD, or RFC3339): %w", err)
				}
				req.ScheduledAt = t.Unix()
			}

			if labels != "" && audience != "" {
				return fmt.Errorf("--labels and --audience are mutually exclusive")
			}

			if labels != "" {
				ids, err := ParseResourceIDListFlag(labels, "label")
				if err != nil {
					return err
				}
				for _, id := range ids {
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

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "create",
				Resource:  "campaign",
				Details:   campaignCreateDetails(req),
			}); ok {
				return err
			}

			campaign, err := client.Campaigns().Create(cmdContext(cmd), req)
			if err != nil {
				return fmt.Errorf("failed to create campaign: %w", err)
			}

			if emitted, err := maybeEmit(cmd, emit, "campaign", campaign.ID, campaign); emitted {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, campaign)
			}

			printAction(cmd, "Created", "campaign", campaign.ID, campaign.Title)
			return nil
		}),
	}

	cmd.Flags().StringVar(&title, "title", "", "Campaign title (required)")
	cmd.Flags().StringVar(&description, "description", "", "Campaign description")
	flagAlias(cmd.Flags(), "description", "desc")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Campaign message (required)")
	cmd.Flags().IntVarP(&inboxID, "inbox-id", "I", 0, "Inbox ID for the campaign (required)")
	flagAlias(cmd.Flags(), "inbox-id", "iid")
	cmd.Flags().IntVar(&senderID, "sender-id", 0, "Sender ID (agent)")
	cmd.Flags().StringVar(&scheduledAt, "scheduled-at", "", "Scheduled time (relative, RFC3339, or YYYY-MM-DD)")
	cmd.Flags().StringVarP(&labels, "labels", "L", "", "Label IDs for targeting (CSV, whitespace, JSON array; or @- / @path) (e.g., 1,2,3)")
	cmd.Flags().StringVar(&audience, "audience", "", "Audience targeting as JSON array (mutually exclusive with --labels)")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "Enable the campaign")
	cmd.Flags().BoolVar(&businessHours, "business-hours", false, "Trigger only during business hours")
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")
	flagAlias(cmd.Flags(), "title", "ttl")
	flagAlias(cmd.Flags(), "sender-id", "sid")
	flagAlias(cmd.Flags(), "scheduled-at", "sch")
	flagAlias(cmd.Flags(), "audience", "aud")
	flagAlias(cmd.Flags(), "enabled", "en")
	flagAlias(cmd.Flags(), "business-hours", "bh")

	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("message")

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
		emit          string
	)

	cmd := &cobra.Command{
		Use:     "update <id>",
		Aliases: []string{"up"},
		Short:   "Update an existing campaign",
		Long: `Update an existing campaign.

The --labels flag accepts comma-separated label IDs for simpler targeting:
  --labels 1,2,3

The --audience flag accepts JSON array of audience targets (mutually exclusive with --labels):
  --audience '[{"type":"Label","id":1}]'`,
		Example: `  # Update campaign with label targeting (simple)
  cw campaigns update 123 --title 'New Title' --labels 1,2,3 --enabled true

  # Update campaign with JSON audience (advanced)
  cw campaigns update 123 --audience '[{"type":"Label","id":1}]' --enabled true`,
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "campaign")
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
				t, err := cli.ParseRelativeTime(scheduledAt, time.Now().UTC())
				if err != nil {
					return fmt.Errorf("invalid scheduled-at format (use relative time, YYYY-MM-DD, or RFC3339): %w", err)
				}
				req.ScheduledAt = t.Unix()
			}

			if labels != "" && audience != "" {
				return fmt.Errorf("--labels and --audience are mutually exclusive")
			}

			if labels != "" {
				ids, err := ParseResourceIDListFlag(labels, "label")
				if err != nil {
					return err
				}
				for _, id := range ids {
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

			req.Enabled = boolPtrIfChanged(cmd, "enabled", enabled)
			req.TriggerOnlyDuringBusinessHours = boolPtrIfChanged(cmd, "business-hours", businessHours)

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "update",
				Resource:  "campaign",
				Details:   campaignUpdateDetails(id, req),
			}); ok {
				return err
			}

			campaign, err := client.Campaigns().Update(cmdContext(cmd), id, req)
			if err != nil {
				return fmt.Errorf("failed to update campaign: %w", err)
			}

			if emitted, err := maybeEmit(cmd, emit, "campaign", campaign.ID, campaign); emitted {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, campaign)
			}

			printAction(cmd, "Updated", "campaign", campaign.ID, campaign.Title)
			return nil
		}),
	}

	cmd.Flags().StringVar(&title, "title", "", "Campaign title")
	cmd.Flags().StringVar(&description, "description", "", "Campaign description")
	flagAlias(cmd.Flags(), "description", "desc")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Campaign message")
	cmd.Flags().IntVar(&senderID, "sender-id", 0, "Sender ID (agent)")
	cmd.Flags().StringVar(&scheduledAt, "scheduled-at", "", "Scheduled time (relative, RFC3339, or YYYY-MM-DD)")
	cmd.Flags().StringVarP(&labels, "labels", "L", "", "Label IDs for targeting (CSV, whitespace, JSON array; or @- / @path) (e.g., 1,2,3)")
	cmd.Flags().StringVar(&audience, "audience", "", "Audience targeting as JSON array (mutually exclusive with --labels)")
	cmd.Flags().BoolVar(&enabled, "enabled", false, "Enable/disable campaign")
	cmd.Flags().BoolVar(&businessHours, "business-hours", false, "Trigger only during business hours")
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")
	flagAlias(cmd.Flags(), "title", "ttl")
	flagAlias(cmd.Flags(), "sender-id", "sid")
	flagAlias(cmd.Flags(), "scheduled-at", "sch")
	flagAlias(cmd.Flags(), "audience", "aud")
	flagAlias(cmd.Flags(), "enabled", "en")
	flagAlias(cmd.Flags(), "business-hours", "bh")

	return cmd
}

func newCampaignsDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm"},
		Short:   "Delete a campaign",
		Example: "cw campaigns delete 123 --force",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "campaign")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "delete",
				Resource:  "campaign",
				Details:   map[string]any{"id": id},
			}); ok {
				return err
			}

			prompt := fmt.Sprintf("Delete campaign %d? (y/N): ", id)
			if !force && !isJSON(cmd) {
				if campaign, err := client.Campaigns().Get(cmdContext(cmd), id); err == nil {
					prompt = fmt.Sprintf("Delete campaign %q (ID: %d)? (y/N): ", campaign.Title, id)
				}
			}

			ok, err := confirmAction(cmd, confirmOptions{
				Prompt:              prompt,
				Expected:            "y",
				CancelMessage:       "Deletion cancelled.",
				Force:               force,
				RequireForceForJSON: true,
			})
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}

			if err := client.Campaigns().Delete(cmdContext(cmd), id); err != nil {
				return fmt.Errorf("failed to delete campaign: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": id})
			}
			printAction(cmd, "Deleted", "campaign", id, "")
			return nil
		}),
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	flagAlias(cmd.Flags(), "force", "fc")

	return cmd
}

func campaignCreateDetails(req api.CreateCampaignRequest) map[string]any {
	details := map[string]any{
		"title":                              req.Title,
		"message":                            req.Message,
		"inbox_id":                           req.InboxID,
		"enabled":                            req.Enabled,
		"trigger_only_during_business_hours": req.TriggerOnlyDuringBusinessHours,
	}
	if req.Description != "" {
		details["description"] = req.Description
	}
	if req.SenderID != 0 {
		details["sender_id"] = req.SenderID
	}
	if req.ScheduledAt != 0 {
		details["scheduled_at"] = req.ScheduledAt
	}
	if len(req.Audience) > 0 {
		details["audience"] = req.Audience
	}
	if len(req.TriggerRules) > 0 {
		details["trigger_rules"] = req.TriggerRules
	}
	return details
}

func campaignUpdateDetails(id int, req api.UpdateCampaignRequest) map[string]any {
	details := map[string]any{
		"id": id,
	}
	if req.Title != "" {
		details["title"] = req.Title
	}
	if req.Description != "" {
		details["description"] = req.Description
	}
	if req.Message != "" {
		details["message"] = req.Message
	}
	if req.SenderID != 0 {
		details["sender_id"] = req.SenderID
	}
	if req.ScheduledAt != 0 {
		details["scheduled_at"] = req.ScheduledAt
	}
	if req.Enabled != nil {
		details["enabled"] = *req.Enabled
	}
	if req.TriggerOnlyDuringBusinessHours != nil {
		details["trigger_only_during_business_hours"] = *req.TriggerOnlyDuringBusinessHours
	}
	if len(req.Audience) > 0 {
		details["audience"] = req.Audience
	}
	if len(req.TriggerRules) > 0 {
		details["trigger_rules"] = req.TriggerRules
	}
	return details
}
