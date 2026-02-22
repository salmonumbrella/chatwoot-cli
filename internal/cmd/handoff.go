package cmd

import (
	"fmt"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/dryrun"
	"github.com/spf13/cobra"
)

func newHandoffCmd() *cobra.Command {
	var (
		agent    string
		team     string
		reason   string
		priority string
	)

	cmd := &cobra.Command{
		Use:     "handoff <conversation-id|url>",
		Aliases: []string{"escalate", "transfer", "ho"},
		Short:   "Escalate a conversation to another agent or team",
		Long: strings.TrimSpace(`
Composite escalation command that:
  1. Sends a private note with the handoff reason
  2. Assigns to the specified agent and/or team
  3. Optionally sets priority

This replaces the three-command sequence of note + assign + update.
`),
		Example: strings.TrimSpace(`
  # Handoff to an agent with reason
  cw handoff 123 --agent 5 --reason "Refund request, needs billing approval"

  # Handoff to a team
  cw handoff 123 --team 2 --reason "Technical issue beyond L1 scope"

  # Handoff with priority escalation
  cw handoff 123 --agent 5 --team 2 --priority urgent --reason "VIP customer, SLA at risk"

  # Handoff using agent/team names
  cw handoff 123 --agent "lily" --team "billing" --reason "Escalating to billing"
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			conversationID, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			if strings.TrimSpace(agent) == "" && strings.TrimSpace(team) == "" {
				return fmt.Errorf("at least one of --agent or --team is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			// Resolve agent/team names to IDs.
			agentID, err := resolveAgentID(ctx, client, agent)
			if err != nil {
				return err
			}
			teamID, err := resolveTeamID(ctx, client, team)
			if err != nil {
				return err
			}

			if priority != "" {
				if priority, err = validatePriority(priority); err != nil {
					return err
				}
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation: "handoff",
				Resource:  "conversation",
				Details: map[string]any{
					"conversation_id": conversationID,
					"agent_id":        agentID,
					"team_id":         teamID,
					"priority":        priority,
					"reason":          reason,
				},
			}); ok {
				return err
			}

			var actions []string

			// Step 1: Send private note with reason.
			if strings.TrimSpace(reason) != "" {
				_, err := client.Messages().Create(ctx, conversationID, reason, true, "outgoing")
				if err != nil {
					return fmt.Errorf("failed to send handoff note: %w", err)
				}
				actions = append(actions, "noted")
			}

			// Step 2: Assign to agent/team.
			_, err = client.Conversations().Assign(ctx, conversationID, agentID, teamID)
			if err != nil {
				return fmt.Errorf("note sent but failed to assign conversation: %w", err)
			}
			actions = append(actions, "assigned")

			// Step 3: Set priority if specified.
			if priority != "" {
				if err := client.Conversations().TogglePriority(ctx, conversationID, priority); err != nil {
					return fmt.Errorf("assigned but failed to set priority: %w", err)
				}
				actions = append(actions, "priority set")
			}

			out := map[string]any{
				"action":          "handoff",
				"conversation_id": conversationID,
				"actions":         actions,
			}
			if agentID > 0 {
				out["agent_id"] = agentID
			}
			if teamID > 0 {
				out["team_id"] = teamID
			}
			if priority != "" {
				out["priority"] = priority
			}
			if reason != "" {
				out["reason"] = reason
			}

			if isAgent(cmd) {
				return printJSON(cmd, agentfmt.ItemEnvelope{
					Kind: "handoff",
					Item: out,
				})
			}
			if isJSON(cmd) {
				return printJSON(cmd, out)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Handed off conversation %d (%s)\n", conversationID, strings.Join(actions, ", "))
			return nil
		}),
	}

	cmd.Flags().StringVar(&agent, "agent", "", "Agent ID, name, or email to assign")
	cmd.Flags().StringVar(&team, "team", "", "Team ID or name to assign")
	cmd.Flags().StringVar(&reason, "reason", "", "Handoff reason (sent as private note)")
	cmd.Flags().StringVar(&priority, "priority", "", "Set priority (urgent|high|medium|low|none)")
	flagAlias(cmd.Flags(), "agent", "ag")
	flagAlias(cmd.Flags(), "priority", "pri")
	flagAlias(cmd.Flags(), "reason", "rs")
	flagAlias(cmd.Flags(), "team", "tm")

	return cmd
}
