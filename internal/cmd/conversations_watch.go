package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newConversationsWatchCmd() *cobra.Command {
	var (
		status   string
		inboxID  int
		interval int
		limit    int
	)

	cmd := &cobra.Command{
		Use:     "watch",
		Aliases: []string{"w"},
		Short:   "Watch conversations in real-time",
		Long:    "Poll for new and updated conversations at regular intervals",
		Example: strings.TrimSpace(`
  # Watch all open conversations
  cw conversations watch --status open

  # Watch specific inbox every 5 seconds
  cw conversations watch --inbox-id 1 --interval 5

  # Watch with custom limit
  cw conversations watch --status open --limit 20
`),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if interval <= 0 {
				return fmt.Errorf("--interval must be greater than 0")
			}

			status, err := validateStatusWithAll(status)
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			// Set up signal handling for graceful shutdown
			ctx, stop := signal.NotifyContext(cmdContext(cmd), os.Interrupt, syscall.SIGTERM)
			defer stop()

			seen := make(map[int]int64) // ID -> last updated timestamp

			if !isJSON(cmd) {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Watching conversations (interval: %ds, press Ctrl+C to stop)...\n\n", interval)
			}

			ticker := time.NewTicker(time.Duration(interval) * time.Second)
			defer ticker.Stop()

			// Initial fetch
			if err := fetchAndDisplayConversations(ctx, cmd, client, status, inboxID, limit, seen); err != nil {
				return err
			}

			for {
				select {
				case <-ctx.Done():
					if !isJSON(cmd) {
						_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nStopped watching.")
					}
					return nil // Not an error - user requested stop
				case <-ticker.C:
					if err := fetchAndDisplayConversations(ctx, cmd, client, status, inboxID, limit, seen); err != nil {
						// Log error but continue watching
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Error fetching: %v\n", err)
					}
				}
			}
		}),
	}

	cmd.Flags().StringVarP(&status, "status", "s", "open", "Filter by status: open, resolved, pending, snoozed, all")
	cmd.Flags().IntVarP(&inboxID, "inbox-id", "I", 0, "Filter by inbox ID")
	cmd.Flags().IntVar(&interval, "interval", 10, "Polling interval in seconds")
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum conversations to display")
	flagAlias(cmd.Flags(), "status", "st")
	flagAlias(cmd.Flags(), "inbox-id", "iid")
	flagAlias(cmd.Flags(), "interval", "iv")
	flagAlias(cmd.Flags(), "limit", "lt")
	registerStaticCompletions(cmd, "status", []string{"open", "resolved", "pending", "snoozed", "all"})

	return cmd
}

func fetchAndDisplayConversations(ctx context.Context, cmd *cobra.Command, client *api.Client, status string, inboxID, limit int, seen map[int]int64) error {
	params := api.ListConversationsParams{
		Status: status,
		Page:   1,
	}
	if inboxID > 0 {
		params.InboxID = strconv.Itoa(inboxID)
	}

	result, err := client.Conversations().List(ctx, params)
	if err != nil {
		return err
	}

	// Filter to only new or updated conversations
	var updated []api.Conversation
	for _, conv := range result.Data.Payload {
		lastUpdated := conv.LastActivityAtTime().Unix()
		if prev, exists := seen[conv.ID]; !exists || lastUpdated > prev {
			updated = append(updated, conv)
			seen[conv.ID] = lastUpdated
		}
	}

	if len(updated) > 0 {
		timestamp := time.Now().Format("15:04:05")
		if limit > 0 && len(updated) > limit {
			updated = updated[:limit]
		}

		if isJSON(cmd) {
			payload := map[string]any{
				"timestamp": timestamp,
				"items":     updated,
			}
			query := outfmt.GetQuery(cmd.Context())
			payloadAny := any(payload)
			if query != "" {
				filtered, err := outfmt.ApplyQuery(payloadAny, query)
				if err != nil {
					return err
				}
				payloadAny = filtered
			}
			if tmpl := outfmt.GetTemplate(cmd.Context()); tmpl != "" {
				return outfmt.WriteTemplate(cmd.OutOrStdout(), payloadAny, tmpl)
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetEscapeHTML(false)
			return enc.Encode(payloadAny)
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[%s] %d update(s):\n", timestamp, len(updated))
		for _, conv := range updated {
			displayID := conv.ID
			if conv.DisplayID != nil {
				displayID = *conv.DisplayID
			}
			priority := "-"
			if conv.Priority != nil {
				priority = *conv.Priority
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  #%d [%s] priority=%s unread=%d\n",
				displayID, conv.Status, priority, conv.Unread)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
	}

	return nil
}
