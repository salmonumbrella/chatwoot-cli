package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

func newConversationsBulkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bulk",
		Aliases: []string{"bk"},
		Short:   "Bulk operations on conversations",
		Long:    "Perform bulk operations on multiple conversations at once",
	}

	cmd.AddCommand(newConversationsBulkResolveCmd())
	cmd.AddCommand(newConversationsBulkAssignCmd())
	cmd.AddCommand(newConversationsBulkAddLabelCmd())
	cmd.AddCommand(newConversationsBatchUpdateCmd())

	return cmd
}

func newConversationsBulkResolveCmd() *cobra.Command {
	var (
		conversationIDs string
		concurrency     int
		progress        bool
		noProgress      bool
	)

	cmd := &cobra.Command{
		Use:     "resolve",
		Aliases: []string{"res"},
		Short:   "Resolve multiple conversations",
		Long:    "Mark multiple conversations as resolved at once",
		Example: strings.TrimSpace(`
  # Resolve multiple conversations
  cw conversations bulk resolve --ids 1,2,3

  # Resolve and output result as JSON
  cw conversations bulk resolve --ids 1,2,3 --output json

  # Resolve with custom concurrency
  cw conversations bulk resolve --ids 1,2,3 --concurrency 10
`),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			ids, err := ParseResourceIDListFlag(conversationIDs, "conversation")
			if err != nil {
				return fmt.Errorf("invalid conversation IDs: %w", err)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			results := runBulkOperation(
				ctx,
				ids,
				int64(concurrency),
				bulkProgressEnabled(cmd, progress, noProgress),
				cmd.ErrOrStderr(),
				func(ctx context.Context, id int) (any, error) {
					result, err := client.Conversations().ToggleStatus(ctx, id, "resolved", 0)
					if err != nil {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Failed to resolve conversation %d: %v\n", id, err)
						return nil, err
					}
					return result, nil
				},
			)

			successCount, failCount := countResults(results)
			output := buildBulkConversationResultRows(results, func(item map[string]any, _ BulkResult) {
				item["status"] = "resolved"
			})

			return writeBulkConversationResult(cmd, "Resolved", successCount, failCount, output)
		}),
	}

	cmd.Flags().StringVar(&conversationIDs, "ids", "", "Conversation IDs (CSV, whitespace, JSON array; or @- / @path)")
	cmd.Flags().IntVar(&concurrency, "concurrency", DefaultConcurrency, "Max concurrent operations")
	cmd.Flags().BoolVar(&progress, "progress", true, "Show progress while running")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "Disable progress output")
	flagAlias(cmd.Flags(), "concurrency", "cc")
	flagAlias(cmd.Flags(), "ids", "id")
	flagAlias(cmd.Flags(), "progress", "prg")
	flagAlias(cmd.Flags(), "no-progress", "npr")
	_ = cmd.MarkFlagRequired("ids")

	return cmd
}

func newConversationsBulkAssignCmd() *cobra.Command {
	var (
		conversationIDs string
		agent           string
		team            string
		agentID         int
		teamID          int
		concurrency     int
		progress        bool
		noProgress      bool
	)

	cmd := &cobra.Command{
		Use:     "assign",
		Aliases: []string{"asgn"},
		Short:   "Assign multiple conversations",
		Long:    "Assign multiple conversations to an agent and/or team at once",
		Example: strings.TrimSpace(`
  # Assign conversations to an agent
  cw conversations bulk assign --ids 1,2,3 --agent 5

  # Assign conversations to a team
  cw conversations bulk assign --ids 1,2,3 --team 2

  # Assign to both agent and team
  cw conversations bulk assign --ids 1,2,3 --agent 5 --team 2

  # Assign with custom concurrency
  cw conversations bulk assign --ids 1,2,3 --agent 5 --concurrency 10
`),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			// Backwards-compat: map deprecated int flags into string flags if set.
			if agent == "" && agentID > 0 {
				agent = fmt.Sprintf("%d", agentID)
			}
			if team == "" && teamID > 0 {
				team = fmt.Sprintf("%d", teamID)
			}

			ids, err := ParseResourceIDListFlag(conversationIDs, "conversation")
			if err != nil {
				return fmt.Errorf("invalid conversation IDs: %w", err)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			resolvedAgentID, err := resolveAgentID(ctx, client, agent)
			if err != nil {
				return err
			}
			resolvedTeamID, err := resolveTeamID(ctx, client, team)
			if err != nil {
				return err
			}

			if resolvedAgentID == 0 && resolvedTeamID == 0 {
				return fmt.Errorf("at least one of --agent or --team is required")
			}

			results := runBulkOperation(
				ctx,
				ids,
				int64(concurrency),
				bulkProgressEnabled(cmd, progress, noProgress),
				cmd.ErrOrStderr(),
				func(ctx context.Context, id int) (any, error) {
					result, err := client.Conversations().Assign(ctx, id, resolvedAgentID, resolvedTeamID)
					if err != nil {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Failed to assign conversation %d: %v\n", id, err)
						return nil, err
					}
					return result, nil
				},
			)

			successCount, failCount := countResults(results)
			output := buildBulkConversationResultRows(results, func(item map[string]any, _ BulkResult) {
				if resolvedAgentID > 0 {
					item["agent_id"] = resolvedAgentID
				}
				if resolvedTeamID > 0 {
					item["team_id"] = resolvedTeamID
				}
			})

			return writeBulkConversationResult(cmd, "Assigned", successCount, failCount, output)
		}),
	}

	cmd.Flags().StringVar(&conversationIDs, "ids", "", "Conversation IDs (CSV, whitespace, JSON array; or @- / @path)")
	cmd.Flags().StringVar(&agent, "agent", "", "Agent ID, name, or email to assign conversations to")
	cmd.Flags().StringVar(&team, "team", "", "Team ID or name to assign conversations to")
	cmd.Flags().IntVar(&agentID, "agent-id", 0, "Agent ID to assign conversations to (deprecated, use --agent)")
	cmd.Flags().IntVar(&teamID, "team-id", 0, "Team ID to assign conversations to (deprecated, use --team)")
	_ = cmd.Flags().MarkHidden("agent-id")
	_ = cmd.Flags().MarkHidden("team-id")
	cmd.Flags().IntVar(&concurrency, "concurrency", DefaultConcurrency, "Max concurrent operations")
	cmd.Flags().BoolVar(&progress, "progress", true, "Show progress while running")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "Disable progress output")
	flagAlias(cmd.Flags(), "agent", "ag")
	flagAlias(cmd.Flags(), "team", "tm")
	flagAlias(cmd.Flags(), "concurrency", "cc")
	flagAlias(cmd.Flags(), "ids", "id")
	flagAlias(cmd.Flags(), "progress", "prg")
	flagAlias(cmd.Flags(), "no-progress", "npr")
	_ = cmd.MarkFlagRequired("ids")

	return cmd
}

func newConversationsBulkAddLabelCmd() *cobra.Command {
	var (
		conversationIDs string
		labels          string
		concurrency     int
		progress        bool
		noProgress      bool
	)

	cmd := &cobra.Command{
		Use:     "add-label",
		Aliases: []string{"add"},
		Short:   "Add labels to multiple conversations",
		Long:    "Add one or more labels to multiple conversations at once",
		Example: strings.TrimSpace(`
  # Add a single label to multiple conversations
  cw conversations bulk add-label --ids 1,2,3 --labels urgent

  # Add multiple labels to multiple conversations
  cw conversations bulk add-label --ids 1,2,3 --labels urgent,bug

  # Add labels with custom concurrency
  cw conversations bulk add-label --ids 1,2,3 --labels urgent --concurrency 10
`),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			ids, err := ParseResourceIDListFlag(conversationIDs, "conversation")
			if err != nil {
				return fmt.Errorf("invalid conversation IDs: %w", err)
			}

			labelList, err := ParseStringListFlag(labels)
			if err != nil {
				return fmt.Errorf("invalid labels: %w", err)
			}
			if len(labelList) == 0 {
				return fmt.Errorf("no valid labels provided")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			results := runBulkOperation(
				ctx,
				ids,
				int64(concurrency),
				bulkProgressEnabled(cmd, progress, noProgress),
				cmd.ErrOrStderr(),
				func(ctx context.Context, id int) (any, error) {
					resultLabels, err := client.Conversations().AddLabels(ctx, id, labelList)
					if err != nil {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Failed to add labels to conversation %d: %v\n", id, err)
						return nil, err
					}
					return resultLabels, nil
				},
			)

			successCount, failCount := countResults(results)
			output := buildBulkConversationResultRows(results, func(item map[string]any, r BulkResult) {
				if r.Data != nil {
					item["labels"] = r.Data
				}
			})

			return writeBulkConversationResult(cmd, "Added labels to", successCount, failCount, output)
		}),
	}

	cmd.Flags().StringVar(&conversationIDs, "ids", "", "Conversation IDs (CSV, whitespace, JSON array; or @- / @path)")
	cmd.Flags().StringVar(&labels, "labels", "", "Labels to add (CSV, whitespace, JSON array; or @- / @path)")
	cmd.Flags().IntVar(&concurrency, "concurrency", DefaultConcurrency, "Max concurrent operations")
	cmd.Flags().BoolVar(&progress, "progress", true, "Show progress while running")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "Disable progress output")
	flagAlias(cmd.Flags(), "concurrency", "cc")
	flagAlias(cmd.Flags(), "ids", "id")
	flagAlias(cmd.Flags(), "labels", "lb")
	flagAlias(cmd.Flags(), "progress", "prg")
	flagAlias(cmd.Flags(), "no-progress", "npr")
	_ = cmd.MarkFlagRequired("ids")
	_ = cmd.MarkFlagRequired("labels")

	return cmd
}

func buildBulkConversationResultRows(results []BulkResult, decorateSuccess func(item map[string]any, r BulkResult)) []map[string]any {
	output := make([]map[string]any, 0, len(results))
	for _, r := range results {
		item := map[string]any{"id": r.ID, "success": r.Success}
		if r.Error != nil {
			item["error"] = r.Error.Error()
		}
		if r.Success && decorateSuccess != nil {
			decorateSuccess(item, r)
		}
		output = append(output, item)
	}
	return output
}

func writeBulkConversationResult(cmd *cobra.Command, action string, successCount, failCount int, results []map[string]any) error {
	if isJSON(cmd) {
		return printJSON(cmd, map[string]any{
			"success_count": successCount,
			"fail_count":    failCount,
			"results":       results,
		})
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %d conversations (%d failed)\n", action, successCount, failCount)
	return nil
}

// BatchUpdateItem represents a single conversation update in a batch operation.
type BatchUpdateItem struct {
	ID         int      `json:"id"`
	Status     string   `json:"status,omitempty"`
	Priority   string   `json:"priority,omitempty"`
	Labels     []string `json:"labels,omitempty"`
	AssigneeID int      `json:"assignee_id,omitempty"`
	TeamID     int      `json:"team_id,omitempty"`
}

// BatchUpdateResult represents the result of a single batch update operation.
type BatchUpdateResult struct {
	ID     int    `json:"id"`
	Action string `json:"action"`
	Status string `json:"status"` // "ok" | "error"
	Error  string `json:"error,omitempty"`
}

// BatchUpdateResponse is the response for the batch-update command.
type BatchUpdateResponse struct {
	Total     int                 `json:"total"`
	Succeeded int                 `json:"succeeded"`
	Failed    int                 `json:"failed"`
	Results   []BatchUpdateResult `json:"results"`
}

// newConversationsBatchUpdateCmd creates the batch-update subcommand.
func newConversationsBatchUpdateCmd() *cobra.Command {
	var concurrency int

	cmd := &cobra.Command{
		Use:     "batch-update",
		Aliases: []string{"bu"},
		Short:   "Update multiple conversations with different operations",
		Long: `Update multiple conversations in parallel with varying operations per conversation.

Reads JSON input from stdin with an array of updates. Each item can specify different
operations (status, priority, labels, assignment).`,
		Example: strings.TrimSpace(`
  # Update multiple conversations with different operations
  echo '[
    {"id": 123, "status": "resolved"},
    {"id": 456, "labels": ["handled"], "assignee_id": 5},
    {"id": 789, "priority": "low"}
  ]' | cw conversations bulk batch-update

  # From a file
  cat updates.json | cw conversations bulk batch-update

  # With custom concurrency
  cat updates.json | cw conversations bulk batch-update --concurrency 10
`),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			// Read input from stdin
			var items []BatchUpdateItem
			decoder := json.NewDecoder(os.Stdin)
			if err := decoder.Decode(&items); err != nil {
				return fmt.Errorf("failed to parse JSON input: %w", err)
			}

			if len(items) == 0 {
				return fmt.Errorf("no updates to process")
			}
			if concurrency <= 0 {
				return fmt.Errorf("--concurrency must be greater than 0")
			}

			// Validate items
			for i, item := range items {
				if item.ID <= 0 {
					return fmt.Errorf("item %d: id must be positive", i)
				}
				// At least one operation must be specified.
				if item.Status == "" && item.Priority == "" && len(item.Labels) == 0 && item.AssigneeID == 0 && item.TeamID == 0 {
					return fmt.Errorf("item %d: at least one operation (status, priority, labels, assignee_id, team_id) is required", i)
				}
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			// Process updates in parallel with bounded concurrency.
			results := make([]BatchUpdateResult, len(items))
			sem := make(chan struct{}, concurrency)
			var wg sync.WaitGroup

			for i, item := range items {
				wg.Add(1)
				go func(idx int, item BatchUpdateItem) {
					defer wg.Done()
					sem <- struct{}{}        // Acquire semaphore.
					defer func() { <-sem }() // Release semaphore.

					result := BatchUpdateResult{
						ID:     item.ID,
						Status: "ok",
					}

					var actions []string
					var firstErr error

					if item.Status != "" {
						_, err := client.Conversations().ToggleStatus(ctx, item.ID, item.Status, 0)
						if err != nil {
							if firstErr == nil {
								firstErr = err
							}
						} else {
							actions = append(actions, "status_changed")
						}
					}

					if item.Priority != "" {
						err := client.Conversations().TogglePriority(ctx, item.ID, item.Priority)
						if err != nil {
							if firstErr == nil {
								firstErr = err
							}
						} else {
							actions = append(actions, "priority_changed")
						}
					}

					if len(item.Labels) > 0 {
						_, err := client.Conversations().AddLabels(ctx, item.ID, item.Labels)
						if err != nil {
							if firstErr == nil {
								firstErr = err
							}
						} else {
							actions = append(actions, "labels_updated")
						}
					}

					if item.AssigneeID > 0 || item.TeamID > 0 {
						_, err := client.Conversations().Assign(ctx, item.ID, item.AssigneeID, item.TeamID)
						if err != nil {
							if firstErr == nil {
								firstErr = err
							}
						} else {
							actions = append(actions, "assigned")
						}
					}

					if len(actions) > 0 {
						result.Action = strings.Join(actions, ",")
					}
					if firstErr != nil {
						result.Status = "error"
						result.Error = firstErr.Error()
					}

					results[idx] = result
				}(i, item)
			}

			wg.Wait()

			var succeeded, failed int
			for _, r := range results {
				if r.Status == "ok" {
					succeeded++
				} else {
					failed++
				}
			}

			response := BatchUpdateResponse{
				Total:     len(items),
				Succeeded: succeeded,
				Failed:    failed,
				Results:   results,
			}

			if isJSON(cmd) {
				return printJSON(cmd, response)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Batch update complete: %d succeeded, %d failed (total: %d)\n", succeeded, failed, len(items))
			if failed > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nFailed updates:")
				for _, r := range results {
					if r.Status == "error" {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Conversation %d: %s\n", r.ID, r.Error)
					}
				}
			}

			return nil
		}),
	}

	cmd.Flags().IntVar(&concurrency, "concurrency", 5, "Maximum concurrent requests")
	flagAlias(cmd.Flags(), "concurrency", "cc")

	return cmd
}
