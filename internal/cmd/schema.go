package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/schema"
	"github.com/spf13/cobra"
)

func newSchemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "schema",
		Aliases: []string{"sc"},
		Short:   "Discover API resource schemas",
		Long:    "List and show schema definitions for Chatwoot API resources",
		Example: strings.TrimSpace(`
  # List available schemas
  cw schema list

  # Show conversation schema
  cw schema show conversation

  # Show schema as JSON (for programmatic use)
  cw schema show conversation -o json
`),
	}

	cmd.AddCommand(newSchemaListCmd())
	cmd.AddCommand(newSchemaShowCmd())

	return cmd
}

func newSchemaListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List available schemas",
		Long:    "List all registered resource schemas with their descriptions",
		Example: strings.TrimSpace(`
  # List all schemas
  cw schema list

  # List as JSON
  cw schema list -o json
`),
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
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
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No schemas registered")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
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
		}),
	}
}

func newSchemaShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <resource>",
		Short: "Show schema for a resource",
		Long:  "Display the full schema definition for a resource type",
		Example: strings.TrimSpace(`
  # Show conversation schema
  cw schema show conversation

  # Show contact schema as JSON
  cw schema show contact -o json

  # Show message schema
  cw schema show message
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
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
			printSchemaText(cmd.OutOrStdout(), name, s)

			return nil
		}),
	}
}

func printSchemaText(out io.Writer, name string, s *schema.Schema) {
	_, _ = fmt.Fprintf(out, "Schema: %s\n", name)
	_, _ = fmt.Fprintf(out, "Type: %s\n", s.Type)
	if s.Description != "" {
		_, _ = fmt.Fprintf(out, "Description: %s\n", s.Description)
	}
	_, _ = fmt.Fprintln(out)

	if len(s.Properties) > 0 {
		_, _ = fmt.Fprintln(out, "Fields:")

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
			printField(out, propName, prop, requiredSet[propName], "  ")
		}
	}

	if len(s.Required) > 0 {
		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintf(out, "Required: %s\n", strings.Join(s.Required, ", "))
	}
}

func printField(out io.Writer, name string, s *schema.Schema, required bool, indent string) {
	reqMarker := ""
	if required {
		reqMarker = " (required)"
	}

	typeName := s.Type
	if s.Items != nil {
		typeName = fmt.Sprintf("array<%s>", s.Items.Type)
	}

	_, _ = fmt.Fprintf(out, "%s%s: %s%s\n", indent, name, typeName, reqMarker)

	if s.Description != "" {
		_, _ = fmt.Fprintf(out, "%s  %s\n", indent, s.Description)
	}

	if len(s.Enum) > 0 {
		_, _ = fmt.Fprintf(out, "%s  Allowed values: %s\n", indent, strings.Join(s.Enum, ", "))
	}
}
