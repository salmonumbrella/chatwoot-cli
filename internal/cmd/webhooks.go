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
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all webhooks",
		Example: "  cw webhooks list",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			webhooks, err := client.Webhooks().List(cmdContext(cmd))
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, webhooks)
			}

			tw := newTabWriterFromCmd(cmd)
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
		}),
	}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "url"},
		"default": {"id", "url", "subscriptions"},
		"debug":   {"id", "url", "subscriptions", "account_id"},
	})

	return cmd
}

func newWebhooksGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <id>",
		Aliases: []string{"g"},
		Short:   "Get a webhook by ID",
		Example: "  cw webhooks get 123",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "webhook")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			webhook, err := client.Webhooks().Get(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, webhook)
			}

			tw := newTabWriterFromCmd(cmd)
			defer func() { _ = tw.Flush() }()

			_, _ = fmt.Fprintln(tw, "ID\tURL\tSUBSCRIPTIONS")
			_, _ = fmt.Fprintf(tw, "%d\t%s\t%s\n",
				webhook.ID,
				webhook.URL,
				strings.Join(webhook.Subscriptions, ", "),
			)

			return nil
		}),
	}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "url"},
		"default": {"id", "url", "subscriptions"},
		"debug":   {"id", "url", "subscriptions", "account_id"},
	})

	return cmd
}

func newWebhooksCreateCmd() *cobra.Command {
	var (
		url           string
		subscriptions []string
		emit          string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"mk"},
		Short:   "Create a new webhook",
		Long: `Create a new webhook subscription.

Available subscription events:
  - conversation_created
  - conversation_status_changed
  - conversation_updated
  - message_created
  - message_updated
  - webwidget_triggered`,
		Example: `  cw webhooks create --url https://example.com/webhook --subscriptions conversation_created,message_created
  cw webhooks create --url https://example.com/webhook --subscriptions message_created`,
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if url == "" {
				return fmt.Errorf("--url is required")
			}

			var err error
			subscriptions, err = func(values []string) ([]string, error) {
				if len(values) == 0 {
					return nil, nil
				}
				if len(values) == 1 {
					return ParseStringListFlag(values[0])
				}
				for _, v := range values {
					v = strings.TrimSpace(v)
					if strings.HasPrefix(v, "@") || strings.HasPrefix(v, "[") {
						return nil, fmt.Errorf("cannot combine @- / @path or JSON array with multiple --subscriptions values")
					}
				}
				return ParseStringListFlag(strings.Join(values, "\n"))
			}(subscriptions)
			if err != nil {
				return fmt.Errorf("invalid --subscriptions value: %w", err)
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

			webhook, err := client.Webhooks().Create(cmdContext(cmd), url, subscriptions)
			if err != nil {
				return err
			}

			if emitted, err := maybeEmit(cmd, emit, "webhook", webhook.ID, webhook); emitted {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, webhook)
			}

			printAction(cmd, "Created", "webhook", webhook.ID, "")
			tw := newTabWriterFromCmd(cmd)
			defer func() { _ = tw.Flush() }()

			_, _ = fmt.Fprintln(tw, "ID\tURL\tSUBSCRIPTIONS")
			_, _ = fmt.Fprintf(tw, "%d\t%s\t%s\n",
				webhook.ID,
				webhook.URL,
				strings.Join(webhook.Subscriptions, ", "),
			)

			return nil
		}),
	}

	cmd.Flags().StringVar(&url, "url", "", "Webhook URL (required)")
	cmd.Flags().StringArrayVar(&subscriptions, "subscriptions", nil, "Subscription events (repeatable, or CSV/whitespace/JSON array, or @- / @path) (required)")
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")
	flagAlias(cmd.Flags(), "url", "wu")

	return cmd
}

func newWebhooksUpdateCmd() *cobra.Command {
	var (
		url           string
		subscriptions []string
		emit          string
	)

	cmd := &cobra.Command{
		Use:     "update <id>",
		Aliases: []string{"up"},
		Short:   "Update a webhook",
		Long: `Update an existing webhook's URL or subscriptions.

Available subscription events:
  - conversation_created
  - conversation_status_changed
  - conversation_updated
  - message_created
  - message_updated
  - webwidget_triggered`,
		Example: `  cw webhooks update 123 --url https://example.com/new-webhook
  cw webhooks update 123 --subscriptions conversation_created,message_created
  cw webhooks update 123 --url https://example.com/new --subscriptions message_created`,
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "webhook")
			if err != nil {
				return err
			}

			subscriptions, err = func(values []string) ([]string, error) {
				if len(values) == 0 {
					return nil, nil
				}
				if len(values) == 1 {
					return ParseStringListFlag(values[0])
				}
				for _, v := range values {
					v = strings.TrimSpace(v)
					if strings.HasPrefix(v, "@") || strings.HasPrefix(v, "[") {
						return nil, fmt.Errorf("cannot combine @- / @path or JSON array with multiple --subscriptions values")
					}
				}
				return ParseStringListFlag(strings.Join(values, "\n"))
			}(subscriptions)
			if err != nil {
				return fmt.Errorf("invalid --subscriptions value: %w", err)
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

			webhook, err := client.Webhooks().Update(cmdContext(cmd), id, url, subscriptions)
			if err != nil {
				return err
			}

			if emitted, err := maybeEmit(cmd, emit, "webhook", webhook.ID, webhook); emitted {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, webhook)
			}

			printAction(cmd, "Updated", "webhook", webhook.ID, "")
			tw := newTabWriterFromCmd(cmd)
			defer func() { _ = tw.Flush() }()

			_, _ = fmt.Fprintln(tw, "ID\tURL\tSUBSCRIPTIONS")
			_, _ = fmt.Fprintf(tw, "%d\t%s\t%s\n",
				webhook.ID,
				webhook.URL,
				strings.Join(webhook.Subscriptions, ", "),
			)

			return nil
		}),
	}

	cmd.Flags().StringVar(&url, "url", "", "New webhook URL")
	cmd.Flags().StringArrayVar(&subscriptions, "subscriptions", nil, "Subscription events (repeatable, or CSV/whitespace/JSON array, or @- / @path)")
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")
	flagAlias(cmd.Flags(), "url", "wu")

	return cmd
}

func newWebhooksDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete a webhook",
		Example: "  cw webhooks delete 123",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "webhook")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			err = client.Webhooks().Delete(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": id})
			}
			printAction(cmd, "Deleted", "webhook", id, "")
			return nil
		}),
	}
}
