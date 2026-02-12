package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/config"
	"github.com/spf13/cobra"
)

func newDashboardCmd() *cobra.Command {
	var contactID int
	var conversationID int
	var page int
	var perPage int
	var noResolve bool
	var noResolveWarning bool
	var resolveWarning string

	cmd := &cobra.Command{
		Use:     "dashboard <name>",
		Aliases: []string{"dash"},
		Short:   "Query a configured dashboard",
		Long: `Query an external dashboard API for contact data.

Dashboards must be configured first using 'cw config dashboard add'.
Run 'cw config dashboard list' to see available dashboards.`,
		Example: `  # Query the orders dashboard for a contact
  cw dashboard orders --contact 180712

  # With pagination
  cw dashboard orders --contact 180712 --page 2 --per-page 20

  # JSON output
  cw dashboard orders --contact 180712 --output json

  # Resolve contact from conversation
  cw dashboard orders --conversation 24445`,
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			dashboardName := args[0]

			if contactID > 0 && conversationID > 0 {
				return fmt.Errorf("--contact and --conversation cannot be used together")
			}
			if contactID == 0 && conversationID == 0 {
				return fmt.Errorf("--contact or --conversation is required")
			}
			if conversationID > 0 {
				resolvedID, err := resolveContactIDFromConversation(cmdContext(cmd), conversationID)
				if err != nil {
					return err
				}
				contactID = resolvedID
			} else if contactID > 0 && !noResolve {
				originalID := contactID
				if resolvedID, ok := tryResolveContactIDFromConversation(cmdContext(cmd), contactID); ok {
					if !noResolveWarning {
						resolveWarning = fmt.Sprintf("Note: --contact %d matched a conversation; using contact ID %d", originalID, resolvedID)
						if !isJSON(cmd) && !flags.Quiet && !flags.Silent {
							_, _ = fmt.Fprintln(cmd.ErrOrStderr(), resolveWarning)
						}
					}
					contactID = resolvedID
				}
			}

			cfg, err := config.GetDashboard(dashboardName)
			if err != nil {
				dashboards, listErr := config.ListDashboards()
				if listErr == nil && len(dashboards) > 0 {
					names := make([]string, 0, len(dashboards))
					for name := range dashboards {
						names = append(names, name)
					}
					sort.Strings(names)
					return fmt.Errorf("dashboard %q not found. Available: %s", dashboardName, strings.Join(names, ", "))
				}
				return fmt.Errorf("dashboard %q not found. Run 'cw config dashboard add --help' to configure one", dashboardName)
			}

			client := api.NewDashboardClient(cfg.Endpoint, cfg.AuthToken)
			result, err := client.Query(cmdContext(cmd), api.DashboardRequest{
				ContactID: contactID,
				Page:      page,
				PerPage:   perPage,
			})
			if err != nil {
				return fmt.Errorf("dashboard query failed: %w", err)
			}

			if isJSON(cmd) {
				if resolveWarning != "" {
					addDashboardWarning(result, resolveWarning)
				}
				return printJSON(cmd, result)
			}

			return renderDashboardResult(cmd, cfg.Name, result)
		}),
	}

	cmd.Flags().IntVar(&contactID, "contact", 0, "Contact ID to query (will resolve if it matches a conversation)")
	cmd.Flags().IntVar(&conversationID, "conversation", 0, "Conversation ID to resolve contact (alternative to --contact)")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&perPage, "per-page", 100, "Results per page")
	cmd.Flags().BoolVar(&noResolve, "no-resolve", false, "Do not resolve --contact as a conversation ID")
	cmd.Flags().BoolVar(&noResolveWarning, "no-resolve-warning", false, "Suppress auto-resolve warning")
	flagAlias(cmd.Flags(), "conversation", "conv")
	flagAlias(cmd.Flags(), "per-page", "pp")
	flagAlias(cmd.Flags(), "contact", "ct")

	return cmd
}

func resolveContactIDFromConversation(ctx context.Context, conversationID int) (int, error) {
	client, err := getClient()
	if err != nil {
		return 0, err
	}

	conv, err := client.Conversations().Get(ctx, conversationID)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve conversation %d: %w", conversationID, err)
	}

	if contactID, ok := extractContactIDFromConversation(conv); ok {
		return contactID, nil
	}

	return 0, fmt.Errorf("conversation %d does not include a contact id", conversationID)
}

func tryResolveContactIDFromConversation(ctx context.Context, conversationID int) (int, bool) {
	contactID, err := resolveContactIDFromConversation(ctx, conversationID)
	if err != nil {
		return 0, false
	}
	return contactID, true
}

func extractContactIDFromConversation(conv *api.Conversation) (int, bool) {
	if conv == nil {
		return 0, false
	}
	if conv.ContactID > 0 {
		return conv.ContactID, true
	}
	if conv.Meta != nil {
		if sender, ok := conv.Meta["sender"].(map[string]any); ok {
			if id, ok := parseAnyInt(sender["id"]); ok && id > 0 {
				return id, true
			}
		}
	}
	return 0, false
}

func parseAnyInt(v any) (int, bool) {
	switch val := v.(type) {
	case float64:
		return int(val), true
	case int:
		return val, true
	case int64:
		return int(val), true
	case json.Number:
		parsed, err := val.Int64()
		if err != nil {
			return 0, false
		}
		return int(parsed), true
	case string:
		parsed, err := strconv.Atoi(val)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func renderDashboardResult(cmd *cobra.Command, displayName string, result map[string]any) error {
	out := cmd.OutOrStdout()

	if displayName != "" {
		_, _ = fmt.Fprintln(out, displayName)
	}

	if customerInfo, ok := result["customer_info"].(map[string]any); ok {
		renderCustomerInfo(out, customerInfo)
		_, _ = fmt.Fprintln(out)
	}

	if items, ok := result["items"].([]any); ok && len(items) > 0 {
		renderItemsTable(cmd, items)
	} else {
		return printJSON(cmd, result)
	}

	if pagination, ok := result["pagination"].(map[string]any); ok {
		renderPagination(out, pagination)
	}

	return nil
}

func addDashboardWarning(result map[string]any, warning string) {
	if result == nil || warning == "" {
		return
	}
	if existing, ok := result["_warnings"]; ok {
		if list, ok := existing.([]any); ok {
			result["_warnings"] = append(list, warning)
			return
		}
		if list, ok := existing.([]string); ok {
			result["_warnings"] = append(list, warning)
			return
		}
	}
	result["_warnings"] = []string{warning}
}

func renderCustomerInfo(out interface{ Write([]byte) (int, error) }, info map[string]any) {
	var parts []string

	if name, ok := info["customer_name"].(string); ok && name != "" {
		parts = append(parts, fmt.Sprintf("Customer: %s", name))
	}
	if tier, ok := info["membership_tier_name"].(string); ok && tier != "" {
		parts = append(parts, fmt.Sprintf("Member: %s", tier))
	}
	if spend, ok := info["total_spend"].(float64); ok {
		parts = append(parts, fmt.Sprintf("Total Spend: $%.0f", spend))
	}

	if len(parts) > 0 {
		_, _ = fmt.Fprintln(out, strings.Join(parts, " | "))
	}
}

func renderItemsTable(cmd *cobra.Command, items []any) {
	if len(items) == 0 {
		return
	}

	firstItem, ok := items[0].(map[string]any)
	if !ok {
		return
	}

	preferredOrder := []string{"number", "id", "name", "status", "order_status", "payment_status", "delivery_status", "total", "order_total", "items", "total_items_count", "date", "created_at", "shopline_created_at"}

	var columns []string
	seen := make(map[string]bool)

	for _, col := range preferredOrder {
		if _, exists := firstItem[col]; exists && !seen[col] {
			columns = append(columns, col)
			seen[col] = true
		}
	}

	// Limit to 6 columns to fit typical terminal width and maintain readability
	if len(columns) > 6 {
		columns = columns[:6]
	}

	if len(columns) == 0 {
		return
	}

	w := newTabWriterFromCmd(cmd)

	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = strings.ToUpper(strings.ReplaceAll(col, "_", " "))
	}
	_, _ = fmt.Fprintln(w, strings.Join(headers, "\t"))

	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		values := make([]string, len(columns))
		for i, col := range columns {
			values[i] = formatValue(row[col])
		}
		_, _ = fmt.Fprintln(w, strings.Join(values, "\t"))
	}

	_ = w.Flush()
}

func formatValue(v any) string {
	if v == nil {
		return "-"
	}
	switch val := v.(type) {
	case string:
		runes := []rune(val)
		if len(runes) > 30 {
			return string(runes[:27]) + "..."
		}
		return val
	case float64:
		if val == float64(int(val)) {
			return fmt.Sprintf("%.0f", val)
		}
		return fmt.Sprintf("%.2f", val)
	case bool:
		if val {
			return "yes"
		}
		return "no"
	case []any:
		if len(val) == 0 {
			return "[]"
		}
		return fmt.Sprintf("[%d items]", len(val))
	case map[string]any:
		if len(val) == 0 {
			return "{}"
		}
		return fmt.Sprintf("{%d keys}", len(val))
	default:
		s := fmt.Sprintf("%v", v)
		if len(s) > 30 {
			return s[:27] + "..."
		}
		return s
	}
}

func renderPagination(out interface{ Write([]byte) (int, error) }, pagination map[string]any) {
	page, _ := pagination["page"].(float64)
	totalPages, _ := pagination["total_pages"].(float64)
	totalRecords, _ := pagination["total_records"].(float64)

	if totalPages > 0 {
		_, _ = fmt.Fprintf(out, "\nPage %.0f/%.0f", page, totalPages)
		if totalRecords > 0 {
			_, _ = fmt.Fprintf(out, " (%.0f total)", totalRecords)
		}
		_, _ = fmt.Fprintln(out)
	}
}
