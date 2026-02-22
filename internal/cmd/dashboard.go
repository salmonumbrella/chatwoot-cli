package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/config"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
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
	var compact bool
	var light bool

	cmd := &cobra.Command{
		Use:     "dashboard <name>",
		Aliases: []string{"dash", "dh"},
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
				// Try prefix and subsequence matching before giving up.
				dashboards, listErr := config.ListDashboards()
				if listErr == nil && len(dashboards) > 0 {
					names := make([]string, 0, len(dashboards))
					for name := range dashboards {
						names = append(names, name)
					}
					sort.Strings(names)

					// Prefix match: find all dashboard names that start with input.
					lower := strings.ToLower(dashboardName)
					var matches []string
					for _, name := range names {
						if strings.HasPrefix(strings.ToLower(name), lower) {
							matches = append(matches, name)
						}
					}

					// Subsequence fallback: if no prefix match, check if input
					// characters appear in order within the name (e.g. "ods" matches
					// "orders" because o-d-s appears as a subsequence of o-r-d-e-r-s).
					if len(matches) == 0 {
						for _, name := range names {
							if isSubsequence(lower, strings.ToLower(name)) {
								matches = append(matches, name)
							}
						}
					}

					switch len(matches) {
					case 1:
						cfg, err = config.GetDashboard(matches[0])
						if err != nil {
							return fmt.Errorf("dashboard %q not found", matches[0])
						}
					case 0:
						return fmt.Errorf("dashboard %q not found. Available: %s", dashboardName, strings.Join(names, ", "))
					default:
						return fmt.Errorf("ambiguous dashboard %q: matches %s", dashboardName, strings.Join(matches, ", "))
					}
				} else {
					return fmt.Errorf("dashboard %q not found. Run 'cw config dashboard add --help' to configure one", dashboardName)
				}
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

			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, lightDashboardResult(result))
			}

			if isJSON(cmd) {
				if resolveWarning != "" {
					addDashboardWarning(result, resolveWarning)
				}
				if compact {
					compacted := compactDashboardResult(result)
					// Preserve _warnings from the original result.
					if w, ok := result["_warnings"]; ok {
						compacted["_warnings"] = w
					}
					result = compacted
				}
				if isAgent(cmd) {
					return printJSON(cmd, dashboardAgentEnvelope(cmd, result))
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
	flagAlias(cmd.Flags(), "no-resolve", "nr")
	cmd.Flags().BoolVar(&noResolveWarning, "no-resolve-warning", false, "Suppress auto-resolve warning")
	flagAlias(cmd.Flags(), "no-resolve-warning", "nrw")
	cmd.Flags().BoolVarP(&compact, "compact", "c", false, "Return only essential fields in JSON/agent output")
	flagAlias(cmd.Flags(), "conversation", "cv")
	flagAlias(cmd.Flags(), "per-page", "pp")
	flagAlias(cmd.Flags(), "contact", "ct")
	flagAlias(cmd.Flags(), "page", "pg")
	flagAlias(cmd.Flags(), "compact", "brief")
	flagAlias(cmd.Flags(), "compact", "summary")
	cmd.Flags().BoolVar(&light, "light", false, "Return compact summary with only the 3 most recent orders")
	flagAlias(cmd.Flags(), "light", "li")

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

// dashboardAgentEnvelope restructures the dashboard result into a ListEnvelope
// so that .it (alias for .items) works in jq expressions, consistent with
// other list commands. Without this, agent mode wraps the result in a
// DataEnvelope where items is nested under .data, breaking shortcodes.
func dashboardAgentEnvelope(cmd *cobra.Command, result map[string]any) agentfmt.ListEnvelope {
	kind := agentfmt.KindFromCommandPath(cmd.CommandPath())

	items := result["items"]
	if items == nil {
		items = []any{}
	}

	meta := make(map[string]any)
	for k, v := range result {
		if k == "items" {
			continue
		}
		meta[k] = v
	}

	return agentfmt.ListEnvelope{
		Kind:  kind,
		Items: items,
		Meta:  meta,
	}
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

// lightDashboardResult returns a strict token-minimized dashboard payload.
// It keeps only:
// - tier: membership tier (when present)
// - n: total records (when present)
// - it: up to 3 recent orders with num/dt/tot/st
func lightDashboardResult(result map[string]any) map[string]any {
	compacted := compactDashboardResult(result)
	out := make(map[string]any)

	if tier, ok := compacted["tier"]; ok {
		out["tier"] = tier
	}

	if pg, ok := compacted["pg"].(map[string]any); ok {
		if totalRecords, ok := pg["total_records"]; ok {
			out["n"] = totalRecords
		}
	}

	rawItems, _ := compacted["items"].([]any)
	limit := len(rawItems)
	if limit > 3 {
		limit = 3
	}
	items := make([]any, 0, limit)
	for i := 0; i < limit; i++ {
		raw, ok := rawItems[i].(map[string]any)
		if !ok {
			continue
		}
		item := make(map[string]any)
		if v, ok := raw["num"]; ok {
			item["num"] = v
		}
		if v, ok := raw["dt"]; ok {
			item["dt"] = v
		}
		if v, ok := raw["tot"]; ok {
			item["tot"] = v
		}
		if v, ok := raw["st"]; ok {
			item["st"] = shortOrderStatus(v)
		}
		items = append(items, item)
	}
	out["it"] = items

	return out
}

func shortOrderStatus(v any) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pending":
		return "p"
	case "confirmed":
		return "c"
	case "completed", "complete":
		return "d"
	case "cancelled", "canceled":
		return "x"
	default:
		return s
	}
}

// compactDashboardResult transforms a full dashboard API response into a
// compact summary with only essential fields.
//
// NOTE: This transformation is specific to Shopline order dashboards. The field
// mapping (shopline_created_at → dt, order_total → tot, etc.) assumes
// Shopline's data structure. For non-Shopline dashboards, --compact will produce
// items with only the fields that happen to match (e.g., "num" is generic).
// If additional dashboard integrations are added, consider making the compact
// field mapping configurable per dashboard.
//
// Result structure:
//
//	{
//	  "tier": "Gold",                    // from customer_info.membership_tier_name
//	  "items": [
//	    {
//	      "num": "ORD-001",
//	      "dt": "2026-01-15",            // shopline_created_at truncated to date
//	      "tot": 1500,                   // order_total
//	      "st": "completed",             // order_status
//	      "pay": "paid",                 // payment_status
//	      "dlv": "delivered",            // delivery_status
//	      "items": 3                     // total_items_count
//	    }
//	  ],
//	  "pg": { ... }                      // pagination preserved as-is
//	}
func compactDashboardResult(result map[string]any) map[string]any {
	out := make(map[string]any)

	// Extract membership tier from customer_info.
	if ci, ok := result["customer_info"].(map[string]any); ok {
		if tier, ok := ci["membership_tier_name"].(string); ok && tier != "" {
			out["tier"] = tier
		}
	}

	// Transform items into compact order summaries.
	items, _ := result["items"].([]any)
	orders := make([]any, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		order := make(map[string]any)
		if v, ok := item["number"]; ok {
			order["num"] = v
		}
		if v, ok := item["shopline_created_at"].(string); ok {
			if len(v) >= 10 {
				order["dt"] = v[:10]
			} else {
				order["dt"] = v
			}
		}
		if v, ok := item["order_total"]; ok {
			order["tot"] = v
		}
		if v, ok := item["order_status"]; ok {
			order["st"] = v
		}
		if v, ok := item["payment_status"]; ok {
			order["pay"] = v
		}
		if v, ok := item["delivery_status"]; ok {
			order["dlv"] = v
		}
		if v, ok := item["total_items_count"]; ok {
			order["items"] = v
		}
		orders = append(orders, order)
	}
	out["items"] = orders

	// Preserve pagination.
	if pagination, ok := result["pagination"]; ok {
		out["pg"] = pagination
	}

	return out
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

// isSubsequence returns true if every character in needle appears in haystack
// in order, though not necessarily consecutively. For example, isSubsequence("ods",
// "orders") is true because o, d, s appear in that order within orders.
func isSubsequence(needle, haystack string) bool {
	needleRunes := []rune(needle)
	if len(needleRunes) == 0 {
		return true
	}
	ni := 0
	for _, ch := range haystack {
		if needleRunes[ni] == ch {
			ni++
			if ni == len(needleRunes) {
				return true
			}
		}
	}
	return false
}
