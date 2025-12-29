package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newAutomationRulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "automation-rules",
		Aliases: []string{"automation", "rules"},
		Short:   "Manage automation rules",
		Long:    "Create, list, update, and delete automation rules in your Chatwoot account",
	}

	cmd.AddCommand(newAutomationRulesListCmd())
	cmd.AddCommand(newAutomationRulesGetCmd())
	cmd.AddCommand(newAutomationRulesCreateCmd())
	cmd.AddCommand(newAutomationRulesUpdateCmd())
	cmd.AddCommand(newAutomationRulesDeleteCmd())

	return cmd
}

func newAutomationRulesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all automation rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			rules, err := client.ListAutomationRules(cmdContext(cmd))
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, rules)
			}

			if len(rules) == 0 {
				fmt.Println("No automation rules found")
				return nil
			}

			w := newTabWriter()
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
		},
	}
}

func newAutomationRulesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get automation rule by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return fmt.Errorf("invalid rule ID: %w", err)
			}

			rule, err := client.GetAutomationRule(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, rule)
			}

			fmt.Printf("ID:          %d\n", rule.ID)
			fmt.Printf("Name:        %s\n", rule.Name)
			fmt.Printf("Event:       %s\n", rule.EventName)
			fmt.Printf("Active:      %t\n", rule.Active)
			fmt.Printf("Description: %s\n", rule.Description)
			fmt.Printf("Conditions:  %d\n", len(rule.Conditions))
			fmt.Printf("Actions:     %d\n", len(rule.Actions))
			return nil
		},
	}
}

func newAutomationRulesCreateCmd() *cobra.Command {
	var (
		name       string
		eventName  string
		conditions string
		actions    string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new automation rule",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			rule, err := client.CreateAutomationRule(cmdContext(cmd), name, eventName, conditionsData, actionsData)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, rule)
			}

			fmt.Printf("Created automation rule #%d: %s\n", rule.ID, rule.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Rule name (required)")
	cmd.Flags().StringVar(&eventName, "event-name", "", "Event name (required)")
	cmd.Flags().StringVar(&conditions, "conditions", "", "Conditions as JSON array (required)")
	cmd.Flags().StringVar(&actions, "actions", "", "Actions as JSON array (required)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("event-name")
	_ = cmd.MarkFlagRequired("conditions")
	_ = cmd.MarkFlagRequired("actions")

	return cmd
}

func newAutomationRulesUpdateCmd() *cobra.Command {
	var (
		name       string
		conditions string
		actions    string
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an automation rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return fmt.Errorf("invalid rule ID: %w", err)
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

			rule, err := client.UpdateAutomationRule(cmdContext(cmd), id, name, conditionsData, actionsData)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, rule)
			}

			fmt.Printf("Updated automation rule #%d: %s\n", rule.ID, rule.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Rule name")
	cmd.Flags().StringVar(&conditions, "conditions", "", "Conditions as JSON array")
	cmd.Flags().StringVar(&actions, "actions", "", "Actions as JSON array")

	return cmd
}

func newAutomationRulesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an automation rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return fmt.Errorf("invalid rule ID: %w", err)
			}

			if err := client.DeleteAutomationRule(cmdContext(cmd), id); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": id})
			}

			fmt.Printf("Deleted automation rule #%d\n", id)
			return nil
		},
	}
}
