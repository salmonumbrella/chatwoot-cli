package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
)

func newConfigStoreKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "store-keys",
		Short: "Manage ecommerce store key mappings",
		Long:  "View and discover custom attribute keys used for ecommerce store identification in contacts",
	}

	listCmd := newConfigStoreKeysListCmd()
	cmd.AddCommand(listCmd)
	cmd.AddCommand(newConfigStoreKeysDiscoverCmd())

	// Default action: delegate to list when no subcommand is given.
	cmd.RunE = listCmd.RunE

	return cmd
}

func newConfigStoreKeysListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List configured store key mappings",
		Example: `  # Show current store key configuration
  cw config store-keys list

  # JSON output
  cw config store-keys list -o json`,
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			raw := os.Getenv(envLightContactStoreMap)
			mappings := parseLightContactStoreMap(raw)

			tierKey := os.Getenv(envLightContactTierKey)
			if tierKey == "" {
				tierKey = defaultLightContactTierKey
			}

			configured := len(mappings) > 0

			if isJSON(cmd) {
				result := map[string]any{
					"configured": configured,
					"tier_key":   tierKey,
					"env_var":    envLightContactStoreMap,
					"mappings":   mappings,
				}
				// Ensure mappings is an empty object, not null.
				if result["mappings"] == nil {
					result["mappings"] = map[string]string{}
				}
				return printJSON(cmd, result)
			}

			if !configured {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(),
					"Store key mappings are not configured.")
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
				_, _ = fmt.Fprintf(cmd.OutOrStdout(),
					"Set %s to define alias:attribute_key pairs:\n", envLightContactStoreMap)
				_, _ = fmt.Fprintln(cmd.OutOrStdout(),
					"  export CW_CONTACT_LIGHT_STORE_KEYS=\"alias1:custom_attr_key_1,alias2:custom_attr_key_2\"")
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
				_, _ = fmt.Fprintln(cmd.OutOrStdout(),
					"Use 'cw config store-keys discover <CONTACT_ID>' to find attribute keys from a contact.")
				return nil
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Tier key: %s\n", tierKey)
			_, _ = fmt.Fprintln(cmd.OutOrStdout())

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ALIAS\tATTRIBUTE KEY")

			// Sort aliases for deterministic output.
			aliases := make([]string, 0, len(mappings))
			for alias := range mappings {
				aliases = append(aliases, alias)
			}
			sort.Strings(aliases)

			for _, alias := range aliases {
				_, _ = fmt.Fprintf(w, "%s\t%s\n", alias, mappings[alias])
			}

			return nil
		}),
	}
}

func newConfigStoreKeysDiscoverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "discover <CONTACT_ID>",
		Short: "Discover store attribute keys from a contact's custom attributes",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented yet")
		}),
	}
}
