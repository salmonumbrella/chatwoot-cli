package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newIntegrationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "integrations",
		Aliases: []string{"integration"},
		Short:   "Manage integrations",
	}

	cmd.AddCommand(newIntegrationsAppsCmd())
	cmd.AddCommand(newIntegrationsHooksCmd())
	cmd.AddCommand(newIntegrationsHookCreateCmd())
	cmd.AddCommand(newIntegrationsHookUpdateCmd())
	cmd.AddCommand(newIntegrationsHookDeleteCmd())

	return cmd
}

func newIntegrationsAppsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "apps",
		Short:   "List available integration apps",
		Example: "chatwoot integrations apps",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			apps, err := client.ListIntegrationApps(cmdContext(cmd))
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, apps)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tNAME\tENABLED\tDESCRIPTION")
			for _, app := range apps {
				enabled := "no"
				if app.Enabled {
					enabled = "yes"
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", app.ID, app.Name, enabled, app.Description)
			}
			return nil
		},
	}
}

func newIntegrationsHooksCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "hooks",
		Short:   "List integration hooks",
		Example: "chatwoot integrations hooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			hooks, err := client.ListIntegrationHooks(cmdContext(cmd))
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, hooks)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tAPP_ID\tINBOX_ID\tACCOUNT_ID")
			for _, hook := range hooks {
				inboxID := "-"
				if hook.InboxID > 0 {
					inboxID = strconv.Itoa(hook.InboxID)
				}
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%d\n", hook.ID, hook.AppID, inboxID, hook.AccountID)
			}
			return nil
		},
	}
}

func newIntegrationsHookCreateCmd() *cobra.Command {
	var appID string
	var inboxID int
	var settingsJSON string

	cmd := &cobra.Command{
		Use:     "hook-create",
		Short:   "Create an integration hook",
		Example: "chatwoot integrations hook-create --app-id slack --inbox-id 1 --settings '{\"webhook_url\":\"https://...\"}'",
		RunE: func(cmd *cobra.Command, args []string) error {
			if appID == "" {
				return fmt.Errorf("--app-id is required")
			}

			var settings map[string]any
			if settingsJSON != "" {
				if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
					return fmt.Errorf("invalid settings JSON: %w", err)
				}
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			hook, err := client.CreateIntegrationHook(cmdContext(cmd), appID, inboxID, settings)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, hook)
			}

			fmt.Printf("Created integration hook %d\n", hook.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app-id", "", "App ID (required)")
	cmd.Flags().IntVar(&inboxID, "inbox-id", 0, "Inbox ID (optional)")
	cmd.Flags().StringVar(&settingsJSON, "settings", "", "Settings as JSON string")

	return cmd
}

func newIntegrationsHookUpdateCmd() *cobra.Command {
	var settingsJSON string

	cmd := &cobra.Command{
		Use:     "hook-update <hook-id>",
		Short:   "Update an integration hook",
		Example: "chatwoot integrations hook-update 123 --settings '{\"webhook_url\":\"https://...\"}'",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hookID, err := validation.ParsePositiveInt(args[0], "hook ID")
			if err != nil {
				return err
			}

			var settings map[string]any
			if settingsJSON != "" {
				if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
					return fmt.Errorf("invalid settings JSON: %w", err)
				}
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			hook, err := client.UpdateIntegrationHook(cmdContext(cmd), hookID, settings)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, hook)
			}

			fmt.Printf("Updated integration hook %d\n", hook.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&settingsJSON, "settings", "", "Settings as JSON string")

	return cmd
}

func newIntegrationsHookDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "hook-delete <hook-id>",
		Short:   "Delete an integration hook",
		Example: "chatwoot integrations hook-delete 123",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hookID, err := validation.ParsePositiveInt(args[0], "hook ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.DeleteIntegrationHook(cmdContext(cmd), hookID); err != nil {
				return err
			}

			if !isJSON(cmd) {
				fmt.Printf("Deleted integration hook %d\n", hookID)
			}
			return nil
		},
	}
}
