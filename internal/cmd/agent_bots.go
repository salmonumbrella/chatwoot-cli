package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
)

func newAgentBotsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agent-bots",
		Aliases: []string{"bots", "ab"},
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
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all agent bots",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			bots, err := client.AgentBots().List(cmdContext(cmd))
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, bots)
			}

			if len(bots) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No agent bots found")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tNAME\tOUTGOING_URL")
			for _, bot := range bots {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\n", bot.ID, bot.Name, bot.OutgoingURL)
			}
			return nil
		}),
	}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name"},
		"default": {"id", "name", "outgoing_url"},
		"debug":   {"id", "name", "description", "outgoing_url", "account_id"},
	})

	return cmd
}

func newAgentBotsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <id>",
		Aliases: []string{"g"},
		Short:   "Get agent bot by ID",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			id, err := parseIDOrURL(args[0], "bot")
			if err != nil {
				return err
			}

			bot, err := client.AgentBots().Get(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, bot)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ID:           %d\n", bot.ID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Name:         %s\n", bot.Name)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Description:  %s\n", bot.Description)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Outgoing URL: %s\n", bot.OutgoingURL)
			return nil
		}),
	}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name"},
		"default": {"id", "name", "outgoing_url"},
		"debug":   {"id", "name", "description", "outgoing_url", "account_id"},
	})

	return cmd
}

func newAgentBotsCreateCmd() *cobra.Command {
	var (
		name        string
		outgoingURL string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"mk"},
		Short:   "Create a new agent bot",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
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

			bot, err := client.AgentBots().Create(cmdContext(cmd), name, outgoingURL)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, bot)
			}

			printAction(cmd, "Created", "agent bot", bot.ID, bot.Name)
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Bot name (required)")
	cmd.Flags().StringVar(&outgoingURL, "outgoing-url", "", "Outgoing webhook URL (required)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("outgoing-url")
	flagAlias(cmd.Flags(), "name", "nm")
	flagAlias(cmd.Flags(), "outgoing-url", "ou")

	return cmd
}

func newAgentBotsUpdateCmd() *cobra.Command {
	var (
		name        string
		outgoingURL string
	)

	cmd := &cobra.Command{
		Use:     "update <id>",
		Aliases: []string{"up"},
		Short:   "Update an agent bot",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
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

			id, err := parseIDOrURL(args[0], "bot")
			if err != nil {
				return err
			}

			bot, err := client.AgentBots().Update(cmdContext(cmd), id, name, outgoingURL)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, bot)
			}

			printAction(cmd, "Updated", "agent bot", bot.ID, bot.Name)
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Bot name")
	cmd.Flags().StringVar(&outgoingURL, "outgoing-url", "", "Outgoing webhook URL")
	flagAlias(cmd.Flags(), "name", "nm")
	flagAlias(cmd.Flags(), "outgoing-url", "ou")

	return cmd
}

func newAgentBotsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm"},
		Short:   "Delete an agent bot",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			id, err := parseIDOrURL(args[0], "bot")
			if err != nil {
				return err
			}

			if err := client.AgentBots().Delete(cmdContext(cmd), id); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": id})
			}

			printAction(cmd, "Deleted", "agent bot", id, "")
			return nil
		}),
	}
}

func newAgentBotsDeleteAvatarCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete-avatar <id>",
		Aliases: []string{"da"},
		Short:   "Remove the avatar from an agent bot",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "bot")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.AgentBots().DeleteAvatar(cmdContext(cmd), id); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": id})
			}

			printAction(cmd, "Deleted", "agent bot avatar", id, "")
			return nil
		}),
	}
}

func newAgentBotsResetTokenCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "reset-token <id>",
		Aliases: []string{"rt"},
		Short:   "Reset the access token for an agent bot",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "bot")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			token, err := client.AgentBots().ResetAccessToken(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"access_token": token})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "New access token for agent bot #%d: %s\n", id, token)
			return nil
		}),
	}
}
