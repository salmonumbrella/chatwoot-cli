package cmd

import (
	"fmt"

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

	return cmd
}

func newAgentsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			agents, err := client.ListAgents(cmdContext(cmd))
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(agents)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tROLE\tAVAILABILITY_STATUS")
			for _, agent := range agents {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
					agent.ID,
					agent.Name,
					agent.Email,
					agent.Role,
					agent.AvailabilityStatus,
				)
			}

			return nil
		},
	}
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
				return printJSON(agent)
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
				return printJSON(agent)
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
				return printJSON(agent)
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

			if !isJSON(cmd) {
				fmt.Printf("Agent %d deleted successfully\n", id)
			}

			return nil
		},
	}
}
