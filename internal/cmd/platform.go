package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newPlatformCmd() *cobra.Command {
	var baseURL string
	var token string

	cmd := &cobra.Command{
		Use:   "platform",
		Short: "Manage Chatwoot via platform APIs",
		Long:  "Use platform APIs to manage accounts and users (requires platform token).",
	}

	cmd.PersistentFlags().StringVar(&baseURL, "base-url", "", "Override Chatwoot base URL")
	cmd.PersistentFlags().StringVar(&token, "token", "", "Platform API token (overrides env/config)")

	cmd.AddCommand(newPlatformAccountsCmd(&baseURL, &token))
	cmd.AddCommand(newPlatformUsersCmd(&baseURL, &token))
	cmd.AddCommand(newPlatformAccountUsersCmd(&baseURL, &token))

	return cmd
}

func newPlatformAccountsCmd(baseURL, token *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Manage accounts via platform API",
	}

	cmd.AddCommand(newPlatformAccountsCreateCmd(baseURL, token))
	cmd.AddCommand(newPlatformAccountsGetCmd(baseURL, token))
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
		Use:   "create",
		Short: "Create an account",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			account, err := client.CreatePlatformAccount(cmdContext(cmd), api.CreatePlatformAccountRequest{
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

			fmt.Printf("Created account %d: %s\n", account.ID, account.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Account name (required)")
	cmd.Flags().StringVar(&locale, "locale", "", "Account locale")
	cmd.Flags().StringVar(&domain, "domain", "", "Account domain")
	cmd.Flags().StringVar(&supportEmail, "support-email", "", "Support email")
	cmd.Flags().StringVar(&status, "status", "", "Account status")
	cmd.Flags().StringVar(&customAttributes, "custom-attributes", "", "Custom attributes JSON")
	cmd.Flags().StringVar(&limits, "limits", "", "Limits JSON")

	return cmd
}

func newPlatformAccountsGetCmd(baseURL, token *string) *cobra.Command {
	return &cobra.Command{
		Use:   "get <account-id>",
		Short: "Get an account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			accountID, err := validation.ParsePositiveInt(args[0], "account ID")
			if err != nil {
				return err
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			account, err := client.GetPlatformAccount(cmdContext(cmd), accountID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, account)
			}

			fmt.Printf("Account %d: %s\n", account.ID, account.Name)
			return nil
		},
	}
}

func newPlatformAccountsDeleteCmd(baseURL, token *string) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <account-id>",
		Short: "Delete an account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			accountID, err := validation.ParsePositiveInt(args[0], "account ID")
			if err != nil {
				return err
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			if err := client.DeletePlatformAccount(cmdContext(cmd), accountID); err != nil {
				return err
			}

			fmt.Printf("Deleted account %d\n", accountID)
			return nil
		},
	}
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
		Use:   "create",
		Short: "Create a user",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			user, err := client.CreatePlatformUser(cmdContext(cmd), api.CreatePlatformUserRequest{
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

			fmt.Printf("Created user %d: %s\n", user.ID, user.Email)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "User name (required)")
	cmd.Flags().StringVar(&displayName, "display-name", "", "User display name")
	cmd.Flags().StringVar(&email, "email", "", "User email (required)")
	cmd.Flags().StringVar(&password, "password", "", "User password (required)")
	cmd.Flags().StringVar(&customAttributes, "custom-attributes", "", "Custom attributes JSON")

	return cmd
}

func newPlatformUsersGetCmd(baseURL, token *string) *cobra.Command {
	return &cobra.Command{
		Use:   "get <user-id>",
		Short: "Get a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, err := validation.ParsePositiveInt(args[0], "user ID")
			if err != nil {
				return err
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			user, err := client.GetPlatformUser(cmdContext(cmd), userID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, user)
			}

			fmt.Printf("User %d: %s\n", user.ID, user.Email)
			return nil
		},
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
		Use:   "update <user-id>",
		Short: "Update a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, err := validation.ParsePositiveInt(args[0], "user ID")
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

			user, err := client.UpdatePlatformUser(cmdContext(cmd), userID, api.UpdatePlatformUserRequest{
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

			fmt.Printf("Updated user %d\n", user.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "User name")
	cmd.Flags().StringVar(&displayName, "display-name", "", "User display name")
	cmd.Flags().StringVar(&email, "email", "", "User email")
	cmd.Flags().StringVar(&password, "password", "", "User password")
	cmd.Flags().StringVar(&customAttributes, "custom-attributes", "", "Custom attributes JSON")

	return cmd
}

func newPlatformUsersDeleteCmd(baseURL, token *string) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <user-id>",
		Short: "Delete a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, err := validation.ParsePositiveInt(args[0], "user ID")
			if err != nil {
				return err
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			if err := client.DeletePlatformUser(cmdContext(cmd), userID); err != nil {
				return err
			}

			fmt.Printf("Deleted user %d\n", userID)
			return nil
		},
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
		Use:   "list <account-id>",
		Short: "List account users",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			accountID, err := validation.ParsePositiveInt(args[0], "account ID")
			if err != nil {
				return err
			}

			client, err := getPlatformClient(*baseURL, *token)
			if err != nil {
				return err
			}

			users, err := client.ListPlatformAccountUsers(cmdContext(cmd), accountID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, users)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tUSER\tROLE")
			for _, user := range users {
				_, _ = fmt.Fprintf(w, "%d\t%d\t%s\n", user.ID, user.UserID, user.Role)
			}
			return nil
		},
	}
}

func newPlatformAccountUsersCreateCmd(baseURL, token *string) *cobra.Command {
	var userID int
	var role string

	cmd := &cobra.Command{
		Use:   "create <account-id>",
		Short: "Add a user to an account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			accountID, err := validation.ParsePositiveInt(args[0], "account ID")
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

			accountUser, err := client.CreatePlatformAccountUser(cmdContext(cmd), accountID, api.CreatePlatformAccountUserRequest{
				UserID: userID,
				Role:   role,
			})
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, accountUser)
			}

			fmt.Printf("Added user %d to account %d\n", userID, accountID)
			return nil
		},
	}

	cmd.Flags().IntVar(&userID, "user-id", 0, "User ID (required)")
	cmd.Flags().StringVar(&role, "role", "", "Role (required)")

	return cmd
}

func newPlatformAccountUsersDeleteCmd(baseURL, token *string) *cobra.Command {
	var userID int

	cmd := &cobra.Command{
		Use:   "delete <account-id>",
		Short: "Remove a user from an account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			accountID, err := validation.ParsePositiveInt(args[0], "account ID")
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

			if err := client.DeletePlatformAccountUser(cmdContext(cmd), accountID, userID); err != nil {
				return err
			}

			fmt.Printf("Removed user %d from account %d\n", userID, accountID)
			return nil
		},
	}

	cmd.Flags().IntVar(&userID, "user-id", 0, "User ID (required)")

	return cmd
}
