package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newCustomFiltersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "custom-filters",
		Aliases: []string{"filters", "cf"},
		Short:   "Manage custom filters",
		Example: `  # List all custom filters
  chatwoot custom-filters list

  # List conversation filters
  chatwoot custom-filters list --type conversation

  # Get a custom filter
  chatwoot custom-filters get 123

  # Create a custom filter
  chatwoot custom-filters create --name "Open Conversations" --type conversation --query '{"status":"open"}'

  # Update a custom filter
  chatwoot custom-filters update 123 --name "Updated Name" --query '{"status":"pending"}'

  # Delete a custom filter
  chatwoot custom-filters delete 123`,
	}

	cmd.AddCommand(newCustomFiltersListCmd())
	cmd.AddCommand(newCustomFiltersGetCmd())
	cmd.AddCommand(newCustomFiltersCreateCmd())
	cmd.AddCommand(newCustomFiltersUpdateCmd())
	cmd.AddCommand(newCustomFiltersDeleteCmd())

	return cmd
}

func newCustomFiltersListCmd() *cobra.Command {
	var filterType string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List custom filters",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			filters, err := client.ListCustomFilters(cmdContext(cmd), filterType)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, filters)
			}

			if len(filters) == 0 {
				fmt.Println("No custom filters found")
				return nil
			}

			w := newTabWriter()
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
		},
	}

	cmd.Flags().StringVar(&filterType, "type", "", "Filter by type: conversation or contact")

	return cmd
}

func newCustomFiltersGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a custom filter by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return fmt.Errorf("invalid ID: %w", err)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			filter, err := client.GetCustomFilter(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, filter)
			}

			queryJSON, _ := json.MarshalIndent(filter.Query, "", "  ")
			fmt.Printf("ID: %d\n", filter.ID)
			fmt.Printf("Name: %s\n", filter.Name)
			fmt.Printf("Type: %s\n", filter.FilterType)
			fmt.Printf("Query:\n%s\n", string(queryJSON))

			return nil
		},
	}
}

func newCustomFiltersCreateCmd() *cobra.Command {
	var (
		name       string
		filterType string
		queryJSON  string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a custom filter",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			filter, err := client.CreateCustomFilter(cmdContext(cmd), name, filterType, query)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, filter)
			}

			fmt.Printf("Created custom filter %d: %s\n", filter.ID, filter.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name for the filter")
	cmd.Flags().StringVar(&filterType, "type", "", "Type: conversation or contact")
	cmd.Flags().StringVar(&queryJSON, "query", "", "Filter query as JSON")

	return cmd
}

func newCustomFiltersUpdateCmd() *cobra.Command {
	var (
		name      string
		queryJSON string
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a custom filter",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return fmt.Errorf("invalid ID: %w", err)
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

			filter, err := client.UpdateCustomFilter(cmdContext(cmd), id, name, query)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, filter)
			}

			fmt.Printf("Updated custom filter %d: %s\n", filter.ID, filter.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name for the filter")
	cmd.Flags().StringVar(&queryJSON, "query", "", "Filter query as JSON")

	return cmd
}

func newCustomFiltersDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a custom filter",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return fmt.Errorf("invalid ID: %w", err)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.DeleteCustomFilter(cmdContext(cmd), id); err != nil {
				return err
			}

			if !isJSON(cmd) {
				fmt.Printf("Deleted custom filter %d\n", id)
			}

			return nil
		},
	}
}
