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
	cmd.AddCommand(newShopifyCmd())
	cmd.AddCommand(newNotionCmd())

	return cmd
}

func newIntegrationsAppsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "apps",
		Short:   "List available integration apps",
		Example: "chatwoot integrations apps",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
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

			w := newTabWriterFromCmd(cmd)
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
		}),
	}
}

func newIntegrationsHooksCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "hooks",
		Short:   "List integration hooks",
		Example: "chatwoot integrations hooks",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
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

			w := newTabWriterFromCmd(cmd)
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
		}),
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
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
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

			printAction(cmd, "Created", "integration hook", hook.ID, "")
			return nil
		}),
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
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
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

			printAction(cmd, "Updated", "integration hook", hook.ID, "")
			return nil
		}),
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
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
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

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": hookID})
			}
			printAction(cmd, "Deleted", "integration hook", hookID, "")
			return nil
		}),
	}
}

func newShopifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shopify",
		Short: "Manage Shopify integration",
	}

	cmd.AddCommand(newShopifyAuthCmd())
	cmd.AddCommand(newShopifyOrdersCmd())
	cmd.AddCommand(newShopifyDeleteCmd())

	return cmd
}

func newShopifyAuthCmd() *cobra.Command {
	var shopDomain string
	var code string

	cmd := &cobra.Command{
		Use:     "auth",
		Short:   "Authenticate Shopify integration",
		Example: "chatwoot integrations shopify auth --shop mystore.myshopify.com --code AUTH_CODE",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if shopDomain == "" {
				return fmt.Errorf("--shop is required")
			}
			if code == "" {
				return fmt.Errorf("--code is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.ShopifyAuth(cmdContext(cmd), shopDomain, code); err != nil {
				return err
			}

			if !isJSON(cmd) {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Shopify integration authenticated successfully")
			}
			return nil
		}),
	}

	cmd.Flags().StringVar(&shopDomain, "shop", "", "Shopify store domain (e.g., mystore.myshopify.com)")
	cmd.Flags().StringVar(&code, "code", "", "OAuth authorization code")

	return cmd
}

func newShopifyOrdersCmd() *cobra.Command {
	var contactID int

	cmd := &cobra.Command{
		Use:     "orders",
		Short:   "List Shopify orders for a contact",
		Example: "chatwoot integrations shopify orders --contact-id 123",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if contactID == 0 {
				return fmt.Errorf("--contact-id is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			orders, err := client.ListShopifyOrders(cmdContext(cmd), contactID)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, orders)
			}

			if len(orders) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No orders found")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tTOTAL\tSTATUS")
			for _, o := range orders {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s %s\t%s\n",
					o.ID, o.Name, o.Email, o.TotalPrice, o.Currency, o.FinancialStatus)
			}
			return nil
		}),
	}

	cmd.Flags().IntVar(&contactID, "contact-id", 0, "Contact ID to get orders for")

	return cmd
}

func newShopifyDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete",
		Short:   "Delete Shopify integration",
		Example: "chatwoot integrations shopify delete",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.DeleteShopifyIntegration(cmdContext(cmd)); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "integration": "shopify"})
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Shopify integration deleted")
			return nil
		}),
	}
}

func newNotionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notion",
		Short: "Manage Notion integration",
	}

	cmd.AddCommand(newNotionDeleteCmd())

	return cmd
}

func newNotionDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete",
		Short:   "Delete Notion integration",
		Example: "chatwoot integrations notion delete",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.DeleteNotionIntegration(cmdContext(cmd)); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "integration": "notion"})
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Notion integration deleted")
			return nil
		}),
	}
}
