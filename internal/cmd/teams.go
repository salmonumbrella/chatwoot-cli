package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func newTeamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "teams",
		Short: "Manage teams",
		Long:  "Create, list, update, and manage teams and their members",
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
	cfg := ListConfig[api.Team]{
		Use:               "list",
		Short:             "List all teams",
		DisablePagination: true,
		EmptyMessage:      "No teams found",
		Fetch: func(ctx context.Context, client *api.Client, _ int, _ int) (ListResult[api.Team], error) {
			teams, err := client.ListTeams(ctx)
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

	return NewListCommand(cfg, func(ctx context.Context) (*api.Client, error) {
		return getClient()
	})
}

func newTeamsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a team by ID",
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

			team, err := client.GetTeam(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, team)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION\tAUTO-ASSIGN\tACCOUNT-ID")
			autoAssign := "no"
			if team.AllowAutoAssign {
				autoAssign = "yes"
			}
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\n", team.ID, team.Name, team.Description, autoAssign, team.AccountID)

			return nil
		},
	}
}

func newTeamsCreateCmd() *cobra.Command {
	var name, description string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new team",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			team, err := client.CreateTeam(cmdContext(cmd), name, description)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, team)
			}

			fmt.Printf("Created team: %s (ID: %d)\n", team.Name, team.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Team name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Team description")

	return cmd
}

func newTeamsUpdateCmd() *cobra.Command {
	var name, description string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
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

			team, err := client.UpdateTeam(cmdContext(cmd), id, name, description)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, team)
			}

			fmt.Printf("Updated team: %s (ID: %d)\n", team.Name, team.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Team name")
	cmd.Flags().StringVar(&description, "description", "", "Team description")

	return cmd
}

func newTeamsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a team",
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

			if err := client.DeleteTeam(cmdContext(cmd), id); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]interface{}{
					"id":      id,
					"deleted": true,
				})
			}

			fmt.Printf("Deleted team ID: %d\n", id)
			return nil
		},
	}
}

func newTeamsMembersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "members <id>",
		Short: "List team members",
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

			members, err := client.ListTeamMembers(cmdContext(cmd), id)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, members)
			}

			w := newTabWriter()
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tROLE\tSTATUS")
			for _, m := range members {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", m.ID, m.Name, m.Email, m.Role, m.AvailabilityStatus)
			}

			return nil
		},
	}
}

func newTeamsMembersAddCmd() *cobra.Command {
	var userIDsStr string

	cmd := &cobra.Command{
		Use:   "members-add <id>",
		Short: "Add members to a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			if userIDsStr == "" {
				return fmt.Errorf("--user-ids is required")
			}

			userIDs, err := parseIntSlice(userIDsStr)
			if err != nil {
				return fmt.Errorf("invalid user IDs: %w", err)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.AddTeamMembers(cmdContext(cmd), id, userIDs); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]interface{}{
					"team_id":     id,
					"added_count": len(userIDs),
					"user_ids":    userIDs,
				})
			}

			fmt.Printf("Added %d member(s) to team ID: %d\n", len(userIDs), id)
			return nil
		},
	}

	cmd.Flags().StringVar(&userIDsStr, "user-ids", "", "Comma-separated user IDs (required)")

	return cmd
}

func newTeamsMembersRemoveCmd() *cobra.Command {
	var userIDsStr string

	cmd := &cobra.Command{
		Use:   "members-remove <id>",
		Short: "Remove members from a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			if userIDsStr == "" {
				return fmt.Errorf("--user-ids is required")
			}

			userIDs, err := parseIntSlice(userIDsStr)
			if err != nil {
				return fmt.Errorf("invalid user IDs: %w", err)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.RemoveTeamMembers(cmdContext(cmd), id, userIDs); err != nil {
				return err
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]interface{}{
					"team_id":       id,
					"removed_count": len(userIDs),
					"user_ids":      userIDs,
				})
			}

			fmt.Printf("Removed %d member(s) from team ID: %d\n", len(userIDs), id)
			return nil
		},
	}

	cmd.Flags().StringVar(&userIDsStr, "user-ids", "", "Comma-separated user IDs (required)")

	return cmd
}

// parseIntSlice parses a comma-separated string into a slice of ints
func parseIntSlice(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	result := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := validation.ParsePositiveInt(part, "agent ID")
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid IDs provided")
	}

	return result, nil
}
