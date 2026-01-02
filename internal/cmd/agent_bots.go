package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
)

func newAgentBotsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agent-bots",
		Aliases: []string{"bots"},
		Short:   "Manage agent bots",
		Long:    "Create, list, update, and delete agent bots in your Chatwoot account",
	}

	cmd.AddCommand(newAgentBotsListCmd())
	cmd.AddCommand(newAgentBotsGetCmd())
	cmd.AddCommand(newAgentBotsCreateCmd())
	cmd.AddCommand(newAgentBotsUpdateCmd())
	cmd.AddCommand(newAgentBotsDeleteCmd())
	cmd.AddCommand(newAgentBotsDeleteAvatarCmd())
	cmd.AddCommand(newAgentBotsResetTokenCmd())

	return cmd
}

func newAgentBotsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all agent bots",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			bots, err := client.ListAgentBots(cmdContext(cmd))
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, bots)
			}

			if len(bots) == 0 {
				fmt.Println("No agent bots found")
				return nil
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tNAME\tOUTGOING_URL")
			for _, bot := range bots {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\n", bot.ID, bot.Name, bot.OutgoingURL)
			}
			return nil
		},
	}
}

func newAgentBotsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get agent bot by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return fmt.Errorf("invalid bot ID: %w", err)
			}

			bot, err := client.GetAgentBot(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, bot)
			}

			fmt.Printf("ID:           %d\n", bot.ID)
			fmt.Printf("Name:         %s\n", bot.Name)
			fmt.Printf("Description:  %s\n", bot.Description)
			fmt.Printf("Outgoing URL: %s\n", bot.OutgoingURL)
			return nil
		},
	}
}

func newAgentBotsCreateCmd() *cobra.Command {
	var (
		name        string
		outgoingURL string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new agent bot",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate outgoing URL if provided
			if outgoingURL != "" {
				if err := validation.ValidateWebhookURL(outgoingURL); err != nil {
					return fmt.Errorf("invalid outgoing URL: %w", err)
				}
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			bot, err := client.CreateAgentBot(cmdContext(cmd), name, outgoingURL)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, bot)
			}

			fmt.Printf("Created agent bot #%d: %s\n", bot.ID, bot.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Bot name (required)")
	cmd.Flags().StringVar(&outgoingURL, "outgoing-url", "", "Outgoing webhook URL (required)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("outgoing-url")

	return cmd
}

func newAgentBotsUpdateCmd() *cobra.Command {
	var (
		name        string
		outgoingURL string
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an agent bot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate outgoing URL if provided
			if outgoingURL != "" {
				if err := validation.ValidateWebhookURL(outgoingURL); err != nil {
					return fmt.Errorf("invalid outgoing URL: %w", err)
				}
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return fmt.Errorf("invalid bot ID: %w", err)
			}

			bot, err := client.UpdateAgentBot(cmdContext(cmd), id, name, outgoingURL)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, bot)
			}

			fmt.Printf("Updated agent bot #%d: %s\n", bot.ID, bot.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Bot name")
	cmd.Flags().StringVar(&outgoingURL, "outgoing-url", "", "Outgoing webhook URL")

	return cmd
}

func newAgentBotsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an agent bot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return fmt.Errorf("invalid bot ID: %w", err)
			}

			if err := client.DeleteAgentBot(cmdContext(cmd), id); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": id})
			}

			fmt.Printf("Deleted agent bot #%d\n", id)
			return nil
		},
	}
}

func newAgentBotsDeleteAvatarCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete-avatar <id>",
		Short: "Remove the avatar from an agent bot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return fmt.Errorf("invalid bot ID: %w", err)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.DeleteAgentBotAvatar(cmdContext(cmd), id); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": id})
			}

			fmt.Printf("Deleted avatar for agent bot #%d\n", id)
			return nil
		},
	}
}

func newAgentBotsResetTokenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset-token <id>",
		Short: "Reset the access token for an agent bot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return fmt.Errorf("invalid bot ID: %w", err)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			token, err := client.ResetAgentBotAccessToken(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"access_token": token})
			}

			fmt.Printf("New access token for agent bot #%d: %s\n", id, token)
			return nil
		},
	}
}
