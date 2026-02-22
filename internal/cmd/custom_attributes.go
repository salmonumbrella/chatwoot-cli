package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

func newCustomAttributesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "custom-attributes",
		Aliases: []string{"attrs", "ca"},
		Short:   "Manage custom attribute definitions",
		Example: `  # List all custom attributes
  cw custom-attributes list

  # List contact custom attributes
  cw custom-attributes list --model contact

  # Get a custom attribute
  cw custom-attributes get 123

  # Create a custom attribute
  cw custom-attributes create --name "Customer ID" --model contact --type text

  # Update a custom attribute
  cw custom-attributes update 123 --name "Updated Name"

  # Delete a custom attribute
  cw custom-attributes delete 123`,
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
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List custom attribute definitions",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			attrs, err := client.CustomAttributes().List(cmdContext(cmd), model)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, attrs)
			}

			if len(attrs) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No custom attributes found")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
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
		}),
	}

	cmd.Flags().StringVar(&model, "model", "", "Filter by model: contact or conversation")
	flagAlias(cmd.Flags(), "model", "mo")

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "attribute_display_name", "attribute_key"},
		"default": {"id", "attribute_display_name", "attribute_key", "attribute_model", "attribute_display_type"},
		"debug":   {"id", "attribute_display_name", "attribute_key", "attribute_model", "attribute_display_type", "default_value", "attribute_values"},
	})

	return cmd
}

func newCustomAttributesGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <id>",
		Aliases: []string{"g"},
		Short:   "Get a custom attribute definition by ID",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "custom attribute")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			attr, err := client.CustomAttributes().Get(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, attr)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ID: %d\n", attr.ID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\n", attr.AttributeDisplayName)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Key: %s\n", attr.AttributeKey)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Model: %s\n", attr.AttributeModel)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Type: %s\n", attr.AttributeDisplayType)
			if attr.DefaultValue != nil {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Default Value: %v\n", attr.DefaultValue)
			}
			if len(attr.AttributeValues) > 0 {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Values: %v\n", attr.AttributeValues)
			}

			return nil
		}),
	}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "attribute_display_name", "attribute_key"},
		"default": {"id", "attribute_display_name", "attribute_key", "attribute_model", "attribute_display_type"},
		"debug":   {"id", "attribute_display_name", "attribute_key", "attribute_model", "attribute_display_type", "default_value", "attribute_values"},
	})

	return cmd
}

func newCustomAttributesCreateCmd() *cobra.Command {
	var (
		name     string
		key      string
		model    string
		attrType string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"mk"},
		Short:   "Create a custom attribute definition",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
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

			attr, err := client.CustomAttributes().Create(cmdContext(cmd), name, key, model, attrType)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, attr)
			}

			printAction(cmd, "Created", "custom attribute", attr.ID, attr.AttributeDisplayName)
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Display name for the attribute")
	cmd.Flags().StringVar(&key, "key", "", "Unique key for the attribute (auto-generated from name if not provided)")
	cmd.Flags().StringVar(&model, "model", "", "Model: contact or conversation")
	cmd.Flags().StringVar(&attrType, "type", "", "Type: text, number, date, list, link, or checkbox")
	flagAlias(cmd.Flags(), "model", "mo")
	flagAlias(cmd.Flags(), "name", "nm")
	flagAlias(cmd.Flags(), "key", "ky")
	flagAlias(cmd.Flags(), "type", "ty")

	return cmd
}

func newCustomAttributesUpdateCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:     "update <id>",
		Aliases: []string{"up"},
		Short:   "Update a custom attribute definition",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "custom attribute")
			if err != nil {
				return err
			}

			if name == "" {
				return fmt.Errorf("--name is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			attr, err := client.CustomAttributes().Update(cmdContext(cmd), id, name)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, attr)
			}

			printAction(cmd, "Updated", "custom attribute", attr.ID, attr.AttributeDisplayName)
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Display name for the attribute")
	flagAlias(cmd.Flags(), "name", "nm")

	return cmd
}

func newCustomAttributesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm"},
		Short:   "Delete a custom attribute definition",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "custom attribute")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.CustomAttributes().Delete(cmdContext(cmd), id); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": id})
			}
			printAction(cmd, "Deleted", "custom attribute", id, "")
			return nil
		}),
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
