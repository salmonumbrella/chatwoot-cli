package cmd

import (
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

// newAssignCmd creates the top-level assign command for quick conversation assignment
func newAssignCmd() *cobra.Command {
	var (
		agent string
		team  string
		light bool
	)

	cmd := &cobra.Command{
		Use:     "assign <conversation-id>",
		Aliases: []string{"reassign", "as"},
		Short:   "Assign a conversation to an agent or team",
		Long: `Assign a conversation to an agent and/or team.

This is a convenience shortcut for 'cw conversations assign'.
At least one of --agent or --team must be specified.`,
		Example: strings.TrimSpace(`
  # Assign to an agent
  cw assign 123 --agent 5

  # Assign to a team
  cw assign 123 --team 2

  # Assign to both agent and team
  cw assign 123 --agent 5 --team 2

  # JSON output
  cw assign 123 --agent 5 --output json

  # Light token-optimized payload
  cw assign 123 --agent 5 --li
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			// Interactive prompts when no flags provided
			if agent == "" && team == "" {
				if isInteractive() {
					selectedAgent, err := promptAgentID(ctx, client)
					if err != nil {
						return err
					}
					if selectedAgent > 0 {
						agent = fmt.Sprintf("%d", selectedAgent)
					}
					selectedTeam, err := promptTeamID(ctx, client)
					if err != nil {
						return err
					}
					if selectedTeam > 0 {
						team = fmt.Sprintf("%d", selectedTeam)
					}
				}
			}

			agentID, err := resolveAgentID(ctx, client, agent)
			if err != nil {
				return err
			}
			teamID, err := resolveTeamID(ctx, client, team)
			if err != nil {
				return err
			}

			if agentID == 0 && teamID == 0 {
				return fmt.Errorf("at least one of --agent or --team is required")
			}

			// Perform the assignment
			if _, err := client.Conversations().Assign(ctx, id, agentID, teamID); err != nil {
				return fmt.Errorf("failed to assign conversation %d: %w", id, err)
			}

			// Fetch updated conversation for output
			conv, err := client.Conversations().Get(ctx, id)
			if err != nil {
				return fmt.Errorf("failed to get conversation %d after assignment: %w", id, err)
			}

			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, buildLightAssignResult(conv.ID, conv.AssigneeID, conv.TeamID))
			}

			if isAgent(cmd) {
				if !flagOrAliasChanged(cmd, "compact-json") {
					cmd.SetContext(outfmt.WithCompact(cmd.Context(), true))
				}
				return printRawJSON(cmd, buildAgentAssignResult(conv))
			}

			if isJSON(cmd) {
				return printJSON(cmd, conv)
			}

			// Text output
			displayID := conv.ID
			if conv.DisplayID != nil {
				displayID = *conv.DisplayID
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversation #%d assigned\n", displayID)
			if conv.AssigneeID != nil {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Agent: %d\n", *conv.AssigneeID)
			}
			if conv.TeamID != nil {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Team:  %d\n", *conv.TeamID)
			}

			return nil
		}),
	}

	cmd.Flags().StringVar(&agent, "agent", "", "Agent ID, name, or email to assign")
	cmd.Flags().StringVar(&team, "team", "", "Team ID or name to assign")
	flagAlias(cmd.Flags(), "agent", "ag")
	flagAlias(cmd.Flags(), "team", "tm")
	cmd.Flags().BoolVar(&light, "light", false, "Return minimal mutation payload")
	flagAlias(cmd.Flags(), "light", "li")

	return cmd
}
