package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/cli"
	"github.com/chatwoot/chatwoot-cli/internal/heuristics"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newConversationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "conversations",
		Aliases: []string{"conv", "c"},
		Short:   "Manage conversations",
		Long:    "List, view, create, and manage Chatwoot conversations",
	}

	cmd.AddCommand(newConversationsListCmd())
	cmd.AddCommand(newConversationsGetCmd())
	cmd.AddCommand(newConversationsCreateCmd())
	cmd.AddCommand(newConversationsFilterCmd())
	cmd.AddCommand(newConversationsMetaCmd())
	cmd.AddCommand(newConversationsCountsCmd())
	cmd.AddCommand(newConversationsToggleStatusCmd())
	cmd.AddCommand(newConversationsResolveCmd())
	cmd.AddCommand(newConversationsTogglePriorityCmd())
	cmd.AddCommand(newConversationsUpdateCmd())
	cmd.AddCommand(newConversationsAssignCmd())
	cmd.AddCommand(newConversationsLabelsCmd())
	cmd.AddCommand(newConversationsLabelsAddCmd())
	cmd.AddCommand(newConversationsLabelsRemoveCmd())
	cmd.AddCommand(newConversationsCustomAttributesCmd())
	cmd.AddCommand(newConversationsContextCmd())
	cmd.AddCommand(newConversationsMarkUnreadCmd())
	cmd.AddCommand(newConversationsMuteCmd())
	cmd.AddCommand(newConversationsUnmuteCmd())
	cmd.AddCommand(newConversationsTranscriptCmd())
	cmd.AddCommand(newConversationsTypingCmd())
	cmd.AddCommand(newConversationsSearchCmd())
	cmd.AddCommand(newConversationsAttachmentsCmd())
	cmd.AddCommand(newConversationsWatchCmd())
	cmd.AddCommand(newConversationsBulkCmd())
	cmd.AddCommand(newConversationsTriageCmd())

	return cmd
}

func printConversationsTable(out io.Writer, conversations []api.Conversation) {
	w := newTabWriter(out)
	_, _ = fmt.Fprintln(w, "ID\tINBOX\tSTATUS\tPRIORITY\tUNREAD\tMSGS\tCREATED\tLAST_ACTIVITY")
	for _, conv := range conversations {
		priority := "-"
		if conv.Priority != nil {
			priority = *conv.Priority
		}
		displayID := conv.ID
		if conv.DisplayID != nil {
			displayID = *conv.DisplayID
		}
		msgs := formatMessageCount(conv.MessagesCount)
		_, _ = fmt.Fprintf(w, "%d\t%d\t%s\t%s\t%d\t%s\t%s\t%s\n",
			displayID,
			conv.InboxID,
			conv.Status,
			priority,
			conv.Unread,
			msgs,
			formatTimestampShort(conv.CreatedAtTime()),
			formatTimestampShort(conv.LastActivityAtTime()),
		)
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
  chatwoot conversations list --status open

  # Filter by inbox ID
  chatwoot conversations list --inbox-id 1

  # Filter by inbox name
  chatwoot conversations list --inbox-id Support

  # JSON output - returns an object with an "items" array
  chatwoot conversations list --output json | jq '.items[0]'

  # Fetch all pages
  chatwoot conversations list --status open --all
`),
		AgentTransform: func(ctx context.Context, client *api.Client, items []api.Conversation) (any, error) {
			summaries := agentfmt.ConversationSummaries(items)
			return resolveConversationSummaries(ctx, client, summaries), nil
		},
		Fetch: func(ctx context.Context, client *api.Client, page, _ int) (ListResult[api.Conversation], error) {
			// Resolve inbox name to ID if provided
			resolvedInboxID := inboxID
			if inboxID != "" {
				id, err := resolveInboxID(ctx, client, inboxID)
				if err != nil {
					return ListResult[api.Conversation]{}, err
				}
				resolvedInboxID = strconv.Itoa(id)
			}

			params := api.ListConversationsParams{
				Status:       status,
				InboxID:      resolvedInboxID,
				AssigneeType: assigneeType,
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

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "status", "inbox_id", "assignee_id"},
		"default": {"id", "status", "priority", "inbox_id", "assignee_id", "team_id", "created_at", "last_activity_at"},
		"debug":   {"id", "status", "priority", "inbox_id", "assignee_id", "team_id", "contact_id", "display_id", "muted", "unread_count", "labels", "meta", "custom_attributes", "created_at", "last_activity_at"},
	})
	registerFieldSchema(cmd, "conversation")

	cmd.Flags().StringVar(&inboxID, "inbox-id", "", "Filter by inbox ID or name")
	cmd.Flags().StringVar(&status, "status", "all", "Filter by status (open|resolved|pending|snoozed|all)")
	cmd.Flags().StringVar(&assigneeType, "assignee-type", "", "Filter by assignee type (me|assigned|unassigned)")
	cmd.Flags().IntVar(&teamID, "team-id", 0, "Filter by team ID")
	cmd.Flags().StringVar(&labels, "labels", "", "Filter by labels (comma-separated)")
	cmd.Flags().StringVar(&search, "search", "", "Filter by search query")
	cmd.Flags().BoolVar(&unreadOnly, "unread-only", false, "Only show conversations with unread messages")
	cmd.Flags().StringVar(&since, "since", "", "Filter by last activity (e.g., yesterday, 2h ago, 2026-01-30)")
	cmd.Flags().BoolVar(&waiting, "waiting", false, "Sort by customer wait time (longest first)")
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

func newConversationsGetCmd() *cobra.Command {
	var withContext bool
	var withMessages bool
	var messageLimit int
	var suggestedActions bool
	var explain bool

	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get conversation details",
		Long:  "Retrieve detailed information about a specific conversation",
		Example: strings.TrimSpace(`
  # Get conversation details
  chatwoot conversations get 123

  # Get conversation as JSON
  chatwoot conversations get 123 --output json

  # Get conversation with recent messages (agent mode)
  chatwoot conversations get 123 --with-messages --output agent

  # Get comprehensive context (agent mode only)
  chatwoot conversations get 123 --context --output agent

  # Get context with limited messages
  chatwoot conversations get 123 --context --message-limit 10 --output agent

  # Get conversation with AI-suggested actions (agent mode)
  chatwoot conversations get 123 --suggested-actions --output agent

  # Get conversation with reasoning hints (agent mode)
  chatwoot conversations get 123 --explain --output agent

  # Get conversation using URL from browser
  chatwoot conversations get https://app.chatwoot.com/app/accounts/1/conversations/123

  # Get the web UI URL for a conversation
  chatwoot conversations get 461 --url
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			if handled, err := handleURLFlag(cmd, "conversations", id); handled {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)
			conv, err := client.Conversations().Get(ctx, id)
			if err != nil {
				return fmt.Errorf("failed to get conversation %d: %w", id, err)
			}

			// Handle --context flag in agent mode
			if withContext && isAgent(cmd) {
				detail := agentfmt.ConversationDetailFromConversation(*conv)
				detail = resolveConversationDetail(ctx, client, detail)

				// Fetch messages
				messages, _ := client.Messages().List(ctx, id)
				if messageLimit > 0 && len(messages) > messageLimit {
					messages = messages[len(messages)-messageLimit:]
				}

				// Fetch contact with relationship
				var contactWithRel *agentfmt.ContactDetailWithRelationship
				if conv.ContactID > 0 {
					contact, err := client.Contacts().Get(ctx, conv.ContactID)
					if err == nil && contact != nil {
						contactDetail := agentfmt.ContactDetailFromContact(*contact)
						convs, _ := client.Contacts().Conversations(ctx, conv.ContactID)
						relationship := agentfmt.ComputeRelationshipSummary(convs)
						contactWithRel = &agentfmt.ContactDetailWithRelationship{
							ContactDetail: contactDetail,
							Relationship:  relationship,
						}
					}
				}

				return printJSON(cmd, agentfmt.ItemEnvelope{
					Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Item: agentfmt.ConversationContext{
						Conversation: detail,
						Messages:     agentfmt.MessageSummaries(messages),
						Contact:      contactWithRel,
					},
				})
			}

			// Handle --with-messages flag in agent mode
			if withMessages && isAgent(cmd) {
				detail := agentfmt.ConversationDetailFromConversation(*conv)
				detail = resolveConversationDetail(ctx, client, detail)

				// Fetch messages
				messages, err := client.Messages().List(ctx, id)
				if err != nil {
					return fmt.Errorf("failed to fetch messages for conversation %d: %w", id, err)
				}

				// Apply message limit (default 20)
				limit := messageLimit
				if limit == 0 {
					limit = 20
				}
				if len(messages) > limit {
					messages = messages[len(messages)-limit:] // Keep most recent
				}

				detailWithMessages := agentfmt.ConversationDetailWithMessages{
					ConversationDetail: detail,
					Messages:           agentfmt.MessageSummaries(messages),
				}
				return printJSON(cmd, agentfmt.ItemEnvelope{
					Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Item: detailWithMessages,
				})
			}

			// Handle --suggested-actions flag in agent mode
			if suggestedActions && isAgent(cmd) {
				detail := agentfmt.ConversationDetailFromConversation(*conv)
				detail = resolveConversationDetail(ctx, client, detail)

				// Fetch messages for heuristics
				messages, _ := client.Messages().List(ctx, id)

				// Fetch contact history if contact ID is available
				var contactHistory []api.Conversation
				if conv.ContactID > 0 {
					contactHistory, _ = client.Contacts().Conversations(ctx, conv.ContactID)
				}

				// Get suggested actions from heuristics
				actions := heuristics.SuggestActions(conv, messages, contactHistory)

				// Build response with suggested actions
				type ConversationDetailWithSuggestedActions struct {
					agentfmt.ConversationDetail
					SuggestedActions []heuristics.SuggestedAction `json:"suggested_actions,omitempty"`
				}

				return printJSON(cmd, agentfmt.ItemEnvelope{
					Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Item: ConversationDetailWithSuggestedActions{
						ConversationDetail: detail,
						SuggestedActions:   actions,
					},
				})
			}

			// Handle --explain flag in agent mode
			if explain && isAgent(cmd) {
				detail := agentfmt.ConversationDetailFromConversation(*conv)
				detail = resolveConversationDetail(ctx, client, detail)

				// Fetch messages for heuristics
				messages, _ := client.Messages().List(ctx, id)

				// Fetch contact history if contact ID is available
				var contactHistory []api.Conversation
				if conv.ContactID > 0 {
					contactHistory, _ = client.Contacts().Conversations(ctx, conv.ContactID)
				}

				// Get analysis from heuristics
				analysis := heuristics.AnalyzeConversation(conv, messages, contactHistory)

				// Build response with explanation (underscore prefix indicates metadata)
				type ConversationDetailWithExplanation struct {
					agentfmt.ConversationDetail
					Explanation *heuristics.Analysis `json:"_explanation,omitempty"`
				}

				return printJSON(cmd, agentfmt.ItemEnvelope{
					Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Item: ConversationDetailWithExplanation{
						ConversationDetail: detail,
						Explanation:        analysis,
					},
				})
			}

			if isAgent(cmd) {
				detail := agentfmt.ConversationDetailFromConversation(*conv)
				detail = resolveConversationDetail(ctx, client, detail)
				payload := agentfmt.ItemEnvelope{
					Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Item: detail,
				}
				return printJSON(cmd, payload)
			}
			if isJSON(cmd) {
				return printJSON(cmd, conv)
			}
			return printConversationDetails(cmd.OutOrStdout(), conv)
		}),
	}

	cmd.Flags().BoolVar(&withMessages, "with-messages", false, "Include recent messages in agent output")
	cmd.Flags().BoolVar(&withContext, "context", false, "Include comprehensive context (messages, contact with relationship) - agent mode only")
	cmd.Flags().IntVar(&messageLimit, "message-limit", 20, "Maximum messages to include (default 20)")
	cmd.Flags().BoolVar(&suggestedActions, "suggested-actions", false, "Include AI-suggested actions in agent output")
	cmd.Flags().BoolVar(&explain, "explain", false, "Include reasoning hints in agent output")
	cmd.Flags().Bool("url", false, "Print the Chatwoot web UI URL for this resource and exit")

	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "status", "inbox_id", "assignee_id"},
		"default": {"id", "status", "priority", "inbox_id", "assignee_id", "team_id", "created_at", "last_activity_at"},
		"debug":   {"id", "status", "priority", "inbox_id", "assignee_id", "team_id", "contact_id", "display_id", "muted", "unread_count", "labels", "meta", "custom_attributes", "created_at", "last_activity_at"},
	})
	registerFieldSchema(cmd, "conversation")

	return cmd
}

func newConversationsCreateCmd() *cobra.Command {
	var inboxID int
	var contactID int
	var message string
	var status string
	var assigneeID int
	var teamID int

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new conversation",
		Long:  "Create a new conversation in an inbox",
		Example: strings.TrimSpace(`
  # Create a conversation
  chatwoot conversations create --inbox-id 1 --contact-id 123

  # Create a conversation with an initial message
  chatwoot conversations create --inbox-id 1 --contact-id 123 --message "Hello!"

  # Create a conversation with status and assignment
  chatwoot conversations create --inbox-id 1 --contact-id 123 --status open --assignee-id 5 --team-id 2
`),
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			if inboxID == 0 {
				if isInteractive() {
					client, err := getClient()
					if err != nil {
						return err
					}
					selected, err := promptInboxID(cmdContext(cmd), client)
					if err != nil {
						return err
					}
					inboxID = selected
				} else {
					return fmt.Errorf("--inbox-id is required")
				}
			}
			if contactID == 0 {
				return fmt.Errorf("--contact-id is required")
			}

			if status != "" {
				if err := validateStatus(status); err != nil {
					return err
				}
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			req := api.CreateConversationRequest{
				InboxID:   inboxID,
				ContactID: contactID,
				Message:   message,
				Status:    status,
			}

			// Use pointer pattern - only set if flag was provided
			if assigneeID > 0 {
				req.Assignee = &assigneeID
			}
			if teamID > 0 {
				req.TeamID = &teamID
			}

			conv, err := client.Conversations().Create(cmdContext(cmd), req)
			if err != nil {
				return fmt.Errorf("failed to create conversation: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, conv)
			}

			displayID := conv.ID
			if conv.DisplayID != nil {
				displayID = *conv.DisplayID
			}
			printAction(cmd, "Created", "conversation", displayID, "")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  ID:     %d\n", conv.ID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Status: %s\n", conv.Status)
			if conv.AssigneeID != nil {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Assignee: %d\n", *conv.AssigneeID)
			}
			if conv.TeamID != nil {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Team: %d\n", *conv.TeamID)
			}

			return nil
		}),
	}

	cmd.Flags().IntVar(&inboxID, "inbox-id", 0, "Inbox ID (required)")
	cmd.Flags().IntVar(&contactID, "contact-id", 0, "Contact ID (required)")
	cmd.Flags().StringVar(&message, "message", "", "Initial message content")
	cmd.Flags().StringVar(&status, "status", "", "Status (open|resolved|pending|snoozed)")
	cmd.Flags().IntVar(&assigneeID, "assignee-id", 0, "Agent ID to assign")
	cmd.Flags().IntVar(&teamID, "team-id", 0, "Team ID to assign")

	return cmd
}

func newConversationsFilterCmd() *cobra.Command {
	var payloadStr string

	cmd := &cobra.Command{
		Use:   "filter",
		Short: "Filter conversations with custom query",
		Long: `Filter conversations using the Chatwoot filter API.

The payload follows the Chatwoot filter API format with an array of filter conditions.
See: https://developers.chatwoot.com/api-reference/conversations/conversations-filter`,
		Example: strings.TrimSpace(`
  # Filter by multiple statuses (open OR pending OR snoozed)
  chatwoot conversations filter --payload '{"payload":[{"attribute_key":"status","filter_operator":"equal_to","values":["open","pending","snoozed"]}]}'

  # Filter by inbox
  chatwoot conversations filter --payload '{"payload":[{"attribute_key":"inbox_id","filter_operator":"equal_to","values":[1]}]}'

  # Combine filters with AND
  chatwoot conversations filter --payload '{"payload":[{"attribute_key":"status","filter_operator":"equal_to","values":["open"],"query_operator":"AND"},{"attribute_key":"inbox_id","filter_operator":"equal_to","values":[1]}]}'

  # Filter operators: equal_to, not_equal_to, contains, does_not_contain
`),
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			if payloadStr == "" {
				return fmt.Errorf("--payload is required")
			}

			var payload map[string]any
			if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
				return fmt.Errorf("invalid JSON payload: %w", err)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			result, err := client.Conversations().Filter(cmdContext(cmd), payload)
			if err != nil {
				return fmt.Errorf("failed to filter conversations: %w", err)
			}

			if isAgent(cmd) {
				summaries := agentfmt.ConversationSummaries(result.Data.Payload)
				summaries = resolveConversationSummaries(cmdContext(cmd), client, summaries)
				payload := agentfmt.ListEnvelope{
					Kind:  agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Items: summaries,
					Meta: map[string]any{
						"total_items": len(summaries),
					},
				}
				return printJSON(cmd, payload)
			}
			if isJSON(cmd) {
				return printJSON(cmd, result.Data.Payload)
			}

			printConversationsTable(cmd.OutOrStdout(), result.Data.Payload)

			return nil
		}),
	}

	cmd.Flags().StringVar(&payloadStr, "payload", "", "JSON payload for filtering (required)")

	return cmd
}

func newConversationsMetaCmd() *cobra.Command {
	var status string
	var inboxID string
	var teamID int
	var labels string
	var search string

	cmd := &cobra.Command{
		Use:   "meta",
		Short: "Get conversations metadata",
		Long:  "Retrieve metadata about conversations (counts by status, etc.)",
		Example: strings.TrimSpace(`
  # Get conversations metadata
  chatwoot conversations meta
`),
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			params := api.ListConversationsParams{
				Status:  status,
				InboxID: inboxID,
				Query:   search,
			}
			if teamID > 0 {
				params.TeamID = strconv.Itoa(teamID)
			}
			if labels != "" {
				params.Labels = splitCommaList(labels)
			}

			meta, err := client.Conversations().Meta(cmdContext(cmd), params)
			if err != nil {
				return fmt.Errorf("failed to get conversations metadata: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, meta)
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Conversations Metadata:")
			for key, value := range meta {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s: %v\n", key, value)
			}

			return nil
		}),
	}

	cmd.Flags().StringVar(&status, "status", "all", "Filter by status (open|resolved|pending|snoozed|all)")
	cmd.Flags().StringVar(&inboxID, "inbox-id", "", "Filter by inbox ID")
	cmd.Flags().IntVar(&teamID, "team-id", 0, "Filter by team ID")
	cmd.Flags().StringVar(&labels, "labels", "", "Filter by labels (comma-separated)")
	cmd.Flags().StringVar(&search, "search", "", "Filter by search query")

	return cmd
}

func newConversationsCountsCmd() *cobra.Command {
	var status string
	var inboxID string
	var teamID int
	var labels string
	var search string

	cmd := &cobra.Command{
		Use:   "counts",
		Short: "Get conversation counts by status",
		Long:  "Get counts of conversations grouped by status (open, pending, resolved, etc.)",
		Example: strings.TrimSpace(`
  # Get all counts
  chatwoot conversations counts

  # Get counts as JSON
  chatwoot conversations counts --output json
`),
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			params := api.ListConversationsParams{
				Status:  status,
				InboxID: inboxID,
				Query:   search,
			}
			if teamID > 0 {
				params.TeamID = strconv.Itoa(teamID)
			}
			if labels != "" {
				params.Labels = splitCommaList(labels)
			}

			meta, err := client.Conversations().Meta(cmdContext(cmd), params)
			if err != nil {
				return fmt.Errorf("failed to get conversation counts: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, meta)
			}

			// Extract counts from nested meta object
			counts, ok := meta["meta"].(map[string]any)
			if !ok {
				return fmt.Errorf("unexpected response format")
			}

			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "STATUS\tCOUNT")
			if mineCount, ok := counts["mine_count"]; ok {
				_, _ = fmt.Fprintf(w, "mine\t%v\n", mineCount)
			}
			if unassigned, ok := counts["unassigned_count"]; ok {
				_, _ = fmt.Fprintf(w, "unassigned\t%v\n", unassigned)
			}
			if assigned, ok := counts["assigned_count"]; ok {
				_, _ = fmt.Fprintf(w, "assigned\t%v\n", assigned)
			}
			if allCount, ok := counts["all_count"]; ok {
				_, _ = fmt.Fprintf(w, "all (open)\t%v\n", allCount)
			}

			return nil
		}),
	}

	cmd.Flags().StringVar(&status, "status", "all", "Filter by status (open|resolved|pending|snoozed|all)")
	cmd.Flags().StringVar(&inboxID, "inbox-id", "", "Filter by inbox ID")
	cmd.Flags().IntVar(&teamID, "team-id", 0, "Filter by team ID")
	cmd.Flags().StringVar(&labels, "labels", "", "Filter by labels (comma-separated)")
	cmd.Flags().StringVar(&search, "search", "", "Filter by search query")

	return cmd
}

func newConversationsToggleStatusCmd() *cobra.Command {
	var status string
	var snoozedUntilStr string

	cmd := &cobra.Command{
		Use:   "toggle-status <id>",
		Short: "Toggle conversation status",
		Long:  "Change the status of a conversation",
		Example: strings.TrimSpace(`
  # Mark conversation as resolved
  chatwoot conversations toggle-status 123 --status resolved

  # Reopen a conversation
  chatwoot conversations toggle-status 123 --status open

  # Snooze until next customer reply (default behavior)
  chatwoot conversations toggle-status 123 --status snoozed

  # Snooze until specific time (RFC3339)
  chatwoot conversations toggle-status 123 --status snoozed --snoozed-until "2025-01-15T10:00:00Z"

  # Snooze until specific time (Unix timestamp)
  chatwoot conversations toggle-status 123 --status snoozed --snoozed-until 1735689600
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			if status == "" {
				return fmt.Errorf("--status is required")
			}

			if err := validateStatus(status); err != nil {
				return err
			}

			// Parse and validate snoozed-until flag
			var snoozedUntil int64
			if snoozedUntilStr != "" {
				if status != "snoozed" {
					return fmt.Errorf("--snoozed-until can only be used with --status snoozed")
				}

				snoozedUntil, err = parseSnoozedUntil(snoozedUntilStr)
				if err != nil {
					return fmt.Errorf("--snoozed-until: %w", err)
				}
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			result, err := client.Conversations().ToggleStatus(cmdContext(cmd), id, status, snoozedUntil)
			if err != nil {
				return fmt.Errorf("failed to toggle status for conversation %d: %w", id, err)
			}

			if isJSON(cmd) {
				// Return payload directly for consistency
				return printJSON(cmd, result.Payload)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversation #%d status updated to: %s\n", result.Payload.ConversationID, result.Payload.CurrentStatus)
			if result.Payload.SnoozedUntil != nil && *result.Payload.SnoozedUntil > 0 {
				snoozedTime := time.Unix(*result.Payload.SnoozedUntil, 0)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Snoozed until: %s\n", formatTimestampWithZone(snoozedTime))
			}

			return nil
		}),
	}

	cmd.Flags().StringVar(&status, "status", "", "New status (open|resolved|pending|snoozed) (required)")
	cmd.Flags().StringVar(&snoozedUntilStr, "snoozed-until", "", "Snooze until time (Unix timestamp, RFC3339, or relative)")
	registerStaticCompletions(cmd, "status", []string{"open", "resolved", "pending", "snoozed"})

	return cmd
}

func newConversationsResolveCmd() *cobra.Command {
	var (
		concurrency int
		progress    bool
		noProgress  bool
	)

	cmd := &cobra.Command{
		Use:   "resolve <id> [id...]",
		Short: "Resolve conversations",
		Long:  "Mark one or more conversations as resolved",
		Example: strings.TrimSpace(`
  # Resolve a single conversation
  chatwoot conversations resolve 123

  # Resolve multiple conversations (space-separated)
  chatwoot conversations resolve 123 456 789

  # Resolve multiple conversations (comma-separated)
  chatwoot conversations resolve 123,456,789

  # Resolve with JSON output
  chatwoot conversations resolve 123 456 --output json
`),
		Args: cobra.MinimumNArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			ids, err := parseIDArgs(args, "conversation")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			// Single ID: simple output
			if len(ids) == 1 {
				result, err := client.Conversations().ToggleStatus(ctx, ids[0], "resolved", 0)
				if err != nil {
					return fmt.Errorf("failed to resolve conversation %d: %w", ids[0], err)
				}

				if isJSON(cmd) {
					return printJSON(cmd, result.Payload)
				}

				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversation #%d resolved\n", result.Payload.ConversationID)
				return nil
			}

			// Multiple IDs: bulk operation
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

			if isJSON(cmd) {
				var output []map[string]any
				for _, r := range results {
					item := map[string]any{"id": r.ID, "success": r.Success}
					if r.Error != nil {
						item["error"] = r.Error.Error()
					}
					if r.Success {
						item["status"] = "resolved"
					}
					output = append(output, item)
				}
				return printJSON(cmd, map[string]any{
					"success_count": successCount,
					"fail_count":    failCount,
					"results":       output,
				})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Resolved %d conversations (%d failed)\n", successCount, failCount)
			return nil
		}),
	}

	cmd.Flags().IntVar(&concurrency, "concurrency", DefaultConcurrency, "Max concurrent operations (for multiple IDs)")
	cmd.Flags().BoolVar(&progress, "progress", true, "Show progress while running (for multiple IDs)")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "Disable progress output")

	return cmd
}

func newConversationsTogglePriorityCmd() *cobra.Command {
	var priority string

	cmd := &cobra.Command{
		Use:   "toggle-priority <id>",
		Short: "Toggle conversation priority",
		Long:  "Change the priority of a conversation",
		Example: strings.TrimSpace(`
  # Set conversation priority to urgent
  chatwoot conversations toggle-priority 123 --priority urgent

  # Set conversation priority to low
  chatwoot conversations toggle-priority 123 --priority low
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			if priority == "" {
				return fmt.Errorf("--priority is required")
			}

			if err := validatePriority(priority); err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.Conversations().TogglePriority(cmdContext(cmd), id, priority); err != nil {
				return fmt.Errorf("failed to toggle priority for conversation %d: %w", id, err)
			}

			// Fetch updated conversation since toggle_priority returns no body
			conv, err := client.Conversations().Get(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get conversation %d after priority update: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, conv)
			}

			displayID := conv.ID
			if conv.DisplayID != nil {
				displayID = *conv.DisplayID
			}
			priorityValue := "none"
			if conv.Priority != nil {
				priorityValue = *conv.Priority
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversation #%d priority updated to: %s\n", displayID, priorityValue)

			return nil
		}),
	}

	cmd.Flags().StringVar(&priority, "priority", "", "New priority (urgent|high|medium|low|none) (required)")
	registerStaticCompletions(cmd, "priority", []string{"urgent", "high", "medium", "low", "none"})

	return cmd
}

func newConversationsUpdateCmd() *cobra.Command {
	var priority string
	var slaPolicyID int

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update conversation attributes",
		Long:  "Update conversation attributes such as priority and SLA policy",
		Example: strings.TrimSpace(`
  # Update conversation priority
  chatwoot conversations update 123 --priority high

  # Assign SLA policy (Enterprise feature)
  chatwoot conversations update 123 --sla-policy-id 5

  # Update both priority and SLA policy
  chatwoot conversations update 123 --priority urgent --sla-policy-id 5
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			// Validate that at least one flag is provided
			if priority == "" && slaPolicyID == 0 {
				return fmt.Errorf("at least one of --priority or --sla-policy-id is required")
			}

			// Validate priority if provided
			if priority != "" {
				if err := validatePriority(priority); err != nil {
					return err
				}
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			conv, err := client.Conversations().Update(cmdContext(cmd), id, priority, slaPolicyID)
			if err != nil {
				return fmt.Errorf("failed to update conversation %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, conv)
			}

			displayID := conv.ID
			if conv.DisplayID != nil {
				displayID = *conv.DisplayID
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversation #%d updated\n", displayID)
			if conv.Priority != nil {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Priority: %s\n", *conv.Priority)
			}
			// Note: SLA policy info may not be in standard conversation response
			if slaPolicyID > 0 {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  SLA Policy ID: %d\n", slaPolicyID)
			}

			return nil
		}),
	}

	cmd.Flags().StringVar(&priority, "priority", "", "Priority (urgent|high|medium|low|none)")
	cmd.Flags().IntVar(&slaPolicyID, "sla-policy-id", 0, "SLA policy ID (Enterprise feature)")
	registerStaticCompletions(cmd, "priority", []string{"urgent", "high", "medium", "low", "none"})

	return cmd
}

func newConversationsAssignCmd() *cobra.Command {
	var (
		agent       string
		team        string
		assigneeID  int
		teamID      int
		concurrency int
		progress    bool
		noProgress  bool
	)

	cmd := &cobra.Command{
		Use:   "assign <id> [id...]",
		Short: "Assign conversations to agent or team",
		Long:  "Assign one or more conversations to an agent and/or team",
		Example: strings.TrimSpace(`
  # Assign to agent
  chatwoot conversations assign 123 --agent 5

  # Assign to team
  chatwoot conversations assign 123 --team 2

  # Assign to both agent and team
  chatwoot conversations assign 123 --agent 5 --team 2

  # Assign multiple conversations (space-separated)
  chatwoot conversations assign 123 456 789 --agent 5

  # Assign multiple conversations (comma-separated)
  chatwoot conversations assign 123,456,789 --agent 5

  # Assign with JSON output
  chatwoot conversations assign 123 456 --agent 5 --output json
`),
		Args: cobra.MinimumNArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			ids, err := parseIDArgs(args, "conversation")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			// Backwards-compat: map deprecated int flags into string flags if set.
			if agent == "" && assigneeID > 0 {
				agent = fmt.Sprintf("%d", assigneeID)
			}
			if team == "" && teamID > 0 {
				team = fmt.Sprintf("%d", teamID)
			}

			// Interactive prompts only for single ID when no flags provided
			if len(ids) == 1 && agent == "" && team == "" {
				if isInteractive() {
					selectedAgent, err := promptAgentID(cmdContext(cmd), client)
					if err != nil {
						return err
					}
					if selectedAgent > 0 {
						agent = fmt.Sprintf("%d", selectedAgent)
					}
					selectedTeam, err := promptTeamID(cmdContext(cmd), client)
					if err != nil {
						return err
					}
					if selectedTeam > 0 {
						team = fmt.Sprintf("%d", selectedTeam)
					}
				}
			}

			ctx := cmdContext(cmd)

			agentID, err := resolveAgentID(ctx, client, agent)
			if err != nil {
				return err
			}
			resolvedTeamID, err := resolveTeamID(ctx, client, team)
			if err != nil {
				return err
			}

			if agentID == 0 && resolvedTeamID == 0 {
				return fmt.Errorf("at least one of --agent or --team is required")
			}

			// Single ID: simple output
			if len(ids) == 1 {
				id := ids[0]
				if _, err := client.Conversations().Assign(ctx, id, agentID, resolvedTeamID); err != nil {
					return fmt.Errorf("failed to assign conversation %d: %w", id, err)
				}

				// Fetch updated conversation since assignments returns the agent/team, not the conversation
				conv, err := client.Conversations().Get(ctx, id)
				if err != nil {
					return fmt.Errorf("failed to get conversation %d after assignment: %w", id, err)
				}

				if isJSON(cmd) {
					return printJSON(cmd, conv)
				}

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
			}

			// Multiple IDs: bulk operation
			results := runBulkOperation(
				ctx,
				ids,
				int64(concurrency),
				bulkProgressEnabled(cmd, progress, noProgress),
				cmd.ErrOrStderr(),
				func(ctx context.Context, id int) (any, error) {
					result, err := client.Conversations().Assign(ctx, id, agentID, resolvedTeamID)
					if err != nil {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Failed to assign conversation %d: %v\n", id, err)
						return nil, err
					}
					return result, nil
				},
			)

			successCount, failCount := countResults(results)

			if isJSON(cmd) {
				var output []map[string]any
				for _, r := range results {
					item := map[string]any{"id": r.ID, "success": r.Success}
					if r.Error != nil {
						item["error"] = r.Error.Error()
					}
					if r.Success {
						if agentID > 0 {
							item["agent_id"] = agentID
						}
						if resolvedTeamID > 0 {
							item["team_id"] = resolvedTeamID
						}
					}
					output = append(output, item)
				}
				return printJSON(cmd, map[string]any{
					"success_count": successCount,
					"fail_count":    failCount,
					"results":       output,
				})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Assigned %d conversations (%d failed)\n", successCount, failCount)
			return nil
		}),
	}

	cmd.Flags().StringVar(&agent, "agent", "", "Agent ID, name, or email to assign")
	cmd.Flags().StringVar(&team, "team", "", "Team ID or name to assign")
	cmd.Flags().IntVar(&concurrency, "concurrency", DefaultConcurrency, "Max concurrent operations (for multiple IDs)")
	cmd.Flags().BoolVar(&progress, "progress", true, "Show progress while running (for multiple IDs)")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "Disable progress output")

	// Keep backwards compatibility with old flag names
	cmd.Flags().IntVar(&assigneeID, "assignee-id", 0, "Agent ID to assign (deprecated, use --agent)")
	cmd.Flags().IntVar(&teamID, "team-id", 0, "Team ID to assign (deprecated, use --team)")
	_ = cmd.Flags().MarkHidden("assignee-id")
	_ = cmd.Flags().MarkHidden("team-id")

	return cmd
}

func newConversationsLabelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "labels <id>",
		Short: "Get conversation labels",
		Long:  "Retrieve labels for a specific conversation",
		Example: strings.TrimSpace(`
  # Get labels for a conversation
  chatwoot conversations labels 123

  # JSON output - returns an object with an "items" array
  chatwoot conversations labels 123 --output json
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

			labels, err := client.Conversations().Labels(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get labels for conversation %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, labels)
			}

			if len(labels) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No labels")
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Labels:")
				for _, label := range labels {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", label)
				}
			}

			return nil
		}),
	}

	return cmd
}

func newConversationsLabelsAddCmd() *cobra.Command {
	var labelsStr string

	cmd := &cobra.Command{
		Use:   "labels-add <id>",
		Short: "Add labels to conversation",
		Long:  "Add one or more labels to a conversation",
		Example: strings.TrimSpace(`
  # Add labels to a conversation
  chatwoot conversations labels-add 123 --labels "bug,urgent"
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			if labelsStr == "" {
				return fmt.Errorf("--labels is required")
			}

			labels := strings.Split(labelsStr, ",")
			for i := range labels {
				labels[i] = strings.TrimSpace(labels[i])
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			resultLabels, err := client.Conversations().AddLabels(cmdContext(cmd), id, labels)
			if err != nil {
				return fmt.Errorf("failed to add labels to conversation %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, resultLabels)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Labels updated for conversation #%d\n", id)
			if len(resultLabels) > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Current labels:")
				for _, label := range resultLabels {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", label)
				}
			}

			return nil
		}),
	}

	cmd.Flags().StringVar(&labelsStr, "labels", "", "Comma-separated list of labels (required)")

	return cmd
}

func newConversationsLabelsRemoveCmd() *cobra.Command {
	var labelsStr string

	cmd := &cobra.Command{
		Use:   "labels-remove <id>",
		Short: "Remove labels from conversation",
		Long:  "Remove one or more labels from a conversation",
		Example: strings.TrimSpace(`
  # Remove labels from a conversation
  chatwoot conversations labels-remove 123 --labels "bug,urgent"
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			if labelsStr == "" {
				return fmt.Errorf("--labels is required")
			}

			labelsToRemove := strings.Split(labelsStr, ",")
			for i := range labelsToRemove {
				labelsToRemove[i] = strings.TrimSpace(labelsToRemove[i])
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			// Get current labels
			currentLabels, err := client.Conversations().Labels(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get current labels for conversation %d: %w", id, err)
			}

			// Build set of labels to remove for O(1) lookup
			removeSet := make(map[string]bool)
			for _, label := range labelsToRemove {
				removeSet[label] = true
			}

			// Filter out labels to remove
			var remainingLabels []string
			for _, label := range currentLabels {
				if !removeSet[label] {
					remainingLabels = append(remainingLabels, label)
				}
			}

			// Update with remaining labels
			resultLabels, err := client.Conversations().AddLabels(cmdContext(cmd), id, remainingLabels)
			if err != nil {
				return fmt.Errorf("failed to update labels for conversation %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, resultLabels)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Labels updated for conversation #%d\n", id)
			if len(resultLabels) > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Current labels:")
				for _, label := range resultLabels {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", label)
				}
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No labels remaining")
			}

			return nil
		}),
	}

	cmd.Flags().StringVar(&labelsStr, "labels", "", "Comma-separated list of labels to remove (required)")

	return cmd
}

func newConversationsCustomAttributesCmd() *cobra.Command {
	var setAttrs []string

	cmd := &cobra.Command{
		Use:   "custom-attributes <id>",
		Short: "Update conversation custom attributes",
		Long:  "Update custom attributes for a conversation",
		Example: strings.TrimSpace(`
  # Set custom attributes
  chatwoot conversations custom-attributes 123 --set priority=high --set source=web

  # JSON output - returns attributes object directly
  chatwoot conversations custom-attributes 123 --set priority=high --output json
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			if len(setAttrs) == 0 {
				return fmt.Errorf("at least one --set key=value is required")
			}

			attrs := make(map[string]any)
			for _, attr := range setAttrs {
				parts := strings.SplitN(attr, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid attribute format: %s (expected key=value)", attr)
				}
				attrs[parts[0]] = parts[1]
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.Conversations().UpdateCustomAttributes(cmdContext(cmd), id, attrs); err != nil {
				return fmt.Errorf("failed to update custom attributes for conversation %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, attrs)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Custom attributes updated for conversation #%d\n", id)
			for key, value := range attrs {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s = %v\n", key, value)
			}

			return nil
		}),
	}

	cmd.Flags().StringArrayVar(&setAttrs, "set", nil, "Set custom attribute (key=value)")

	return cmd
}

func newConversationsContextCmd() *cobra.Command {
	var embedImages bool

	cmd := &cobra.Command{
		Use:   "context <id>",
		Short: "Get full conversation context for AI",
		Long: `Get complete conversation context optimized for AI consumption.

Includes conversation metadata, contact info, all messages, and optionally
embeds images as base64 data URIs that AI vision models can consume directly.`,
		Example: strings.TrimSpace(`
  # Get conversation context
  chatwoot conversations context 123

  # Use conversation URL from browser
  chatwoot conversations context https://app.chatwoot.com/app/accounts/1/conversations/123

  # Get context with embedded images (for AI vision)
  chatwoot conversations context 123 --embed-images

  # Pipe to AI for draft response
  chatwoot conversations context 123 --embed-images --output json | ai-tool
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

			ctx, err := client.Context().GetConversation(cmdContext(cmd), id, embedImages)
			if err != nil {
				return fmt.Errorf("failed to get conversation context: %w", err)
			}

			if isAgent(cmd) {
				var detail any
				if ctx.Conversation != nil {
					convDetail := agentfmt.ConversationDetailFromConversation(*ctx.Conversation)
					convDetail = resolveConversationDetail(cmdContext(cmd), client, convDetail)
					detail = convDetail
				}

				var contactDetail any
				if ctx.Contact != nil {
					contactDetail = agentfmt.ContactDetailFromContact(*ctx.Contact)
				}

				var contactLabels []string
				var contactInboxes []contextInboxSummary
				if ctx.Contact != nil && ctx.Contact.ID > 0 {
					labels, err := client.Contacts().Labels(cmdContext(cmd), ctx.Contact.ID)
					if err == nil {
						contactLabels = labels
					}
					inboxes, err := client.Contacts().ContactableInboxes(cmdContext(cmd), ctx.Contact.ID)
					if err == nil {
						contactInboxes = contextInboxSummaries(inboxes)
					}
				}

				messages, publicCount, privateCount, embeddedCount := contextMessageSummaries(ctx.Messages)
				meta := map[string]any{
					"conversation_id":  id,
					"message_count":    len(ctx.Messages),
					"public_messages":  publicCount,
					"private_messages": privateCount,
					"embed_images":     embedImages,
				}
				if embeddedCount > 0 {
					meta["embedded_attachments"] = embeddedCount
				}

				item := map[string]any{
					"conversation": detail,
					"contact":      contactDetail,
					"messages":     messages,
					"summary":      ctx.Summary,
					"meta":         meta,
				}
				if len(contactLabels) > 0 {
					item["contact_labels"] = contactLabels
				}
				if len(contactInboxes) > 0 {
					item["contact_inboxes"] = contactInboxes
				}

				payload := agentfmt.ItemEnvelope{
					Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Item: item,
				}
				return printJSON(cmd, payload)
			}
			if isJSON(cmd) {
				return printJSON(cmd, ctx)
			}

			// Human-readable output
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "=== Conversation #%d ===\n", id)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Summary: %s\n\n", ctx.Summary)

			if ctx.Contact != nil {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Customer: %s\n", ctx.Contact.Name)
				if ctx.Contact.Email != "" {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Email: %s\n", ctx.Contact.Email)
				}
				if ctx.Contact.PhoneNumber != "" {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Phone: %s\n", ctx.Contact.PhoneNumber)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "--- Messages ---")
			for _, msg := range ctx.Messages {
				sender := "Customer"
				if msg.MessageType == 1 {
					sender = "Agent"
				}
				if msg.Private {
					sender = "Private Note"
				}

				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", sender, msg.Content)

				for _, att := range msg.Attachments {
					if att.Embedded != "" {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  📎 [%s - embedded as base64]\n", att.FileType)
					} else {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  📎 [%s - %s]\n", att.FileType, att.DataURL)
					}
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
			}

			return nil
		}),
	}

	cmd.Flags().BoolVar(&embedImages, "embed-images", false, "Embed images as base64 data URIs for AI vision")

	return cmd
}

type contextMessageAttachmentSummary struct {
	ID       int    `json:"id"`
	FileType string `json:"file_type,omitempty"`
	DataURL  string `json:"data_url,omitempty"`
	FileSize int    `json:"file_size,omitempty"`
	Embedded string `json:"embedded,omitempty"`
}

type contextInboxSummary struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ChannelType string `json:"channel_type,omitempty"`
}

type contextMessageSummary struct {
	ID          int                               `json:"id"`
	Type        string                            `json:"type"`
	Private     bool                              `json:"private"`
	Content     string                            `json:"content"`
	ContentType string                            `json:"content_type,omitempty"`
	SenderType  string                            `json:"sender_type,omitempty"`
	CreatedAt   *agentfmt.Timestamp               `json:"created_at,omitempty"`
	Attachments []contextMessageAttachmentSummary `json:"attachments,omitempty"`
}

func contextMessageSummaries(messages []api.MessageWithEmbeddings) ([]contextMessageSummary, int, int, int) {
	if len(messages) == 0 {
		return nil, 0, 0, 0
	}
	out := make([]contextMessageSummary, 0, len(messages))
	publicCount := 0
	privateCount := 0
	embeddedCount := 0

	for _, msg := range messages {
		if msg.Private {
			privateCount++
		} else {
			publicCount++
		}
		summary := contextMessageSummary{
			ID:          msg.ID,
			Type:        messageTypeNameFromValue(msg.MessageType),
			Private:     msg.Private,
			Content:     msg.Content,
			ContentType: msg.ContentType,
			SenderType:  msg.SenderType,
			CreatedAt:   agentTimestampFromUnix(msg.CreatedAt),
		}
		if len(msg.Attachments) > 0 {
			attachments := make([]contextMessageAttachmentSummary, 0, len(msg.Attachments))
			for _, att := range msg.Attachments {
				if att.Embedded != "" {
					embeddedCount++
				}
				attachments = append(attachments, contextMessageAttachmentSummary{
					ID:       att.ID,
					FileType: att.FileType,
					DataURL:  att.DataURL,
					FileSize: att.FileSize,
					Embedded: att.Embedded,
				})
			}
			summary.Attachments = attachments
		}
		out = append(out, summary)
	}
	return out, publicCount, privateCount, embeddedCount
}

func contextInboxSummaries(inboxes []api.Inbox) []contextInboxSummary {
	if len(inboxes) == 0 {
		return nil
	}
	out := make([]contextInboxSummary, 0, len(inboxes))
	for _, inbox := range inboxes {
		out = append(out, contextInboxSummary{
			ID:          inbox.ID,
			Name:        inbox.Name,
			ChannelType: inbox.ChannelType,
		})
	}
	return out
}

func messageTypeNameFromValue(value int) string {
	switch value {
	case api.MessageTypeIncoming:
		return "incoming"
	case api.MessageTypeOutgoing:
		return "outgoing"
	case api.MessageTypeActivity:
		return "activity"
	case api.MessageTypeTemplate:
		return "template"
	default:
		return "unknown"
	}
}

func agentTimestampFromUnix(unix int64) *agentfmt.Timestamp {
	if unix == 0 {
		return nil
	}
	return &agentfmt.Timestamp{
		Unix: unix,
		ISO:  time.Unix(unix, 0).UTC().Format(time.RFC3339),
	}
}

func newConversationsMarkUnreadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mark-unread <id>",
		Short: "Mark conversation as unread",
		Long: `Mark a conversation as unread for all agents.

This resets the agent_last_seen_at timestamp, making the conversation appear
as unread in the inbox for all agents (not just the current user).`,
		Example: strings.TrimSpace(`
  # Mark a single conversation as unread
  chatwoot conversations mark-unread 123

  # Mark multiple conversations as unread
  for id in 123 124 125; do chatwoot conversations mark-unread $id; done
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

			// Get initial state to verify change
			beforeConv, err := client.Conversations().Get(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get conversation %d: %w", id, err)
			}
			initialUnread := beforeConv.Unread

			if err := client.Conversations().MarkUnread(cmdContext(cmd), id); err != nil {
				return fmt.Errorf("failed to mark conversation %d as unread: %w", id, err)
			}

			// Fetch updated conversation to verify the operation succeeded
			afterConv, err := client.Conversations().Get(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get conversation %d after marking unread: %w", id, err)
			}

			// Verify the operation didn't fail silently (count should not decrease)
			// Note: The API resets agent_last_seen_at timestamp, which may not always increment Unread
			if afterConv.Unread < initialUnread {
				return fmt.Errorf("mark-unread operation appears to have failed (unread count decreased from %d to %d)", initialUnread, afterConv.Unread)
			}

			if isJSON(cmd) {
				return printJSON(cmd, afterConv)
			}

			displayID := afterConv.ID
			if afterConv.DisplayID != nil {
				displayID = *afterConv.DisplayID
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversation #%d marked as unread (unread count: %d)\n", displayID, afterConv.Unread)
			return nil
		}),
	}

	return cmd
}

func newConversationsSearchCmd() *cobra.Command {
	var page int
	var all bool
	var maxPages int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search conversations by message content",
		Long:  "Search conversations by message content across all conversations",
		Example: strings.TrimSpace(`
  # Search for conversations mentioning "password reset"
  chatwoot conversations search "password reset"

  # Search with pagination
  chatwoot conversations search "refund" --page 2

  # Fetch all matching conversations
  chatwoot conversations search "error" --all

  # JSON output
  chatwoot conversations search "billing" -o json
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			query := args[0]
			if query == "" {
				return fmt.Errorf("search query cannot be empty")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if all {
				return searchAllConversations(cmd, client, query, maxPages)
			}

			result, err := client.Conversations().Search(cmdContext(cmd), query, page)
			if err != nil {
				return fmt.Errorf("failed to search conversations: %w", err)
			}

			conversations := result.Data.Payload

			if isAgent(cmd) {
				summaries := agentfmt.ConversationSummaries(conversations)
				summaries = resolveConversationSummaries(cmdContext(cmd), client, summaries)
				payload := agentfmt.SearchEnvelope{
					Kind:    agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Query:   query,
					Results: summaries,
					Summary: map[string]int{"conversations": len(conversations)},
					Meta: map[string]any{
						"page": page,
						"meta": result.Data.Meta,
					},
				}
				return printJSON(cmd, payload)
			}
			if isJSON(cmd) {
				return printJSON(cmd, conversations)
			}

			if len(conversations) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No conversations found matching your query")
				return nil
			}

			printConversationsTable(cmd.OutOrStdout(), conversations)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nPage %d (%d conversations)\n", page, len(conversations))
			return nil
		}),
	}

	cmd.Flags().IntVarP(&page, "page", "p", 1, "Page number")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages")
	cmd.Flags().IntVar(&maxPages, "max-pages", 100, "Maximum pages to fetch with --all")

	return cmd
}

func searchAllConversations(cmd *cobra.Command, client *api.Client, query string, maxPages int) error {
	var allConversations []api.Conversation
	currentPage := 1
	pagesFetched := 0

	for {
		if pagesFetched >= maxPages {
			return fmt.Errorf("safety limit reached: fetched %d pages (%d conversations). Use --max-pages to increase", maxPages, len(allConversations))
		}

		if currentPage > 1 {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Fetching page %d...\n", currentPage) //nolint:errcheck
		}

		result, err := client.Conversations().Search(cmdContext(cmd), query, currentPage)
		if err != nil {
			return fmt.Errorf("failed to search conversations: %w", err)
		}

		conversations := result.Data.Payload
		totalPages := int(result.Data.Meta.TotalPages)

		if len(conversations) == 0 {
			break
		}

		allConversations = append(allConversations, conversations...)
		pagesFetched++

		if totalPages > 0 && currentPage >= totalPages {
			break
		}

		currentPage++
	}

	if isAgent(cmd) {
		summaries := agentfmt.ConversationSummaries(allConversations)
		summaries = resolveConversationSummaries(cmdContext(cmd), client, summaries)
		payload := agentfmt.SearchEnvelope{
			Kind:    agentfmt.KindFromCommandPath(cmd.CommandPath()),
			Query:   query,
			Results: summaries,
			Summary: map[string]int{"conversations": len(allConversations)},
			Meta: map[string]any{
				"pages_fetched": pagesFetched,
				"total_items":   len(allConversations),
				"all":           true,
			},
		}
		return printJSON(cmd, payload)
	}
	if isJSON(cmd) {
		return printJSON(cmd, allConversations)
	}

	if len(allConversations) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No conversations found matching your query")
		return nil
	}

	printConversationsTable(cmd.OutOrStdout(), allConversations)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nTotal: %d conversations (%d pages)\n", len(allConversations), pagesFetched)
	return nil
}

func newConversationsAttachmentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attachments <conversation-id>",
		Short: "List attachments in a conversation",
		Long:  "List all attachments (files, images) in a conversation",
		Example: strings.TrimSpace(`
  # List attachments in a conversation
  chatwoot conversations attachments 123

  # JSON output with URLs
  chatwoot conversations attachments 123 -o json
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

			attachments, err := client.Conversations().Attachments(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get attachments for conversation %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, attachments)
			}

			if len(attachments) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No attachments found")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
			_, _ = fmt.Fprintln(w, "ID\tTYPE\tSIZE\tURL")
			for _, att := range attachments {
				size := formatFileSize(att.FileSize)
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", att.ID, att.FileType, size, att.DataURL)
			}
			_ = w.Flush()

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nTotal: %d attachments\n", len(attachments))
			return nil
		}),
	}

	return cmd
}

func newConversationsMuteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mute <id>",
		Short: "Mute a conversation",
		Long: `Mute a conversation to stop receiving notifications.

Muted conversations will not trigger desktop or push notifications for new messages.`,
		Example: strings.TrimSpace(`
  # Mute a conversation
  chatwoot conversations mute 123

  # Mute and output as JSON
  chatwoot conversations mute 123 --output json
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

			if err := client.Conversations().ToggleMute(cmdContext(cmd), id, true); err != nil {
				return fmt.Errorf("failed to mute conversation %d: %w", id, err)
			}

			// Fetch updated conversation to verify the operation succeeded
			conv, err := client.Conversations().Get(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get conversation %d after muting: %w", id, err)
			}

			// Verify the operation succeeded
			if !conv.Muted {
				return fmt.Errorf("mute operation failed: conversation is still unmuted")
			}

			if isJSON(cmd) {
				return printJSON(cmd, conv)
			}

			displayID := conv.ID
			if conv.DisplayID != nil {
				displayID = *conv.DisplayID
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversation #%d muted (muted: %t)\n", displayID, conv.Muted)
			return nil
		}),
	}

	return cmd
}

func newConversationsUnmuteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unmute <id>",
		Short: "Unmute a conversation",
		Long: `Unmute a conversation to resume receiving notifications.

Unmuted conversations will trigger desktop and push notifications for new messages.`,
		Example: strings.TrimSpace(`
  # Unmute a conversation
  chatwoot conversations unmute 123

  # Unmute and output as JSON
  chatwoot conversations unmute 123 --output json
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

			if err := client.Conversations().ToggleMute(cmdContext(cmd), id, false); err != nil {
				return fmt.Errorf("failed to unmute conversation %d: %w", id, err)
			}

			// Fetch updated conversation to verify the operation succeeded
			conv, err := client.Conversations().Get(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get conversation %d after unmuting: %w", id, err)
			}

			// Verify the operation succeeded
			if conv.Muted {
				return fmt.Errorf("unmute operation failed: conversation is still muted")
			}

			if isJSON(cmd) {
				return printJSON(cmd, conv)
			}

			displayID := conv.ID
			if conv.DisplayID != nil {
				displayID = *conv.DisplayID
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversation #%d unmuted (muted: %t)\n", displayID, conv.Muted)
			return nil
		}),
	}

	return cmd
}

func newConversationsTranscriptCmd() *cobra.Command {
	var (
		email      string
		limit      int
		maxPages   int
		publicOnly bool
	)

	cmd := &cobra.Command{
		Use:   "transcript <id>",
		Short: "Render or send a conversation transcript",
		Long: `Render a conversation transcript to stdout, or send it via email.

When --email is provided, the transcript is sent via Chatwoot and no
message content is printed. Without --email, the transcript is rendered
locally with private notes included by default.`,
		Example: strings.TrimSpace(`
  # Render transcript to stdout (includes private notes)
  chatwoot conversations transcript 123

  # Render public-only transcript
  chatwoot conversations transcript 123 --public-only

  # Limit to the most recent messages
  chatwoot conversations transcript 123 --limit 200

  # Send transcript to an email address
  chatwoot conversations transcript 123 --email user@example.com
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			if cmd.Flags().Changed("limit") && limit < 1 {
				return fmt.Errorf("--limit must be at least 1")
			}
			if cmd.Flags().Changed("max-pages") && maxPages < 1 {
				return fmt.Errorf("--max-pages must be at least 1")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if email != "" {
				if err := client.Conversations().Transcript(cmdContext(cmd), id, email); err != nil {
					return fmt.Errorf("failed to send transcript for conversation %d: %w", id, err)
				}

				if isJSON(cmd) {
					return printJSON(cmd, map[string]any{
						"conversation_id": id,
						"email":           email,
						"status":          "sent",
					})
				}

				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Transcript for conversation #%d sent to %s\n", id, email)
				return nil
			}

			conv, err := client.Conversations().Get(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get conversation %d: %w", id, err)
			}

			var messages []api.Message
			if limit > 0 {
				messages, err = client.Messages().ListWithLimit(cmdContext(cmd), id, limit, maxPages)
			} else {
				messages, err = client.Messages().ListAllWithMaxPages(cmdContext(cmd), id, maxPages)
			}
			if err != nil {
				return fmt.Errorf("failed to list messages for conversation %d: %w", id, err)
			}

			filtered, publicCount, privateCount := filterTranscriptMessages(messages, publicOnly)
			sort.SliceStable(filtered, func(i, j int) bool {
				if filtered[i].CreatedAt == filtered[j].CreatedAt {
					return filtered[i].ID < filtered[j].ID
				}
				return filtered[i].CreatedAt < filtered[j].CreatedAt
			})

			meta := map[string]any{
				"conversation_id":   id,
				"total_messages":    len(messages),
				"public_messages":   publicCount,
				"private_messages":  privateCount,
				"included_messages": len(filtered),
				"public_only":       publicOnly,
			}
			if limit > 0 {
				meta["limit"] = limit
			}

			if isAgent(cmd) {
				detail := agentfmt.ConversationDetailFromConversation(*conv)
				detail = resolveConversationDetail(cmdContext(cmd), client, detail)
				wrapped := make([]agentfmt.MessageSummaryWithPosition, len(filtered))
				for i, msg := range filtered {
					summary := agentfmt.MessageSummaryFromMessage(msg)
					wrapped[i] = agentfmt.MessageSummaryWithPosition{
						MessageSummary: summary,
						Position:       i + 1,
						TotalMessages:  len(filtered),
					}
				}
				payload := agentfmt.ItemEnvelope{
					Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Item: map[string]any{
						"conversation": detail,
						"messages":     wrapped,
						"meta":         meta,
					},
				}
				return printJSON(cmd, payload)
			}
			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{
					"conversation": conv,
					"messages":     filtered,
					"meta":         meta,
				})
			}

			writeTranscript(cmd.OutOrStdout(), conv, filtered, publicOnly, publicCount, privateCount, limit)
			return nil
		}),
	}

	cmd.Flags().StringVar(&email, "email", "", "Email address to send transcript to")
	cmd.Flags().IntVar(&limit, "limit", 0, "Limit the number of messages to include (default: all)")
	cmd.Flags().IntVar(&maxPages, "max-pages", 100, "Maximum pages to fetch when listing messages")
	cmd.Flags().BoolVar(&publicOnly, "public-only", false, "Exclude private notes from the transcript")

	return cmd
}

func filterTranscriptMessages(messages []api.Message, publicOnly bool) ([]api.Message, int, int) {
	if len(messages) == 0 {
		return nil, 0, 0
	}
	filtered := make([]api.Message, 0, len(messages))
	publicCount := 0
	privateCount := 0

	for _, msg := range messages {
		if msg.Private {
			privateCount++
		} else {
			publicCount++
		}
		if publicOnly && msg.Private {
			continue
		}
		filtered = append(filtered, msg)
	}
	return filtered, publicCount, privateCount
}

func transcriptActor(msg api.Message) string {
	if msg.Private {
		if msg.Sender != nil && msg.Sender.Name != "" {
			return fmt.Sprintf("Private note (%s)", msg.Sender.Name)
		}
		return "Private note"
	}
	switch msg.MessageType {
	case api.MessageTypeIncoming:
		if msg.Sender != nil && msg.Sender.Name != "" {
			return fmt.Sprintf("Customer (%s)", msg.Sender.Name)
		}
		return "Customer"
	case api.MessageTypeOutgoing:
		if msg.Sender != nil && msg.Sender.Name != "" {
			return fmt.Sprintf("Agent (%s)", msg.Sender.Name)
		}
		return "Agent"
	case api.MessageTypeActivity:
		return "Activity"
	case api.MessageTypeTemplate:
		return "Template"
	default:
		if msg.Sender != nil && msg.Sender.Name != "" {
			return msg.Sender.Name
		}
		return msg.MessageTypeName()
	}
}

func writeTranscript(out io.Writer, conv *api.Conversation, messages []api.Message, publicOnly bool, publicCount, privateCount, limit int) {
	displayID := conv.ID
	if conv.DisplayID != nil {
		displayID = *conv.DisplayID
	}
	summary := agentfmt.ConversationSummaryFromConversation(*conv)

	_, _ = fmt.Fprintf(out, "Conversation #%d\n", displayID)
	_, _ = fmt.Fprintf(out, "ID: %d\n", conv.ID)
	_, _ = fmt.Fprintf(out, "Status: %s\n", conv.Status)
	if conv.Priority != nil {
		_, _ = fmt.Fprintf(out, "Priority: %s\n", *conv.Priority)
	}
	_, _ = fmt.Fprintf(out, "Inbox ID: %d\n", conv.InboxID)
	if summary.Contact != nil && (summary.Contact.Name != "" || summary.Contact.Email != "" || summary.Contact.Phone != "") {
		label := summary.Contact.Name
		if label == "" {
			if conv.ContactID > 0 {
				label = fmt.Sprintf("Contact %d", conv.ContactID)
			} else {
				label = "Contact"
			}
		}
		_, _ = fmt.Fprintf(out, "Contact: %s", label)
		if summary.Contact.Email != "" {
			_, _ = fmt.Fprintf(out, " <%s>", summary.Contact.Email)
		}
		if summary.Contact.Phone != "" {
			_, _ = fmt.Fprintf(out, " (%s)", summary.Contact.Phone)
		}
		if conv.ContactID > 0 {
			_, _ = fmt.Fprintf(out, " [id %d]", conv.ContactID)
		}
		_, _ = fmt.Fprintln(out)
	} else if conv.ContactID > 0 {
		_, _ = fmt.Fprintf(out, "Contact ID: %d\n", conv.ContactID)
	}
	if conv.AssigneeID != nil {
		_, _ = fmt.Fprintf(out, "Assignee ID: %d\n", *conv.AssigneeID)
	}
	if conv.TeamID != nil {
		_, _ = fmt.Fprintf(out, "Team ID: %d\n", *conv.TeamID)
	}
	_, _ = fmt.Fprintf(out, "Created: %s\n", formatTimestamp(conv.CreatedAtTime()))
	if conv.LastActivityAt > 0 {
		_, _ = fmt.Fprintf(out, "Last activity: %s\n", formatTimestamp(conv.LastActivityAtTime()))
	}
	_, _ = fmt.Fprintf(out, "Messages: %d (public %d, private %d)\n", len(messages), publicCount, privateCount)
	if publicOnly {
		_, _ = fmt.Fprintln(out, "Public only: true")
	}
	if limit > 0 {
		_, _ = fmt.Fprintf(out, "Limit: %d\n", limit)
	}

	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "--- Transcript ---")

	for _, msg := range messages {
		_, _ = fmt.Fprintf(out, "[%s] %s\n", formatTimestamp(msg.CreatedAtTime()), transcriptActor(msg))

		content := strings.TrimSpace(msg.Content)
		if content == "" {
			content = "(no content)"
		}
		for _, line := range strings.Split(content, "\n") {
			_, _ = fmt.Fprintf(out, "  %s\n", line)
		}

		for _, att := range msg.Attachments {
			label := att.FileType
			if label == "" {
				label = "attachment"
			}
			if att.DataURL != "" {
				_, _ = fmt.Fprintf(out, "  [attachment] %s %s\n", label, att.DataURL)
			} else {
				_, _ = fmt.Fprintf(out, "  [attachment] %s\n", label)
			}
		}
		_, _ = fmt.Fprintln(out)
	}
}

func newConversationsTypingCmd() *cobra.Command {
	var (
		typingOn  bool
		isPrivate bool
	)

	cmd := &cobra.Command{
		Use:   "typing <id>",
		Short: "Toggle typing indicator for a conversation",
		Long: `Toggle the typing indicator for a conversation.

This shows or hides the "agent is typing" indicator that the customer sees.
Use --private to show the typing indicator only to other agents (for private notes).`,
		Example: strings.TrimSpace(`
  # Show typing indicator to customer
  chatwoot conversations typing 123 --on

  # Hide typing indicator
  chatwoot conversations typing 123

  # Show typing indicator for private note (visible only to agents)
  chatwoot conversations typing 123 --on --private
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

			if err := client.Conversations().ToggleTyping(cmdContext(cmd), id, typingOn, isPrivate); err != nil {
				return fmt.Errorf("failed to toggle typing status for conversation %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{
					"conversation_id": id,
					"typing":          typingOn,
					"private":         isPrivate,
				})
			}

			status := "off"
			if typingOn {
				status = "on"
			}
			visibility := "public"
			if isPrivate {
				visibility = "private"
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Typing indicator for conversation #%d: %s (%s)\n", id, status, visibility)
			return nil
		}),
	}

	cmd.Flags().BoolVar(&typingOn, "on", false, "Turn typing indicator on (default: off)")
	cmd.Flags().BoolVar(&isPrivate, "private", false, "Show typing indicator only to agents (for private notes)")

	return cmd
}

func formatFileSize(bytes int) string {
	if bytes == 0 {
		return "-"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatMessageCount formats a message count for display
func formatMessageCount(count int) string {
	if count == 0 {
		return "-"
	}
	return fmt.Sprintf("[%d msgs]", count)
}

// formatPosition formats a position indicator as [position/total]
func formatPosition(position, total int) string {
	return fmt.Sprintf("[%d/%d]", position, total)
}

const maxFutureYears = 10 * 365 * 24 * 60 * 60 // 10 years in seconds

// parseSnoozedUntil parses a snoozed-until value as Unix timestamp, RFC3339, or relative time.
func parseSnoozedUntil(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("snoozed-until cannot be empty")
	}

	// Try parsing as Unix timestamp first
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return validateSnoozedUntil(ts, "timestamp")
	}

	t, err := cli.ParseRelativeTime(s, time.Now().UTC())
	if err != nil {
		return 0, fmt.Errorf("invalid format (use Unix timestamp, RFC3339, or relative time): %w", err)
	}

	return validateSnoozedUntil(t.Unix(), "time")
}

func validateSnoozedUntil(ts int64, label string) (int64, error) {
	if ts <= 0 {
		return 0, fmt.Errorf("timestamp must be positive, got %d", ts)
	}
	// Validate reasonable timestamp range (not too far in past or future)
	now := time.Now().UTC().Unix()
	if ts < now {
		if label == "timestamp" {
			return 0, fmt.Errorf("timestamp %d is in the past", ts)
		}
		return 0, fmt.Errorf("time %q is in the past", time.Unix(ts, 0).Format(time.RFC3339))
	}
	// Prevent absurdly far future timestamps (max 10 years from now)
	maxFuture := now + maxFutureYears
	if ts > maxFuture {
		if label == "timestamp" {
			return 0, fmt.Errorf("timestamp %d is too far in the future (max 10 years)", ts)
		}
		return 0, fmt.Errorf("time %q is too far in the future (max 10 years)", time.Unix(ts, 0).Format(time.RFC3339))
	}
	return ts, nil
}

func newConversationsWatchCmd() *cobra.Command {
	var (
		status   string
		inboxID  int
		interval int
		limit    int
	)

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch conversations in real-time",
		Long:  "Poll for new and updated conversations at regular intervals",
		Example: strings.TrimSpace(`
  # Watch all open conversations
  chatwoot conversations watch --status open

  # Watch specific inbox every 5 seconds
  chatwoot conversations watch --inbox-id 1 --interval 5

  # Watch with custom limit
  chatwoot conversations watch --status open --limit 20
`),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
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

	cmd.Flags().StringVar(&status, "status", "open", "Filter by status: open, resolved, pending, snoozed, all")
	cmd.Flags().IntVar(&inboxID, "inbox-id", 0, "Filter by inbox ID")
	cmd.Flags().IntVar(&interval, "interval", 10, "Polling interval in seconds")
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum conversations to display")
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

func newConversationsBulkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bulk",
		Short: "Bulk operations on conversations",
		Long:  "Perform bulk operations on multiple conversations at once",
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
		Use:   "resolve",
		Short: "Resolve multiple conversations",
		Long:  "Mark multiple conversations as resolved at once",
		Example: strings.TrimSpace(`
  # Resolve multiple conversations
  chatwoot conversations bulk resolve --ids 1,2,3

  # Resolve and output result as JSON
  chatwoot conversations bulk resolve --ids 1,2,3 --output json

  # Resolve with custom concurrency
  chatwoot conversations bulk resolve --ids 1,2,3 --concurrency 10
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

			// Build JSON output if needed
			var output []map[string]any
			for _, r := range results {
				item := map[string]any{"id": r.ID, "success": r.Success}
				if r.Error != nil {
					item["error"] = r.Error.Error()
				}
				if r.Success {
					item["status"] = "resolved"
				}
				output = append(output, item)
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{
					"success_count": successCount,
					"fail_count":    failCount,
					"results":       output,
				})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Resolved %d conversations (%d failed)\n", successCount, failCount)
			return nil
		}),
	}

	cmd.Flags().StringVar(&conversationIDs, "ids", "", "Conversation IDs (CSV, whitespace, JSON array; or @- / @path)")
	cmd.Flags().IntVar(&concurrency, "concurrency", DefaultConcurrency, "Max concurrent operations")
	cmd.Flags().BoolVar(&progress, "progress", true, "Show progress while running")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "Disable progress output")
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
		Use:   "assign",
		Short: "Assign multiple conversations",
		Long:  "Assign multiple conversations to an agent and/or team at once",
		Example: strings.TrimSpace(`
  # Assign conversations to an agent
  chatwoot conversations bulk assign --ids 1,2,3 --agent 5

  # Assign conversations to a team
  chatwoot conversations bulk assign --ids 1,2,3 --team 2

  # Assign to both agent and team
  chatwoot conversations bulk assign --ids 1,2,3 --agent 5 --team 2

  # Assign with custom concurrency
  chatwoot conversations bulk assign --ids 1,2,3 --agent 5 --concurrency 10
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

			// Build JSON output
			var output []map[string]any
			for _, r := range results {
				item := map[string]any{"id": r.ID, "success": r.Success}
				if r.Error != nil {
					item["error"] = r.Error.Error()
				}
				if r.Success {
					if resolvedAgentID > 0 {
						item["agent_id"] = resolvedAgentID
					}
					if resolvedTeamID > 0 {
						item["team_id"] = resolvedTeamID
					}
				}
				output = append(output, item)
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{
					"success_count": successCount,
					"fail_count":    failCount,
					"results":       output,
				})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Assigned %d conversations (%d failed)\n", successCount, failCount)
			return nil
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
		Use:   "add-label",
		Short: "Add labels to multiple conversations",
		Long:  "Add one or more labels to multiple conversations at once",
		Example: strings.TrimSpace(`
  # Add a single label to multiple conversations
  chatwoot conversations bulk add-label --ids 1,2,3 --labels urgent

  # Add multiple labels to multiple conversations
  chatwoot conversations bulk add-label --ids 1,2,3 --labels urgent,bug

  # Add labels with custom concurrency
  chatwoot conversations bulk add-label --ids 1,2,3 --labels urgent --concurrency 10
`),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			ids, err := ParseResourceIDListFlag(conversationIDs, "conversation")
			if err != nil {
				return fmt.Errorf("invalid conversation IDs: %w", err)
			}

			labelList := strings.Split(labels, ",")
			var filtered []string
			for _, l := range labelList {
				l = strings.TrimSpace(l)
				if l != "" {
					filtered = append(filtered, l)
				}
			}
			labelList = filtered

			if len(labelList) == 0 {
				return fmt.Errorf("no valid labels provided after filtering empty values")
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

			// Build JSON output
			var output []map[string]any
			for _, r := range results {
				item := map[string]any{"id": r.ID, "success": r.Success}
				if r.Error != nil {
					item["error"] = r.Error.Error()
				}
				if r.Success && r.Data != nil {
					item["labels"] = r.Data
				}
				output = append(output, item)
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{
					"success_count": successCount,
					"fail_count":    failCount,
					"results":       output,
				})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added labels to %d conversations (%d failed)\n", successCount, failCount)
			return nil
		}),
	}

	cmd.Flags().StringVar(&conversationIDs, "ids", "", "Conversation IDs (CSV, whitespace, JSON array; or @- / @path)")
	cmd.Flags().StringVar(&labels, "labels", "", "Comma-separated labels to add")
	cmd.Flags().IntVar(&concurrency, "concurrency", DefaultConcurrency, "Max concurrent operations")
	cmd.Flags().BoolVar(&progress, "progress", true, "Show progress while running")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "Disable progress output")
	_ = cmd.MarkFlagRequired("ids")
	_ = cmd.MarkFlagRequired("labels")

	return cmd
}

// BatchUpdateItem represents a single conversation update in a batch operation
type BatchUpdateItem struct {
	ID         int      `json:"id"`
	Status     string   `json:"status,omitempty"`
	Priority   string   `json:"priority,omitempty"`
	Labels     []string `json:"labels,omitempty"`
	AssigneeID int      `json:"assignee_id,omitempty"`
	TeamID     int      `json:"team_id,omitempty"`
}

// BatchUpdateResult represents the result of a single batch update operation
type BatchUpdateResult struct {
	ID     int    `json:"id"`
	Action string `json:"action"`
	Status string `json:"status"` // "ok" | "error"
	Error  string `json:"error,omitempty"`
}

// BatchUpdateResponse is the response for the batch-update command
type BatchUpdateResponse struct {
	Total     int                 `json:"total"`
	Succeeded int                 `json:"succeeded"`
	Failed    int                 `json:"failed"`
	Results   []BatchUpdateResult `json:"results"`
}

// newConversationsBatchUpdateCmd creates the batch-update subcommand
func newConversationsBatchUpdateCmd() *cobra.Command {
	var concurrency int

	cmd := &cobra.Command{
		Use:   "batch-update",
		Short: "Update multiple conversations with different operations",
		Long: `Update multiple conversations in parallel with varying operations per conversation.

Reads JSON input from stdin with an array of updates. Each item can specify different
operations (status, priority, labels, assignment).`,
		Example: strings.TrimSpace(`
  # Update multiple conversations with different operations
  echo '[
    {"id": 123, "status": "resolved"},
    {"id": 456, "labels": ["handled"], "assignee_id": 5},
    {"id": 789, "priority": "low"}
  ]' | chatwoot conversations bulk batch-update

  # From a file
  cat updates.json | chatwoot conversations bulk batch-update

  # With custom concurrency
  cat updates.json | chatwoot conversations bulk batch-update --concurrency 10
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

			// Validate items
			for i, item := range items {
				if item.ID <= 0 {
					return fmt.Errorf("item %d: id must be positive", i)
				}
				// At least one operation must be specified
				if item.Status == "" && item.Priority == "" && len(item.Labels) == 0 && item.AssigneeID == 0 && item.TeamID == 0 {
					return fmt.Errorf("item %d: at least one operation (status, priority, labels, assignee_id, team_id) is required", i)
				}
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			// Process updates in parallel with bounded concurrency
			results := make([]BatchUpdateResult, len(items))
			sem := make(chan struct{}, concurrency)
			var wg sync.WaitGroup

			for i, item := range items {
				wg.Add(1)
				go func(idx int, item BatchUpdateItem) {
					defer wg.Done()
					sem <- struct{}{}        // Acquire semaphore
					defer func() { <-sem }() // Release semaphore

					result := BatchUpdateResult{
						ID:     item.ID,
						Status: "ok",
					}

					var actions []string
					var firstErr error

					// Apply status change
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

					// Apply priority change
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

					// Apply labels
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

					// Apply assignment
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

			// Count successes and failures
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

			// Text output
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

	return cmd
}

// triageItem represents a conversation needing attention with context
type triageItem struct {
	ID          int                  `json:"id"`
	InboxID     int                  `json:"inbox_id"`
	Status      string               `json:"status"`
	Unread      int                  `json:"unread"`
	WaitTime    string               `json:"wait_time"`
	LastMessage string               `json:"last_message"`
	ContactName string               `json:"contact_name"`
	InboxName   string               `json:"inbox_name,omitempty"`
	Explanation *heuristics.Analysis `json:"_explanation,omitempty"`
}

func newConversationsTriageCmd() *cobra.Command {
	var limit int
	var inboxID int
	var explain bool

	cmd := &cobra.Command{
		Use:   "triage",
		Short: "Show conversations needing attention, sorted by urgency",
		Long: `Show conversations needing attention, sorted by urgency.

Fetches open and pending conversations with unread messages, sorted by how long
the customer has been waiting (oldest first = longest waiting = most urgent).`,
		Example: strings.TrimSpace(`
  # Show top 20 conversations needing attention
  chatwoot conversations triage

  # Show top 10 with detailed reasoning
  chatwoot conversations triage --limit 10 --explain

  # Filter by inbox
  chatwoot conversations triage --inbox 5

  # JSON output for agent processing
  chatwoot conversations triage --output json
`),
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			// Fetch open conversations
			params := api.ListConversationsParams{
				Status: "open",
			}
			if inboxID > 0 {
				params.InboxID = strconv.Itoa(inboxID)
			}

			result, err := client.Conversations().List(ctx, params)
			if err != nil {
				return fmt.Errorf("failed to list conversations: %w", err)
			}

			// Filter to only conversations with unread messages
			var candidates []api.Conversation
			for _, conv := range result.Data.Payload {
				if conv.Unread > 0 {
					candidates = append(candidates, conv)
				}
			}

			// Sort by LastActivityAt ascending (oldest first = longest waiting)
			sort.Slice(candidates, func(i, j int) bool {
				return candidates[i].LastActivityAt < candidates[j].LastActivityAt
			})

			// Apply limit
			if limit > 0 && len(candidates) > limit {
				candidates = candidates[:limit]
			}

			// Build triage items with message previews
			items := make([]triageItem, 0, len(candidates))
			for _, conv := range candidates {
				item := triageItem{
					ID:       conv.ID,
					InboxID:  conv.InboxID,
					Status:   conv.Status,
					Unread:   conv.Unread,
					WaitTime: formatWaitTime(conv.LastActivityAt),
				}

				// Extract contact name from meta
				item.ContactName = getSenderNameFromMeta(conv.Meta)
				if item.ContactName == "" {
					item.ContactName = fmt.Sprintf("Contact #%d", conv.ContactID)
				}

				// Fetch last message preview
				messages, err := client.Messages().List(ctx, conv.ID)
				if err == nil && len(messages) > 0 {
					// Get the last incoming message
					for i := len(messages) - 1; i >= 0; i-- {
						if messages[i].MessageType == api.MessageTypeIncoming {
							item.LastMessage = truncateString(normalizeMessagePreview(messages[i].Content), 100)
							break
						}
					}
					// Fallback to last message if no incoming found
					if item.LastMessage == "" {
						item.LastMessage = truncateString(normalizeMessagePreview(messages[len(messages)-1].Content), 100)
					}
				}

				// Include heuristics analysis if requested
				if explain {
					var contactHistory []api.Conversation
					if conv.ContactID > 0 {
						contactHistory, _ = client.Contacts().Conversations(ctx, conv.ContactID)
					}
					item.Explanation = heuristics.AnalyzeConversation(&conv, messages, contactHistory)
				}

				items = append(items, item)
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"items": items})
			}

			// Text output
			if len(items) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No conversations needing attention")
				return nil
			}

			w := newTabWriterFromCmd(cmd)
			_, _ = fmt.Fprintln(w, "ID\tINBOX\tCONTACT\tWAITING\tUNREAD\tLAST MESSAGE")
			for _, item := range items {
				_, _ = fmt.Fprintf(w, "%d\t%d\t%s\t%s\t%d\t%s\n",
					item.ID,
					item.InboxID,
					item.ContactName,
					item.WaitTime,
					item.Unread,
					item.LastMessage,
				)
			}
			_ = w.Flush()

			return nil
		}),
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum conversations to show")
	cmd.Flags().IntVar(&inboxID, "inbox", 0, "Filter by inbox ID")
	cmd.Flags().BoolVar(&explain, "explain", false, "Include reasoning hints (agent mode)")

	return cmd
}

// formatWaitTime formats the time since the given Unix timestamp as human-readable.
func formatWaitTime(unixTimestamp int64) string {
	if unixTimestamp == 0 {
		return "unknown"
	}
	lastActivity := time.Unix(unixTimestamp, 0)
	waitDuration := time.Since(lastActivity)

	if waitDuration < time.Hour {
		return fmt.Sprintf("%dm", int(waitDuration.Minutes()))
	}
	if waitDuration < 24*time.Hour {
		hours := int(waitDuration.Hours())
		minutes := int(waitDuration.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	days := int(waitDuration.Hours()) / 24
	hours := int(waitDuration.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}

// normalizeMessagePreview normalizes a message for display as a preview.
func normalizeMessagePreview(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}
