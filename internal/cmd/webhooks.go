package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
)

func newWebhooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "webhooks",
		Aliases: []string{"webhook", "wh"},
		Short:   "Manage webhooks",
		Long:    "Manage webhook subscriptions for receiving event notifications",
	}

	cmd.AddCommand(newWebhooksListCmd())
	cmd.AddCommand(newWebhooksGetCmd())
	cmd.AddCommand(newWebhooksCreateCmd())
	cmd.AddCommand(newWebhooksUpdateCmd())
	cmd.AddCommand(newWebhooksDeleteCmd())

	return cmd
}

func newWebhooksListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all webhooks",
		Example: "  chatwoot webhooks list",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			webhooks, err := client.ListWebhooks(cmdContext(cmd))
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, webhooks)
			}

			tw := newTabWriter()
			defer func() { _ = tw.Flush() }()

			_, _ = fmt.Fprintln(tw, "ID\tURL\tSUBSCRIPTIONS")
			for _, wh := range webhooks {
				_, _ = fmt.Fprintf(tw, "%d\t%s\t%s\n",
					wh.ID,
					wh.URL,
					strings.Join(wh.Subscriptions, ", "),
				)
			}

			return nil
		},
	}
}

func newWebhooksGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get <id>",
		Aliases: []string{"show"},
		Short:   "Get a webhook by ID",
		Example: "  chatwoot webhooks get 123",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return fmt.Errorf("invalid webhook ID: %s", args[0])
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			webhook, err := client.GetWebhook(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, webhook)
			}

			tw := newTabWriter()
			defer func() { _ = tw.Flush() }()

			_, _ = fmt.Fprintln(tw, "ID\tURL\tSUBSCRIPTIONS")
			_, _ = fmt.Fprintf(tw, "%d\t%s\t%s\n",
				webhook.ID,
				webhook.URL,
				strings.Join(webhook.Subscriptions, ", "),
			)

			return nil
		},
	}
}

func newWebhooksCreateCmd() *cobra.Command {
	var (
		url           string
		subscriptions []string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new webhook",
		Long: `Create a new webhook subscription.

Available subscription events:
  - conversation_created
  - conversation_status_changed
  - conversation_updated
  - message_created
  - message_updated
  - webwidget_triggered`,
		Example: `  chatwoot webhooks create --url https://example.com/webhook --subscriptions conversation_created,message_created
  chatwoot webhooks create --url https://example.com/webhook --subscriptions message_created`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if url == "" {
				return fmt.Errorf("--url is required")
			}
			if len(subscriptions) == 0 {
				return fmt.Errorf("--subscriptions is required")
			}

			// Validate webhook URL before sending to API
			if err := validation.ValidateWebhookURL(url); err != nil {
				return fmt.Errorf("invalid webhook URL: %w", err)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			webhook, err := client.CreateWebhook(cmdContext(cmd), url, subscriptions)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, webhook)
			}

			fmt.Printf("Created webhook %d\n", webhook.ID)
			tw := newTabWriter()
			defer func() { _ = tw.Flush() }()

			_, _ = fmt.Fprintln(tw, "ID\tURL\tSUBSCRIPTIONS")
			_, _ = fmt.Fprintf(tw, "%d\t%s\t%s\n",
				webhook.ID,
				webhook.URL,
				strings.Join(webhook.Subscriptions, ", "),
			)

			return nil
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "Webhook URL (required)")
	cmd.Flags().StringSliceVar(&subscriptions, "subscriptions", nil, "Comma-separated list of events to subscribe to (required)")

	return cmd
}

func newWebhooksUpdateCmd() *cobra.Command {
	var (
		url           string
		subscriptions []string
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a webhook",
		Long: `Update an existing webhook's URL or subscriptions.

Available subscription events:
  - conversation_created
  - conversation_status_changed
  - conversation_updated
  - message_created
  - message_updated
  - webwidget_triggered`,
		Example: `  chatwoot webhooks update 123 --url https://example.com/new-webhook
  chatwoot webhooks update 123 --subscriptions conversation_created,message_created
  chatwoot webhooks update 123 --url https://example.com/new --subscriptions message_created`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return fmt.Errorf("invalid webhook ID: %s", args[0])
			}

			if url == "" && len(subscriptions) == 0 {
				return fmt.Errorf("at least one of --url or --subscriptions must be provided")
			}

			// Validate webhook URL if provided
			if url != "" {
				if err := validation.ValidateWebhookURL(url); err != nil {
					return fmt.Errorf("invalid webhook URL: %w", err)
				}
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			webhook, err := client.UpdateWebhook(cmdContext(cmd), id, url, subscriptions)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, webhook)
			}

			fmt.Printf("Updated webhook %d\n", webhook.ID)
			tw := newTabWriter()
			defer func() { _ = tw.Flush() }()

			_, _ = fmt.Fprintln(tw, "ID\tURL\tSUBSCRIPTIONS")
			_, _ = fmt.Fprintf(tw, "%d\t%s\t%s\n",
				webhook.ID,
				webhook.URL,
				strings.Join(webhook.Subscriptions, ", "),
			)

			return nil
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "New webhook URL")
	cmd.Flags().StringSliceVar(&subscriptions, "subscriptions", nil, "Comma-separated list of events to subscribe to")

	return cmd
}

func newWebhooksDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete a webhook",
		Example: "  chatwoot webhooks delete 123",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return fmt.Errorf("invalid webhook ID: %s", args[0])
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			err = client.DeleteWebhook(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": id})
			}
			fmt.Printf("Deleted webhook %d\n", id)
			return nil
		},
	}
}
