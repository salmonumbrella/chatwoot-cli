package cmd

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

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
				// Guard nil map before assigning to any-typed field
				// (typed nil in interface is non-nil; same class as null-slice bug).
				if mappings == nil {
					mappings = map[string]string{}
				}
				result := map[string]any{
					"configured": configured,
					"tier_key":   tierKey,
					"env_var":    envLightContactStoreMap,
					"mappings":   mappings,
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
		Short: "Auto-discover store keys from a contact's custom attributes",
		Long: `Fetch a contact and inspect its custom_attributes for URL-shaped values
that look like ecommerce store admin links. Outputs the discovered keys
and a suggested CW_CONTACT_LIGHT_STORE_KEYS configuration.`,
		Example: `  # Discover store keys from contact 42
  cw config store-keys discover 42

  # JSON output
  cw config store-keys discover 42 -j`,
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			contactID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid contact ID: %w", err)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			contact, err := client.Contacts().Get(cmdContext(cmd), contactID)
			if err != nil {
				return err
			}

			discovered := discoverStoreKeys(contact.CustomAttributes)

			if isJSON(cmd) {
				result := map[string]any{
					"contact_id":      contactID,
					"discovered_keys": discovered,
				}
				if len(discovered) > 0 {
					result["suggested_config"] = buildSuggestedConfig(discovered)
				}
				return printJSON(cmd, result)
			}

			if len(discovered) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No ecommerce store keys found in contact's custom attributes.")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintf(w, "Found %d store key(s) in contact %d:\n", len(discovered), contactID)
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintln(w, "ATTRIBUTE KEY\tVALUE")
			for _, dk := range discovered {
				_, _ = fmt.Fprintf(w, "%s\t%s\n", dk.Key, dk.Value)
			}
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintln(w, "Suggested config:")
			_, _ = fmt.Fprintf(w, "  export %s='%s'\n", envLightContactStoreMap, buildSuggestedConfig(discovered))

			return nil
		}),
	}
}

type discoveredKey struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func discoverStoreKeys(ca map[string]any) []discoveredKey {
	if len(ca) == 0 {
		return []discoveredKey{}
	}

	keys := make([]string, 0, len(ca))
	for k := range ca {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var discovered []discoveredKey
	for _, key := range keys {
		v, ok := ca[key].(string)
		if !ok || v == "" {
			continue
		}
		if !looksLikeStoreURL(v) {
			continue
		}
		discovered = append(discovered, discoveredKey{Key: key, Value: v})
	}
	return discovered
}

func looksLikeStoreURL(v string) bool {
	if !strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "https://") {
		return false
	}
	parts := strings.SplitN(v, "://", 2)
	if len(parts) < 2 {
		return false
	}
	path := parts[1]
	segments := strings.Split(path, "/")
	return len(segments) >= 3 && segments[len(segments)-1] != ""
}

func buildSuggestedConfig(keys []discoveredKey) string {
	parts := make([]string, 0, len(keys))
	for i, dk := range keys {
		alias := fmt.Sprintf("store%d", i+1)
		parts = append(parts, alias+":"+dk.Key)
	}
	return strings.Join(parts, ",")
}
