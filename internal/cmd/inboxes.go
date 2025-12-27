package cmd

import (
	"context"
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
				return printJSON(inbox)
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
		name        string
		channelType string
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

			inbox, err := client.CreateInbox(cmdContext(cmd), name, channelType)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(inbox)
			}

			fmt.Printf("Created inbox %d: %s (%s)\n", inbox.ID, inbox.Name, inbox.ChannelType)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Inbox name (required)")
	cmd.Flags().StringVar(&channelType, "channel-type", "", "Channel type (required)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("channel-type")

	return cmd
}

func newInboxesUpdateCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an inbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			if name == "" {
				return fmt.Errorf("at least one field must be provided to update")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			inbox, err := client.UpdateInbox(cmdContext(cmd), id, name)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(inbox)
			}

			fmt.Printf("Updated inbox %d: %s\n", inbox.ID, inbox.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Inbox name")

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
				return printJSON(bot)
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
