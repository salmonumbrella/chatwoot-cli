package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chatwoot/chatwoot-cli/internal/api"
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

	return NewListCommand(cfg, func(ctx context.Context) (*api.Client, error) {
		return getClient()
	})
}

func newInboxesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get inbox details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			w := newTabWriter()
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
		},
	}
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
		RunE: func(cmd *cobra.Command, args []string) error {
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

			if cmd.Flags().Changed("greeting-enabled") {
				req.GreetingEnabled = &greetingEnabled
			}
			if cmd.Flags().Changed("enable-email-collect") {
				req.EnableEmailCollect = &enableEmailCollect
			}
			if cmd.Flags().Changed("csat-survey-enabled") {
				req.CSATSurveyEnabled = &csatSurveyEnabled
			}
			if cmd.Flags().Changed("enable-auto-assignment") {
				req.EnableAutoAssignment = &enableAutoAssignment
			}
			if cmd.Flags().Changed("working-hours-enabled") {
				req.WorkingHoursEnabled = &workingHoursEnabled
			}
			if cmd.Flags().Changed("allow-messages-after-resolved") {
				req.AllowMessagesAfterResolved = &allowMessagesAfterResolved
			}
			if cmd.Flags().Changed("lock-to-single-conversation") {
				req.LockToSingleConversation = &lockToSingleConversation
			}
			if cmd.Flags().Changed("out-of-office-enabled") {
				req.OutOfOfficeEnabled = &outOfOfficeEnabled
			}
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

			inbox, err := client.CreateInbox(cmdContext(cmd), req)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, inbox)
			}

			fmt.Printf("Created inbox %d: %s (%s)\n", inbox.ID, inbox.Name, inbox.ChannelType)
			return nil
		},
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
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			if name == "" &&
				!cmd.Flags().Changed("greeting-enabled") &&
				greetingMessage == "" &&
				!cmd.Flags().Changed("enable-email-collect") &&
				!cmd.Flags().Changed("csat-survey-enabled") &&
				!cmd.Flags().Changed("enable-auto-assignment") &&
				autoAssignmentConfig == "" &&
				!cmd.Flags().Changed("working-hours-enabled") &&
				timezone == "" &&
				!cmd.Flags().Changed("allow-messages-after-resolved") &&
				!cmd.Flags().Changed("lock-to-single-conversation") &&
				portalID == 0 &&
				senderNameType == "" &&
				outOfOfficeMessage == "" &&
				!cmd.Flags().Changed("out-of-office-enabled") {
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
			if cmd.Flags().Changed("greeting-enabled") {
				req.GreetingEnabled = &greetingEnabled
			}
			if cmd.Flags().Changed("enable-email-collect") {
				req.EnableEmailCollect = &enableEmailCollect
			}
			if cmd.Flags().Changed("csat-survey-enabled") {
				req.CSATSurveyEnabled = &csatSurveyEnabled
			}
			if cmd.Flags().Changed("enable-auto-assignment") {
				req.EnableAutoAssignment = &enableAutoAssignment
			}
			if cmd.Flags().Changed("working-hours-enabled") {
				req.WorkingHoursEnabled = &workingHoursEnabled
			}
			if cmd.Flags().Changed("allow-messages-after-resolved") {
				req.AllowMessagesAfterResolved = &allowMessagesAfterResolved
			}
			if cmd.Flags().Changed("lock-to-single-conversation") {
				req.LockToSingleConversation = &lockToSingleConversation
			}
			if cmd.Flags().Changed("out-of-office-enabled") {
				req.OutOfOfficeEnabled = &outOfOfficeEnabled
			}
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

			inbox, err := client.UpdateInbox(cmdContext(cmd), id, req)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, inbox)
			}

			fmt.Printf("Updated inbox %d: %s\n", inbox.ID, inbox.Name)
			return nil
		},
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
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.DeleteInbox(cmdContext(cmd), id); err != nil {
				return err
			}

			fmt.Printf("Deleted inbox %d\n", id)
			return nil
		},
	}
}

func newInboxesAgentBotCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agent-bot <id>",
		Short: "Get the agent bot assigned to an inbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			w := newTabWriter()
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
		},
	}
}

func newInboxesSetAgentBotCmd() *cobra.Command {
	var botID int

	cmd := &cobra.Command{
		Use:   "set-agent-bot <id>",
		Short: "Assign an agent bot to an inbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			if err := client.SetInboxAgentBot(cmdContext(cmd), id, botID); err != nil {
				return err
			}

			fmt.Printf("Assigned agent bot %d to inbox %d\n", botID, id)
			return nil
		},
	}

	cmd.Flags().IntVar(&botID, "bot-id", 0, "Agent bot ID (required)")
	_ = cmd.MarkFlagRequired("bot-id")

	return cmd
}
