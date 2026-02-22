package cmd

import (
	"context"
	"fmt"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/spf13/cobra"
)

func newAgentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agents",
		Aliases: []string{"agent", "a"},
		Short:   "Manage agents",
		Long:    "List, create, update, and delete agents in your Chatwoot account",
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
	var light bool

	cfg := ListConfig[api.Agent]{
		Use:               "list",
		Short:             "List all agents",
		DisablePagination: true,
		EmptyMessage:      "No agents found",
		AgentTransform: func(_ context.Context, _ *api.Client, items []api.Agent) (any, error) {
			if light {
				return buildLightAgents(items), nil
			}
			return nil, nil
		},
		JSONTransform: func(_ context.Context, _ *api.Client, items []api.Agent) (any, error) {
			if !light {
				return items, nil
			}
			return buildLightAgents(items), nil
		},
		ForceJSON: func(_ *cobra.Command) bool {
			return light
		},
		Fetch: func(ctx context.Context, client *api.Client, _ int, _ int) (ListResult[api.Agent], error) {
			agents, err := client.Agents().List(ctx)
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

	cmd := NewListCommand(cfg, func(ctx context.Context) (*api.Client, error) {
		return getClient()
	})
	cmd.Aliases = []string{"ls"}

	cmd.Flags().BoolVar(&light, "light", false, "Return minimal agent payload for lookup")
	flagAlias(cmd.Flags(), "light", "li")

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name", "email"},
		"default": {"id", "name", "email", "role"},
		"debug":   {"id", "name", "email", "role", "availability_status", "thumbnail", "confirmed_at"},
	})
	registerFieldSchema(cmd, "agent")

	return cmd
}

func newAgentsGetCmd() *cobra.Command {
	var emit string

	cmd := &cobra.Command{
		Use:     "get <id>",
		Aliases: []string{"g"},
		Short:   "Get agent by ID",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "agent")
			if err != nil {
				return err
			}

			mode, err := normalizeEmitFlag(emit)
			if err != nil {
				return err
			}
			if mode == "id" || mode == "url" {
				_, err := maybeEmit(cmd, mode, "agent", id, nil)
				return err
			}

			if handled, err := handleURLFlag(cmd, "agents", id); handled {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			agent, err := client.Agents().Get(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if emitted, err := maybeEmit(cmd, emit, "agent", agent.ID, agent); emitted {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, agent)
			}
			return printAgentDetails(cmd.OutOrStdout(), agent)
		}),
	}

	cmd.Flags().Bool("url", false, "Print the Chatwoot web UI URL for this resource and exit")
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name", "email"},
		"default": {"id", "name", "email", "role"},
		"debug":   {"id", "name", "email", "role", "availability_status", "thumbnail", "confirmed_at"},
	})
	registerFieldSchema(cmd, "agent")

	return cmd
}

func newAgentsCreateCmd() *cobra.Command {
	var (
		name  string
		email string
		role  string
		emit  string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"mk"},
		Short:   "Create a new agent",
		Long:    "Create a new agent with the specified name, email, and role (agent or admin)",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
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

			agent, err := client.Agents().Create(cmdContext(cmd), name, email, role)
			if err != nil {
				return err
			}

			if emitted, err := maybeEmit(cmd, emit, "agent", agent.ID, agent); emitted {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, agent)
			}

			w := newTabWriterFromCmd(cmd)
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
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Agent name")
	cmd.Flags().StringVar(&email, "email", "", "Agent email address")
	cmd.Flags().StringVar(&role, "role", "", "Agent role: agent|admin")
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")

	return cmd
}

func newAgentsUpdateCmd() *cobra.Command {
	var (
		name string
		role string
		emit string
	)

	cmd := &cobra.Command{
		Use:     "update <id>",
		Aliases: []string{"up"},
		Short:   "Update an agent",
		Long:    "Update an agent's name and/or role",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "agent")
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

			agent, err := client.Agents().Update(cmdContext(cmd), id, name, role)
			if err != nil {
				return err
			}

			if emitted, err := maybeEmit(cmd, emit, "agent", agent.ID, agent); emitted {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, agent)
			}

			w := newTabWriterFromCmd(cmd)
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
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "New agent name")
	cmd.Flags().StringVar(&role, "role", "", "New agent role: agent|admin")
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")

	return cmd
}

func newAgentsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm"},
		Short:   "Delete an agent",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "agent")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.Agents().Delete(cmdContext(cmd), id); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"deleted": true, "id": id})
			}
			printAction(cmd, "Deleted", "agent", id, "")
			return nil
		}),
	}
}

func newAgentsBulkCreateCmd() *cobra.Command {
	var emails string

	cmd := &cobra.Command{
		Use:     "bulk-create",
		Aliases: []string{"bc"},
		Short:   "Create multiple agents at once",
		Long:    "Create multiple agents at once by providing a list of email addresses",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if emails == "" {
				return fmt.Errorf("--emails is required")
			}

			validEmails, err := ParseStringListFlag(emails)
			if err != nil {
				return fmt.Errorf("invalid --emails value: %w", err)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			agents, err := client.Agents().BulkCreate(cmdContext(cmd), validEmails)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, agents)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created %d agents:\n", len(agents))
			w := newTabWriterFromCmd(cmd)
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
		}),
	}

	cmd.Flags().StringVar(&emails, "emails", "", "Email addresses (CSV/whitespace/JSON array, or @- / @path) (required)")
	_ = cmd.MarkFlagRequired("emails")

	return cmd
}
