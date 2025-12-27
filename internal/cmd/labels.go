package cmd

import (
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newLabelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "labels",
		Aliases: []string{"label"},
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
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all labels",
		Long:  "List all labels in the account",
		Example: strings.TrimSpace(`
  # List all labels
  chatwoot labels list

  # JSON output
  chatwoot labels list -o json
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			labels, err := client.ListLabels(cmdContext(cmd))
			if err != nil {
				return fmt.Errorf("failed to list labels: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(labels)
			}

			if len(labels) == 0 {
				fmt.Println("No labels found")
				return nil
			}

			w := newTabWriter()
			_, _ = fmt.Fprintln(w, "ID\tTITLE\tCOLOR\tDESCRIPTION")
			for _, label := range labels {
				desc := label.Description
				if len(desc) > 40 {
					desc = desc[:37] + "..."
				}
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", label.ID, label.Title, label.Color, desc)
			}
			_ = w.Flush()

			return nil
		},
	}

	return cmd
}

func newLabelsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get label details",
		Long:  "Get details of a specific label",
		Example: strings.TrimSpace(`
  # Get label details
  chatwoot labels get 123

  # JSON output
  chatwoot labels get 123 -o json
`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "label ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			label, err := client.GetLabel(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get label %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(label)
			}

			fmt.Printf("Label #%d\n", label.ID)
			fmt.Printf("  Title:       %s\n", label.Title)
			fmt.Printf("  Color:       %s\n", label.Color)
			fmt.Printf("  Description: %s\n", label.Description)
			fmt.Printf("  Sidebar:     %t\n", label.ShowOnSidebar)

			return nil
		},
	}

	return cmd
}

func newLabelsCreateCmd() *cobra.Command {
	var title, description, color string
	var showOnSidebar bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new label",
		Long:  "Create a new account-level label",
		Example: strings.TrimSpace(`
  # Create a simple label
  chatwoot labels create --title "Bug Report"

  # Create a label with all options
  chatwoot labels create --title "Urgent" --color "#FF0000" --description "High priority issues" --show-on-sidebar
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if title == "" {
				return fmt.Errorf("--title is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			label, err := client.CreateLabel(cmdContext(cmd), title, description, color, showOnSidebar)
			if err != nil {
				return fmt.Errorf("failed to create label: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(label)
			}

			fmt.Printf("Created label #%d: %s\n", label.ID, label.Title)
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Label title (required)")
	cmd.Flags().StringVar(&description, "description", "", "Label description")
	cmd.Flags().StringVar(&color, "color", "", "Label color (hex, e.g., #FF0000)")
	cmd.Flags().BoolVar(&showOnSidebar, "show-on-sidebar", false, "Show label on sidebar")

	return cmd
}

func newLabelsUpdateCmd() *cobra.Command {
	var title, description, color string
	var showOnSidebar bool
	var showOnSidebarSet bool

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a label",
		Long:  "Update an existing label",
		Example: strings.TrimSpace(`
  # Update label title
  chatwoot labels update 123 --title "New Title"

  # Update label color
  chatwoot labels update 123 --color "#00FF00"
`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "label ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			var sidebarPtr *bool
			if showOnSidebarSet {
				sidebarPtr = &showOnSidebar
			}

			label, err := client.UpdateLabel(cmdContext(cmd), id, title, description, color, sidebarPtr)
			if err != nil {
				return fmt.Errorf("failed to update label %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(label)
			}

			fmt.Printf("Updated label #%d: %s\n", label.ID, label.Title)
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Label title")
	cmd.Flags().StringVar(&description, "description", "", "Label description")
	cmd.Flags().StringVar(&color, "color", "", "Label color (hex)")
	cmd.Flags().BoolVar(&showOnSidebar, "show-on-sidebar", false, "Show label on sidebar")
	cmd.PreRun = func(_ *cobra.Command, _ []string) {
		showOnSidebarSet = cmd.Flags().Changed("show-on-sidebar")
	}

	return cmd
}

func newLabelsDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a label",
		Long:  "Delete an account label",
		Example: strings.TrimSpace(`
  # Delete a label
  chatwoot labels delete 123
`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "label ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.DeleteLabel(cmdContext(cmd), id); err != nil {
				return fmt.Errorf("failed to delete label %d: %w", id, err)
			}

			fmt.Printf("Deleted label #%d\n", id)
			return nil
		},
	}

	return cmd
}
