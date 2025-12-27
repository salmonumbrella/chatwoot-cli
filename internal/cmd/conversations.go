package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
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
	cmd.AddCommand(newConversationsTogglePriorityCmd())
	cmd.AddCommand(newConversationsUpdateCmd())
	cmd.AddCommand(newConversationsAssignCmd())
	cmd.AddCommand(newConversationsLabelsCmd())
	cmd.AddCommand(newConversationsLabelsAddCmd())
	cmd.AddCommand(newConversationsCustomAttributesCmd())
	cmd.AddCommand(newConversationsContextCmd())
	cmd.AddCommand(newConversationsMarkUnreadCmd())
	cmd.AddCommand(newConversationsMuteCmd())
	cmd.AddCommand(newConversationsUnmuteCmd())
	cmd.AddCommand(newConversationsSearchCmd())
	cmd.AddCommand(newConversationsAttachmentsCmd())

	return cmd
}

func printConversationsTable(conversations []api.Conversation) {
	w := newTabWriter()
	_, _ = fmt.Fprintln(w, "ID\tINBOX\tSTATUS\tPRIORITY\tUNREAD\tCREATED")
	for _, conv := range conversations {
		priority := "-"
		if conv.Priority != nil {
			priority = *conv.Priority
		}
		displayID := conv.ID
		if conv.DisplayID != nil {
			displayID = *conv.DisplayID
		}
		_, _ = fmt.Fprintf(w, "%d\t%d\t%s\t%s\t%d\t%s\n",
			displayID,
			conv.InboxID,
			conv.Status,
			priority,
			conv.Unread,
			conv.CreatedAtTime().Format("2006-01-02 15:04"),
		)
	}
	_ = w.Flush()
}

func newConversationsListCmd() *cobra.Command {
	var inboxID string
	var status string
	var page int
	var all bool
	var maxPages int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List conversations",
		Long:  "List conversations filtered by status and inbox",
		Example: strings.TrimSpace(`
  # List open conversations
  chatwoot conversations list --status open

  # JSON output - returns array directly
  chatwoot conversations list --output json | jq '.[0]'

  # Fetch all pages
  chatwoot conversations list --status open --all
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if page < 1 {
				return fmt.Errorf("page must be >= 1")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			var allConversations []api.Conversation
			currentPage := page
			totalFetched := 0
			pagesFetched := 0

			for {
				if all && pagesFetched >= maxPages {
					return fmt.Errorf("safety limit reached: fetched %d pages (%d conversations). Use --max-pages to increase the limit", maxPages, totalFetched)
				}

				// Show progress indicator when fetching multiple pages
				if all && currentPage > page {
					fmt.Fprintf(os.Stderr, "Fetching page %d...\n", currentPage)
				}

				result, err := client.ListConversations(cmdContext(cmd), status, inboxID, currentPage)
				if err != nil {
					return fmt.Errorf("failed to list conversations: %w", err)
				}

				conversations := result.Data.Payload
				totalPages := int(result.Data.Meta.TotalPages)

				// Stop if no results or if we've reached the total pages from API
				if len(conversations) == 0 || (totalPages > 0 && currentPage >= totalPages) {
					break
				}

				allConversations = append(allConversations, conversations...)
				totalFetched += len(conversations)

				if !all {
					// Single page mode - output and exit
					if isJSON(cmd) {
						// Return array directly for easier jq processing
						return printJSON(conversations)
					}

					printConversationsTable(conversations)

					fmt.Printf("\nPage %d (%d conversations)\n", currentPage, len(conversations))
					return nil
				}

				currentPage++
				pagesFetched++
			}

			// --all mode: output all fetched conversations
			if isJSON(cmd) {
				// Return array directly for easier jq processing
				return printJSON(allConversations)
			}

			printConversationsTable(allConversations)

			fmt.Printf("\nTotal: %d conversations (%d pages)\n", totalFetched, pagesFetched)
			return nil
		},
	}

	cmd.Flags().StringVar(&inboxID, "inbox-id", "", "Filter by inbox ID")
	cmd.Flags().StringVar(&status, "status", "all", "Filter by status (open|resolved|pending|snoozed|all)")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages")
	cmd.Flags().IntVar(&maxPages, "max-pages", 100, "Maximum number of pages to fetch when using --all")

	return cmd
}

func newConversationsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get conversation details",
		Long:  "Retrieve detailed information about a specific conversation",
		Example: strings.TrimSpace(`
  # Get conversation details
  chatwoot conversations get 123

  # Get conversation as JSON
  chatwoot conversations get 123 --output json
`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "conversation ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			conv, err := client.GetConversation(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get conversation %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(conv)
			}

			displayID := conv.ID
			if conv.DisplayID != nil {
				displayID = *conv.DisplayID
			}
			fmt.Printf("Conversation #%d\n", displayID)
			fmt.Printf("  ID:         %d\n", conv.ID)
			fmt.Printf("  Inbox ID:   %d\n", conv.InboxID)
			fmt.Printf("  Contact ID: %d\n", conv.ContactID)
			fmt.Printf("  Status:     %s\n", conv.Status)
			if conv.Priority != nil {
				fmt.Printf("  Priority:   %s\n", *conv.Priority)
			}
			if conv.AssigneeID != nil {
				fmt.Printf("  Assignee:   %d\n", *conv.AssigneeID)
			}
			if conv.TeamID != nil {
				fmt.Printf("  Team:       %d\n", *conv.TeamID)
			}
			fmt.Printf("  Unread:     %d\n", conv.Unread)
			fmt.Printf("  Muted:      %t\n", conv.Muted)
			fmt.Printf("  Created:    %s\n", conv.CreatedAtTime().Format("2006-01-02 15:04:05"))
			if len(conv.Labels) > 0 {
				fmt.Printf("  Labels:     %s\n", strings.Join(conv.Labels, ", "))
			}

			return nil
		},
	}

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
		RunE: func(cmd *cobra.Command, _ []string) error {
			if inboxID == 0 {
				return fmt.Errorf("--inbox-id is required")
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

			conv, err := client.CreateConversation(cmdContext(cmd), req)
			if err != nil {
				return fmt.Errorf("failed to create conversation: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(conv)
			}

			displayID := conv.ID
			if conv.DisplayID != nil {
				displayID = *conv.DisplayID
			}
			fmt.Printf("Created conversation #%d\n", displayID)
			fmt.Printf("  ID:     %d\n", conv.ID)
			fmt.Printf("  Status: %s\n", conv.Status)
			if conv.AssigneeID != nil {
				fmt.Printf("  Assignee: %d\n", *conv.AssigneeID)
			}
			if conv.TeamID != nil {
				fmt.Printf("  Team: %d\n", *conv.TeamID)
			}

			return nil
		},
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
		RunE: func(cmd *cobra.Command, _ []string) error {
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

			result, err := client.FilterConversations(cmdContext(cmd), payload)
			if err != nil {
				return fmt.Errorf("failed to filter conversations: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(result.Data.Payload)
			}

			printConversationsTable(result.Data.Payload)

			return nil
		},
	}

	cmd.Flags().StringVar(&payloadStr, "payload", "", "JSON payload for filtering (required)")

	return cmd
}

func newConversationsMetaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "meta",
		Short: "Get conversations metadata",
		Long:  "Retrieve metadata about conversations (counts by status, etc.)",
		Example: strings.TrimSpace(`
  # Get conversations metadata
  chatwoot conversations meta
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			meta, err := client.GetConversationsMeta(cmdContext(cmd))
			if err != nil {
				return fmt.Errorf("failed to get conversations metadata: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(meta)
			}

			fmt.Println("Conversations Metadata:")
			for key, value := range meta {
				fmt.Printf("  %s: %v\n", key, value)
			}

			return nil
		},
	}

	return cmd
}

func newConversationsCountsCmd() *cobra.Command {
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			meta, err := client.GetConversationsMeta(cmdContext(cmd))
			if err != nil {
				return fmt.Errorf("failed to get conversation counts: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(meta)
			}

			// Extract counts from nested meta object
			counts, ok := meta["meta"].(map[string]any)
			if !ok {
				return fmt.Errorf("unexpected response format")
			}

			w := newTabWriter()
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
		},
	}

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
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "conversation ID")
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

			result, err := client.ToggleConversationStatus(cmdContext(cmd), id, status, snoozedUntil)
			if err != nil {
				return fmt.Errorf("failed to toggle status for conversation %d: %w", id, err)
			}

			if isJSON(cmd) {
				// Return payload directly for consistency
				return printJSON(result.Payload)
			}

			fmt.Printf("Conversation #%d status updated to: %s\n", result.Payload.ConversationID, result.Payload.CurrentStatus)
			if result.Payload.SnoozedUntil != nil && *result.Payload.SnoozedUntil > 0 {
				snoozedTime := time.Unix(*result.Payload.SnoozedUntil, 0)
				fmt.Printf("Snoozed until: %s\n", snoozedTime.Format("2006-01-02 15:04:05 MST"))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&status, "status", "", "New status (open|resolved|pending|snoozed) (required)")
	cmd.Flags().StringVar(&snoozedUntilStr, "snoozed-until", "", "Snooze until time (Unix timestamp or RFC3339 format)")

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
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "conversation ID")
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

			if err := client.ToggleConversationPriority(cmdContext(cmd), id, priority); err != nil {
				return fmt.Errorf("failed to toggle priority for conversation %d: %w", id, err)
			}

			// Fetch updated conversation since toggle_priority returns no body
			conv, err := client.GetConversation(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get conversation %d after priority update: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(conv)
			}

			displayID := conv.ID
			if conv.DisplayID != nil {
				displayID = *conv.DisplayID
			}
			priorityValue := "none"
			if conv.Priority != nil {
				priorityValue = *conv.Priority
			}
			fmt.Printf("Conversation #%d priority updated to: %s\n", displayID, priorityValue)

			return nil
		},
	}

	cmd.Flags().StringVar(&priority, "priority", "", "New priority (urgent|high|medium|low|none) (required)")

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
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "conversation ID")
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

			conv, err := client.UpdateConversation(cmdContext(cmd), id, priority, slaPolicyID)
			if err != nil {
				return fmt.Errorf("failed to update conversation %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(conv)
			}

			displayID := conv.ID
			if conv.DisplayID != nil {
				displayID = *conv.DisplayID
			}
			fmt.Printf("Conversation #%d updated\n", displayID)
			if conv.Priority != nil {
				fmt.Printf("  Priority: %s\n", *conv.Priority)
			}
			// Note: SLA policy info may not be in standard conversation response
			if slaPolicyID > 0 {
				fmt.Printf("  SLA Policy ID: %d\n", slaPolicyID)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&priority, "priority", "", "Priority (urgent|high|medium|low|none)")
	cmd.Flags().IntVar(&slaPolicyID, "sla-policy-id", 0, "SLA policy ID (Enterprise feature)")

	return cmd
}

func newConversationsAssignCmd() *cobra.Command {
	var assigneeID int
	var teamID int

	cmd := &cobra.Command{
		Use:   "assign <id>",
		Short: "Assign conversation to agent or team",
		Long:  "Assign a conversation to an agent and/or team",
		Example: strings.TrimSpace(`
  # Assign to agent
  chatwoot conversations assign 123 --assignee-id 5

  # Assign to team
  chatwoot conversations assign 123 --team-id 2

  # Assign to both agent and team
  chatwoot conversations assign 123 --assignee-id 5 --team-id 2
`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "conversation ID")
			if err != nil {
				return err
			}

			if assigneeID == 0 && teamID == 0 {
				return fmt.Errorf("at least one of --assignee-id or --team-id is required")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if _, err := client.AssignConversation(cmdContext(cmd), id, assigneeID, teamID); err != nil {
				return fmt.Errorf("failed to assign conversation %d: %w", id, err)
			}

			// Fetch updated conversation since assignments returns the agent/team, not the conversation
			conv, err := client.GetConversation(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get conversation %d after assignment: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(conv)
			}

			displayID := conv.ID
			if conv.DisplayID != nil {
				displayID = *conv.DisplayID
			}
			fmt.Printf("Conversation #%d assigned\n", displayID)
			if conv.AssigneeID != nil {
				fmt.Printf("  Agent: %d\n", *conv.AssigneeID)
			}
			if conv.TeamID != nil {
				fmt.Printf("  Team:  %d\n", *conv.TeamID)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&assigneeID, "assignee-id", 0, "Agent ID to assign")
	cmd.Flags().IntVar(&teamID, "team-id", 0, "Team ID to assign")

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

  # JSON output - returns array directly
  chatwoot conversations labels 123 --output json
`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "conversation ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			labels, err := client.GetConversationLabels(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get labels for conversation %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(labels)
			}

			if len(labels) == 0 {
				fmt.Println("No labels")
			} else {
				fmt.Println("Labels:")
				for _, label := range labels {
					fmt.Printf("  - %s\n", label)
				}
			}

			return nil
		},
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
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "conversation ID")
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

			resultLabels, err := client.AddConversationLabels(cmdContext(cmd), id, labels)
			if err != nil {
				return fmt.Errorf("failed to add labels to conversation %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(resultLabels)
			}

			fmt.Printf("Labels updated for conversation #%d\n", id)
			if len(resultLabels) > 0 {
				fmt.Println("Current labels:")
				for _, label := range resultLabels {
					fmt.Printf("  - %s\n", label)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&labelsStr, "labels", "", "Comma-separated list of labels (required)")

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
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "conversation ID")
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

			if err := client.UpdateConversationCustomAttributes(cmdContext(cmd), id, attrs); err != nil {
				return fmt.Errorf("failed to update custom attributes for conversation %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(attrs)
			}

			fmt.Printf("Custom attributes updated for conversation #%d\n", id)
			for key, value := range attrs {
				fmt.Printf("  %s = %v\n", key, value)
			}

			return nil
		},
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

  # Get context with embedded images (for AI vision)
  chatwoot conversations context 123 --embed-images

  # Pipe to AI for draft response
  chatwoot conversations context 123 --embed-images --output json | ai-tool
`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "conversation ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx, err := client.GetConversationContext(cmdContext(cmd), id, embedImages)
			if err != nil {
				return fmt.Errorf("failed to get conversation context: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(ctx)
			}

			// Human-readable output
			fmt.Printf("=== Conversation #%d ===\n", id)
			fmt.Printf("Summary: %s\n\n", ctx.Summary)

			if ctx.Contact != nil {
				fmt.Printf("Customer: %s\n", ctx.Contact.Name)
				if ctx.Contact.Email != "" {
					fmt.Printf("Email: %s\n", ctx.Contact.Email)
				}
				if ctx.Contact.PhoneNumber != "" {
					fmt.Printf("Phone: %s\n", ctx.Contact.PhoneNumber)
				}
				fmt.Println()
			}

			fmt.Println("--- Messages ---")
			for _, msg := range ctx.Messages {
				sender := "Customer"
				if msg.MessageType == 1 {
					sender = "Agent"
				}
				if msg.Private {
					sender = "Private Note"
				}

				fmt.Printf("[%s] %s\n", sender, msg.Content)

				for _, att := range msg.Attachments {
					if att.Embedded != "" {
						fmt.Printf("  📎 [%s - embedded as base64]\n", att.FileType)
					} else {
						fmt.Printf("  📎 [%s - %s]\n", att.FileType, att.DataURL)
					}
				}
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&embedImages, "embed-images", false, "Embed images as base64 data URIs for AI vision")

	return cmd
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
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "conversation ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			// Get initial state to verify change
			beforeConv, err := client.GetConversation(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get conversation %d: %w", id, err)
			}
			initialUnread := beforeConv.Unread

			if err := client.MarkConversationUnread(cmdContext(cmd), id); err != nil {
				return fmt.Errorf("failed to mark conversation %d as unread: %w", id, err)
			}

			// Fetch updated conversation to verify the operation succeeded
			afterConv, err := client.GetConversation(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get conversation %d after marking unread: %w", id, err)
			}

			// Verify the operation didn't fail silently (count should not decrease)
			// Note: The API resets agent_last_seen_at timestamp, which may not always increment Unread
			if afterConv.Unread < initialUnread {
				return fmt.Errorf("mark-unread operation appears to have failed (unread count decreased from %d to %d)", initialUnread, afterConv.Unread)
			}

			if isJSON(cmd) {
				return printJSON(afterConv)
			}

			displayID := afterConv.ID
			if afterConv.DisplayID != nil {
				displayID = *afterConv.DisplayID
			}
			fmt.Printf("Conversation #%d marked as unread (unread count: %d)\n", displayID, afterConv.Unread)
			return nil
		},
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
		RunE: func(cmd *cobra.Command, args []string) error {
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

			result, err := client.SearchConversations(cmdContext(cmd), query, page)
			if err != nil {
				return fmt.Errorf("failed to search conversations: %w", err)
			}

			conversations := result.Data.Payload

			if isJSON(cmd) {
				return printJSON(conversations)
			}

			if len(conversations) == 0 {
				fmt.Println("No conversations found matching your query")
				return nil
			}

			printConversationsTable(conversations)
			fmt.Printf("\nPage %d (%d conversations)\n", page, len(conversations))
			return nil
		},
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
			fmt.Fprintf(os.Stderr, "Fetching page %d...\n", currentPage)
		}

		result, err := client.SearchConversations(cmdContext(cmd), query, currentPage)
		if err != nil {
			return fmt.Errorf("failed to search conversations: %w", err)
		}

		conversations := result.Data.Payload
		totalPages := int(result.Data.Meta.TotalPages)

		if len(conversations) == 0 || (totalPages > 0 && currentPage >= totalPages) {
			break
		}

		allConversations = append(allConversations, conversations...)
		currentPage++
		pagesFetched++
	}

	if isJSON(cmd) {
		return printJSON(allConversations)
	}

	if len(allConversations) == 0 {
		fmt.Println("No conversations found matching your query")
		return nil
	}

	printConversationsTable(allConversations)
	fmt.Printf("\nTotal: %d conversations (%d pages)\n", len(allConversations), pagesFetched)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "conversation ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			attachments, err := client.GetConversationAttachments(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get attachments for conversation %d: %w", id, err)
			}

			if isJSON(cmd) {
				return printJSON(attachments)
			}

			if len(attachments) == 0 {
				fmt.Println("No attachments found")
				return nil
			}

			w := newTabWriter()
			_, _ = fmt.Fprintln(w, "ID\tTYPE\tSIZE\tURL")
			for _, att := range attachments {
				size := formatFileSize(att.FileSize)
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", att.ID, att.FileType, size, att.DataURL)
			}
			_ = w.Flush()

			fmt.Printf("\nTotal: %d attachments\n", len(attachments))
			return nil
		},
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
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "conversation ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.ToggleMuteConversation(cmdContext(cmd), id, true); err != nil {
				return fmt.Errorf("failed to mute conversation %d: %w", id, err)
			}

			// Fetch updated conversation to verify the operation succeeded
			conv, err := client.GetConversation(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get conversation %d after muting: %w", id, err)
			}

			// Verify the operation succeeded
			if !conv.Muted {
				return fmt.Errorf("mute operation failed: conversation is still unmuted")
			}

			if isJSON(cmd) {
				return printJSON(conv)
			}

			displayID := conv.ID
			if conv.DisplayID != nil {
				displayID = *conv.DisplayID
			}
			fmt.Printf("Conversation #%d muted (muted: %t)\n", displayID, conv.Muted)
			return nil
		},
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
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "conversation ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.ToggleMuteConversation(cmdContext(cmd), id, false); err != nil {
				return fmt.Errorf("failed to unmute conversation %d: %w", id, err)
			}

			// Fetch updated conversation to verify the operation succeeded
			conv, err := client.GetConversation(cmdContext(cmd), id)
			if err != nil {
				return fmt.Errorf("failed to get conversation %d after unmuting: %w", id, err)
			}

			// Verify the operation succeeded
			if conv.Muted {
				return fmt.Errorf("unmute operation failed: conversation is still muted")
			}

			if isJSON(cmd) {
				return printJSON(conv)
			}

			displayID := conv.ID
			if conv.DisplayID != nil {
				displayID = *conv.DisplayID
			}
			fmt.Printf("Conversation #%d unmuted (muted: %t)\n", displayID, conv.Muted)
			return nil
		},
	}

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

const maxFutureYears = 10 * 365 * 24 * 60 * 60 // 10 years in seconds

// parseSnoozedUntil parses a snoozed-until value as either Unix timestamp (seconds) or RFC3339 datetime
func parseSnoozedUntil(s string) (int64, error) {
	// Try parsing as Unix timestamp first
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		if ts <= 0 {
			return 0, fmt.Errorf("timestamp must be positive, got %d", ts)
		}
		// Validate reasonable timestamp range (not too far in past or future)
		now := time.Now().Unix()
		if ts < now {
			return 0, fmt.Errorf("timestamp %d is in the past", ts)
		}
		// Prevent absurdly far future timestamps (max 10 years from now)
		maxFuture := now + maxFutureYears
		if ts > maxFuture {
			return 0, fmt.Errorf("timestamp %d is too far in the future (max 10 years)", ts)
		}
		return ts, nil
	}

	// Try parsing as RFC3339
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0, fmt.Errorf("invalid format (use Unix timestamp or RFC3339): %w", err)
	}

	ts := t.Unix()
	now := time.Now().Unix()
	if ts < now {
		return 0, fmt.Errorf("time %q is in the past", s)
	}
	// Prevent absurdly far future timestamps (max 10 years from now)
	maxFuture := now + maxFutureYears
	if ts > maxFuture {
		return 0, fmt.Errorf("time %q is too far in the future (max 10 years)", s)
	}

	return ts, nil
}
