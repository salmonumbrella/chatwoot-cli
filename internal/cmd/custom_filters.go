package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newCustomFiltersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "custom-filters",
		Aliases: []string{"filters", "cf"},
		Short:   "Manage custom filters",
		Example: `  # List all custom filters
  cw custom-filters list

  # List conversation filters
  cw custom-filters list --type conversation

  # Get a custom filter
  cw custom-filters get 123

  # Create a custom filter
  cw custom-filters create --name "Open Conversations" --type conversation --query '{"status":"open"}'

  # Update a custom filter
  cw custom-filters update 123 --name "Updated Name" --query '{"status":"pending"}'

  # Delete a custom filter
  cw custom-filters delete 123`,
	}

	cmd.AddCommand(newCustomFiltersListCmd())
	cmd.AddCommand(newCustomFiltersGetCmd())
	cmd.AddCommand(newCustomFiltersCreateCmd())
	cmd.AddCommand(newCustomFiltersUpdateCmd())
	cmd.AddCommand(newCustomFiltersDeleteCmd())

	return cmd
}

func newCustomFiltersListCmd() *cobra.Command {
	var (
		filterType string
		light      bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List custom filters",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			filters, err := client.CustomFilters().List(cmdContext(cmd), filterType)
			if err != nil {
				return err
			}

			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, buildLightCustomFilters(filters))
			}

			if isJSON(cmd) {
				return printJSON(cmd, filters)
			}

			if len(filters) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No custom filters found")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tTYPE\tQUERY")
			for _, filter := range filters {
				queryJSON, _ := json.Marshal(filter.Query)
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
					filter.ID,
					filter.Name,
					filter.FilterType,
					string(queryJSON),
				)
			}

			return nil
		}),
	}

	cmd.Flags().StringVar(&filterType, "type", "", "Filter by type: conversation or contact")
	flagAlias(cmd.Flags(), "type", "ty")
	cmd.Flags().BoolVar(&light, "light", false, "Return minimal custom filter payload for lookup")
	flagAlias(cmd.Flags(), "light", "li")

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name", "filter_type"},
		"default": {"id", "name", "filter_type", "query"},
		"debug":   {"id", "name", "filter_type", "query"},
	})

	return cmd
}

func newCustomFiltersGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <id>",
		Aliases: []string{"g"},
		Short:   "Get a custom filter by ID",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "custom filter")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			filter, err := client.CustomFilters().Get(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, filter)
			}

			queryJSON, _ := json.MarshalIndent(filter.Query, "", "  ")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ID: %d\n", filter.ID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\n", filter.Name)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Type: %s\n", filter.FilterType)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Query:\n%s\n", string(queryJSON))

			return nil
		}),
	}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name", "filter_type"},
		"default": {"id", "name", "filter_type", "query"},
		"debug":   {"id", "name", "filter_type", "query"},
	})

	return cmd
}

func newCustomFiltersCreateCmd() *cobra.Command {
	var (
		name       string
		filterType string
		queryJSON  string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"mk"},
		Short:   "Create a custom filter",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if filterType == "" {
				return fmt.Errorf("--type is required")
			}
			if queryJSON == "" {
				return fmt.Errorf("--query is required")
			}

			var query map[string]any
			if err := json.Unmarshal([]byte(queryJSON), &query); err != nil {
				return fmt.Errorf("invalid query JSON: %w", err)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			filter, err := client.CustomFilters().Create(cmdContext(cmd), name, filterType, query)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, filter)
			}

			printAction(cmd, "Created", "custom filter", filter.ID, filter.Name)
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Name for the filter")
	cmd.Flags().StringVar(&filterType, "type", "", "Type: conversation or contact")
	cmd.Flags().StringVar(&queryJSON, "query", "", "Filter query as JSON")
	flagAlias(cmd.Flags(), "name", "nm")
	flagAlias(cmd.Flags(), "type", "ty")
	flagAlias(cmd.Flags(), "query", "sq")

	return cmd
}

func newCustomFiltersUpdateCmd() *cobra.Command {
	var (
		name      string
		queryJSON string
	)

	cmd := &cobra.Command{
		Use:     "update <id>",
		Aliases: []string{"up"},
		Short:   "Update a custom filter",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "custom filter")
			if err != nil {
				return err
			}

			if name == "" && queryJSON == "" {
				return fmt.Errorf("at least one of --name or --query is required")
			}

			var query map[string]any
			if queryJSON != "" {
				if err := json.Unmarshal([]byte(queryJSON), &query); err != nil {
					return fmt.Errorf("invalid query JSON: %w", err)
				}
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			filter, err := client.CustomFilters().Update(cmdContext(cmd), id, name, query)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, filter)
			}

			printAction(cmd, "Updated", "custom filter", filter.ID, filter.Name)
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Name for the filter")
	cmd.Flags().StringVar(&queryJSON, "query", "", "Filter query as JSON")
	flagAlias(cmd.Flags(), "name", "nm")
	flagAlias(cmd.Flags(), "query", "sq")

	return cmd
}

func newCustomFiltersDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm"},
		Short:   "Delete a custom filter",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "custom filter")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.CustomFilters().Delete(cmdContext(cmd), id); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": id})
			}
			printAction(cmd, "Deleted", "custom filter", id, "")
			return nil
		}),
	}
}
