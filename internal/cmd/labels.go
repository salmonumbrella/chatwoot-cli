package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/dryrun"
	"github.com/spf13/cobra"
)

func newLabelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "labels",
		Aliases: []string{"label", "l"},
		Short:   "Manage account labels",
		Long:    "Create, list, update, and delete account-level labels",
	}

	cmd.AddCommand(newLabelsListCmd())
	cmd.AddCommand(newLabelsGetCmd())
	cmd.AddCommand(newLabelsCreateCmd())
	cmd.AddCommand(newLabelsUpdateCmd())
	cmd.AddCommand(newLabelsDeleteCmd())

	return cmd
}

func newLabelsListCmd() *cobra.Command {
	var light bool

	cfg := ListConfig[api.Label]{
		Use:               "list",
		Short:             "List all labels",
		Long:              "List all labels in the account",
		DisablePagination: true,
		EmptyMessage:      "No labels found",
		Example: strings.TrimSpace(`
  # List all labels
  cw labels list

  # JSON output
  cw labels list -o json
`),
		AgentTransform: func(_ context.Context, _ *api.Client, items []api.Label) (any, error) {
			if light {
				return buildLightLabels(items), nil
			}
			return nil, nil
		},
		JSONTransform: func(_ context.Context, _ *api.Client, items []api.Label) (any, error) {
			if !light {
				return items, nil
			}
			return buildLightLabels(items), nil
		},
		ForceJSON: func(_ *cobra.Command) bool {
			return light
		},
		Fetch: func(ctx context.Context, client *api.Client, _ int, _ int) (ListResult[api.Label], error) {
			labels, err := client.Labels().List(ctx)
			if err != nil {
				return ListResult[api.Label]{}, fmt.Errorf("failed to list labels: %w", err)
			}
			return ListResult[api.Label]{Items: labels, HasMore: false}, nil
		},
		Headers: []string{"ID", "TITLE", "COLOR", "DESCRIPTION"},
		RowFunc: func(label api.Label) []string {
			desc := label.Description
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}
			return []string{
				fmt.Sprintf("%d", label.ID),
				label.Title,
				label.Color,
				desc,
			}
		},
	}

	cmd := NewListCommand(cfg, func(ctx context.Context) (*api.Client, error) {
		return getClient()
	})
	cmd.Aliases = []string{"ls"}

	cmd.Flags().BoolVar(&light, "light", false, "Return minimal label payload for lookup")
	flagAlias(cmd.Flags(), "light", "li")

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "title"},
		"default": {"id", "title", "color", "show_on_sidebar"},
		"debug":   {"id", "title", "description", "color", "show_on_sidebar"},
	})
	registerFieldSchema(cmd, "label")

	return cmd
}

func newLabelsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <id>",
		Aliases: []string{"g"},
		Short:   "Get label details",
		Long:    "Get details of a specific label",
		Example: strings.TrimSpace(`
  # Get label details
  cw labels get 123

  # JSON output
  cw labels get 123 -o json
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "label")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			label, err := client.Labels().Get(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get label %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, label)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Label #%d\n", label.ID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Title:       %s\n", label.Title)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Color:       %s\n", label.Color)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Description: %s\n", label.Description)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Sidebar:     %t\n", label.ShowOnSidebar)

			return nil
		}),
	}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "title"},
		"default": {"id", "title", "color", "show_on_sidebar"},
		"debug":   {"id", "title", "description", "color", "show_on_sidebar"},
	})
	registerFieldSchema(cmd, "label")

	return cmd
}

func newLabelsCreateCmd() *cobra.Command {
	var title, description, color string
	var showOnSidebar bool

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"mk"},
		Short:   "Create a new label",
		Long:    "Create a new account-level label",
		Example: strings.TrimSpace(`
  # Create a simple label
  cw labels create --title "Bug Report"

  # Create a label with all options
  cw labels create --title "Urgent" --color "#FF0000" --description "High priority issues" --show-on-sidebar
`),
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			if title == "" {
				return fmt.Errorf("--title is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "create",
				Resource:  "label",
				Details: map[string]any{
					"title":           title,
					"description":     description,
					"color":           color,
					"show_on_sidebar": showOnSidebar,
				},
			}); ok {
				return err
			}

			label, err := client.Labels().Create(cmdContext(cmd), title, description, color, showOnSidebar)
			if err != nil {
				return fmt.Errorf("failed to create label: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, label)
			}

			printAction(cmd, "Created", "label", label.ID, label.Title)
			return nil
		}),
	}

	cmd.Flags().StringVar(&title, "title", "", "Label title (required)")
	cmd.Flags().StringVar(&description, "description", "", "Label description")
	flagAlias(cmd.Flags(), "description", "desc")
	cmd.Flags().StringVar(&color, "color", "", "Label color (hex, e.g., #FF0000)")
	cmd.Flags().BoolVar(&showOnSidebar, "show-on-sidebar", false, "Show label on sidebar")
	flagAlias(cmd.Flags(), "title", "ttl")
	flagAlias(cmd.Flags(), "color", "hex")
	flagAlias(cmd.Flags(), "show-on-sidebar", "sos")

	return cmd
}

func newLabelsUpdateCmd() *cobra.Command {
	var title, description, color string
	var showOnSidebar bool

	cmd := &cobra.Command{
		Use:     "update <id>",
		Aliases: []string{"up"},
		Short:   "Update a label",
		Long:    "Update an existing label",
		Example: strings.TrimSpace(`
  # Update label title
  cw labels update 123 --title "New Title"

  # Update label color
  cw labels update 123 --color "#00FF00"
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "label")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			sidebarPtr := boolPtrIfChanged(cmd, "show-on-sidebar", showOnSidebar)

			details := map[string]any{
				"id": id,
			}
			if title != "" {
				details["title"] = title
			}
			if description != "" {
				details["description"] = description
			}
			if color != "" {
				details["color"] = color
			}
			if sidebarPtr != nil {
				details["show_on_sidebar"] = *sidebarPtr
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "update",
				Resource:  "label",
				Details:   details,
			}); ok {
				return err
			}

			label, err := client.Labels().Update(cmdContext(cmd), id, title, description, color, sidebarPtr)
			if err != nil {
				return fmt.Errorf("failed to update label %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, label)
			}

			printAction(cmd, "Updated", "label", label.ID, label.Title)
			return nil
		}),
	}

	cmd.Flags().StringVar(&title, "title", "", "Label title")
	cmd.Flags().StringVar(&description, "description", "", "Label description")
	flagAlias(cmd.Flags(), "description", "desc")
	cmd.Flags().StringVar(&color, "color", "", "Label color (hex)")
	cmd.Flags().BoolVar(&showOnSidebar, "show-on-sidebar", false, "Show label on sidebar")
	flagAlias(cmd.Flags(), "title", "ttl")
	flagAlias(cmd.Flags(), "color", "hex")
	flagAlias(cmd.Flags(), "show-on-sidebar", "sos")

	return cmd
}

func newLabelsDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm"},
		Short:   "Delete a label",
		Long:    "Delete an account label",
		Example: strings.TrimSpace(`
  # Delete a label
  cw labels delete 123
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "label")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "delete",
				Resource:  "label",
				Details:   map[string]any{"id": id},
			}); ok {
				return err
			}

			if err := client.Labels().Delete(cmdContext(cmd), id); err != nil {
				return fmt.Errorf("failed to delete label %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": id})
			}
			printAction(cmd, "Deleted", "label", id, "")
			return nil
		}),
	}

	return cmd
}
