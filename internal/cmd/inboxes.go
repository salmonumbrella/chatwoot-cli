package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/dryrun"
	"github.com/spf13/cobra"
)

func newInboxesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "inboxes",
		Aliases: []string{"inbox", "in"},
		Short:   "Manage inboxes",
		Long:    "List, create, update, and delete inboxes in your Chatwoot account",
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
	cmd.AddCommand(newInboxesStatsCmd())

	return cmd
}

func newInboxesListCmd() *cobra.Command {
	var light bool

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
		AgentTransform: func(_ context.Context, _ *api.Client, items []api.Inbox) (any, error) {
			if light {
				return buildLightInboxes(items), nil
			}
			return nil, nil
		},
		JSONTransform: func(_ context.Context, _ *api.Client, items []api.Inbox) (any, error) {
			if !light {
				return items, nil
			}
			return buildLightInboxes(items), nil
		},
		ForceJSON: func(_ *cobra.Command) bool {
			return light
		},
		Fetch: func(ctx context.Context, client *api.Client, page, pageSize int) (ListResult[api.Inbox], error) {
			inboxes, err := client.Inboxes().List(ctx)
			if err != nil {
				return ListResult[api.Inbox]{}, err
			}
			return ListResult[api.Inbox]{Items: inboxes, HasMore: false}, nil
		},
	}

	cmd := NewListCommand(cfg, func(ctx context.Context) (*api.Client, error) {
		return getClient()
	})
	cmd.Aliases = []string{"ls"}

	cmd.Flags().BoolVar(&light, "light", false, "Return minimal inbox payload for lookup")
	flagAlias(cmd.Flags(), "light", "li")

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name", "channel_type"},
		"default": {"id", "name", "channel_type", "website_url", "greeting_enabled"},
		"debug":   {"id", "name", "channel_type", "avatar_url", "website_url", "greeting_enabled", "greeting_message", "enable_auto_assignment"},
	})
	registerFieldSchema(cmd, "inbox")

	return cmd
}

func newInboxesGetCmd() *cobra.Command {
	var emit string

	cmd := &cobra.Command{
		Use:     "get <id>",
		Aliases: []string{"g"},
		Short:   "Get inbox details",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "inbox")
			if err != nil {
				return err
			}

			mode, err := normalizeEmitFlag(emit)
			if err != nil {
				return err
			}
			if mode == "id" || mode == "url" {
				_, err := maybeEmit(cmd, mode, "inbox", id, nil)
				return err
			}

			if handled, err := handleURLFlag(cmd, "inboxes", id); handled {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			inbox, err := client.Inboxes().Get(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if emitted, err := maybeEmit(cmd, emit, "inbox", inbox.ID, inbox); emitted {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, inbox)
			}
			return printInboxDetails(cmd.OutOrStdout(), inbox)
		}),
	}

	cmd.Flags().Bool("url", false, "Print the Chatwoot web UI URL for this resource and exit")
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name", "channel_type"},
		"default": {"id", "name", "channel_type", "website_url", "greeting_enabled"},
		"debug":   {"id", "name", "channel_type", "avatar_url", "website_url", "greeting_enabled", "greeting_message", "enable_auto_assignment"},
	})
	registerFieldSchema(cmd, "inbox")

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
		emit                       string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"mk"},
		Short:   "Create a new inbox",
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

			inbox, err := client.Inboxes().Create(cmdContext(cmd), req)
			if err != nil {
				return err
			}

			if emitted, err := maybeEmit(cmd, emit, "inbox", inbox.ID, inbox); emitted {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, inbox)
			}

			printAction(cmd, "Created", "inbox", inbox.ID, fmt.Sprintf("%s (%s)", inbox.Name, inbox.ChannelType))
			return nil
		}),
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Inbox name (required)")
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
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")
	flagAlias(cmd.Flags(), "channel-type", "chn")
	flagAlias(cmd.Flags(), "greeting-enabled", "ge")
	flagAlias(cmd.Flags(), "greeting-message", "gm")
	flagAlias(cmd.Flags(), "enable-email-collect", "eec")
	flagAlias(cmd.Flags(), "csat-survey-enabled", "cse")
	flagAlias(cmd.Flags(), "enable-auto-assignment", "eaa")
	flagAlias(cmd.Flags(), "auto-assignment-config", "aac")
	flagAlias(cmd.Flags(), "working-hours-enabled", "whe")
	flagAlias(cmd.Flags(), "timezone", "tz")
	flagAlias(cmd.Flags(), "allow-messages-after-resolved", "mar")
	flagAlias(cmd.Flags(), "lock-to-single-conversation", "ltsc")
	flagAlias(cmd.Flags(), "portal-id", "pid")
	flagAlias(cmd.Flags(), "sender-name-type", "snt")
	flagAlias(cmd.Flags(), "out-of-office-message", "oom")
	flagAlias(cmd.Flags(), "out-of-office-enabled", "ooe")
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
		emit                       string
	)

	cmd := &cobra.Command{
		Use:     "update <id>",
		Aliases: []string{"up"},
		Short:   "Update an inbox",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "inbox")
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

			inbox, err := client.Inboxes().Update(cmdContext(cmd), id, req)
			if err != nil {
				return err
			}

			if emitted, err := maybeEmit(cmd, emit, "inbox", inbox.ID, inbox); emitted {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, inbox)
			}

			printAction(cmd, "Updated", "inbox", inbox.ID, inbox.Name)
			return nil
		}),
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Inbox name")
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
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")
	flagAlias(cmd.Flags(), "greeting-enabled", "ge")
	flagAlias(cmd.Flags(), "greeting-message", "gm")
	flagAlias(cmd.Flags(), "enable-email-collect", "eec")
	flagAlias(cmd.Flags(), "csat-survey-enabled", "cse")
	flagAlias(cmd.Flags(), "enable-auto-assignment", "eaa")
	flagAlias(cmd.Flags(), "auto-assignment-config", "aac")
	flagAlias(cmd.Flags(), "working-hours-enabled", "whe")
	flagAlias(cmd.Flags(), "timezone", "tz")
	flagAlias(cmd.Flags(), "allow-messages-after-resolved", "mar")
	flagAlias(cmd.Flags(), "lock-to-single-conversation", "ltsc")
	flagAlias(cmd.Flags(), "portal-id", "pid")
	flagAlias(cmd.Flags(), "sender-name-type", "snt")
	flagAlias(cmd.Flags(), "out-of-office-message", "oom")
	flagAlias(cmd.Flags(), "out-of-office-enabled", "ooe")

	return cmd
}

func newInboxesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm"},
		Short:   "Delete an inbox",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "inbox")
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

			if err := client.Inboxes().Delete(cmdContext(cmd), id); err != nil {
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
		Use:     "agent-bot <id>",
		Aliases: []string{"bot"},
		Short:   "Get the agent bot assigned to an inbox",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "inbox")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			bot, err := client.Inboxes().GetAgentBot(cmdContext(cmd), id)
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
		Use:     "set-agent-bot <id>",
		Aliases: []string{"sab"},
		Short:   "Assign an agent bot to an inbox",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "inbox")
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

			if err := client.Inboxes().SetAgentBot(cmdContext(cmd), id, botID); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Assigned agent bot %d to inbox %d\n", botID, id)
			return nil
		}),
	}

	cmd.Flags().IntVar(&botID, "bot-id", 0, "Agent bot ID (required)")
	flagAlias(cmd.Flags(), "bot-id", "bid")
	_ = cmd.MarkFlagRequired("bot-id")

	return cmd
}

// inboxTriageItem represents a single inbox in the cross-inbox triage overview
type inboxTriageItem struct {
	InboxID        int    `json:"inbox_id"`
	InboxName      string `json:"inbox_name"`
	OpenCount      int    `json:"open_count"`
	PendingCount   int    `json:"pending_count"`
	UnreadCount    int    `json:"unread_count"`
	OldestWaiting  string `json:"oldest_waiting,omitempty"`
	OldestContact  string `json:"oldest_contact,omitempty"`
	OldestWaitTime string `json:"oldest_wait_time,omitempty"`
}

func newInboxesTriageCmd() *cobra.Command {
	var (
		status string
		limit  int
	)

	cmd := &cobra.Command{
		Use:   "triage [id]",
		Short: "Get conversations with enriched context for triage",
		Long: `Returns conversations for an inbox with contact info and last message for decision-making.

When called without an inbox ID, shows a cross-inbox overview of all inboxes
sorted by urgency (highest unread first, then most open).

When called with an inbox ID, shows detailed triage info for that specific inbox.`,
		Args: cobra.MaximumNArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			// Cross-inbox mode: no ID provided
			if len(args) == 0 {
				return runCrossInboxTriage(cmd, client, ctx)
			}

			// Single inbox mode
			id, err := parseIDOrURL(args[0], "inbox")
			if err != nil {
				return err
			}

			triage, err := client.Inboxes().Triage(ctx, id, status, limit)
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
	flagAlias(cmd.Flags(), "status", "st")
	flagAlias(cmd.Flags(), "limit", "lt")

	return cmd
}

// runCrossInboxTriage shows a health overview of all inboxes sorted by urgency
func runCrossInboxTriage(cmd *cobra.Command, client *api.Client, ctx context.Context) error {
	// Fetch all inboxes
	inboxes, err := client.Inboxes().List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list inboxes: %w", err)
	}

	if len(inboxes) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No inboxes found")
		return nil
	}

	// Fetch all conversations to calculate stats per inbox
	convResult, err := client.Conversations().List(ctx, api.ListConversationsParams{
		Status: "all",
	})
	if err != nil {
		return fmt.Errorf("failed to list conversations: %w", err)
	}

	// Build stats per inbox
	inboxStats := make(map[int]*inboxTriageItem)
	for _, inbox := range inboxes {
		inboxStats[inbox.ID] = &inboxTriageItem{
			InboxID:   inbox.ID,
			InboxName: inbox.Name,
		}
	}

	now := time.Now().Unix()
	// Track oldest waiting conversation per inbox
	oldestWaiting := make(map[int]*api.Conversation)

	for i := range convResult.Data.Payload {
		conv := &convResult.Data.Payload[i]
		stats, ok := inboxStats[conv.InboxID]
		if !ok {
			continue // conversation belongs to an inbox not in our list
		}

		switch conv.Status {
		case "open":
			stats.OpenCount++
		case "pending":
			stats.PendingCount++
		}

		if conv.Unread > 0 {
			stats.UnreadCount += conv.Unread
		}

		// Track oldest waiting (open or pending with unread)
		if (conv.Status == "open" || conv.Status == "pending") && conv.Unread > 0 {
			existing := oldestWaiting[conv.InboxID]
			if existing == nil || conv.LastActivityAt < existing.LastActivityAt {
				oldestWaiting[conv.InboxID] = conv
			}
		}
	}

	// Calculate wait times for oldest waiting conversations
	for inboxID, conv := range oldestWaiting {
		stats := inboxStats[inboxID]
		if conv.LastActivityAt > 0 {
			waitSeconds := now - conv.LastActivityAt
			stats.OldestWaitTime = formatDuration(waitSeconds)
			stats.OldestWaiting = fmt.Sprintf("#%d", conv.ID)
			// We don't have contact info readily available, leave OldestContact empty
		}
	}

	// Build result slice and sort by urgency: highest unread first, then most open
	items := make([]inboxTriageItem, 0, len(inboxes))
	for _, inbox := range inboxes {
		items = append(items, *inboxStats[inbox.ID])
	}

	// Sort by unread (desc), then open (desc)
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			// Higher unread first
			if items[j].UnreadCount > items[i].UnreadCount {
				items[i], items[j] = items[j], items[i]
			} else if items[j].UnreadCount == items[i].UnreadCount && items[j].OpenCount > items[i].OpenCount {
				// Same unread, higher open first
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	if isJSON(cmd) {
		return printJSON(cmd, map[string]any{"items": items})
	}

	// Text output
	w := newTabWriterFromCmd(cmd)
	_, _ = fmt.Fprintln(w, "INBOX\tOPEN\tPENDING\tUNREAD\tOLDEST WAITING")
	for _, item := range items {
		oldestInfo := "-"
		if item.OldestWaiting != "" {
			oldestInfo = fmt.Sprintf("%s (%s)", item.OldestWaiting, item.OldestWaitTime)
		}
		_, _ = fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%s\n",
			item.InboxName, item.OpenCount, item.PendingCount, item.UnreadCount, oldestInfo)
	}
	_ = w.Flush()

	return nil
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
			id, err := parseIDOrURL(args[0], "inbox")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			campaigns, err := client.Inboxes().Campaigns(cmdContext(cmd), id)
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
		Use:     "sync-templates <id>",
		Aliases: []string{"sync"},
		Short:   "Sync WhatsApp templates for an inbox",
		Long:    "Sync WhatsApp message templates from the WhatsApp Business API for this inbox",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "inbox")
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

			if err := client.Inboxes().SyncTemplates(cmdContext(cmd), id); err != nil {
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
			id, err := parseIDOrURL(args[0], "inbox")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			health, err := client.Inboxes().Health(cmdContext(cmd), id)
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
		Use:     "delete-avatar <id>",
		Aliases: []string{"da"},
		Short:   "Remove the inbox avatar",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "inbox")
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

			if err := client.Inboxes().DeleteAvatar(cmdContext(cmd), id); err != nil {
				return err
			}

			printAction(cmd, "Deleted", "inbox avatar", id, "")
			return nil
		}),
	}
}

func newInboxesCSATTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "csat-template <id>",
		Aliases: []string{"cst"},
		Short:   "Get or set CSAT survey template for an inbox",
		Args:    cobra.ExactArgs(1),
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

func newInboxesStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats <id>",
		Short: "Get inbox statistics",
		Long:  "Returns inbox health metrics: open count, pending count, unread messages, average wait time",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "inbox")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			// Get inbox info
			inbox, err := client.Inboxes().Get(ctx, id)
			if err != nil {
				return fmt.Errorf("failed to get inbox: %w", err)
			}

			// Get conversations for this inbox
			result, err := client.Conversations().List(ctx, api.ListConversationsParams{
				InboxID: fmt.Sprintf("%d", id),
				Status:  "all",
			})
			if err != nil {
				return fmt.Errorf("failed to list conversations: %w", err)
			}

			// Calculate stats
			var openCount, pendingCount, totalUnread int
			var totalWaitTime int64
			var waitingCount int
			now := time.Now().Unix()

			for _, conv := range result.Data.Payload {
				switch conv.Status {
				case "open":
					openCount++
				case "pending":
					pendingCount++
				}
				totalUnread += conv.Unread

				// Calculate wait time for open/pending conversations
				// Use time since last activity as approximation
				if conv.Status == "open" || conv.Status == "pending" {
					if conv.LastActivityAt > 0 {
						totalWaitTime += now - conv.LastActivityAt
						waitingCount++
					}
				}
			}

			avgWaitSeconds := int64(0)
			if waitingCount > 0 {
				avgWaitSeconds = totalWaitTime / int64(waitingCount)
			}

			stats := map[string]any{
				"inbox_id":         inbox.ID,
				"inbox_name":       inbox.Name,
				"open_count":       openCount,
				"pending_count":    pendingCount,
				"unread_count":     totalUnread,
				"avg_wait_seconds": avgWaitSeconds,
				"waiting_count":    waitingCount,
			}

			if isJSON(cmd) {
				return printJSON(cmd, stats)
			}

			// Text output
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Inbox: %s (ID: %d)\n", inbox.Name, inbox.ID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Open: %d | Pending: %d | Unread: %d\n", openCount, pendingCount, totalUnread)
			if waitingCount > 0 {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Avg wait: %s (%d waiting)\n", formatDuration(avgWaitSeconds), waitingCount)
			}
			return nil
		}),
	}
	return cmd
}

func formatDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	hours := seconds / 3600
	mins := (seconds % 3600) / 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, mins)
}

func newInboxesCSATTemplateGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get <inbox-id>",
		Aliases: []string{"g"},
		Short:   "Get the CSAT survey template for an inbox",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "inbox")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			template, err := client.Inboxes().CSATTemplate(cmdContext(cmd), id)
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
			id, err := parseIDOrURL(args[0], "inbox")
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

			template, err := client.Inboxes().CreateCSATTemplate(cmdContext(cmd), id, question, message)
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
	cmd.Flags().StringVarP(&message, "message", "m", "", "Survey message (required)")
	flagAlias(cmd.Flags(), "question", "qu")
	_ = cmd.MarkFlagRequired("question")
	_ = cmd.MarkFlagRequired("message")

	return cmd
}
