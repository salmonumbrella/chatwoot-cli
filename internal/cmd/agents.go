package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newAgentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Manage agents",
		Long:  "List, create, update, and delete agents in your Chatwoot account",
	}

	cmd.AddCommand(newAgentsListCmd())
	cmd.AddCommand(newAgentsGetCmd())
	cmd.AddCommand(newAgentsCreateCmd())
	cmd.AddCommand(newAgentsUpdateCmd())
	cmd.AddCommand(newAgentsDeleteCmd())
	cmd.AddCommand(newAgentsBulkCreateCmd())

	return cmd
}

func newAgentsListCmd() *cobra.Command {
	cfg := ListConfig[api.Agent]{
		Use:               "list",
		Short:             "List all agents",
		DisablePagination: true,
		EmptyMessage:      "No agents found",
		Fetch: func(ctx context.Context, client *api.Client, _ int, _ int) (ListResult[api.Agent], error) {
			agents, err := client.ListAgents(ctx)
			if err != nil {
				return ListResult[api.Agent]{}, err
			}
			return ListResult[api.Agent]{Items: agents, HasMore: false}, nil
		},
		Headers: []string{"ID", "NAME", "EMAIL", "ROLE", "AVAILABILITY_STATUS"},
		RowFunc: func(agent api.Agent) []string {
			return []string{
				fmt.Sprintf("%d", agent.ID),
				agent.Name,
				agent.Email,
				agent.Role,
				agent.AvailabilityStatus,
			}
		},
	}

	return NewListCommand(cfg, func(ctx context.Context) (*api.Client, error) {
		return getClient()
	})
}

func newAgentsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get agent by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			agent, err := client.GetAgent(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, agent)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tROLE\tAVAILABILITY_STATUS")
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
				agent.ID,
				agent.Name,
				agent.Email,
				agent.Role,
				agent.AvailabilityStatus,
			)

			return nil
		},
	}
}

func newAgentsCreateCmd() *cobra.Command {
	var (
		name  string
		email string
		role  string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new agent",
		Long:  "Create a new agent with the specified name, email, and role (agent or admin)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if email == "" {
				return fmt.Errorf("--email is required")
			}
			if role == "" {
				return fmt.Errorf("--role is required")
			}
			if role != "agent" && role != "admin" {
				return fmt.Errorf("role must be 'agent' or 'admin'")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			agent, err := client.CreateAgent(cmdContext(cmd), name, email, role)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, agent)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tROLE\tAVAILABILITY_STATUS")
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
				agent.ID,
				agent.Name,
				agent.Email,
				agent.Role,
				agent.AvailabilityStatus,
			)

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Agent name")
	cmd.Flags().StringVar(&email, "email", "", "Agent email address")
	cmd.Flags().StringVar(&role, "role", "", "Agent role: agent|admin")

	return cmd
}

func newAgentsUpdateCmd() *cobra.Command {
	var (
		name string
		role string
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an agent",
		Long:  "Update an agent's name and/or role",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			if name == "" && role == "" {
				return fmt.Errorf("at least one of --name or --role must be provided")
			}

			if role != "" && role != "agent" && role != "admin" {
				return fmt.Errorf("role must be 'agent' or 'admin'")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			agent, err := client.UpdateAgent(cmdContext(cmd), id, name, role)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, agent)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tROLE\tAVAILABILITY_STATUS")
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
				agent.ID,
				agent.Name,
				agent.Email,
				agent.Role,
				agent.AvailabilityStatus,
			)

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New agent name")
	cmd.Flags().StringVar(&role, "role", "", "New agent role: agent|admin")

	return cmd
}

func newAgentsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.DeleteAgent(cmdContext(cmd), id); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": id})
			}
			fmt.Printf("Agent %d deleted successfully\n", id)
			return nil
		},
	}
}

func newAgentsBulkCreateCmd() *cobra.Command {
	var emails string

	cmd := &cobra.Command{
		Use:   "bulk-create",
		Short: "Create multiple agents at once",
		Long:  "Create multiple agents at once by providing a comma-separated list of email addresses",
		RunE: func(cmd *cobra.Command, args []string) error {
			if emails == "" {
				return fmt.Errorf("--emails is required")
			}

			// Split emails by comma and trim whitespace
			emailList := strings.Split(emails, ",")
			for i := range emailList {
				emailList[i] = strings.TrimSpace(emailList[i])
			}

			// Filter out empty strings
			var validEmails []string
			for _, email := range emailList {
				if email != "" {
					validEmails = append(validEmails, email)
				}
			}

			if len(validEmails) == 0 {
				return fmt.Errorf("at least one email address is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			agents, err := client.BulkCreateAgents(cmdContext(cmd), validEmails)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, agents)
			}

			fmt.Printf("Created %d agents:\n", len(agents))
			w := newTabWriter()
			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tROLE")
			for _, agent := range agents {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
					agent.ID,
					agent.Name,
					agent.Email,
					agent.Role,
				)
			}
			_ = w.Flush()

			return nil
		},
	}

	cmd.Flags().StringVar(&emails, "emails", "", "Comma-separated list of email addresses (required)")
	_ = cmd.MarkFlagRequired("emails")

	return cmd
}
