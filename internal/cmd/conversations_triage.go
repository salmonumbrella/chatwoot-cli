package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/heuristics"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

// triageItem represents a conversation needing attention with context.
type triageItem struct {
	ID          int                  `json:"id"`
	InboxID     int                  `json:"inbox_id"`
	Status      string               `json:"status"`
	Unread      int                  `json:"unread"`
	WaitTime    string               `json:"wait_time"`
	LastMessage string               `json:"last_message"`
	ContactName string               `json:"contact_name"`
	Explanation *heuristics.Analysis `json:"_explanation,omitempty"`
}

type lightTriageItem struct {
	ID          int                  `json:"id"`
	Status      string               `json:"st,omitempty"`
	InboxID     *int                 `json:"ib,omitempty"`
	Unread      int                  `json:"ur"`
	LastMessage string               `json:"lm,omitempty"`
	ContactName string               `json:"cnm,omitempty"`
	Explanation *heuristics.Analysis `json:"_exp,omitempty"`
}

func buildLightTriageItems(items []triageItem, includeInbox bool) []lightTriageItem {
	if len(items) == 0 {
		return []lightTriageItem{}
	}

	out := make([]lightTriageItem, 0, len(items))
	for _, item := range items {
		light := lightTriageItem{
			ID:          item.ID,
			Status:      shortStatus(item.Status),
			Unread:      item.Unread,
			LastMessage: item.LastMessage,
			ContactName: item.ContactName,
		}
		if includeInbox {
			inboxID := item.InboxID
			light.InboxID = &inboxID
		}
		if item.Explanation != nil {
			light.Explanation = item.Explanation
		}
		out = append(out, light)
	}
	return out
}

func newConversationsTriageCmd() *cobra.Command {
	var limit int
	var inboxID int
	var explain bool
	var brief bool
	var light bool

	cmd := &cobra.Command{
		Use:     "triage",
		Aliases: []string{"tri"},
		Short:   "Show conversations needing attention, sorted by urgency",
		Long: `Show conversations needing attention, sorted by urgency.

Fetches open and pending conversations with unread messages, sorted by how long
the customer has been waiting (oldest first = longest waiting = most urgent).`,
		Example: strings.TrimSpace(`
  # Show top 20 conversations needing attention
  cw conversations triage

  # Show top 10 with detailed reasoning
  cw conversations triage --limit 10 --explain

  # Filter by inbox
  cw conversations triage --inbox 5

  # JSON output for agent processing
  cw conversations triage --output json
`),
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)
			statuses := []string{"open", "pending"}
			seen := make(map[int]struct{})
			candidates := make([]api.Conversation, 0)

			for _, status := range statuses {
				params := api.ListConversationsParams{Status: status}
				if inboxID > 0 {
					params.InboxID = strconv.Itoa(inboxID)
				}

				result, err := client.Conversations().List(ctx, params)
				if err != nil {
					return fmt.Errorf("failed to list %s conversations: %w", status, err)
				}

				for _, conv := range result.Data.Payload {
					if conv.Unread <= 0 {
						continue
					}
					if _, exists := seen[conv.ID]; exists {
						continue
					}
					seen[conv.ID] = struct{}{}
					candidates = append(candidates, conv)
				}
			}

			// Sort by LastActivityAt ascending (oldest first = longest waiting).
			sort.Slice(candidates, func(i, j int) bool {
				return candidates[i].LastActivityAt < candidates[j].LastActivityAt
			})

			if limit > 0 && len(candidates) > limit {
				candidates = candidates[:limit]
			}

			items := make([]triageItem, 0, len(candidates))
			for _, conv := range candidates {
				item := triageItem{
					ID:       conv.ID,
					InboxID:  conv.InboxID,
					Status:   conv.Status,
					Unread:   conv.Unread,
					WaitTime: formatWaitTime(conv.LastActivityAt),
				}

				item.ContactName = getSenderNameFromMeta(conv.Meta)
				if item.ContactName == "" {
					item.ContactName = fmt.Sprintf("Contact #%d", conv.ContactID)
				}

				var messages []api.Message
				if brief {
					item.LastMessage = truncateString(normalizeMessagePreview(extractLastNonActivityMessage(conv)), 100)
				} else {
					var msgErr error
					messages, msgErr = client.Messages().List(ctx, conv.ID)
					if msgErr == nil && len(messages) > 0 {
						for i := len(messages) - 1; i >= 0; i-- {
							if messages[i].MessageType == api.MessageTypeIncoming {
								item.LastMessage = truncateString(normalizeMessagePreview(messages[i].Content), 100)
								break
							}
						}
						if item.LastMessage == "" {
							item.LastMessage = truncateString(normalizeMessagePreview(messages[len(messages)-1].Content), 100)
						}
					}
				}

				if explain && !brief {
					var contactHistory []api.Conversation
					if conv.ContactID > 0 {
						contactHistory, _ = client.Contacts().Conversations(ctx, conv.ContactID)
					}
					item.Explanation = heuristics.AnalyzeConversation(&conv, messages, contactHistory)
				}

				items = append(items, item)
			}

			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, map[string]any{
					"items": buildLightTriageItems(items, inboxID <= 0),
				})
			}

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{"items": items})
			}

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
	cmd.Flags().BoolVar(&brief, "brief", false, "Skip per-conversation API calls; use last_non_activity_message from list response")
	cmd.Flags().BoolVar(&light, "light", false, "Return minimal triage payload for lookup")
	flagAlias(cmd.Flags(), "inbox", "ib")
	flagAlias(cmd.Flags(), "explain", "exp")
	flagAlias(cmd.Flags(), "limit", "lt")
	flagAlias(cmd.Flags(), "brief", "br")
	flagAlias(cmd.Flags(), "light", "li")

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
