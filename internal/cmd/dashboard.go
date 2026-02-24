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
	"github.com/chatwoot/chatwoot-cli/internal/dryrun"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newDashboardCmd() *cobra.Command {
	var contactID int
	var conversationID int
	var page int
	var perPage int
	var lineItems bool
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

  # Include line items for each order (compact support JSON)
  cw dashboard orders --contact 180712 --lni

  # Link an order to a contact (may merge contacts)
  cw dashboard link orders --contact 180712 --order-number SO20240215001 --force

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
			if lineItems && light {
				return fmt.Errorf("--line-items and --light cannot be used together")
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

			resolvedDashboardName, cfg, err := resolveDashboardConfig(dashboardName)
			if err != nil {
				return err
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
			if lineItems {
				if err := enrichDashboardLineItems(cmdContext(cmd), client, result); err != nil {
					return fmt.Errorf("dashboard line-item enrichment failed: %w", err)
				}
			}
			if shouldAddDashboardNoOrdersPath(resolvedDashboardName, cfg, result) {
				if err := addDashboardNoOrdersPath(cmdContext(cmd), resolvedDashboardName, contactID, result); err != nil {
					addDashboardWarning(result, fmt.Sprintf("no-orders support path unavailable: %v", err))
				}
			}
			if lineItems {
				if resolveWarning != "" {
					addDashboardWarning(result, resolveWarning)
				}
				// Line-items mode is agent-focused: always emit compact JSON payload.
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				if !flagOrAliasChanged(cmd, "compact-json") {
					cmd.SetContext(outfmt.WithCompact(cmd.Context(), true))
				}
				return printRawJSON(cmd, lineItemsDashboardResult(result))
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
	cmd.Flags().BoolVar(&lineItems, "line-items", false, "Fetch and attach line_items/order_metadata for each order")
	flagAlias(cmd.Flags(), "line-items", "lni")
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
	flagAlias(cmd.Flags(), "light", "lt")
	cmd.AddCommand(newDashboardLinkCmd())

	return cmd
}

func resolveDashboardConfig(dashboardName string) (string, *config.DashboardConfig, error) {
	cfg, err := config.GetDashboard(dashboardName)
	if err == nil {
		return dashboardName, cfg, nil
	}

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
				return "", nil, fmt.Errorf("dashboard %q not found", matches[0])
			}
			return matches[0], cfg, nil
		case 0:
			return "", nil, fmt.Errorf("dashboard %q not found. Available: %s", dashboardName, strings.Join(names, ", "))
		default:
			return "", nil, fmt.Errorf("ambiguous dashboard %q: matches %s", dashboardName, strings.Join(matches, ", "))
		}
	}

	return "", nil, fmt.Errorf("dashboard %q not found. Run 'cw config dashboard add --help' to configure one", dashboardName)
}

func newDashboardLinkCmd() *cobra.Command {
	var contactID int
	var orderNumber string
	var force bool

	cmd := &cobra.Command{
		Use:   "link <name>",
		Short: "Link an order to a contact via dashboard API",
		Long: `Link a Shopline order number to a Chatwoot contact.

Warning: this operation may trigger contact merges in Chatwoot and cannot be undone.`,
		Example: `  # Link an order to a contact (recommended for scripts)
  cw dashboard link orders --contact 180712 --order-number SO20240215001 --force

  # Same with short aliases
  cw dh link ods --ct 180712 --on SO20240215001 --force`,
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			dashboardName := args[0]

			if contactID <= 0 {
				return fmt.Errorf("--contact is required")
			}
			orderNumber = strings.TrimSpace(orderNumber)
			if orderNumber == "" {
				return fmt.Errorf("--order-number is required")
			}

			if ok, err := maybeDryRun(cmd, &dryrun.Preview{
				Operation:   "link",
				Resource:    "order",
				Description: "Link Shopline order number to Chatwoot contact (may merge contacts).",
				Details: map[string]any{
					"dashboard":    dashboardName,
					"contact_id":   contactID,
					"order_number": orderNumber,
				},
				Warnings: []string{"This operation may merge contacts and cannot be undone."},
			}); ok {
				return err
			}

			if err := requireForceForJSON(cmd, force); err != nil {
				return err
			}

			ok, err := confirmAction(cmd, confirmOptions{
				Prompt:              yellow("Type 'link' to confirm (this may merge contacts and cannot be undone): "),
				Expected:            "link",
				CancelMessage:       "Order link cancelled.",
				Force:               force,
				RequireForceForJSON: true,
			})
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}

			_, cfg, err := resolveDashboardConfig(dashboardName)
			if err != nil {
				return err
			}

			client := api.NewDashboardClient(cfg.Endpoint, cfg.AuthToken)
			result, err := client.LinkOrderToContact(cmdContext(cmd), orderNumber, contactID)
			if err != nil {
				return fmt.Errorf("failed to link order %q to contact %d: %w", orderNumber, contactID, err)
			}

			payload := map[string]any{
				"order_number":        orderNumber,
				"customer_id":         result.CustomerID,
				"chatwoot_contact_id": result.ChatwootContactID,
			}

			if isJSON(cmd) {
				return printJSON(cmd, payload)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Linked order %s to contact %d (customer: %s)\n", orderNumber, result.ChatwootContactID, result.CustomerID)
			return nil
		}),
	}

	cmd.Flags().IntVar(&contactID, "contact", 0, "Chatwoot contact ID to link the order to (required)")
	cmd.Flags().StringVar(&orderNumber, "order-number", "", "Shopline order number to link (required)")
	cmd.Flags().BoolVarP(&force, "force", "F", false, "Skip confirmation prompt (required for --output json)")
	flagAlias(cmd.Flags(), "contact", "ct")
	flagAlias(cmd.Flags(), "order-number", "on")

	return cmd
}

func enrichDashboardLineItems(ctx context.Context, client *api.DashboardClient, result map[string]any) error {
	if client == nil || result == nil {
		return nil
	}

	items, ok := result["items"].([]any)
	if !ok || len(items) == 0 {
		return nil
	}

	for i, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		orderID, ok := dashboardOrderID(item["id"])
		if !ok {
			continue
		}

		detail, err := client.QueryOrderDetail(ctx, orderID)
		if err != nil {
			return fmt.Errorf("order %s: %w", orderID, err)
		}

		if lineItems, ok := detail["line_items"]; ok {
			item["line_items"] = lineItems
		}
		if metadata, ok := detail["order_metadata"]; ok {
			item["order_metadata"] = metadata
		}
		if order, ok := detail["order"].(map[string]any); ok {
			for key, value := range order {
				if _, exists := item[key]; !exists {
					item[key] = value
				}
			}
		}

		items[i] = item
	}

	result["items"] = items
	return nil
}

func dashboardOrderID(v any) (string, bool) {
	switch val := v.(type) {
	case string:
		id := strings.TrimSpace(val)
		return id, id != ""
	case json.Number:
		id := strings.TrimSpace(val.String())
		return id, id != ""
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10), true
		}
		id := strings.TrimSpace(fmt.Sprintf("%v", val))
		return id, id != ""
	case int:
		return strconv.Itoa(val), true
	case int64:
		return strconv.FormatInt(val, 10), true
	default:
		id := strings.TrimSpace(fmt.Sprintf("%v", val))
		if id == "" || id == "<nil>" {
			return "", false
		}
		return id, true
	}
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

func shouldAddDashboardNoOrdersPath(dashboardName string, cfg *config.DashboardConfig, result map[string]any) bool {
	if cfg == nil || result == nil {
		return false
	}
	normalizedName := strings.ToLower(strings.TrimSpace(dashboardName))
	endpoint := strings.ToLower(strings.TrimSpace(cfg.Endpoint))
	name := strings.ToLower(strings.TrimSpace(cfg.Name))
	isOrdersDashboard := strings.Contains(endpoint, "/chatwoot/contact/orders") ||
		strings.Contains(name, "order") ||
		strings.Contains(normalizedName, "order") ||
		normalizedName == "ods"
	if !isOrdersDashboard {
		return false
	}
	items, ok := result["items"].([]any)
	return !ok || len(items) == 0
}

func addDashboardNoOrdersPath(ctx context.Context, dashboardName string, contactID int, result map[string]any) error {
	if result == nil {
		return nil
	}

	linkCheck := map[string]any{
		"source":          "chatwoot_contact.custom_attributes",
		"shopline_linked": false,
		"shopify_linked":  false,
		"matched_keys":    []string{},
		"lookup":          "ok",
	}

	path := "no_orders_contact_unlinked"

	if contactID > 0 {
		client, err := getClient()
		if err != nil {
			linkCheck["lookup"] = "error"
			linkCheck["error"] = err.Error()
			path = "no_orders_link_status_unknown"
		} else {
			contact, err := client.Contacts().Get(ctx, contactID)
			if err != nil {
				linkCheck["lookup"] = "error"
				linkCheck["error"] = err.Error()
				path = "no_orders_link_status_unknown"
			} else {
				shoplineLinked, shopifyLinked, keys := detectCommerceAccountLinks(contact.CustomAttributes)
				linkCheck["shopline_linked"] = shoplineLinked
				linkCheck["shopify_linked"] = shopifyLinked
				linkCheck["matched_keys"] = keys
				if shoplineLinked || shopifyLinked {
					path = "no_orders_contact_linked_check_store_api"
				}
			}
		}
	} else {
		linkCheck["lookup"] = "skipped"
		path = "no_orders_contact_id_missing"
	}

	steps := []any{
		map[string]any{
			"id":   "check_link",
			"do":   "Check Chatwoot contact custom_attributes for Shopline/Shopify account IDs.",
			"cmd":  fmt.Sprintf("cw contacts get %d -o json --jq '.custom_attributes'", contactID),
			"when": "always",
		},
		map[string]any{
			"id":   "query_store_orders",
			"do":   "If linked, query store orders directly (Shopify CLI command or Shopline API).",
			"cmd":  fmt.Sprintf("cw integrations shopify orders --contact-id %d", contactID),
			"when": "if_linked",
		},
		map[string]any{
			"id":   "request_order_screenshot",
			"do":   "If still unresolved, ask the customer for a screenshot showing order number/details.",
			"when": "if_no_orders",
		},
		map[string]any{
			"id":   "link_contact_account",
			"do":   "Link a known order number to this contact (may merge contacts).",
			"cmd":  fmt.Sprintf("cw dashboard link %s --contact %d --order-number <ORDER_NUMBER> --force -o json", dashboardCommandName(dashboardName), contactID),
			"when": "after_verification",
		},
	}

	result["support_path"] = map[string]any{
		"status":     "no_orders_found",
		"path":       path,
		"contact_id": contactID,
		"link_check": linkCheck,
		"steps":      steps,
	}

	return nil
}

func dashboardCommandName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "orders"
	}
	return name
}

func detectCommerceAccountLinks(attrs map[string]any) (shoplineLinked, shopifyLinked bool, matchedKeys []string) {
	if len(attrs) == 0 {
		return false, false, []string{}
	}

	for key, value := range attrs {
		if !dashboardAttrHasValue(value) {
			continue
		}
		lk := strings.ToLower(strings.TrimSpace(key))
		lv := strings.ToLower(strings.TrimSpace(dashboardAttrString(value)))

		isShopline := containsAnySubstring(lk, []string{"shopline", "shop_line", "sl_"}) ||
			containsAnySubstring(lv, []string{"shopline"})
		isShopify := containsAnySubstring(lk, []string{"shopify", "myshopify", "sf_"}) ||
			containsAnySubstring(lv, []string{"shopify", "myshopify"})

		if !isShopline && !isShopify {
			continue
		}
		if isShopline {
			shoplineLinked = true
		}
		if isShopify {
			shopifyLinked = true
		}
		matchedKeys = append(matchedKeys, key)
	}

	sort.Strings(matchedKeys)
	return shoplineLinked, shopifyLinked, matchedKeys
}

func dashboardAttrHasValue(v any) bool {
	switch val := v.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(val) != ""
	case json.Number:
		return strings.TrimSpace(val.String()) != "" && val.String() != "0"
	case float64:
		return val != 0
	case int:
		return val != 0
	case int64:
		return val != 0
	case bool:
		return val
	case []any:
		return len(val) > 0
	case map[string]any:
		return len(val) > 0
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v)) != ""
	}
}

func dashboardAttrString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case json.Number:
		return val.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func containsAnySubstring(s string, subs []string) bool {
	for _, sub := range subs {
		if sub != "" && strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func appendCompactNoOrdersPath(out map[string]any, result map[string]any) {
	if out == nil || result == nil {
		return
	}
	supportPath, ok := result["support_path"].(map[string]any)
	if !ok {
		return
	}

	if path, ok := supportPath["path"]; ok {
		out["path"] = path
	}
	if contactID, ok := supportPath["contact_id"]; ok {
		out["cid"] = contactID
	}

	if linkCheck, ok := supportPath["link_check"].(map[string]any); ok {
		lk := make(map[string]any)
		if v, ok := linkCheck["shopline_linked"]; ok {
			lk["sl"] = v
		}
		if v, ok := linkCheck["shopify_linked"]; ok {
			lk["sf"] = v
		}
		if keys, ok := linkCheck["matched_keys"]; ok {
			switch kv := keys.(type) {
			case []any:
				if len(kv) > 0 {
					lk["k"] = kv
				}
			case []string:
				if len(kv) > 0 {
					anyKeys := make([]any, 0, len(kv))
					for _, key := range kv {
						anyKeys = append(anyKeys, key)
					}
					lk["k"] = anyKeys
				}
			}
		}
		if len(lk) > 0 {
			out["lk"] = lk
		}
	}

	if steps, ok := supportPath["steps"].([]any); ok {
		next := make([]any, 0, len(steps))
		for _, raw := range steps {
			step, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if id, ok := step["id"]; ok {
				next = append(next, id)
			}
		}
		if len(next) > 0 {
			out["nx"] = next
		}
	}
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
	appendCompactNoOrdersPath(out, result)

	return out
}

// lineItemsDashboardResult returns a compact order+line-items payload for support workflows.
// It keeps only high-signal fields needed for customer support decisions.
func lineItemsDashboardResult(result map[string]any) map[string]any {
	rawItems, _ := result["items"].([]any)
	items := make([]any, 0, len(rawItems))
	for _, raw := range rawItems {
		order, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		item := make(map[string]any)
		if v, ok := order["id"]; ok {
			item["id"] = v
		}
		if v, ok := order["number"]; ok {
			item["num"] = v
		}
		if v, ok := order["shopline_created_at"].(string); ok && v != "" {
			if len(v) >= 10 {
				item["dt"] = v[:10]
			} else {
				item["dt"] = v
			}
		} else if v, ok := order["created_at"].(string); ok && v != "" {
			if len(v) >= 10 {
				item["dt"] = v[:10]
			} else {
				item["dt"] = v
			}
		}
		if v, ok := order["order_total"]; ok {
			item["tot"] = v
		}
		if v, ok := order["order_status"]; ok {
			item["st"] = shortOrderStatus(v)
		}
		if v, ok := order["payment_status"]; ok {
			item["pay"] = v
		}
		if v, ok := order["delivery_status"]; ok {
			item["dlv"] = v
		}
		if v, ok := order["line_items"].([]any); ok {
			item["li"] = compactSupportLineItems(v)
		}
		items = append(items, item)
	}
	out := map[string]any{"it": items}
	appendCompactNoOrdersPath(out, result)
	if warnings, ok := result["_warnings"]; ok {
		out["_warnings"] = warnings
	}
	return out
}

func compactSupportLineItems(lineItems []any) []any {
	out := make([]any, 0, len(lineItems))
	for _, raw := range lineItems {
		line, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		item := make(map[string]any)

		if v, ok := line["product_name"]; ok {
			item["p"] = v
		}

		colorway, _ := line["colorway"].(string)
		size, _ := line["size"].(string)
		colorway = strings.TrimSpace(colorway)
		size = strings.TrimSpace(size)
		switch {
		case colorway != "" && size != "":
			item["v"] = colorway + " / " + size
		case colorway != "":
			item["v"] = colorway
		case size != "":
			item["v"] = size
		}

		if v, ok := line["quantity"]; ok {
			item["q"] = v
		}
		if v, ok := line["total_ntd"]; ok {
			item["tot"] = v
		} else if v, ok := line["price_ntd"]; ok {
			item["tot"] = v
		}
		if v, ok := line["is_refunded"]; ok {
			item["rf"] = v
		}
		if v, ok := line["inventory_quantity"]; ok {
			item["inv"] = v
		}

		out = append(out, item)
	}
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
	if supportPath, ok := result["support_path"]; ok {
		out["support_path"] = supportPath
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

	preferredOrder := []string{"number", "id", "name", "status", "order_status", "payment_status", "delivery_status", "total", "order_total", "line_items", "items", "total_items_count", "date", "created_at", "shopline_created_at"}

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
		if _, hasLineItems := firstItem["line_items"]; hasLineItems && !stringInSlice(columns, "line_items") {
			columns[len(columns)-1] = "line_items"
		}
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

func stringInSlice(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
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
