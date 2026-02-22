package cmd

import (
	"context"
	"fmt"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/spf13/cobra"
)

func newTeamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "teams",
		Aliases: []string{"team", "t"},
		Short:   "Manage teams",
		Long:    "Create, list, update, and manage teams and their members",
	}

	cmd.AddCommand(newTeamsListCmd())
	cmd.AddCommand(newTeamsGetCmd())
	cmd.AddCommand(newTeamsCreateCmd())
	cmd.AddCommand(newTeamsUpdateCmd())
	cmd.AddCommand(newTeamsDeleteCmd())
	cmd.AddCommand(newTeamsMembersCmd())
	cmd.AddCommand(newTeamsMembersAddCmd())
	cmd.AddCommand(newTeamsMembersRemoveCmd())

	return cmd
}

func newTeamsListCmd() *cobra.Command {
	var light bool

	cfg := ListConfig[api.Team]{
		Use:               "list",
		Short:             "List all teams",
		DisablePagination: true,
		EmptyMessage:      "No teams found",
		AgentTransform: func(_ context.Context, _ *api.Client, items []api.Team) (any, error) {
			if light {
				return buildLightTeams(items), nil
			}
			return nil, nil
		},
		JSONTransform: func(_ context.Context, _ *api.Client, items []api.Team) (any, error) {
			if !light {
				return items, nil
			}
			return buildLightTeams(items), nil
		},
		ForceJSON: func(_ *cobra.Command) bool {
			return light
		},
		Fetch: func(ctx context.Context, client *api.Client, _ int, _ int) (ListResult[api.Team], error) {
			teams, err := client.Teams().List(ctx)
			if err != nil {
				return ListResult[api.Team]{}, err
			}
			return ListResult[api.Team]{Items: teams, HasMore: false}, nil
		},
		Headers: []string{"ID", "NAME", "DESCRIPTION", "AUTO-ASSIGN"},
		RowFunc: func(team api.Team) []string {
			autoAssign := "no"
			if team.AllowAutoAssign {
				autoAssign = "yes"
			}
			return []string{
				fmt.Sprintf("%d", team.ID),
				team.Name,
				team.Description,
				autoAssign,
			}
		},
	}

	cmd := NewListCommand(cfg, func(ctx context.Context) (*api.Client, error) {
		return getClient()
	})
	cmd.Aliases = []string{"ls"}

	cmd.Flags().BoolVar(&light, "light", false, "Return minimal team payload for lookup")
	flagAlias(cmd.Flags(), "light", "li")

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name"},
		"default": {"id", "name", "description", "allow_auto_assign"},
		"debug":   {"id", "name", "description", "allow_auto_assign", "account_id"},
	})
	registerFieldSchema(cmd, "team")

	return cmd
}

func newTeamsGetCmd() *cobra.Command {
	var emit string

	cmd := &cobra.Command{
		Use:     "get <id>",
		Aliases: []string{"g"},
		Short:   "Get a team by ID",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "team")
			if err != nil {
				return err
			}

			mode, err := normalizeEmitFlag(emit)
			if err != nil {
				return err
			}
			if mode == "id" || mode == "url" {
				_, err := maybeEmit(cmd, mode, "team", id, nil)
				return err
			}

			if handled, err := handleURLFlag(cmd, "teams", id); handled {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			team, err := client.Teams().Get(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if emitted, err := maybeEmit(cmd, emit, "team", team.ID, team); emitted {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, team)
			}
			return printTeamDetails(cmd.OutOrStdout(), team)
		}),
	}

	cmd.Flags().Bool("url", false, "Print the Chatwoot web UI URL for this resource and exit")
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name"},
		"default": {"id", "name", "description", "allow_auto_assign"},
		"debug":   {"id", "name", "description", "allow_auto_assign", "account_id"},
	})
	registerFieldSchema(cmd, "team")

	return cmd
}

func newTeamsCreateCmd() *cobra.Command {
	var name, description, emit string

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"mk"},
		Short:   "Create a new team",
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			team, err := client.Teams().Create(cmdContext(cmd), name, description)
			if err != nil {
				return err
			}

			if emitted, err := maybeEmit(cmd, emit, "team", team.ID, team); emitted {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, team)
			}

			printAction(cmd, "Created", "team", team.ID, team.Name)
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Team name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Team description")
	flagAlias(cmd.Flags(), "description", "desc")
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")
	flagAlias(cmd.Flags(), "name", "nm")

	return cmd
}

func newTeamsUpdateCmd() *cobra.Command {
	var name, description, emit string

	cmd := &cobra.Command{
		Use:     "update <id>",
		Aliases: []string{"up"},
		Short:   "Update a team",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "team")
			if err != nil {
				return err
			}

			if name == "" && description == "" {
				return fmt.Errorf("at least one of --name or --description must be provided")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			team, err := client.Teams().Update(cmdContext(cmd), id, name, description)
			if err != nil {
				return err
			}

			if emitted, err := maybeEmit(cmd, emit, "team", team.ID, team); emitted {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, team)
			}

			printAction(cmd, "Updated", "team", team.ID, team.Name)
			return nil
		}),
	}

	cmd.Flags().StringVar(&name, "name", "", "Team name")
	cmd.Flags().StringVar(&description, "description", "", "Team description")
	flagAlias(cmd.Flags(), "description", "desc")
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")
	flagAlias(cmd.Flags(), "name", "nm")

	return cmd
}

func newTeamsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm"},
		Short:   "Delete a team",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "team")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.Teams().Delete(cmdContext(cmd), id); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]interface{}{
					"id":      id,
					"deleted": true,
				})
			}

			printAction(cmd, "Deleted", "team", id, "")
			return nil
		}),
	}
}

func newTeamsMembersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "members <id>",
		Short: "List team members",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "team")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			members, err := client.Teams().ListMembers(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, members)
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tROLE\tSTATUS")
			for _, m := range members {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", m.ID, m.Name, m.Email, m.Role, m.AvailabilityStatus)
			}

			return nil
		}),
	}
}

func newTeamsMembersAddCmd() *cobra.Command {
	var userIDsStr string

	cmd := &cobra.Command{
		Use:     "members-add <id>",
		Aliases: []string{"ma"},
		Short:   "Add members to a team",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "team")
			if err != nil {
				return err
			}

			if userIDsStr == "" {
				return fmt.Errorf("--user-ids is required")
			}

			userIDs, err := ParseResourceIDListFlag(userIDsStr, "agent")
			if err != nil {
				return fmt.Errorf("invalid user IDs: %w", err)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.Teams().AddMembers(cmdContext(cmd), id, userIDs); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]interface{}{
					"team_id":     id,
					"added_count": len(userIDs),
					"user_ids":    userIDs,
				})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added %d member(s) to team ID: %d\n", len(userIDs), id)
			return nil
		}),
	}

	cmd.Flags().StringVar(&userIDsStr, "user-ids", "", "User IDs (CSV, whitespace, JSON array; or @- / @path) (required)")
	flagAlias(cmd.Flags(), "user-ids", "uids")

	return cmd
}

func newTeamsMembersRemoveCmd() *cobra.Command {
	var userIDsStr string

	cmd := &cobra.Command{
		Use:     "members-remove <id>",
		Aliases: []string{"mr"},
		Short:   "Remove members from a team",
		Args:    cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "team")
			if err != nil {
				return err
			}

			if userIDsStr == "" {
				return fmt.Errorf("--user-ids is required")
			}

			userIDs, err := ParseResourceIDListFlag(userIDsStr, "agent")
			if err != nil {
				return fmt.Errorf("invalid user IDs: %w", err)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.Teams().RemoveMembers(cmdContext(cmd), id, userIDs); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]interface{}{
					"team_id":       id,
					"removed_count": len(userIDs),
					"user_ids":      userIDs,
				})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed %d member(s) from team ID: %d\n", len(userIDs), id)
			return nil
		}),
	}

	cmd.Flags().StringVar(&userIDsStr, "user-ids", "", "User IDs (CSV, whitespace, JSON array; or @- / @path) (required)")
	flagAlias(cmd.Flags(), "user-ids", "uids")

	return cmd
}
