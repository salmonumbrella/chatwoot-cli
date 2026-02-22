package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/spf13/cobra"
)

func newPlatformCmd() *cobra.Command {
	var baseURL string
	var token string

	cmd := &cobra.Command{
		Use:     "platform",
		Aliases: []string{"pf"},
		Short:   "Manage Chatwoot via platform APIs",
		Long:    "Use platform APIs to manage accounts and users (requires platform token).",
	}

	cmd.PersistentFlags().StringVar(&baseURL, "base-url", "", "Override Chatwoot base URL")
	cmd.PersistentFlags().StringVar(&token, "token", "", "Platform API token (overrides env/config)")
	flagAlias(cmd.PersistentFlags(), "base-url", "bu")
	flagAlias(cmd.PersistentFlags(), "token", "tk")

	cmd.AddCommand(newPlatformAccountsCmd(&baseURL, &token))
	cmd.AddCommand(newPlatformUsersCmd(&baseURL, &token))
	cmd.AddCommand(newPlatformAccountUsersCmd(&baseURL, &token))
	cmd.AddCommand(newPlatformAgentBotsCmd(&baseURL, &token))

	return cmd
}

func newPlatformAccountsCmd(baseURL, token *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Manage accounts via platform API",
	}

	cmd.AddCommand(newPlatformAccountsCreateCmd(baseURL, token))
	cmd.AddCommand(newPlatformAccountsGetCmd(baseURL, token))
	cmd.AddCommand(newPlatformAccountsUpdateCmd(baseURL, token))
	cmd.AddCommand(newPlatformAccountsDeleteCmd(baseURL, token))

	return cmd
}

func newPlatformAccountsCreateCmd(baseURL, token *string) *cobra.Command {
	var (
		name             string
		locale           string
		domain           string
		supportEmail     string
		status           string
		customAttributes string
		limits           string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"mk"},
		Short:   "Create an account",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			var attrs map[string]any
			if customAttributes != "" {
				if err := json.Unmarshal([]byte(customAttributes), &attrs); err != nil {
					return fmt.Errorf("invalid custom-attributes JSON: %w", err)
				}
			}

			var limitsMap map[string]any
			if limits != "" {
				if err := json.Unmarshal([]byte(limits), &limitsMap); err != nil {
					return fmt.Errorf("invalid limits JSON: %w", err)
				}
			}

			account, err := client.Platform().CreateAccount(cmdContext(cmd), api.CreatePlatformAccountRequest{
				Name:             name,
				Locale:           locale,
				Domain:           domain,
				SupportEmail:     supportEmail,
				Status:           status,
				CustomAttributes: attrs,
				Limits:           limitsMap,
			})
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, account)
			}

			printAction(cmd, "Created", "account", account.ID, account.Name)
			return nil
		}),
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Account name (required)")
	cmd.Flags().StringVar(&locale, "locale", "", "Account locale")
	cmd.Flags().StringVar(&domain, "domain", "", "Account domain")
	cmd.Flags().StringVar(&supportEmail, "support-email", "", "Support email")
	cmd.Flags().StringVar(&status, "status", "", "Account status")
	cmd.Flags().StringVar(&customAttributes, "custom-attributes", "", "Custom attributes JSON")
	cmd.Flags().StringVar(&limits, "limits", "", "Limits JSON")
	flagAlias(cmd.Flags(), "locale", "lc")
	flagAlias(cmd.Flags(), "domain", "dom")
	flagAlias(cmd.Flags(), "support-email", "se")
	flagAlias(cmd.Flags(), "status", "st")
	flagAlias(cmd.Flags(), "custom-attributes", "ca")
	flagAlias(cmd.Flags(), "limits", "lim")

	return cmd
}

func newPlatformAccountsGetCmd(baseURL, token *string) *cobra.Command {
	return &cobra.Command{
		Use:     "get <account-id>",
		Aliases: []string{"g"},
		Short:   "Get an account",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			accountID, err := parseIDOrURL(args[0], "account")
			if err != nil {
				return err
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			account, err := client.Platform().GetAccount(cmdContext(cmd), accountID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, account)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Account %d: %s\n", account.ID, account.Name)
			return nil
		}),
	}
}

func newPlatformAccountsDeleteCmd(baseURL, token *string) *cobra.Command {
	return &cobra.Command{
		Use:     "delete <account-id>",
		Aliases: []string{"rm"},
		Short:   "Delete an account",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			accountID, err := parseIDOrURL(args[0], "account")
			if err != nil {
				return err
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			if err := client.Platform().DeleteAccount(cmdContext(cmd), accountID); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": accountID})
			}
			printAction(cmd, "Deleted", "account", accountID, "")
			return nil
		}),
	}
}

func newPlatformAccountsUpdateCmd(baseURL, token *string) *cobra.Command {
	var (
		name   string
		locale string
		domain string
		status string
	)

	cmd := &cobra.Command{
		Use:     "update <account-id>",
		Aliases: []string{"up"},
		Short:   "Update an account",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			accountID, err := parseIDOrURL(args[0], "account")
			if err != nil {
				return err
			}

			if name == "" && locale == "" && domain == "" && status == "" {
				return fmt.Errorf("at least one field must be provided to update")
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			account, err := client.Platform().UpdateAccount(cmdContext(cmd), accountID, api.UpdatePlatformAccountRequest{
				Name:   name,
				Locale: locale,
				Domain: domain,
				Status: status,
			})
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, account)
			}

			printAction(cmd, "Updated", "account", account.ID, account.Name)
			return nil
		}),
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Account name")
	cmd.Flags().StringVar(&locale, "locale", "", "Account locale")
	cmd.Flags().StringVar(&domain, "domain", "", "Account domain")
	cmd.Flags().StringVar(&status, "status", "", "Account status")
	flagAlias(cmd.Flags(), "locale", "lc")
	flagAlias(cmd.Flags(), "domain", "dom")
	flagAlias(cmd.Flags(), "status", "st")

	return cmd
}

func newPlatformUsersCmd(baseURL, token *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Manage users via platform API",
	}

	cmd.AddCommand(newPlatformUsersCreateCmd(baseURL, token))
	cmd.AddCommand(newPlatformUsersGetCmd(baseURL, token))
	cmd.AddCommand(newPlatformUsersUpdateCmd(baseURL, token))
	cmd.AddCommand(newPlatformUsersDeleteCmd(baseURL, token))
	cmd.AddCommand(newPlatformUsersLoginCmd(baseURL, token))

	return cmd
}

func newPlatformUsersCreateCmd(baseURL, token *string) *cobra.Command {
	var (
		name             string
		displayName      string
		email            string
		password         string
		customAttributes string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"mk"},
		Short:   "Create a user",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if name == "" || email == "" || password == "" {
				return fmt.Errorf("--name, --email, and --password are required")
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			var attrs map[string]any
			if customAttributes != "" {
				if err := json.Unmarshal([]byte(customAttributes), &attrs); err != nil {
					return fmt.Errorf("invalid custom-attributes JSON: %w", err)
				}
			}

			user, err := client.Platform().CreateUser(cmdContext(cmd), api.CreatePlatformUserRequest{
				Name:             name,
				DisplayName:      displayName,
				Email:            email,
				Password:         password,
				CustomAttributes: attrs,
			})
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, user)
			}

			printAction(cmd, "Created", "user", user.ID, user.Email)
			return nil
		}),
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "User name (required)")
	cmd.Flags().StringVar(&displayName, "display-name", "", "User display name")
	cmd.Flags().StringVar(&email, "email", "", "User email (required)")
	cmd.Flags().StringVar(&password, "password", "", "User password (required)")
	cmd.Flags().StringVar(&customAttributes, "custom-attributes", "", "Custom attributes JSON")
	flagAlias(cmd.Flags(), "display-name", "dn")
	flagAlias(cmd.Flags(), "email", "em")
	flagAlias(cmd.Flags(), "custom-attributes", "ca")
	flagAlias(cmd.Flags(), "password", "pw")

	return cmd
}

func newPlatformUsersGetCmd(baseURL, token *string) *cobra.Command {
	return &cobra.Command{
		Use:     "get <user-id>",
		Aliases: []string{"g"},
		Short:   "Get a user",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			userID, err := parseIDOrURL(args[0], "user")
			if err != nil {
				return err
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			user, err := client.Platform().GetUser(cmdContext(cmd), userID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, user)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "User %d: %s\n", user.ID, user.Email)
			return nil
		}),
	}
}

func newPlatformUsersUpdateCmd(baseURL, token *string) *cobra.Command {
	var (
		name             string
		displayName      string
		email            string
		password         string
		customAttributes string
	)

	cmd := &cobra.Command{
		Use:     "update <user-id>",
		Aliases: []string{"up"},
		Short:   "Update a user",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			userID, err := parseIDOrURL(args[0], "user")
			if err != nil {
				return err
			}

			if name == "" && displayName == "" && email == "" && password == "" && customAttributes == "" {
				return fmt.Errorf("at least one field must be provided to update")
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			var attrs map[string]any
			if customAttributes != "" {
				if err := json.Unmarshal([]byte(customAttributes), &attrs); err != nil {
					return fmt.Errorf("invalid custom-attributes JSON: %w", err)
				}
			}

			user, err := client.Platform().UpdateUser(cmdContext(cmd), userID, api.UpdatePlatformUserRequest{
				Name:             name,
				DisplayName:      displayName,
				Email:            email,
				Password:         password,
				CustomAttributes: attrs,
			})
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, user)
			}

			printAction(cmd, "Updated", "user", user.ID, user.Email)
			return nil
		}),
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "User name")
	cmd.Flags().StringVar(&displayName, "display-name", "", "User display name")
	cmd.Flags().StringVar(&email, "email", "", "User email")
	cmd.Flags().StringVar(&password, "password", "", "User password")
	cmd.Flags().StringVar(&customAttributes, "custom-attributes", "", "Custom attributes JSON")
	flagAlias(cmd.Flags(), "display-name", "dn")
	flagAlias(cmd.Flags(), "email", "em")
	flagAlias(cmd.Flags(), "custom-attributes", "ca")
	flagAlias(cmd.Flags(), "password", "pw")

	return cmd
}

func newPlatformUsersDeleteCmd(baseURL, token *string) *cobra.Command {
	return &cobra.Command{
		Use:     "delete <user-id>",
		Aliases: []string{"rm"},
		Short:   "Delete a user",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			userID, err := parseIDOrURL(args[0], "user")
			if err != nil {
				return err
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			if err := client.Platform().DeleteUser(cmdContext(cmd), userID); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": userID})
			}
			printAction(cmd, "Deleted", "user", userID, "")
			return nil
		}),
	}
}

func newPlatformUsersLoginCmd(baseURL, token *string) *cobra.Command {
	return &cobra.Command{
		Use:   "login <user-id>",
		Short: "Get SSO login URL for a user",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			userID, err := parseIDOrURL(args[0], "user")
			if err != nil {
				return err
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			login, err := client.Platform().GetUserLogin(cmdContext(cmd), userID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, login)
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), login.URL)
			return nil
		}),
	}
}

func newPlatformAccountUsersCmd(baseURL, token *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account-users",
		Short: "Manage account users via platform API",
	}

	cmd.AddCommand(newPlatformAccountUsersListCmd(baseURL, token))
	cmd.AddCommand(newPlatformAccountUsersCreateCmd(baseURL, token))
	cmd.AddCommand(newPlatformAccountUsersDeleteCmd(baseURL, token))

	return cmd
}

func newPlatformAccountUsersListCmd(baseURL, token *string) *cobra.Command {
	return &cobra.Command{
		Use:     "list <account-id>",
		Aliases: []string{"ls"},
		Short:   "List account users",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			accountID, err := parseIDOrURL(args[0], "account")
			if err != nil {
				return err
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			users, err := client.Platform().ListAccountUsers(cmdContext(cmd), accountID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, users)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tUSER\tROLE")
			for _, user := range users {
				_, _ = fmt.Fprintf(w, "%d\t%d\t%s\n", user.ID, user.UserID, user.Role)
			}
			return nil
		}),
	}
}

func newPlatformAccountUsersCreateCmd(baseURL, token *string) *cobra.Command {
	var userID int
	var role string

	cmd := &cobra.Command{
		Use:     "create <account-id>",
		Aliases: []string{"mk"},
		Short:   "Add a user to an account",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			accountID, err := parseIDOrURL(args[0], "account")
			if err != nil {
				return err
			}
			if userID == 0 {
				return fmt.Errorf("--user-id is required")
			}
			if role == "" {
				return fmt.Errorf("--role is required")
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			accountUser, err := client.Platform().CreateAccountUser(cmdContext(cmd), accountID, api.CreatePlatformAccountUserRequest{
				UserID: userID,
				Role:   role,
			})
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, accountUser)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added user %d to account %d\n", userID, accountID)
			return nil
		}),
	}

	cmd.Flags().IntVar(&userID, "user-id", 0, "User ID (required)")
	cmd.Flags().StringVar(&role, "role", "", "Role (required)")
	flagAlias(cmd.Flags(), "user-id", "uid")
	flagAlias(cmd.Flags(), "role", "rl")

	return cmd
}

func newPlatformAccountUsersDeleteCmd(baseURL, token *string) *cobra.Command {
	var userID int

	cmd := &cobra.Command{
		Use:     "delete <account-id>",
		Aliases: []string{"rm"},
		Short:   "Remove a user from an account",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			accountID, err := parseIDOrURL(args[0], "account")
			if err != nil {
				return err
			}
			if userID == 0 {
				return fmt.Errorf("--user-id is required")
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			if err := client.Platform().DeleteAccountUser(cmdContext(cmd), accountID, userID); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "account_id": accountID, "user_id": userID})
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed user %d from account %d\n", userID, accountID)
			return nil
		}),
	}

	cmd.Flags().IntVar(&userID, "user-id", 0, "User ID (required)")
	flagAlias(cmd.Flags(), "user-id", "uid")

	return cmd
}

func newPlatformAgentBotsCmd(baseURL, token *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agent-bots",
		Aliases: []string{"ab"},
		Short:   "Manage agent bots via platform API",
	}

	cmd.AddCommand(newPlatformAgentBotsListCmd(baseURL, token))
	cmd.AddCommand(newPlatformAgentBotsGetCmd(baseURL, token))
	cmd.AddCommand(newPlatformAgentBotsCreateCmd(baseURL, token))
	cmd.AddCommand(newPlatformAgentBotsUpdateCmd(baseURL, token))
	cmd.AddCommand(newPlatformAgentBotsDeleteCmd(baseURL, token))

	return cmd
}

func newPlatformAgentBotsListCmd(baseURL, token *string) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all platform agent bots",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			bots, err := client.PlatformAgentBots().List(cmdContext(cmd))
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, bots)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tNAME\tTYPE\tURL")
			for _, bot := range bots {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", bot.ID, bot.Name, bot.BotType, bot.OutgoingURL)
			}
			return nil
		}),
	}
}

func newPlatformAgentBotsGetCmd(baseURL, token *string) *cobra.Command {
	return &cobra.Command{
		Use:     "get <bot-id>",
		Aliases: []string{"g"},
		Short:   "Get a platform agent bot",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			botID, err := parseIDOrURL(args[0], "bot")
			if err != nil {
				return err
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			bot, err := client.PlatformAgentBots().Get(cmdContext(cmd), botID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, bot)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Agent Bot %d: %s\n", bot.ID, bot.Name)
			if bot.Description != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Description: %s\n", bot.Description)
			}
			if bot.BotType != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Type: %s\n", bot.BotType)
			}
			if bot.OutgoingURL != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "URL: %s\n", bot.OutgoingURL)
			}
			return nil
		}),
	}
}

func newPlatformAgentBotsCreateCmd(baseURL, token *string) *cobra.Command {
	var (
		name        string
		description string
		outgoingURL string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"mk"},
		Short:   "Create a platform agent bot",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			bot, err := client.PlatformAgentBots().Create(cmdContext(cmd), api.CreatePlatformAgentBotRequest{
				Name:        name,
				Description: description,
				OutgoingURL: outgoingURL,
			})
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

	cmd.Flags().StringVarP(&name, "name", "n", "", "Agent bot name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Agent bot description")
	flagAlias(cmd.Flags(), "description", "desc")
	cmd.Flags().StringVar(&outgoingURL, "outgoing-url", "", "Webhook URL for bot events")
	flagAlias(cmd.Flags(), "outgoing-url", "ou")

	return cmd
}

func newPlatformAgentBotsUpdateCmd(baseURL, token *string) *cobra.Command {
	var (
		name        string
		description string
		outgoingURL string
	)

	cmd := &cobra.Command{
		Use:     "update <bot-id>",
		Aliases: []string{"up"},
		Short:   "Update a platform agent bot",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			botID, err := parseIDOrURL(args[0], "bot")
			if err != nil {
				return err
			}

			if name == "" && description == "" && outgoingURL == "" {
				return fmt.Errorf("at least one field must be provided to update")
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			bot, err := client.PlatformAgentBots().Update(cmdContext(cmd), botID, api.UpdatePlatformAgentBotRequest{
				Name:        name,
				Description: description,
				OutgoingURL: outgoingURL,
			})
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

	cmd.Flags().StringVarP(&name, "name", "n", "", "Agent bot name")
	cmd.Flags().StringVar(&description, "description", "", "Agent bot description")
	flagAlias(cmd.Flags(), "description", "desc")
	cmd.Flags().StringVar(&outgoingURL, "outgoing-url", "", "Webhook URL for bot events")
	flagAlias(cmd.Flags(), "outgoing-url", "ou")

	return cmd
}

func newPlatformAgentBotsDeleteCmd(baseURL, token *string) *cobra.Command {
	return &cobra.Command{
		Use:     "delete <bot-id>",
		Aliases: []string{"rm"},
		Short:   "Delete a platform agent bot",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			botID, err := parseIDOrURL(args[0], "bot")
			if err != nil {
				return err
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			if err := client.PlatformAgentBots().Delete(cmdContext(cmd), botID); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": botID})
			}
			printAction(cmd, "Deleted", "agent bot", botID, "")
			return nil
		}),
	}
}
