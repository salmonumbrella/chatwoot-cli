package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newCustomAttributesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "custom-attributes",
		Aliases: []string{"attrs", "ca"},
		Short:   "Manage custom attribute definitions",
		Example: `  # List all custom attributes
  chatwoot custom-attributes list

  # List contact custom attributes
  chatwoot custom-attributes list --model contact

  # Get a custom attribute
  chatwoot custom-attributes get 123

  # Create a custom attribute
  chatwoot custom-attributes create --name "Customer ID" --model contact --type text

  # Update a custom attribute
  chatwoot custom-attributes update 123 --name "Updated Name"

  # Delete a custom attribute
  chatwoot custom-attributes delete 123`,
	}

	cmd.AddCommand(newCustomAttributesListCmd())
	cmd.AddCommand(newCustomAttributesGetCmd())
	cmd.AddCommand(newCustomAttributesCreateCmd())
	cmd.AddCommand(newCustomAttributesUpdateCmd())
	cmd.AddCommand(newCustomAttributesDeleteCmd())

	return cmd
}

func newCustomAttributesListCmd() *cobra.Command {
	var model string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List custom attribute definitions",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			attrs, err := client.ListCustomAttributes(cmdContext(cmd), model)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(attrs)
			}

			if len(attrs) == 0 {
				fmt.Println("No custom attributes found")
				return nil
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tKEY\tMODEL\tTYPE")
			for _, attr := range attrs {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
					attr.ID,
					attr.AttributeDisplayName,
					attr.AttributeKey,
					attr.AttributeModel,
					attr.AttributeDisplayType,
				)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&model, "model", "", "Filter by model: contact or conversation")

	return cmd
}

func newCustomAttributesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a custom attribute definition by ID",
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

			attr, err := client.GetCustomAttribute(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(attr)
			}

			fmt.Printf("ID: %d\n", attr.ID)
			fmt.Printf("Name: %s\n", attr.AttributeDisplayName)
			fmt.Printf("Key: %s\n", attr.AttributeKey)
			fmt.Printf("Model: %s\n", attr.AttributeModel)
			fmt.Printf("Type: %s\n", attr.AttributeDisplayType)
			if attr.DefaultValue != nil {
				fmt.Printf("Default Value: %v\n", attr.DefaultValue)
			}
			if len(attr.AttributeValues) > 0 {
				fmt.Printf("Values: %v\n", attr.AttributeValues)
			}

			return nil
		},
	}
}

func newCustomAttributesCreateCmd() *cobra.Command {
	var (
		name     string
		key      string
		model    string
		attrType string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a custom attribute definition",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if model == "" {
				return fmt.Errorf("--model is required")
			}
			if attrType == "" {
				return fmt.Errorf("--type is required")
			}

			// Auto-generate key from name if not provided
			if key == "" {
				key = generateAttributeKey(name)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			attr, err := client.CreateCustomAttribute(cmdContext(cmd), name, key, model, attrType)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(attr)
			}

			fmt.Printf("Created custom attribute %d: %s\n", attr.ID, attr.AttributeDisplayName)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Display name for the attribute")
	cmd.Flags().StringVar(&key, "key", "", "Unique key for the attribute (auto-generated from name if not provided)")
	cmd.Flags().StringVar(&model, "model", "", "Model: contact or conversation")
	cmd.Flags().StringVar(&attrType, "type", "", "Type: text, number, date, list, link, or checkbox")

	return cmd
}

func newCustomAttributesUpdateCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a custom attribute definition",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return fmt.Errorf("invalid ID: %w", err)
			}

			if name == "" {
				return fmt.Errorf("--name is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			attr, err := client.UpdateCustomAttribute(cmdContext(cmd), id, name)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(attr)
			}

			fmt.Printf("Updated custom attribute %d: %s\n", attr.ID, attr.AttributeDisplayName)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Display name for the attribute")

	return cmd
}

func newCustomAttributesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a custom attribute definition",
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

			if err := client.DeleteCustomAttribute(cmdContext(cmd), id); err != nil {
				return err
			}

			if !isJSON(cmd) {
				fmt.Printf("Deleted custom attribute %d\n", id)
			}

			return nil
		},
	}
}

// generateAttributeKey converts a display name to a valid attribute key
// by converting to lowercase, replacing spaces and special chars with underscores
func generateAttributeKey(name string) string {
	// Convert to lowercase
	key := strings.ToLower(name)

	// Replace spaces and special characters with underscores
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	key = reg.ReplaceAllString(key, "_")

	// Remove leading/trailing underscores
	key = strings.Trim(key, "_")

	return key
}
