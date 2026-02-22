package cmd

import (
	"fmt"

	"github.com/chatwoot/chatwoot-cli/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config",
		Aliases: []string{"cfg"},
		Short:   "Manage CLI configuration",
	}

	cmd.AddCommand(newConfigProfilesCmd())
	cmd.AddCommand(newConfigDashboardCmd())

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
		Aliases: []string{"ls"},
		Short:   "List configured profiles",
		Example: "cw config profiles list",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
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
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No profiles configured. Run 'cw auth login' to add one.")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "CURRENT\tPROFILE\tBASE_URL")
			for _, profile := range profiles {
				marker := ""
				if profile == current {
					marker = "*"
				}
				baseURL := "-"
				if account, err := config.LoadProfile(profile); err == nil && account.BaseURL != "" {
					baseURL = account.BaseURL
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", marker, profile, baseURL)
			}

			return nil
		}),
	}
}

func newProfilesUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "use <name>",
		Short:   "Switch active profile",
		Example: "cw config profiles use staging",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			name := args[0]
			account, err := config.LoadProfile(name)
			if err != nil {
				return fmt.Errorf("profile %q not found: %w", name, err)
			}
			if err := config.SetCurrentProfile(name); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Current profile: %s (%s)\n", name, account.BaseURL)
			return nil
		}),
	}
}

func newProfilesShowCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:     "show",
		Short:   "Show profile details",
		Example: "cw config profiles show --name staging",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
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

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Profile: %s\n", name)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Base URL: %s\n", account.BaseURL)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Account ID: %d\n", account.AccountID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  API Token: %s\n", maskToken(account.APIToken))
			if account.PlatformToken != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Platform Token: %s\n", maskToken(account.PlatformToken))
			}
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Profile name (defaults to current)")
	flagAlias(cmd.Flags(), "name", "nm")

	return cmd
}

func newProfilesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <name>",
		Aliases: []string{"rm"},
		Short:   "Delete a profile",
		Example: "cw config profiles delete staging",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := config.DeleteProfile(name); err != nil {
				return err
			}
			printAction(cmd, "Deleted", "profile", name, "")
			return nil
		}),
	}
}
