package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newAutomationRulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "automation-rules",
		Aliases: []string{"automation", "rules", "ar"},
		Short:   "Manage automation rules",
		Long:    "Create, list, update, and delete automation rules in your Chatwoot account",
	}

	cmd.AddCommand(newAutomationRulesListCmd())
	cmd.AddCommand(newAutomationRulesGetCmd())
	cmd.AddCommand(newAutomationRulesCreateCmd())
	cmd.AddCommand(newAutomationRulesUpdateCmd())
	cmd.AddCommand(newAutomationRulesDeleteCmd())
	cmd.AddCommand(newAutomationRulesCloneCmd())

	return cmd
}

func newAutomationRulesListCmd() *cobra.Command {
	var light bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all automation rules",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			rules, err := client.AutomationRules().List(cmdContext(cmd))
			if err != nil {
				return err
			}

			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, buildLightAutomationRules(rules))
			}

			if isJSON(cmd) {
				return printJSON(cmd, rules)
			}

			if len(rules) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No automation rules found")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()
			_, _ = fmt.Fprintln(w, "ID\tNAME\tEVENT\tACTIVE")
			for _, rule := range rules {
				active := "no"
				if rule.Active {
					active = "yes"
				}
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", rule.ID, rule.Name, rule.EventName, active)
			}
			return nil
		}),
	}

	cmd.Flags().BoolVar(&light, "light", false, "Return minimal automation rule payload for lookup")
	flagAlias(cmd.Flags(), "light", "li")

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name", "event_name"},
		"default": {"id", "name", "event_name", "active"},
		"debug":   {"id", "name", "description", "event_name", "conditions", "actions", "active", "account_id"},
	})

	return cmd
}

func newAutomationRulesGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <id>",
		Aliases: []string{"g"},
		Short:   "Get automation rule by ID",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			id, err := parseIDOrURL(args[0], "rule")
			if err != nil {
				return err
			}

			rule, err := client.AutomationRules().Get(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, rule)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ID:          %d\n", rule.ID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Name:        %s\n", rule.Name)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Event:       %s\n", rule.EventName)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Active:      %t\n", rule.Active)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Description: %s\n", rule.Description)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conditions:  %d\n", len(rule.Conditions))
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Actions:     %d\n", len(rule.Actions))
			return nil
		}),
	}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name", "event_name"},
		"default": {"id", "name", "event_name", "active"},
		"debug":   {"id", "name", "description", "event_name", "conditions", "actions", "active", "account_id"},
	})

	return cmd
}

func newAutomationRulesCreateCmd() *cobra.Command {
	var (
		name       string
		eventName  string
		conditions string
		actions    string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"mk"},
		Short:   "Create a new automation rule",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			var conditionsData []map[string]any
			if err := json.Unmarshal([]byte(conditions), &conditionsData); err != nil {
				return fmt.Errorf("invalid conditions JSON: %w", err)
			}

			var actionsData []map[string]any
			if err := json.Unmarshal([]byte(actions), &actionsData); err != nil {
				return fmt.Errorf("invalid actions JSON: %w", err)
			}

			rule, err := client.AutomationRules().Create(cmdContext(cmd), name, eventName, conditionsData, actionsData)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, rule)
			}

			printAction(cmd, "Created", "automation rule", rule.ID, rule.Name)
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Rule name (required)")
	cmd.Flags().StringVar(&eventName, "event-name", "", "Event name (required)")
	cmd.Flags().StringVar(&conditions, "conditions", "", "Conditions as JSON array (required)")
	cmd.Flags().StringVar(&actions, "actions", "", "Actions as JSON array (required)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("event-name")
	_ = cmd.MarkFlagRequired("conditions")
	_ = cmd.MarkFlagRequired("actions")
	flagAlias(cmd.Flags(), "name", "nm")
	flagAlias(cmd.Flags(), "event-name", "ev")
	flagAlias(cmd.Flags(), "conditions", "cnd")
	flagAlias(cmd.Flags(), "actions", "act")

	return cmd
}

func newAutomationRulesUpdateCmd() *cobra.Command {
	var (
		name       string
		conditions string
		actions    string
		active     string
	)

	cmd := &cobra.Command{
		Use:     "update <id>",
		Aliases: []string{"up"},
		Short:   "Update an automation rule",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			id, err := parseIDOrURL(args[0], "rule")
			if err != nil {
				return err
			}

			var conditionsData []map[string]any
			if conditions != "" {
				if err := json.Unmarshal([]byte(conditions), &conditionsData); err != nil {
					return fmt.Errorf("invalid conditions JSON: %w", err)
				}
			}

			var actionsData []map[string]any
			if actions != "" {
				if err := json.Unmarshal([]byte(actions), &actionsData); err != nil {
					return fmt.Errorf("invalid actions JSON: %w", err)
				}
			}

			var activeBool *bool
			if cmd.Flags().Changed("active") {
				switch active {
				case "true", "yes", "1":
					b := true
					activeBool = &b
				case "false", "no", "0":
					b := false
					activeBool = &b
				default:
					return fmt.Errorf("invalid --active value %q: must be true/false", active)
				}
			}

			rule, err := client.AutomationRules().Update(cmdContext(cmd), id, name, conditionsData, actionsData, activeBool)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, rule)
			}

			printAction(cmd, "Updated", "automation rule", rule.ID, rule.Name)
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Rule name")
	cmd.Flags().StringVar(&conditions, "conditions", "", "Conditions as JSON array")
	cmd.Flags().StringVar(&actions, "actions", "", "Actions as JSON array")
	cmd.Flags().StringVar(&active, "active", "", "Enable or disable the rule (true/false)")
	flagAlias(cmd.Flags(), "name", "nm")
	flagAlias(cmd.Flags(), "conditions", "cnd")
	flagAlias(cmd.Flags(), "actions", "act")
	flagAlias(cmd.Flags(), "active", "on")

	return cmd
}

func newAutomationRulesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm"},
		Short:   "Delete an automation rule",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			id, err := parseIDOrURL(args[0], "rule")
			if err != nil {
				return err
			}

			if err := client.AutomationRules().Delete(cmdContext(cmd), id); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": id})
			}

			printAction(cmd, "Deleted", "automation rule", id, "")
			return nil
		}),
	}
}

func newAutomationRulesCloneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clone <id>",
		Short: "Clone an existing automation rule",
		Long:  "Create a copy of an existing automation rule. The cloned rule will be inactive by default.",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "rule")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			rule, err := client.AutomationRules().Clone(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, rule)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Cloned automation rule #%d to new rule #%d: %s\n", id, rule.ID, rule.Name)
			return nil
		}),
	}
}
