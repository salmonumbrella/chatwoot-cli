package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/schema"
	"github.com/spf13/cobra"
)

func newSchemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Discover API resource schemas",
		Long:  "List and show schema definitions for Chatwoot API resources",
		Example: strings.TrimSpace(`
  # List available schemas
  chatwoot schema list

  # Show conversation schema
  chatwoot schema show conversation

  # Show schema as JSON (for programmatic use)
  chatwoot schema show conversation -o json
`),
	}

	cmd.AddCommand(newSchemaListCmd())
	cmd.AddCommand(newSchemaShowCmd())

	return cmd
}

func newSchemaListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available schemas",
		Long:  "List all registered resource schemas with their descriptions",
		Example: strings.TrimSpace(`
  # List all schemas
  chatwoot schema list

  # List as JSON
  chatwoot schema list -o json
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			names := schema.List()

			if isJSON(cmd) {
				// Build a list of schema summaries
				type schemaSummary struct {
					Name        string `json:"name"`
					Description string `json:"description"`
				}
				summaries := make([]schemaSummary, 0, len(names))
				for _, name := range names {
					s, _ := schema.Get(name)
					summaries = append(summaries, schemaSummary{
						Name:        name,
						Description: s.Description,
					})
				}
				return printJSON(cmd, summaries)
			}

			if len(names) == 0 {
				fmt.Println("No schemas registered")
				return nil
			}

			w := newTabWriter()
			_, _ = fmt.Fprintln(w, "RESOURCE\tDESCRIPTION")
			for _, name := range names {
				s, _ := schema.Get(name)
				desc := s.Description
				if len(desc) > 60 {
					desc = desc[:57] + "..."
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\n", name, desc)
			}
			_ = w.Flush()

			return nil
		},
	}
}

func newSchemaShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <resource>",
		Short: "Show schema for a resource",
		Long:  "Display the full schema definition for a resource type",
		Example: strings.TrimSpace(`
  # Show conversation schema
  chatwoot schema show conversation

  # Show contact schema as JSON
  chatwoot schema show contact -o json

  # Show message schema
  chatwoot schema show message
`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			s, err := schema.Get(name)
			if err != nil {
				available := schema.List()
				return fmt.Errorf("schema %q not found; available: %s", name, strings.Join(available, ", "))
			}

			if isJSON(cmd) {
				return printJSON(cmd, s)
			}

			// Pretty print the schema
			printSchemaText(name, s)

			return nil
		},
	}
}

func printSchemaText(name string, s *schema.Schema) {
	fmt.Printf("Schema: %s\n", name)
	fmt.Printf("Type: %s\n", s.Type)
	if s.Description != "" {
		fmt.Printf("Description: %s\n", s.Description)
	}
	fmt.Println()

	if len(s.Properties) > 0 {
		fmt.Println("Fields:")

		// Sort property names for consistent output
		propNames := make([]string, 0, len(s.Properties))
		for propName := range s.Properties {
			propNames = append(propNames, propName)
		}
		sort.Strings(propNames)

		// Create a set of required fields for quick lookup
		requiredSet := make(map[string]bool)
		for _, req := range s.Required {
			requiredSet[req] = true
		}

		for _, propName := range propNames {
			prop := s.Properties[propName]
			printField(propName, prop, requiredSet[propName], "  ")
		}
	}

	if len(s.Required) > 0 {
		fmt.Println()
		fmt.Printf("Required: %s\n", strings.Join(s.Required, ", "))
	}
}

func printField(name string, s *schema.Schema, required bool, indent string) {
	reqMarker := ""
	if required {
		reqMarker = " (required)"
	}

	typeName := s.Type
	if s.Items != nil {
		typeName = fmt.Sprintf("array<%s>", s.Items.Type)
	}

	fmt.Printf("%s%s: %s%s\n", indent, name, typeName, reqMarker)

	if s.Description != "" {
		fmt.Printf("%s  %s\n", indent, s.Description)
	}

	if len(s.Enum) > 0 {
		fmt.Printf("%s  Allowed values: %s\n", indent, strings.Join(s.Enum, ", "))
	}
}
