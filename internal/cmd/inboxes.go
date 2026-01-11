package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/dryrun"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newInboxesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inboxes",
		Short: "Manage inboxes",
		Long:  "List, create, update, and delete inboxes in your Chatwoot account",
	}

	cmd.AddCommand(newInboxesListCmd())
	cmd.AddCommand(newInboxesGetCmd())
	cmd.AddCommand(newInboxesCreateCmd())
	cmd.AddCommand(newInboxesUpdateCmd())
	cmd.AddCommand(newInboxesDeleteCmd())
	cmd.AddCommand(newInboxesAgentBotCmd())
	cmd.AddCommand(newInboxesSetAgentBotCmd())
	cmd.AddCommand(newInboxesTriageCmd())
	cmd.AddCommand(newInboxesCampaignsCmd())
	cmd.AddCommand(newInboxesSyncTemplatesCmd())
	cmd.AddCommand(newInboxesHealthCmd())
	cmd.AddCommand(newInboxesDeleteAvatarCmd())
	cmd.AddCommand(newInboxesCSATTemplateCmd())

	return cmd
}

func newInboxesListCmd() *cobra.Command {
	cfg := ListConfig[api.Inbox]{
		Use:     "list",
		Short:   "List all inboxes",
		Headers: []string{"ID", "NAME", "CHANNEL TYPE", "AUTO ASSIGN"},
		RowFunc: func(inbox api.Inbox) []string {
			return []string{
				fmt.Sprintf("%d", inbox.ID),
				inbox.Name,
				inbox.ChannelType,
				fmt.Sprintf("%v", inbox.EnableAutoAssignment),
			}
		},
		EmptyMessage: "No inboxes found",
		Fetch: func(ctx context.Context, client *api.Client, page, pageSize int) (ListResult[api.Inbox], error) {
			inboxes, err := client.ListInboxes(ctx)
			if err != nil {
				return ListResult[api.Inbox]{}, err
			}
			return ListResult[api.Inbox]{Items: inboxes, HasMore: false}, nil
		},
	}

	cmd := NewListCommand(cfg, func(ctx context.Context) (*api.Client, error) {
		return getClient()
	})

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name", "channel_type"},
		"default": {"id", "name", "channel_type", "website_url", "greeting_enabled"},
		"debug":   {"id", "name", "channel_type", "avatar_url", "website_url", "greeting_enabled", "greeting_message", "enable_auto_assignment"},
	})

	return cmd
}

func newInboxesGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get inbox details",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			inbox, err := client.GetInbox(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, inbox)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintf(w, "ID:\t%d\n", inbox.ID)
			_, _ = fmt.Fprintf(w, "Name:\t%s\n", inbox.Name)
			_, _ = fmt.Fprintf(w, "Channel Type:\t%s\n", inbox.ChannelType)
			_, _ = fmt.Fprintf(w, "Auto Assignment:\t%v\n", inbox.EnableAutoAssignment)
			_, _ = fmt.Fprintf(w, "Greeting Enabled:\t%v\n", inbox.GreetingEnabled)
			if inbox.GreetingMessage != "" {
				_, _ = fmt.Fprintf(w, "Greeting Message:\t%s\n", inbox.GreetingMessage)
			}
			if inbox.WebsiteURL != "" {
				_, _ = fmt.Fprintf(w, "Website URL:\t%s\n", inbox.WebsiteURL)
			}

			return nil
		}),
	}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name", "channel_type"},
		"default": {"id", "name", "channel_type", "website_url", "greeting_enabled"},
		"debug":   {"id", "name", "channel_type", "avatar_url", "website_url", "greeting_enabled", "greeting_message", "enable_auto_assignment"},
	})

	return cmd
}

func newInboxesCreateCmd() *cobra.Command {
	var (
		name                       string
		channelType                string
		greetingEnabled            bool
		greetingMessage            string
		enableEmailCollect         bool
		csatSurveyEnabled          bool
		enableAutoAssignment       bool
		autoAssignmentConfig       string
		workingHoursEnabled        bool
		timezone                   string
		allowMessagesAfterResolved bool
		lockToSingleConversation   bool
		portalID                   int
		senderNameType             string
		outOfOfficeMessage         string
		outOfOfficeEnabled         bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new inbox",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("name is required")
			}
			if channelType == "" {
				return fmt.Errorf("channel-type is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			req := api.CreateInboxRequest{
				Name:        name,
				ChannelType: channelType,
				InboxSettings: api.InboxSettings{
					GreetingMessage:    greetingMessage,
					Timezone:           timezone,
					SenderNameType:     senderNameType,
					OutOfOfficeMessage: outOfOfficeMessage,
				},
			}

			req.GreetingEnabled = boolPtrIfChanged(cmd, "greeting-enabled", greetingEnabled)
			req.EnableEmailCollect = boolPtrIfChanged(cmd, "enable-email-collect", enableEmailCollect)
			req.CSATSurveyEnabled = boolPtrIfChanged(cmd, "csat-survey-enabled", csatSurveyEnabled)
			req.EnableAutoAssignment = boolPtrIfChanged(cmd, "enable-auto-assignment", enableAutoAssignment)
			req.WorkingHoursEnabled = boolPtrIfChanged(cmd, "working-hours-enabled", workingHoursEnabled)
			req.AllowMessagesAfterResolved = boolPtrIfChanged(cmd, "allow-messages-after-resolved", allowMessagesAfterResolved)
			req.LockToSingleConversation = boolPtrIfChanged(cmd, "lock-to-single-conversation", lockToSingleConversation)
			req.OutOfOfficeEnabled = boolPtrIfChanged(cmd, "out-of-office-enabled", outOfOfficeEnabled)
			if portalID > 0 {
				req.PortalID = &portalID
			}
			if autoAssignmentConfig != "" {
				var cfg map[string]any
				if err := json.Unmarshal([]byte(autoAssignmentConfig), &cfg); err != nil {
					return fmt.Errorf("invalid auto-assignment-config JSON: %w", err)
				}
				req.AutoAssignmentConfig = cfg
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "create",
				Resource:  "inbox",
				Details:   inboxCreateDetails(req),
			}); ok {
				return err
			}

			inbox, err := client.CreateInbox(cmdContext(cmd), req)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, inbox)
			}

			printAction(cmd, "Created", "inbox", inbox.ID, fmt.Sprintf("%s (%s)", inbox.Name, inbox.ChannelType))
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Inbox name (required)")
	cmd.Flags().StringVar(&channelType, "channel-type", "", "Channel type (required)")
	cmd.Flags().BoolVar(&greetingEnabled, "greeting-enabled", false, "Enable greeting message")
	cmd.Flags().StringVar(&greetingMessage, "greeting-message", "", "Greeting message")
	cmd.Flags().BoolVar(&enableEmailCollect, "enable-email-collect", false, "Enable email collection")
	cmd.Flags().BoolVar(&csatSurveyEnabled, "csat-survey-enabled", false, "Enable CSAT survey")
	cmd.Flags().BoolVar(&enableAutoAssignment, "enable-auto-assignment", false, "Enable auto-assignment")
	cmd.Flags().StringVar(&autoAssignmentConfig, "auto-assignment-config", "", "Auto-assignment config JSON")
	cmd.Flags().BoolVar(&workingHoursEnabled, "working-hours-enabled", false, "Enable working hours")
	cmd.Flags().StringVar(&timezone, "timezone", "", "Timezone (e.g. America/New_York)")
	cmd.Flags().BoolVar(&allowMessagesAfterResolved, "allow-messages-after-resolved", false, "Allow messages after resolved")
	cmd.Flags().BoolVar(&lockToSingleConversation, "lock-to-single-conversation", false, "Lock to single conversation")
	cmd.Flags().IntVar(&portalID, "portal-id", 0, "Help center portal ID")
	cmd.Flags().StringVar(&senderNameType, "sender-name-type", "", "Sender name type")
	cmd.Flags().StringVar(&outOfOfficeMessage, "out-of-office-message", "", "Out of office message")
	cmd.Flags().BoolVar(&outOfOfficeEnabled, "out-of-office-enabled", false, "Enable out of office message")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("channel-type")

	return cmd
}

func newInboxesUpdateCmd() *cobra.Command {
	var (
		name                       string
		greetingEnabled            bool
		greetingMessage            string
		enableEmailCollect         bool
		csatSurveyEnabled          bool
		enableAutoAssignment       bool
		autoAssignmentConfig       string
		workingHoursEnabled        bool
		timezone                   string
		allowMessagesAfterResolved bool
		lockToSingleConversation   bool
		portalID                   int
		senderNameType             string
		outOfOfficeMessage         string
		outOfOfficeEnabled         bool
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an inbox",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			if name == "" &&
				greetingMessage == "" &&
				autoAssignmentConfig == "" &&
				timezone == "" &&
				portalID == 0 &&
				senderNameType == "" &&
				outOfOfficeMessage == "" &&
				!anyFlagChanged(cmd,
					"greeting-enabled",
					"enable-email-collect",
					"csat-survey-enabled",
					"enable-auto-assignment",
					"working-hours-enabled",
					"allow-messages-after-resolved",
					"lock-to-single-conversation",
					"out-of-office-enabled",
				) {
				return fmt.Errorf("at least one field must be provided to update")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			req := api.UpdateInboxRequest{
				Name: name,
				InboxSettings: api.InboxSettings{
					GreetingMessage:    greetingMessage,
					Timezone:           timezone,
					SenderNameType:     senderNameType,
					OutOfOfficeMessage: outOfOfficeMessage,
				},
			}
			req.GreetingEnabled = boolPtrIfChanged(cmd, "greeting-enabled", greetingEnabled)
			req.EnableEmailCollect = boolPtrIfChanged(cmd, "enable-email-collect", enableEmailCollect)
			req.CSATSurveyEnabled = boolPtrIfChanged(cmd, "csat-survey-enabled", csatSurveyEnabled)
			req.EnableAutoAssignment = boolPtrIfChanged(cmd, "enable-auto-assignment", enableAutoAssignment)
			req.WorkingHoursEnabled = boolPtrIfChanged(cmd, "working-hours-enabled", workingHoursEnabled)
			req.AllowMessagesAfterResolved = boolPtrIfChanged(cmd, "allow-messages-after-resolved", allowMessagesAfterResolved)
			req.LockToSingleConversation = boolPtrIfChanged(cmd, "lock-to-single-conversation", lockToSingleConversation)
			req.OutOfOfficeEnabled = boolPtrIfChanged(cmd, "out-of-office-enabled", outOfOfficeEnabled)
			if portalID > 0 {
				req.PortalID = &portalID
			}
			if autoAssignmentConfig != "" {
				var cfg map[string]any
				if err := json.Unmarshal([]byte(autoAssignmentConfig), &cfg); err != nil {
					return fmt.Errorf("invalid auto-assignment-config JSON: %w", err)
				}
				req.AutoAssignmentConfig = cfg
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "update",
				Resource:  "inbox",
				Details:   inboxUpdateDetails(id, req),
			}); ok {
				return err
			}

			inbox, err := client.UpdateInbox(cmdContext(cmd), id, req)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, inbox)
			}

			printAction(cmd, "Updated", "inbox", inbox.ID, inbox.Name)
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Inbox name")
	cmd.Flags().BoolVar(&greetingEnabled, "greeting-enabled", false, "Enable greeting message")
	cmd.Flags().StringVar(&greetingMessage, "greeting-message", "", "Greeting message")
	cmd.Flags().BoolVar(&enableEmailCollect, "enable-email-collect", false, "Enable email collection")
	cmd.Flags().BoolVar(&csatSurveyEnabled, "csat-survey-enabled", false, "Enable CSAT survey")
	cmd.Flags().BoolVar(&enableAutoAssignment, "enable-auto-assignment", false, "Enable auto-assignment")
	cmd.Flags().StringVar(&autoAssignmentConfig, "auto-assignment-config", "", "Auto-assignment config JSON")
	cmd.Flags().BoolVar(&workingHoursEnabled, "working-hours-enabled", false, "Enable working hours")
	cmd.Flags().StringVar(&timezone, "timezone", "", "Timezone (e.g. America/New_York)")
	cmd.Flags().BoolVar(&allowMessagesAfterResolved, "allow-messages-after-resolved", false, "Allow messages after resolved")
	cmd.Flags().BoolVar(&lockToSingleConversation, "lock-to-single-conversation", false, "Lock to single conversation")
	cmd.Flags().IntVar(&portalID, "portal-id", 0, "Help center portal ID")
	cmd.Flags().StringVar(&senderNameType, "sender-name-type", "", "Sender name type")
	cmd.Flags().StringVar(&outOfOfficeMessage, "out-of-office-message", "", "Out of office message")
	cmd.Flags().BoolVar(&outOfOfficeEnabled, "out-of-office-enabled", false, "Enable out of office message")

	return cmd
}

func newInboxesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an inbox",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "delete",
				Resource:  "inbox",
				Details:   map[string]any{"id": id},
			}); ok {
				return err
			}

			if err := client.DeleteInbox(cmdContext(cmd), id); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": id})
			}
			printAction(cmd, "Deleted", "inbox", id, "")
			return nil
		}),
	}
}

func newInboxesAgentBotCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agent-bot <id>",
		Short: "Get the agent bot assigned to an inbox",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			bot, err := client.GetInboxAgentBot(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, bot)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintf(w, "ID:\t%d\n", bot.ID)
			_, _ = fmt.Fprintf(w, "Name:\t%s\n", bot.Name)
			if bot.Description != "" {
				_, _ = fmt.Fprintf(w, "Description:\t%s\n", bot.Description)
			}
			if bot.OutgoingURL != "" {
				_, _ = fmt.Fprintf(w, "Outgoing URL:\t%s\n", bot.OutgoingURL)
			}

			return nil
		}),
	}
}

func newInboxesSetAgentBotCmd() *cobra.Command {
	var botID int

	cmd := &cobra.Command{
		Use:   "set-agent-bot <id>",
		Short: "Assign an agent bot to an inbox",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			if botID == 0 {
				return fmt.Errorf("bot-id is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "update",
				Resource:  "inbox_agent_bot",
				Details: map[string]any{
					"id":     id,
					"bot_id": botID,
				},
			}); ok {
				return err
			}

			if err := client.SetInboxAgentBot(cmdContext(cmd), id, botID); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Assigned agent bot %d to inbox %d\n", botID, id)
			return nil
		}),
	}

	cmd.Flags().IntVar(&botID, "bot-id", 0, "Agent bot ID (required)")
	_ = cmd.MarkFlagRequired("bot-id")

	return cmd
}

func newInboxesTriageCmd() *cobra.Command {
	var (
		status string
		limit  int
	)

	cmd := &cobra.Command{
		Use:   "triage <id>",
		Short: "Get conversations with enriched context for triage",
		Long:  "Returns conversations for an inbox with contact info and last message for decision-making",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			triage, err := client.GetInboxTriage(cmdContext(cmd), id, status, limit)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, triage)
			}

			// Text output
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Inbox: %s (ID: %d)\n", triage.InboxName, triage.InboxID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Summary: %d open, %d pending, %d unread\n\n",
				triage.Summary.Open, triage.Summary.Pending, triage.Summary.Unread)

			if len(triage.Conversations) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No conversations found")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
			_, _ = fmt.Fprintln(w, "ID\tCONTACT\tSTATUS\tUNREAD\tLAST MESSAGE")
			for _, conv := range triage.Conversations {
				contactName := conv.Contact.Name
				if contactName == "" {
					contactName = fmt.Sprintf("Contact #%d", conv.Contact.ID)
				}

				lastMsg := ""
				if conv.LastMessage != nil {
					lastMsg = truncateString(conv.LastMessage.Content, 40)
				}

				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%s\n",
					conv.ID, contactName, conv.Status, conv.UnreadCount, lastMsg)
			}
			_ = w.Flush()

			return nil
		}),
	}

	cmd.Flags().StringVar(&status, "status", "open", "Filter by status (open, pending, resolved, snoozed, all)")
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum number of conversations to return")

	return cmd
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if maxLen < 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func newInboxesCampaignsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "campaigns <id>",
		Short: "List campaigns for an inbox",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			campaigns, err := client.GetInboxCampaigns(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, campaigns)
			}

			if len(campaigns) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No campaigns found for this inbox")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
			_, _ = fmt.Fprintln(w, "ID\tTITLE\tTYPE\tENABLED")
			for _, c := range campaigns {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%v\n",
					c.ID, c.Title, c.CampaignType, c.Enabled)
			}
			_ = w.Flush()

			return nil
		}),
	}
}

func newInboxesSyncTemplatesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync-templates <id>",
		Short: "Sync WhatsApp templates for an inbox",
		Long:  "Sync WhatsApp message templates from the WhatsApp Business API for this inbox",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "update",
				Resource:  "inbox_templates",
				Details:   map[string]any{"id": id},
			}); ok {
				return err
			}

			if err := client.SyncInboxTemplates(cmdContext(cmd), id); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Successfully synced templates for inbox %d\n", id)
			return nil
		}),
	}
}

func newInboxesHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health <id>",
		Short: "Get WhatsApp Cloud API health status for an inbox",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			health, err := client.GetInboxHealth(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, health)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			for k, v := range health {
				_, _ = fmt.Fprintf(w, "%s:\t%v\n", k, v)
			}

			return nil
		}),
	}
}

func newInboxesDeleteAvatarCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete-avatar <id>",
		Short: "Remove the inbox avatar",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "delete",
				Resource:  "inbox_avatar",
				Details:   map[string]any{"id": id},
			}); ok {
				return err
			}

			if err := client.DeleteInboxAvatar(cmdContext(cmd), id); err != nil {
				return err
			}

			printAction(cmd, "Deleted", "inbox avatar", id, "")
			return nil
		}),
	}
}

func newInboxesCSATTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "csat-template <id>",
		Short: "Get or set CSAT survey template for an inbox",
		Args:  cobra.ExactArgs(1),
	}

	cmd.AddCommand(newInboxesCSATTemplateGetCmd())
	cmd.AddCommand(newInboxesCSATTemplateSetCmd())

	return cmd
}

func inboxCreateDetails(req api.CreateInboxRequest) map[string]any {
	details := map[string]any{
		"name":         req.Name,
		"channel_type": req.ChannelType,
	}
	addInboxSettingsDetails(details, req.InboxSettings)
	return details
}

func inboxUpdateDetails(id int, req api.UpdateInboxRequest) map[string]any {
	details := map[string]any{
		"id": id,
	}
	if req.Name != "" {
		details["name"] = req.Name
	}
	addInboxSettingsDetails(details, req.InboxSettings)
	return details
}

func addInboxSettingsDetails(details map[string]any, settings api.InboxSettings) {
	if settings.GreetingEnabled != nil {
		details["greeting_enabled"] = *settings.GreetingEnabled
	}
	if settings.GreetingMessage != "" {
		details["greeting_message"] = settings.GreetingMessage
	}
	if settings.EnableEmailCollect != nil {
		details["enable_email_collect"] = *settings.EnableEmailCollect
	}
	if settings.CSATSurveyEnabled != nil {
		details["csat_survey_enabled"] = *settings.CSATSurveyEnabled
	}
	if settings.EnableAutoAssignment != nil {
		details["enable_auto_assignment"] = *settings.EnableAutoAssignment
	}
	if settings.AutoAssignmentConfig != nil {
		details["auto_assignment_config"] = settings.AutoAssignmentConfig
	}
	if settings.WorkingHoursEnabled != nil {
		details["working_hours_enabled"] = *settings.WorkingHoursEnabled
	}
	if settings.Timezone != "" {
		details["timezone"] = settings.Timezone
	}
	if settings.AllowMessagesAfterResolved != nil {
		details["allow_messages_after_resolved"] = *settings.AllowMessagesAfterResolved
	}
	if settings.LockToSingleConversation != nil {
		details["lock_to_single_conversation"] = *settings.LockToSingleConversation
	}
	if settings.PortalID != nil {
		details["portal_id"] = *settings.PortalID
	}
	if settings.SenderNameType != "" {
		details["sender_name_type"] = settings.SenderNameType
	}
	if settings.OutOfOfficeMessage != "" {
		details["out_of_office_message"] = settings.OutOfOfficeMessage
	}
	if settings.OutOfOfficeEnabled != nil {
		details["out_of_office_enabled"] = *settings.OutOfOfficeEnabled
	}
}

func newInboxesCSATTemplateGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <inbox-id>",
		Short: "Get the CSAT survey template for an inbox",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			template, err := client.GetInboxCSATTemplate(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, template)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintf(w, "ID:\t%d\n", template.ID)
			_, _ = fmt.Fprintf(w, "Question:\t%s\n", template.Question)
			_, _ = fmt.Fprintf(w, "Message:\t%s\n", template.Message)

			return nil
		}),
	}
}

func newInboxesCSATTemplateSetCmd() *cobra.Command {
	var (
		question string
		message  string
	)

	cmd := &cobra.Command{
		Use:   "set <inbox-id>",
		Short: "Create or update the CSAT survey template for an inbox",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			if question == "" {
				return fmt.Errorf("question is required")
			}
			if message == "" {
				return fmt.Errorf("message is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "update",
				Resource:  "inbox_csat_template",
				Details: map[string]any{
					"id":       id,
					"question": question,
					"message":  message,
				},
			}); ok {
				return err
			}

			template, err := client.CreateInboxCSATTemplate(cmdContext(cmd), id, question, message)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, template)
			}

			printAction(cmd, "Updated", "inbox CSAT template", id, "")
			return nil
		}),
	}

	cmd.Flags().StringVar(&question, "question", "", "Survey question (required)")
	cmd.Flags().StringVar(&message, "message", "", "Survey message (required)")
	_ = cmd.MarkFlagRequired("question")
	_ = cmd.MarkFlagRequired("message")

	return cmd
}
