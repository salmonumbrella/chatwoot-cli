package cmd

import (
	"fmt"
	"sort"

	"github.com/chatwoot/chatwoot-cli/internal/config"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newConfigDashboardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Manage dashboard integrations",
		Long:  "Configure external dashboard API endpoints for querying contact data",
	}

	cmd.AddCommand(newDashboardAddCmd())
	cmd.AddCommand(newDashboardListCmd())
	cmd.AddCommand(newDashboardShowCmd())
	cmd.AddCommand(newDashboardRemoveCmd())

	return cmd
}

func newDashboardAddCmd() *cobra.Command {
	var endpoint string
	var authToken string
	var name string

	cmd := &cobra.Command{
		Use:   "add <dashboard-name>",
		Short: "Add a dashboard integration",
		Long:  "Configure an external dashboard API endpoint",
		Example: `  # Add an orders dashboard
  cw config dashboard add orders \
    --endpoint https://api.example.com/api/public/chatwoot/contact/orders \
    --auth-token mytoken123 \
    --name "Customer Orders"`,
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			dashboardName := args[0]

			if endpoint == "" {
				return fmt.Errorf("--endpoint is required")
			}
			if err := validation.ValidateChatwootURL(endpoint); err != nil {
				return fmt.Errorf("invalid endpoint URL: %w", err)
			}
			if authToken == "" {
				return fmt.Errorf("--auth-token is required")
			}

			displayName := name
			if displayName == "" {
				displayName = dashboardName
			}

			cfg := &config.DashboardConfig{
				Name:      displayName,
				Endpoint:  endpoint,
				AuthToken: authToken,
			}

			if err := config.SetDashboard(dashboardName, cfg); err != nil {
				return fmt.Errorf("failed to save dashboard: %w", err)
			}

			printAction(cmd, "Added", "dashboard", dashboardName, displayName)
			return nil
		}),
	}

	cmd.Flags().StringVar(&endpoint, "endpoint", "", "Full URL to the dashboard API endpoint (required)")
	cmd.Flags().StringVar(&authToken, "auth-token", "", "Token for Basic auth (required)")
	cmd.Flags().StringVar(&name, "name", "", "Display name for the dashboard (defaults to dashboard-name)")
	_ = cmd.MarkFlagRequired("endpoint")
	_ = cmd.MarkFlagRequired("auth-token")
	flagAlias(cmd.Flags(), "endpoint", "ep")
	flagAlias(cmd.Flags(), "auth-token", "at")
	flagAlias(cmd.Flags(), "name", "nm")

	return cmd
}

func newDashboardListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List configured dashboards",
		Example: "cw config dashboard list",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			dashboards, err := config.ListDashboards()
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, dashboards)
			}

			if len(dashboards) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No dashboards configured. Run 'cw config dashboard add --help' to add one.")
				return nil
			}

			names := make([]string, 0, len(dashboards))
			for name := range dashboards {
				names = append(names, name)
			}
			sort.Strings(names)

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "NAME\tDISPLAY NAME\tENDPOINT")
			for _, name := range names {
				cfg := dashboards[name]
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", name, cfg.Name, cfg.Endpoint)
			}

			return nil
		}),
	}
}

func newDashboardShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "show <dashboard-name>",
		Short:   "Show dashboard configuration",
		Example: "cw config dashboard show orders",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := config.GetDashboard(name)
			if err != nil {
				return fmt.Errorf("dashboard %q not found", name)
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{
					"name":       name,
					"display":    cfg.Name,
					"endpoint":   cfg.Endpoint,
					"auth_token": cfg.AuthToken,
				})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Dashboard: %s\n", name)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Display Name: %s\n", cfg.Name)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Endpoint: %s\n", cfg.Endpoint)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Auth Token: %s\n", cfg.AuthToken)

			return nil
		}),
	}
}

func newDashboardRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <dashboard-name>",
		Short:   "Remove a dashboard integration",
		Example: "cw config dashboard remove orders",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if err := config.DeleteDashboard(name); err != nil {
				return fmt.Errorf("failed to remove dashboard %q: %w", name, err)
			}

			printAction(cmd, "Removed", "dashboard", name, "")
			return nil
		}),
	}
}
