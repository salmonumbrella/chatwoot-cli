package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
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
	cmd.AddCommand(newConversationsFollowCmd())
	cmd.AddCommand(newConversationsBulkCmd())
	cmd.AddCommand(newConversationsTriageCmd())

	return cmd
}

func newConversationsGetCmd() *cobra.Command {
	var withContext bool
	var withMessages bool
	var messageLimit int
	var suggestedActions bool
	var explain bool
	var emit string
	var light bool

	cmd := &cobra.Command{
		Use:     "get <id>",
		Aliases: []string{"g"},
		Short:   "Get conversation details",
		Long:    "Retrieve detailed information about a specific conversation",
		Example: strings.TrimSpace(`
  # Get conversation details
  cw conversations get 123

  # Get conversation as JSON
  cw conversations get 123 --output json

  # Get conversation with recent messages (agent mode)
  cw conversations get 123 --with-messages --output agent

  # Get comprehensive context (agent mode only)
  cw conversations get 123 --context --output agent

  # Get context with limited messages
  cw conversations get 123 --context --message-limit 10 --output agent

  # Get conversation with AI-suggested actions (agent mode)
  cw conversations get 123 --suggested-actions --output agent

  # Get conversation with reasoning hints (agent mode)
  cw conversations get 123 --explain --output agent

  # Get conversation using URL from browser
  cw conversations get https://app.chatwoot.com/app/accounts/1/conversations/123

  # Get the web UI URL for a conversation
  cw conversations get 461 --url
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				return err
			}

			mode, err := normalizeEmitFlag(emit)
			if err != nil {
				return err
			}
			if mode == "id" || mode == "url" {
				_, err := maybeEmit(cmd, mode, "conversation", id, nil)
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

			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, buildLightConversationGet(*conv))
			}

			// Keep agent-mode behavior intact (context enrichment, suggested actions, etc.).
			if !isAgent(cmd) {
				if emitted, err := maybeEmit(cmd, emit, "conversation", id, conv); emitted {
					return err
				}
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
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")
	flagAlias(cmd.Flags(), "with-messages", "wm")
	flagAlias(cmd.Flags(), "context", "ctx")
	flagAlias(cmd.Flags(), "message-limit", "ml")
	flagAlias(cmd.Flags(), "suggested-actions", "sa")
	flagAlias(cmd.Flags(), "explain", "exp")
	cmd.Flags().BoolVar(&light, "light", false, "Return minimal conversation payload for lookup")
	flagAlias(cmd.Flags(), "light", "li")

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
	var emit string

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"mk"},
		Short:   "Create a new conversation",
		Long:    "Create a new conversation in an inbox",
		Example: strings.TrimSpace(`
  # Create a conversation
  cw c mk -I 1 -C 123

  # Create a conversation with an initial message
  cw c mk -I 1 -C 123 -m "Hello!"

  # Create a conversation with status and assignment
  cw c mk -I 1 -C 123 -s open --aid 5 --tid 2
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

			var err error
			if status != "" {
				if status, err = validateStatus(status); err != nil {
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

			if emitted, err := maybeEmit(cmd, emit, "conversation", conv.ID, conv); emitted {
				return err
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

	cmd.Flags().IntVarP(&inboxID, "inbox-id", "I", 0, "Inbox ID (required)")
	cmd.Flags().IntVarP(&contactID, "contact-id", "C", 0, "Contact ID (required)")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Initial message content")
	cmd.Flags().StringVarP(&status, "status", "s", "", "Status (open|resolved|pending|snoozed)")
	cmd.Flags().IntVar(&assigneeID, "assignee-id", 0, "Agent ID to assign")
	cmd.Flags().IntVar(&teamID, "team-id", 0, "Team ID to assign")
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")
	flagAlias(cmd.Flags(), "contact-id", "cid")
	flagAlias(cmd.Flags(), "inbox-id", "iid")
	flagAlias(cmd.Flags(), "assignee-id", "aid")
	flagAlias(cmd.Flags(), "team-id", "tid")
	flagAlias(cmd.Flags(), "status", "st")

	return cmd
}

// filterPayloadShortcodes maps short keys to their full Chatwoot filter API equivalents.
var filterPayloadShortcodes = map[string]string{
	"ak": "attribute_key",
	"fo": "filter_operator",
	"v":  "values",
	"qo": "query_operator",
}

// filterOperatorShortcodes maps short operator names to their full equivalents.
var filterOperatorShortcodes = map[string]string{
	"eq": "equal_to",
	"ne": "not_equal_to",
	"co": "contains",
	"nc": "does_not_contain",
	"ip": "is_present",
	"np": "is_not_present",
}

// filterAttributeShortcodes maps short attribute names to their full equivalents.
var filterAttributeShortcodes = map[string]string{
	"st": "status",
	"ii": "inbox_id",
	"ai": "assignee_id",
	"ti": "team_id",
	"pr": "priority",
	"lb": "label_list",
}

// expandFilterPayload expands shortcode keys and values in a filter payload.
// It looks for the "payload" (or "pl") key containing an array of filter conditions,
// and expands shortcodes in each condition's keys and values.
func expandFilterPayload(payload map[string]any) map[string]any {
	// Resolve "pl" shortcode for the top-level key
	if arr, ok := payload["pl"]; ok {
		if _, exists := payload["payload"]; !exists {
			payload["payload"] = arr
			delete(payload, "pl")
		}
	}

	arr, ok := payload["payload"].([]any)
	if !ok {
		return payload
	}

	for i, item := range arr {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		arr[i] = expandFilterCondition(obj)
	}
	payload["payload"] = arr
	return payload
}

// expandFilterCondition expands shortcode keys and values in a single filter condition.
func expandFilterCondition(cond map[string]any) map[string]any {
	expanded := make(map[string]any, len(cond))
	for k, v := range cond {
		fullKey := k
		if fk, ok := filterPayloadShortcodes[k]; ok {
			// Only expand if the full key doesn't already exist
			if _, exists := cond[fk]; !exists {
				fullKey = fk
			}
		}

		switch fullKey {
		case "attribute_key":
			if s, ok := v.(string); ok {
				if expanded, ok := filterAttributeShortcodes[s]; ok {
					v = expanded
				}
			}
		case "filter_operator":
			if s, ok := v.(string); ok {
				if expanded, ok := filterOperatorShortcodes[s]; ok {
					v = expanded
				}
			}
		case "query_operator":
			if s, ok := v.(string); ok {
				v = strings.ToUpper(s)
			}
		}

		expanded[fullKey] = v
	}
	return expanded
}

func newConversationsFilterCmd() *cobra.Command {
	var payloadStr string
	var page int
	var all bool
	var maxPages int
	var folderID int
	var light bool

	cmd := &cobra.Command{
		Use:     "filter",
		Aliases: []string{"f"},
		Short:   "Filter conversations with custom query",
		Long: `Filter conversations using the Chatwoot filter API.

The payload follows the Chatwoot filter API format with an array of filter conditions.
Accepts a full object, a bare array (auto-wrapped), or stdin via --payload -.
Use --folder to load a saved custom filter by ID.

Shortcodes are supported for terser filter queries:
  Keys:       ak → attribute_key, fo → filter_operator, v → values, qo → query_operator
  Operators:  eq → equal_to, ne → not_equal_to, co → contains, nc → does_not_contain,
              ip → is_present, np → is_not_present
  Attributes: st → status, ii → inbox_id, ai → assignee_id, ti → team_id,
              pr → priority, lb → label_list

See: https://developers.chatwoot.com/api-reference/conversations/conversations-filter`,
		Example: strings.TrimSpace(`
  # Filter by multiple statuses (open OR pending OR snoozed)
  cw conversations filter --payload '{"payload":[{"attribute_key":"status","filter_operator":"equal_to","values":["open","pending","snoozed"]}]}'

  # Same filter using shortcodes
  cw c f --pl '[{"ak":"st","fo":"eq","v":["open","pending","snoozed"]}]'

  # Filter by inbox
  cw conversations filter --payload '{"payload":[{"attribute_key":"inbox_id","filter_operator":"equal_to","values":[1]}]}'

  # Same filter using shortcodes
  cw c f --pl '[{"ak":"ii","fo":"eq","v":[1]}]'

  # Combine filters with AND (shortcodes)
  cw c f --pl '[{"ak":"st","fo":"eq","v":["open"],"qo":"and"},{"ak":"ii","fo":"eq","v":[1]}]'

  # Payload as bare array (auto-wrapped)
  cw conversations filter --payload '[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]}]'

  # Payload from stdin
  echo '{"payload":[...]}' | cw conversations filter --payload -

  # Use a saved custom filter/folder
  cw conversations filter --folder 1
  cw conversations filter --folder 1 --all

  # Filter with pagination
  cw conversations filter --payload '...' --page 2

  # Fetch all pages
  cw conversations filter --payload '...' --all

  # Filter operators: equal_to, not_equal_to, contains, does_not_contain
  # Shortcodes:       eq,       ne,           co,       nc
`),
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			if folderID > 0 && payloadStr != "" {
				return fmt.Errorf("cannot use both --folder and --payload")
			}
			if folderID == 0 && payloadStr == "" {
				return fmt.Errorf("--payload or --folder is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			var payload map[string]any
			if folderID > 0 {
				filter, ferr := client.CustomFilters().Get(cmdContext(cmd), folderID)
				if ferr != nil {
					return fmt.Errorf("failed to get custom filter %d: %w", folderID, ferr)
				}
				payload = filter.Query
			} else {
				if payloadStr == "-" {
					data, rerr := io.ReadAll(os.Stdin)
					if rerr != nil {
						return fmt.Errorf("failed to read payload from stdin: %w", rerr)
					}
					payloadStr = string(data)
				}

				if err = json.Unmarshal([]byte(payloadStr), &payload); err != nil {
					// Try parsing as array and wrapping
					var arr []any
					if arrErr := json.Unmarshal([]byte(payloadStr), &arr); arrErr != nil {
						return fmt.Errorf("invalid JSON payload: %w", err)
					}
					payload = map[string]any{"payload": arr}
				}
			}

			payload = expandFilterPayload(payload)

			if all {
				return filterAllConversations(cmd, client, payload, maxPages, light)
			}

			result, err := client.Conversations().Filter(cmdContext(cmd), payload, page)
			if err != nil {
				return fmt.Errorf("failed to filter conversations: %w", err)
			}
			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, buildLightConversationLookups(result.Data.Payload))
			}

			if isAgent(cmd) {
				summaries := agentfmt.ConversationSummaries(result.Data.Payload)
				summaries = resolveConversationSummaries(cmdContext(cmd), client, summaries)
				envelope := agentfmt.ListEnvelope{
					Kind:  agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Items: summaries,
					Meta: map[string]any{
						"total_items": len(summaries),
						"page":        page,
						"meta":        result.Data.Meta,
					},
				}
				return printJSON(cmd, envelope)
			}
			if isJSON(cmd) {
				return printJSON(cmd, result.Data.Payload)
			}

			printConversationsTable(cmd.OutOrStdout(), result.Data.Payload)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nPage %d (%d conversations)\n", page, len(result.Data.Payload))

			return nil
		}),
	}

	cmd.Flags().StringVar(&payloadStr, "payload", "", "JSON payload for filtering (object, array, or - for stdin)")
	cmd.Flags().IntVar(&folderID, "folder", 0, "Custom filter/folder ID to use as filter query")
	cmd.Flags().IntVarP(&page, "page", "p", 1, "Page number")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Fetch all pages")
	cmd.Flags().IntVarP(&maxPages, "max-pages", "M", 100, "Maximum pages to fetch with --all")
	cmd.Flags().BoolVar(&light, "light", false, "Return minimal conversation payload for lookup")
	flagAlias(cmd.Flags(), "payload", "pl")
	flagAlias(cmd.Flags(), "folder", "view")
	flagAlias(cmd.Flags(), "max-pages", "mp")
	flagAlias(cmd.Flags(), "light", "li")

	return cmd
}

func filterAllConversations(cmd *cobra.Command, client *api.Client, payload map[string]any, maxPages int, light bool) error {
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

		result, err := client.Conversations().Filter(cmdContext(cmd), payload, currentPage)
		if err != nil {
			return fmt.Errorf("failed to filter conversations: %w", err)
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

	if light {
		cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
		return printRawJSON(cmd, buildLightConversationLookups(allConversations))
	}

	if isAgent(cmd) {
		summaries := agentfmt.ConversationSummaries(allConversations)
		summaries = resolveConversationSummaries(cmdContext(cmd), client, summaries)
		envelope := agentfmt.ListEnvelope{
			Kind:  agentfmt.KindFromCommandPath(cmd.CommandPath()),
			Items: summaries,
			Meta: map[string]any{
				"pages_fetched": pagesFetched,
				"total_items":   len(allConversations),
				"all":           true,
			},
		}
		return printJSON(cmd, envelope)
	}
	if isJSON(cmd) {
		return printJSON(cmd, allConversations)
	}

	if len(allConversations) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No conversations found matching your filter")
		return nil
	}

	printConversationsTable(cmd.OutOrStdout(), allConversations)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nTotal: %d conversations (%d pages)\n", len(allConversations), pagesFetched)
	return nil
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
  cw conversations meta
`),
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			normalizedStatus, err := validateStatusWithAll(status)
			if err != nil {
				return err
			}

			params := api.ListConversationsParams{
				Status:  normalizedStatus,
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

	cmd.Flags().StringVarP(&status, "status", "s", "all", "Filter by status (open|resolved|pending|snoozed|all)")
	cmd.Flags().StringVarP(&inboxID, "inbox-id", "I", "", "Filter by inbox ID")
	cmd.Flags().IntVar(&teamID, "team-id", 0, "Filter by team ID")
	cmd.Flags().StringVar(&labels, "labels", "", "Filter by labels (comma-separated)")
	cmd.Flags().StringVar(&search, "search", "", "Filter by search query")
	flagAlias(cmd.Flags(), "status", "st")
	flagAlias(cmd.Flags(), "inbox-id", "iid")
	flagAlias(cmd.Flags(), "team-id", "tid")
	flagAlias(cmd.Flags(), "labels", "lb")
	flagAlias(cmd.Flags(), "search", "sq")

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
  cw conversations counts

  # Get counts as JSON
  cw conversations counts --output json
`),
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			normalizedStatus, err := validateStatusWithAll(status)
			if err != nil {
				return err
			}

			params := api.ListConversationsParams{
				Status:  normalizedStatus,
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

	cmd.Flags().StringVarP(&status, "status", "s", "all", "Filter by status (open|resolved|pending|snoozed|all)")
	cmd.Flags().StringVarP(&inboxID, "inbox-id", "I", "", "Filter by inbox ID")
	cmd.Flags().IntVar(&teamID, "team-id", 0, "Filter by team ID")
	cmd.Flags().StringVar(&labels, "labels", "", "Filter by labels (comma-separated)")
	cmd.Flags().StringVar(&search, "search", "", "Filter by search query")
	flagAlias(cmd.Flags(), "status", "st")
	flagAlias(cmd.Flags(), "inbox-id", "iid")
	flagAlias(cmd.Flags(), "team-id", "tid")
	flagAlias(cmd.Flags(), "labels", "lb")
	flagAlias(cmd.Flags(), "search", "sq")

	return cmd
}

func newConversationsToggleStatusCmd() *cobra.Command {
	var status string
	var snoozedUntilStr string
	var light bool

	cmd := &cobra.Command{
		Use:     "toggle-status <id>",
		Aliases: []string{"ts"},
		Short:   "Toggle conversation status",
		Long:    "Change the status of a conversation",
		Example: strings.TrimSpace(`
  # Mark conversation as resolved
  cw conversations toggle-status 123 --status resolved

  # Reopen a conversation
  cw conversations toggle-status 123 --status open

  # Snooze until next customer reply (default behavior)
  cw conversations toggle-status 123 --status snoozed

  # Snooze until specific time (RFC3339)
  cw conversations toggle-status 123 --status snoozed --snoozed-until "2025-01-15T10:00:00Z"

  # Snooze until specific time (Unix timestamp)
  cw conversations toggle-status 123 --status snoozed --snoozed-until 1735689600
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

			if status, err = validateStatus(status); err != nil {
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

			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, buildLightToggleStatusResult(result.Payload.ConversationID, result.Payload.CurrentStatus, result.Payload.SnoozedUntil))
			}

			if isAgent(cmd) {
				if !flagOrAliasChanged(cmd, "compact-json") {
					cmd.SetContext(outfmt.WithCompact(cmd.Context(), true))
				}
				item := map[string]any{
					"id": result.Payload.ConversationID,
					"ok": result.Payload.Success,
					"st": shortStatus(result.Payload.CurrentStatus),
				}
				if result.Payload.SnoozedUntil != nil && *result.Payload.SnoozedUntil > 0 {
					item["su"] = *result.Payload.SnoozedUntil
				}
				return printRawJSON(cmd, item)
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

	cmd.Flags().StringVarP(&status, "status", "s", "", "New status (open|resolved|pending|snoozed) (required)")
	cmd.Flags().StringVar(&snoozedUntilStr, "snoozed-until", "", "Snooze until time (Unix timestamp, RFC3339, or relative)")
	cmd.Flags().BoolVar(&light, "light", false, "Return minimal mutation payload")
	flagAlias(cmd.Flags(), "status", "st")
	flagAlias(cmd.Flags(), "snoozed-until", "su")
	flagAlias(cmd.Flags(), "light", "li")
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
  cw conversations resolve 123

  # Resolve multiple conversations (space-separated)
  cw conversations resolve 123 456 789

  # Resolve multiple conversations (comma-separated)
  cw conversations resolve 123,456,789

  # Resolve with JSON output
  cw conversations resolve 123 456 --output json
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
	flagAlias(cmd.Flags(), "concurrency", "cc")
	flagAlias(cmd.Flags(), "progress", "prg")
	flagAlias(cmd.Flags(), "no-progress", "npr")

	return cmd
}

func newConversationsTogglePriorityCmd() *cobra.Command {
	var priority string
	var light bool

	cmd := &cobra.Command{
		Use:     "toggle-priority <id>",
		Aliases: []string{"tp"},
		Short:   "Toggle conversation priority",
		Long:    "Change the priority of a conversation",
		Example: strings.TrimSpace(`
  # Set conversation priority to urgent
  cw conversations toggle-priority 123 --priority urgent

  # Set conversation priority to low
  cw conversations toggle-priority 123 --priority low
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

			if priority, err = validatePriority(priority); err != nil {
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

			priorityValue := "none"
			if conv.Priority != nil {
				priorityValue = *conv.Priority
			}
			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, buildLightTogglePriorityResult(conv.ID, priorityValue))
			}

			if isAgent(cmd) {
				if !flagOrAliasChanged(cmd, "compact-json") {
					cmd.SetContext(outfmt.WithCompact(cmd.Context(), true))
				}
				item := map[string]any{
					"id":  conv.ID,
					"pri": shortPriority(priorityValue),
				}
				if status := shortStatus(conv.Status); status != "" {
					item["st"] = status
				}
				if conv.InboxID > 0 {
					item["ib"] = conv.InboxID
				}
				if conv.Unread > 0 {
					item["ur"] = conv.Unread
				}
				return printRawJSON(cmd, item)
			}

			if isJSON(cmd) {
				return printJSON(cmd, conv)
			}

			displayID := conv.ID
			if conv.DisplayID != nil {
				displayID = *conv.DisplayID
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Conversation #%d priority updated to: %s\n", displayID, priorityValue)

			return nil
		}),
	}

	cmd.Flags().StringVar(&priority, "priority", "", "New priority (urgent|high|medium|low|none) (required)")
	cmd.Flags().BoolVar(&light, "light", false, "Return minimal mutation payload")
	flagAlias(cmd.Flags(), "priority", "pri")
	flagAlias(cmd.Flags(), "light", "li")
	registerStaticCompletions(cmd, "priority", []string{"urgent", "high", "medium", "low", "none"})

	return cmd
}

func newConversationsUpdateCmd() *cobra.Command {
	var priority string
	var slaPolicyID int
	var emit string

	cmd := &cobra.Command{
		Use:     "update <id>",
		Aliases: []string{"up"},
		Short:   "Update conversation attributes",
		Long:    "Update conversation attributes such as priority and SLA policy",
		Example: strings.TrimSpace(`
  # Update conversation priority
  cw conversations update 123 --priority high

  # Assign SLA policy (Enterprise feature)
  cw conversations update 123 --sla-policy-id 5

  # Update both priority and SLA policy
  cw conversations update 123 --priority urgent --sla-policy-id 5
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
				if priority, err = validatePriority(priority); err != nil {
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

			if emitted, err := maybeEmit(cmd, emit, "conversation", conv.ID, conv); emitted {
				return err
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
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit: json|id|url (overrides normal text output)")
	flagAlias(cmd.Flags(), "priority", "pri")
	flagAlias(cmd.Flags(), "sla-policy-id", "sla")
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
		light       bool
	)

	cmd := &cobra.Command{
		Use:   "assign <id> [id...]",
		Short: "Assign conversations to agent or team",
		Long:  "Assign one or more conversations to an agent and/or team",
		Example: strings.TrimSpace(`
  # Assign to agent
  cw conversations assign 123 --agent 5

  # Assign to team
  cw conversations assign 123 --team 2

  # Assign to both agent and team
  cw conversations assign 123 --agent 5 --team 2

  # Assign multiple conversations (space-separated)
  cw conversations assign 123 456 789 --agent 5

  # Assign multiple conversations (comma-separated)
  cw conversations assign 123,456,789 --agent 5

  # Assign with JSON output
  cw conversations assign 123 456 --agent 5 --output json
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

			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, buildLightBulkAssignResult(results, successCount, failCount, agentID, resolvedTeamID))
			}

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
	cmd.Flags().BoolVar(&light, "light", false, "Return minimal mutation payload")
	flagAlias(cmd.Flags(), "agent", "ag")
	flagAlias(cmd.Flags(), "team", "tm")
	flagAlias(cmd.Flags(), "concurrency", "cc")
	flagAlias(cmd.Flags(), "progress", "prg")
	flagAlias(cmd.Flags(), "no-progress", "npr")
	flagAlias(cmd.Flags(), "light", "li")

	return cmd
}

func newConversationsLabelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "labels <id>",
		Short: "Get conversation labels",
		Long:  "Retrieve labels for a specific conversation",
		Example: strings.TrimSpace(`
  # Get labels for a conversation
  cw conversations labels 123

  # JSON output - returns an object with an "items" array
  cw conversations labels 123 --output json
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
		Use:     "labels-add <id>",
		Aliases: []string{"la"},
		Short:   "Add labels to conversation",
		Long:    "Add one or more labels to a conversation",
		Example: strings.TrimSpace(`
  # Add labels to a conversation
  cw conversations labels-add 123 --labels "bug,urgent"
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

			labels, err := ParseStringListFlag(labelsStr)
			if err != nil {
				return fmt.Errorf("invalid labels: %w", err)
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

	cmd.Flags().StringVar(&labelsStr, "labels", "", "Labels (CSV, whitespace, JSON array; or @- / @path) (required)")
	flagAlias(cmd.Flags(), "labels", "lb")

	return cmd
}

func newConversationsLabelsRemoveCmd() *cobra.Command {
	var labelsStr string

	cmd := &cobra.Command{
		Use:     "labels-remove <id>",
		Aliases: []string{"lr"},
		Short:   "Remove labels from conversation",
		Long:    "Remove one or more labels from a conversation",
		Example: strings.TrimSpace(`
  # Remove labels from a conversation
  cw conversations labels-remove 123 --labels "bug,urgent"
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

			labelsToRemove, err := ParseStringListFlag(labelsStr)
			if err != nil {
				return fmt.Errorf("invalid labels: %w", err)
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

	cmd.Flags().StringVar(&labelsStr, "labels", "", "Labels to remove (CSV, whitespace, JSON array; or @- / @path) (required)")
	flagAlias(cmd.Flags(), "labels", "lb")

	return cmd
}

func newConversationsCustomAttributesCmd() *cobra.Command {
	var setAttrs []string

	cmd := &cobra.Command{
		Use:     "custom-attributes <id>",
		Aliases: []string{"ca"},
		Short:   "Update conversation custom attributes",
		Long:    "Update custom attributes for a conversation",
		Example: strings.TrimSpace(`
  # Set custom attributes
  cw conversations custom-attributes 123 --set priority=high --set source=web

  # JSON output - returns attributes object directly
  cw conversations custom-attributes 123 --set priority=high --output json
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
	flagAlias(cmd.Flags(), "set", "kv")

	return cmd
}

func newConversationsContextCmd() *cobra.Command {
	var embedImages bool
	var light bool

	cmd := &cobra.Command{
		Use:   "context <id>",
		Short: "Get full conversation context for AI",
		Long: `Get complete conversation context optimized for AI consumption.

Includes conversation metadata, contact info, all messages, and optionally
embeds images as base64 data URIs that AI vision models can consume directly.`,
		Example: strings.TrimSpace(`
  # Get conversation context
  cw conversations context 123

  # Use conversation URL from browser
  cw conversations context https://app.chatwoot.com/app/accounts/1/conversations/123

  # Get context with embedded images (for AI vision)
  cw conversations context 123 --embed-images

  # Get lightweight context (id/status/inbox/contact/messages only)
  cw conversations context 123 --light --compact-json

  # Pipe to AI for draft response
  cw conversations context 123 --embed-images --output json | ai-tool
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

			requestEmbeddedImages := embedImages && !light
			ctx, err := client.Context().GetConversation(cmdContext(cmd), id, requestEmbeddedImages)
			if err != nil {
				return fmt.Errorf("failed to get conversation context: %w", err)
			}

			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, buildLightConversationContext(id, ctx))
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
	cmd.Flags().BoolVar(&light, "light", false, "Return minimal context payload (id, st, inbox, contact, msgs)")
	flagAlias(cmd.Flags(), "embed-images", "embed")
	flagAlias(cmd.Flags(), "embed-images", "em")
	flagAlias(cmd.Flags(), "light", "li")

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

func contextInboxSummaries(contactInboxes []api.ContactInbox) []contextInboxSummary {
	if len(contactInboxes) == 0 {
		return nil
	}
	out := make([]contextInboxSummary, 0, len(contactInboxes))
	for _, contactInbox := range contactInboxes {
		inbox := contactInbox.Inbox
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
		Use:     "mark-unread <id>",
		Aliases: []string{"mu"},
		Short:   "Mark conversation as unread",
		Long: `Mark a conversation as unread for all agents.

This resets the agent_last_seen_at timestamp, making the conversation appear
as unread in the inbox for all agents (not just the current user).`,
		Example: strings.TrimSpace(`
  # Mark a single conversation as unread
  cw conversations mark-unread 123

  # Mark multiple conversations as unread
  for id in 123 124 125; do cw conversations mark-unread $id; done
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
	var light bool

	cmd := &cobra.Command{
		Use:     "search <query>",
		Aliases: []string{"q"},
		Short:   "Search conversations by message content",
		Long:    "Search conversations by message content across all conversations",
		Example: strings.TrimSpace(`
  # Search for conversations mentioning "password reset"
  cw conversations search "password reset"

  # Search with pagination
  cw conversations search "refund" --page 2

  # Fetch all matching conversations
  cw conversations search "error" --all

  # JSON output
  cw conversations search "billing" -o json
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
				return searchAllConversations(cmd, client, query, maxPages, light)
			}

			result, err := client.Conversations().Search(cmdContext(cmd), query, page)
			if err != nil {
				return fmt.Errorf("failed to search conversations: %w", err)
			}

			conversations := result.Data.Payload
			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, buildLightConversationLookups(conversations))
			}

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
	cmd.Flags().IntVarP(&maxPages, "max-pages", "M", 100, "Maximum pages to fetch with --all")
	cmd.Flags().BoolVar(&light, "light", false, "Return minimal conversation payload for lookup")
	flagAlias(cmd.Flags(), "max-pages", "mp")
	flagAlias(cmd.Flags(), "light", "li")

	return cmd
}

func searchAllConversations(cmd *cobra.Command, client *api.Client, query string, maxPages int, light bool) error {
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

	if light {
		cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
		return printRawJSON(cmd, buildLightConversationLookups(allConversations))
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
  cw conversations attachments 123

  # JSON output with URLs
  cw conversations attachments 123 -o json
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
  cw conversations mute 123

  # Mute and output as JSON
  cw conversations mute 123 --output json
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
  cw conversations unmute 123

  # Unmute and output as JSON
  cw conversations unmute 123 --output json
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
		Use:     "transcript <id>",
		Aliases: []string{"tr"},
		Short:   "Render or send a conversation transcript",
		Long: `Render a conversation transcript to stdout, or send it via email.

When --email is provided, the transcript is sent via Chatwoot and no
message content is printed. Without --email, the transcript is rendered
locally with private notes included by default.`,
		Example: strings.TrimSpace(`
  # Render transcript to stdout (includes private notes)
  cw conversations transcript 123

  # Render public-only transcript
  cw conversations transcript 123 --public-only

  # Limit to the most recent messages
  cw conversations transcript 123 --limit 200

  # Send transcript to an email address
  cw conversations transcript 123 --email user@example.com
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
	cmd.Flags().IntVarP(&maxPages, "max-pages", "M", 100, "Maximum pages to fetch when listing messages")
	cmd.Flags().BoolVar(&publicOnly, "public-only", false, "Exclude private notes from the transcript")
	flagAlias(cmd.Flags(), "max-pages", "mp")
	flagAlias(cmd.Flags(), "public-only", "pub")
	flagAlias(cmd.Flags(), "email", "em")
	flagAlias(cmd.Flags(), "limit", "lt")

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
  cw conversations typing 123 --on

  # Hide typing indicator
  cw conversations typing 123

  # Show typing indicator for private note (visible only to agents)
  cw conversations typing 123 --on --private
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
	cmd.Flags().BoolVarP(&isPrivate, "private", "P", false, "Show typing indicator only to agents (for private notes)")

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
