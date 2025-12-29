package cmd

import (
	"fmt"

	"github.com/chatwoot/chatwoot-cli/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
	}

	cmd.AddCommand(newConfigProfilesCmd())

	return cmd
}

func newConfigProfilesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profiles",
		Short: "Manage auth profiles",
	}

	cmd.AddCommand(newProfilesListCmd())
	cmd.AddCommand(newProfilesUseCmd())
	cmd.AddCommand(newProfilesShowCmd())
	cmd.AddCommand(newProfilesDeleteCmd())

	return cmd
}

func newProfilesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List configured profiles",
		Example: "chatwoot config profiles list",
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := config.ListProfiles()
			if err != nil {
				return err
			}
			current, _ := config.CurrentProfile()

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{
					"current":  current,
					"profiles": profiles,
				})
			}

			if len(profiles) == 0 {
				fmt.Println("No profiles configured. Run 'chatwoot auth login' to add one.")
				return nil
			}

			for _, profile := range profiles {
				marker := ""
				if profile == current {
					marker = "*"
				}
				fmt.Printf("%s%s\n", marker, profile)
			}

			return nil
		},
	}
}

func newProfilesUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "use <name>",
		Short:   "Switch active profile",
		Example: "chatwoot config profiles use staging",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if _, err := config.LoadProfile(name); err != nil {
				return fmt.Errorf("profile %q not found: %w", name, err)
			}
			if err := config.SetCurrentProfile(name); err != nil {
				return err
			}
			fmt.Printf("Current profile set to %s\n", name)
			return nil
		},
	}
}

func newProfilesShowCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:     "show",
		Short:   "Show profile details",
		Example: "chatwoot config profiles show --name staging",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				current, err := config.CurrentProfile()
				if err != nil {
					return err
				}
				name = current
			}

			account, err := config.LoadProfile(name)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{
					"profile":        name,
					"base_url":       account.BaseURL,
					"account_id":     account.AccountID,
					"api_token":      maskToken(account.APIToken),
					"platform_token": maskToken(account.PlatformToken),
				})
			}

			fmt.Printf("Profile: %s\n", name)
			fmt.Printf("  Base URL: %s\n", account.BaseURL)
			fmt.Printf("  Account ID: %d\n", account.AccountID)
			fmt.Printf("  API Token: %s\n", maskToken(account.APIToken))
			if account.PlatformToken != "" {
				fmt.Printf("  Platform Token: %s\n", maskToken(account.PlatformToken))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Profile name (defaults to current)")

	return cmd
}

func newProfilesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <name>",
		Short:   "Delete a profile",
		Example: "chatwoot config profiles delete staging",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := config.DeleteProfile(name); err != nil {
				return err
			}
			fmt.Printf("Deleted profile %s\n", name)
			return nil
		},
	}
}
