package cmd

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/cli"
	"github.com/spf13/cobra"
)

func printConversationsTable(out io.Writer, conversations []api.Conversation) {
	w := newTabWriter(out)
	_, _ = fmt.Fprintln(w, "ID\tINBOX\tSTATUS\tPRIORITY\tUNREAD\tMSGS\tCREATED\tLAST_ACTIVITY")
	for _, conv := range conversations {
		_, _ = fmt.Fprintln(w, strings.Join(conversationRow(conv), "\t"))
	}
	_ = w.Flush()
}

func newConversationsListCmd() *cobra.Command {
	var inboxID string
	var status string
	var assigneeType string
	var teamID int
	var labels string
	var search string
	var unreadOnly bool
	var since string
	var waiting bool
	var light bool

	cfg := ListConfig[api.Conversation]{
		Use:               "list",
		Short:             "List conversations",
		Long:              "List conversations filtered by status and inbox",
		EmptyMessage:      "",
		DisableLimit:      true,
		DefaultMaxPages:   100,
		Headers:           []string{"ID", "INBOX", "STATUS", "PRIORITY", "UNREAD", "MSGS", "CREATED", "LAST_ACTIVITY"},
		RowFunc:           conversationRow,
		AfterOutput:       conversationsListSummary,
		DisablePagination: false,
		Example: strings.TrimSpace(`
  # List open conversations
  cw conversations list --status open

  # Filter by inbox ID
  cw conversations list --inbox-id 1

  # Filter by inbox name
  cw conversations list --inbox-id Support

  # JSON output - returns an object with an "items" array
  cw conversations list --output json | jq '.items[0]'

  # Fetch all pages
  cw conversations list --status open --all
`),
		AgentTransform: func(ctx context.Context, client *api.Client, items []api.Conversation) (any, error) {
			if light {
				return buildLightConversationLookups(items), nil
			}
			summaries := agentfmt.ConversationSummaries(items)
			return resolveConversationSummaries(ctx, client, summaries), nil
		},
		JSONTransform: func(_ context.Context, _ *api.Client, items []api.Conversation) (any, error) {
			if !light {
				return items, nil
			}
			return buildLightConversationLookups(items), nil
		},
		ForceJSON: func(_ *cobra.Command) bool {
			return light
		},
		Fetch: func(ctx context.Context, client *api.Client, page, _ int) (ListResult[api.Conversation], error) {
			// Normalize status prefix (e.g. "o" → "open").
			normalizedStatus, err := validateStatusWithAll(status)
			if err != nil {
				return ListResult[api.Conversation]{}, err
			}

			// Normalize assignee-type prefix (e.g. "u" → "unassigned").
			normalizedAssigneeType, err := validateAssigneeType(assigneeType)
			if err != nil {
				return ListResult[api.Conversation]{}, err
			}

			// Resolve inbox name to ID if provided.
			resolvedInboxID := inboxID
			if inboxID != "" {
				id, err := resolveInboxID(ctx, client, inboxID)
				if err != nil {
					return ListResult[api.Conversation]{}, err
				}
				resolvedInboxID = strconv.Itoa(id)
			}

			params := api.ListConversationsParams{
				Status:       normalizedStatus,
				InboxID:      resolvedInboxID,
				AssigneeType: normalizedAssigneeType,
				Query:        search,
				Page:         page,
			}
			if teamID > 0 {
				params.TeamID = strconv.Itoa(teamID)
			}
			if labels != "" {
				params.Labels = splitCommaList(labels)
			}
			result, err := client.Conversations().List(ctx, params)
			if err != nil {
				return ListResult[api.Conversation]{}, fmt.Errorf("failed to list conversations: %w", err)
			}

			items := result.Data.Payload
			if unreadOnly {
				filtered := make([]api.Conversation, 0, len(items))
				for _, conv := range items {
					if conv.Unread > 0 {
						filtered = append(filtered, conv)
					}
				}
				items = filtered
			}
			if since != "" {
				sinceTime, err := cli.ParseRelativeTime(since, time.Now())
				if err != nil {
					return ListResult[api.Conversation]{}, fmt.Errorf("invalid --since value: %w", err)
				}
				filtered := make([]api.Conversation, 0, len(items))
				for _, conv := range items {
					if conv.LastActivityAtTime().After(sinceTime) || conv.LastActivityAtTime().Equal(sinceTime) {
						filtered = append(filtered, conv)
					}
				}
				items = filtered
			}
			if waiting {
				// Sort by customer wait time (longest waiting first).
				// Wait time is approximated by oldest LastActivityAt, since conversations
				// with older last activity have been waiting longer for a response.
				sort.Slice(items, func(i, j int) bool {
					return items[i].LastActivityAt < items[j].LastActivityAt
				})
			}

			totalPages := int(result.Data.Meta.TotalPages)
			hasMore := totalPages > 0 && page < totalPages
			return ListResult[api.Conversation]{Items: items, HasMore: hasMore}, nil
		},
	}

	cmd := NewListCommand(cfg, func(ctx context.Context) (*api.Client, error) {
		return getClient()
	})
	cmd.Aliases = []string{"ls"}

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "status", "inbox_id", "assignee_id"},
		"default": {"id", "status", "priority", "inbox_id", "assignee_id", "team_id", "created_at", "last_activity_at"},
		"debug":   {"id", "status", "priority", "inbox_id", "assignee_id", "team_id", "contact_id", "display_id", "muted", "unread_count", "labels", "meta", "custom_attributes", "created_at", "last_activity_at"},
	})
	registerFieldSchema(cmd, "conversation")

	cmd.Flags().StringVarP(&inboxID, "inbox-id", "I", "", "Filter by inbox ID or name")
	cmd.Flags().StringVarP(&status, "status", "s", "all", "Filter by status (open|resolved|pending|snoozed|all)")
	cmd.Flags().StringVar(&assigneeType, "assignee-type", "", "Filter by assignee type (me|assigned|unassigned)")
	cmd.Flags().IntVar(&teamID, "team-id", 0, "Filter by team ID")
	cmd.Flags().StringVarP(&labels, "labels", "L", "", "Filter by labels (comma-separated)")
	cmd.Flags().StringVar(&search, "search", "", "Filter by search query")
	cmd.Flags().BoolVar(&unreadOnly, "unread-only", false, "Only show conversations with unread messages")
	cmd.Flags().StringVarP(&since, "since", "S", "", "Filter by last activity (e.g., yesterday, 2h ago, 2026-01-30)")
	cmd.Flags().BoolVar(&waiting, "waiting", false, "Sort by customer wait time (longest first)")
	cmd.Flags().BoolVar(&light, "light", false, "Return minimal conversation payload for lookup")
	flagAlias(cmd.Flags(), "status", "st")
	flagAlias(cmd.Flags(), "inbox-id", "iid")
	flagAlias(cmd.Flags(), "assignee-type", "at")
	flagAlias(cmd.Flags(), "team-id", "tid")
	flagAlias(cmd.Flags(), "unread-only", "unread")
	flagAlias(cmd.Flags(), "waiting", "wt")
	flagAlias(cmd.Flags(), "light", "li")
	flagAlias(cmd.Flags(), "search", "sq")
	registerStaticCompletions(cmd, "status", []string{"open", "resolved", "pending", "snoozed", "all"})
	registerStaticCompletions(cmd, "assignee-type", []string{"me", "assigned", "unassigned"})

	return cmd
}

func conversationRow(conv api.Conversation) []string {
	priority := "-"
	if conv.Priority != nil {
		priority = *conv.Priority
	}
	displayID := conv.ID
	if conv.DisplayID != nil {
		displayID = *conv.DisplayID
	}
	return []string{
		fmt.Sprintf("%d", displayID),
		fmt.Sprintf("%d", conv.InboxID),
		conv.Status,
		priority,
		fmt.Sprintf("%d", conv.Unread),
		formatMessageCount(conv.MessagesCount),
		formatTimestampShort(conv.CreatedAtTime()),
		formatTimestampShort(conv.LastActivityAtTime()),
	}
}

func conversationsListSummary(cmd *cobra.Command, summary ListSummary) error {
	if summary.All {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nTotal: %d conversations (%d pages)\n", summary.TotalItems, summary.PagesFetched)
		return nil
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nPage %d (%d conversations)\n", summary.Page, summary.TotalItems)
	return nil
}
