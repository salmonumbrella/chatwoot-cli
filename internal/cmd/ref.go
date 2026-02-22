package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/urlparse"
	"github.com/spf13/cobra"
)

type refItem struct {
	Input   string `json:"input"`
	Type    string `json:"type"`
	ID      int    `json:"id"`
	TypedID string `json:"typed_id"`

	// URL is the Chatwoot UI URL when it can be constructed for this type.
	URL string `json:"url,omitempty"`

	// Actions suggests follow-up commands for agents to run next. These suggestions
	// never trigger extra API calls; they are derived from the resolved reference.
	Actions []refAction `json:"actions,omitempty"`

	// Probe metadata (present only when we had to probe).
	Probed []string `json:"probed,omitempty"`
}

type refAction struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Argv        []string         `json:"argv"`
	Destructive bool             `json:"destructive,omitempty"`
	Notes       string           `json:"notes,omitempty"`
	Inputs      []refActionInput `json:"inputs,omitempty"`
}

type refActionInput struct {
	Name     string `json:"name"`
	Prompt   string `json:"prompt,omitempty"`
	Required bool   `json:"required,omitempty"`
}

func newRefCmd() *cobra.Command {
	var (
		typeFlag string
		tryFlags []string
		noProbe  bool
		emit     string
	)

	cmd := &cobra.Command{
		Use:   "ref <id|#id|type:id|url>",
		Short: "Resolve a reference into a canonical typed ID (agent-friendly)",
		Long: strings.TrimSpace(`
Normalize identifiers (IDs, #ID shorthands, typed prefixes, or Chatwoot UI URLs)
into a canonical typed ID form for agent workflows.

If the input is a bare numeric ID without a type, this command can probe
one or more resource types to determine what the ID refers to.

Examples:
  cw ref 123
  cw ref #123
  cw ref conversation:123
  cw ref https://app.chatwoot.com/app/accounts/1/conversations/123
  cw ref 123 --try conversation --try contact
  cw ref 123 --type contact
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			input := strings.TrimSpace(args[0])
			if input == "" {
				return fmt.Errorf("invalid ref: empty input")
			}

			if emit == "" {
				if isJSON(cmd) || isAgent(cmd) {
					emit = "json"
				} else {
					emit = "id"
				}
			}
			emit = strings.ToLower(strings.TrimSpace(emit))
			switch emit {
			case "json", "id", "url":
			default:
				return fmt.Errorf("invalid --emit %q: must be one of json, id, url", emit)
			}

			// 1) URL input: parse without probing.
			if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
				parsed, err := urlparse.Parse(input)
				if err != nil {
					return fmt.Errorf("failed to parse URL: %w", err)
				}
				if !parsed.HasResourceID() {
					return fmt.Errorf("URL must include a resource ID (e.g., /conversations/123)")
				}
				item, err := buildRefItem(input, parsed.ResourceType, parsed.ResourceID, nil, emit != "id")
				if err != nil {
					return err
				}
				return emitRef(cmd, item, emit)
			}

			// 2) Typed ID input: infer the resource type from the prefix and parse.
			if prefix, _, ok := strings.Cut(input, ":"); ok && !strings.Contains(input, "://") {
				if inferred, err := normalizeRefResourceType(prefix); err == nil {
					id, err := parseIDOrURL(input, inferred)
					if err != nil {
						return err
					}
					item, err := buildRefItem(input, inferred, id, nil, emit != "id")
					if err != nil {
						return err
					}
					return emitRef(cmd, item, emit)
				}
			}

			// 3) Explicit type flag.
			if strings.TrimSpace(typeFlag) != "" {
				rt, err := normalizeRefResourceType(typeFlag)
				if err != nil {
					return err
				}
				id, err := parseIDOrURL(input, rt)
				if err != nil {
					return err
				}
				item, err := buildRefItem(input, rt, id, nil, emit != "id")
				if err != nil {
					return err
				}
				return emitRef(cmd, item, emit)
			}

			// 4) Bare ID: either default (no-probe) or probe.
			id, err := parseIDOrURL(input, "")
			if err != nil {
				return err
			}

			if noProbe {
				item, err := buildRefItem(input, "conversation", id, nil, emit != "id")
				if err != nil {
					return err
				}
				return emitRef(cmd, item, emit)
			}

			tryTypes, err := parseTryTypes(tryFlags)
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			matches, err := probeID(cmdContext(cmd), client, id, tryTypes)
			if err != nil {
				return err
			}
			if len(matches) == 0 {
				return api.NewStructuredErrorWithContext(api.ErrNotFound, fmt.Sprintf("ID %d not found in probed resource types", id), map[string]any{
					"id":     id,
					"probed": tryTypes,
				})
			}
			if len(matches) > 1 {
				var typed []string
				for _, m := range matches {
					typed = append(typed, fmt.Sprintf("%s:%d", prefixForType(m), id))
				}
				sort.Strings(typed)

				e := api.NewStructuredErrorWithContext(api.ErrValidation, fmt.Sprintf("ambiguous ID %d: matches multiple resource types (%s)", id, strings.Join(typed, ", ")), map[string]any{
					"id":      id,
					"matches": typed,
					"probed":  tryTypes,
				})
				e.Suggestion = "Specify --type (e.g. --type contact) or pass a typed ID (e.g. contact:123)"
				return e
			}

			item, err := buildRefItem(input, matches[0], id, tryTypes, emit != "id")
			if err != nil {
				return err
			}
			return emitRef(cmd, item, emit)
		}),
	}

	cmd.Flags().StringVarP(&typeFlag, "type", "T", "", "Resource type to assume for bare IDs (skips probing)")
	cmd.Flags().StringSliceVar(&tryFlags, "try", nil, "Resource types to probe for bare IDs (repeatable; default: conversation, contact)")
	flagAlias(cmd.Flags(), "try", "tr")
	cmd.Flags().BoolVar(&noProbe, "no-probe", false, "Do not probe bare IDs; default them to conversations")
	flagAlias(cmd.Flags(), "no-probe", "np")
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Emit format: json|id|url (defaults: id for text output; json for json/agent)")

	return cmd
}

func emitRef(cmd *cobra.Command, item refItem, emit string) error {
	switch emit {
	case "id":
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), item.TypedID)
		return nil
	case "url":
		if strings.TrimSpace(item.URL) == "" {
			return api.NewStructuredErrorWithContext(api.ErrValidation, fmt.Sprintf("no URL available for %s:%d", prefixForType(item.Type), item.ID), map[string]any{
				"type": item.Type,
				"id":   item.ID,
			})
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), item.URL)
		return nil
	case "json":
		if isAgent(cmd) {
			payload := agentfmt.ItemEnvelope{
				Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
				Item: item,
			}
			return printJSON(cmd, payload)
		}
		return printJSON(cmd, item)
	default:
		return fmt.Errorf("unknown emit %q", emit)
	}
}

func buildRefItem(input, resourceType string, id int, probed []string, includeURL bool) (refItem, error) {
	typedID := fmt.Sprintf("%s:%d", prefixForType(resourceType), id)
	item := refItem{
		Input:   input,
		Type:    resourceType,
		ID:      id,
		TypedID: typedID,
		Actions: actionsForType(resourceType, typedID),
	}

	if len(probed) > 0 {
		cp := append([]string(nil), probed...)
		sort.Strings(cp)
		item.Probed = cp
	}

	if includeURL {
		if plural, ok := uiPluralForType(resourceType); ok {
			u, err := resourceURL(plural, id)
			if err != nil {
				return refItem{}, err
			}
			item.URL = u
		}
	}

	return item, nil
}

func actionsForType(resourceType string, typedID string) []refAction {
	// Keep this list tight: only suggest actions that exist and are commonly useful.
	// Placeholders are allowed when the underlying command requires extra input.
	switch resourceType {
	case "conversation":
		return []refAction{
			{ID: "open", Title: "Open details", Argv: []string{"cw", "open", typedID}},
			{ID: "ctx", Title: "Get context (messages + contact)", Argv: []string{"cw", "ctx", typedID}},
			{ID: "comment", Title: "Send a public reply", Argv: []string{"cw", "comment", typedID, "$text"}, Inputs: []refActionInput{{Name: "text", Prompt: "Message text", Required: true}}},
			{ID: "note", Title: "Add a private note", Argv: []string{"cw", "note", typedID, "$text"}, Inputs: []refActionInput{{Name: "text", Prompt: "Note text", Required: true}}},
			{
				ID:    "assign",
				Title: "Assign to an agent or team",
				Argv:  []string{"cw", "assign", typedID, "--agent", "$agent"},
				Inputs: []refActionInput{
					{Name: "agent", Prompt: "Agent identifier (id, name, or email)", Required: true},
				},
				Notes: "You can also use --team $team instead of --agent.",
			},
			{ID: "close", Title: "Close conversation", Argv: []string{"cw", "close", typedID}, Destructive: true},
			{ID: "reopen", Title: "Reopen conversation", Argv: []string{"cw", "reopen", typedID}},
		}

	case "contact":
		return []refAction{
			{ID: "open", Title: "Open details", Argv: []string{"cw", "open", typedID}},
			{ID: "get", Title: "Get contact", Argv: []string{"cw", "contacts", "get", typedID}},
			{ID: "conversations", Title: "List conversations for contact", Argv: []string{"cw", "contacts", "conversations", typedID}},
			{ID: "update", Title: "Update contact", Argv: []string{"cw", "contacts", "update", typedID, "--name", "$name"}, Inputs: []refActionInput{{Name: "name", Prompt: "Contact name", Required: true}}},
		}

	case "inbox":
		return []refAction{
			{ID: "open", Title: "Open details", Argv: []string{"cw", "open", typedID}},
			{ID: "get", Title: "Get inbox", Argv: []string{"cw", "inboxes", "get", typedID}},
			{ID: "members", Title: "List inbox members", Argv: []string{"cw", "inbox-members", "list", typedID}},
		}

	case "team":
		return []refAction{
			{ID: "open", Title: "Open details", Argv: []string{"cw", "open", typedID}},
			{ID: "get", Title: "Get team", Argv: []string{"cw", "teams", "get", typedID}},
		}

	case "agent":
		return []refAction{
			{ID: "open", Title: "Open details", Argv: []string{"cw", "open", typedID}},
			{ID: "get", Title: "Get agent", Argv: []string{"cw", "agents", "get", typedID}},
		}

	case "campaign":
		return []refAction{
			{ID: "open", Title: "Open details", Argv: []string{"cw", "open", typedID}},
			{ID: "get", Title: "Get campaign", Argv: []string{"cw", "campaigns", "get", typedID}},
			{ID: "update", Title: "Update campaign", Argv: []string{"cw", "campaigns", "update", typedID, "--title", "$title"}, Inputs: []refActionInput{{Name: "title", Prompt: "Campaign title", Required: true}}},
			{ID: "delete", Title: "Delete campaign", Argv: []string{"cw", "campaigns", "delete", typedID, "--force"}, Destructive: true},
		}

	case "label":
		return []refAction{
			{ID: "get", Title: "Get label", Argv: []string{"cw", "labels", "get", typedID}},
			{ID: "update", Title: "Update label", Argv: []string{"cw", "labels", "update", typedID, "--title", "$title"}, Inputs: []refActionInput{{Name: "title", Prompt: "Label title", Required: true}}},
			{ID: "delete", Title: "Delete label", Argv: []string{"cw", "labels", "delete", typedID}, Destructive: true},
		}

	case "canned response":
		return []refAction{
			{ID: "get", Title: "Get canned response", Argv: []string{"cw", "canned-responses", "get", typedID}},
			{ID: "update", Title: "Update canned response", Argv: []string{"cw", "canned-responses", "update", typedID, "--content", "$content"}, Inputs: []refActionInput{{Name: "content", Prompt: "Response content", Required: true}}},
			{ID: "delete", Title: "Delete canned response", Argv: []string{"cw", "canned-responses", "delete", typedID}, Destructive: true},
		}

	case "rule":
		return []refAction{
			{ID: "get", Title: "Get automation rule", Argv: []string{"cw", "automation-rules", "get", typedID}},
			{ID: "clone", Title: "Clone automation rule", Argv: []string{"cw", "automation-rules", "clone", typedID}},
			{ID: "update", Title: "Update automation rule", Argv: []string{"cw", "automation-rules", "update", typedID, "--name", "$name"}, Inputs: []refActionInput{{Name: "name", Prompt: "Rule name", Required: true}}},
			{ID: "delete", Title: "Delete automation rule", Argv: []string{"cw", "automation-rules", "delete", typedID}, Destructive: true},
		}

	case "bot":
		return []refAction{
			{ID: "get", Title: "Get agent bot", Argv: []string{"cw", "agent-bots", "get", typedID}},
			{ID: "update", Title: "Update agent bot", Argv: []string{"cw", "agent-bots", "update", typedID, "--name", "$name"}, Inputs: []refActionInput{{Name: "name", Prompt: "Bot name", Required: true}}},
			{ID: "reset-token", Title: "Reset bot access token", Argv: []string{"cw", "agent-bots", "reset-token", typedID}, Destructive: true},
			{ID: "delete", Title: "Delete agent bot", Argv: []string{"cw", "agent-bots", "delete", typedID}, Destructive: true},
		}

	case "webhook":
		return []refAction{
			{ID: "get", Title: "Get webhook", Argv: []string{"cw", "webhooks", "get", typedID}},
			{ID: "delete", Title: "Delete webhook", Argv: []string{"cw", "webhooks", "delete", typedID}, Destructive: true},
		}

	case "custom attribute":
		return []refAction{
			{ID: "get", Title: "Get custom attribute", Argv: []string{"cw", "custom-attributes", "get", typedID}},
			{ID: "delete", Title: "Delete custom attribute", Argv: []string{"cw", "custom-attributes", "delete", typedID}, Destructive: true},
		}

	case "custom filter":
		return []refAction{
			{ID: "get", Title: "Get custom filter", Argv: []string{"cw", "custom-filters", "get", typedID}},
			{ID: "delete", Title: "Delete custom filter", Argv: []string{"cw", "custom-filters", "delete", typedID}, Destructive: true},
		}

	case "account":
		// Note: "account" here is the platform account ID, not "cw account get" (which has no ID).
		return []refAction{
			{ID: "platform-get", Title: "Get platform account", Argv: []string{"cw", "platform", "accounts", "get", typedID}},
			{ID: "platform-update", Title: "Update platform account", Argv: []string{"cw", "platform", "accounts", "update", typedID, "--name", "$name"}, Inputs: []refActionInput{{Name: "name", Prompt: "Account name", Required: true}}},
			{ID: "platform-delete", Title: "Delete platform account", Argv: []string{"cw", "platform", "accounts", "delete", typedID}, Destructive: true},
		}

	case "user":
		return []refAction{
			{ID: "platform-get", Title: "Get platform user", Argv: []string{"cw", "platform", "users", "get", typedID}},
			{ID: "platform-update", Title: "Update platform user", Argv: []string{"cw", "platform", "users", "update", typedID, "--email", "$email"}, Inputs: []refActionInput{{Name: "email", Prompt: "User email", Required: true}}},
			{ID: "platform-delete", Title: "Delete platform user", Argv: []string{"cw", "platform", "users", "delete", typedID}, Destructive: true},
			{ID: "platform-login", Title: "Get platform SSO login URL", Argv: []string{"cw", "platform", "users", "login", typedID}},
		}

	case "article":
		return []refAction{
			{
				ID:    "get",
				Title: "Get portal article (requires portal slug)",
				Argv:  []string{"cw", "portals", "articles", "get", "<portal-slug>", typedID},
				Notes: "Portal article commands require a portal slug (e.g. \"help\").",
			},
		}

	case "hook":
		return []refAction{
			{ID: "update", Title: "Update integration hook", Argv: []string{"cw", "integrations", "hook-update", typedID, "--settings", "$json"}, Inputs: []refActionInput{{Name: "json", Prompt: "Settings JSON object (stringified)", Required: true}}},
			{ID: "delete", Title: "Delete integration hook", Argv: []string{"cw", "integrations", "hook-delete", typedID}, Destructive: true},
		}
	}

	return nil
}

func parseTryTypes(inputs []string) ([]string, error) {
	if len(inputs) == 0 {
		return []string{"conversation", "contact"}, nil
	}

	var out []string
	for _, in := range inputs {
		for _, part := range strings.Split(in, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			rt, err := normalizeRefResourceType(part)
			if err != nil {
				return nil, err
			}
			out = append(out, rt)
		}
	}

	if len(out) == 0 {
		return []string{"conversation", "contact"}, nil
	}

	// Dedup stable.
	seen := make(map[string]struct{}, len(out))
	var dedup []string
	for _, t := range out {
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		dedup = append(dedup, t)
	}
	return dedup, nil
}

func normalizeRefResourceType(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("resource type cannot be empty")
	}

	switch canonicalResourceName(input) {
	case "conversation", "conversations", "conv", "c":
		return "conversation", nil
	case "contact", "contacts":
		return "contact", nil
	case "inbox", "inboxes":
		return "inbox", nil
	case "team", "teams":
		return "team", nil
	case "agent", "agents":
		return "agent", nil
	case "user", "users":
		// Platform "users" are distinct from account "agents". This enables typed IDs like "user:123"
		// to map to platform commands (chatwoot platform users ...).
		return "user", nil
	case "campaign", "campaigns":
		return "campaign", nil
	case "label", "labels":
		return "label", nil
	case "cannedresponse", "cannedresponses", "canned":
		return "canned response", nil
	case "rule", "rules", "automationrule", "automationrules":
		return "rule", nil
	case "bot", "bots", "agentbot", "agentbots":
		return "bot", nil
	case "webhook", "webhooks":
		return "webhook", nil
	case "customattribute", "customattributes":
		return "custom attribute", nil
	case "customfilter", "customfilters", "filter", "filters":
		return "custom filter", nil
	case "account", "accounts":
		return "account", nil
	case "article", "articles":
		return "article", nil
	case "hook", "hooks":
		return "hook", nil
	default:
		return "", fmt.Errorf("invalid resource type %q", input)
	}
}

func probeID(ctx context.Context, client *api.Client, id int, tryTypes []string) ([]string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		t   string
		err error
	}

	results := make(chan result, len(tryTypes))
	var wg sync.WaitGroup

	for _, t := range tryTypes {
		t := t
		wg.Add(1)
		go func() {
			defer wg.Done()

			var err error
			switch t {
			case "conversation":
				_, err = client.Conversations().Get(ctx, id)
			case "contact":
				_, err = client.Contacts().Get(ctx, id)
			case "inbox":
				_, err = client.Inboxes().Get(ctx, id)
			case "team":
				_, err = client.Teams().Get(ctx, id)
			case "agent":
				_, err = client.Agents().Get(ctx, id)
			case "campaign":
				_, err = client.Campaigns().Get(ctx, id)
			case "label":
				_, err = client.Labels().Get(ctx, id)
			case "canned response":
				_, err = client.CannedResponses().Get(ctx, id)
			case "rule":
				_, err = client.AutomationRules().Get(ctx, id)
			case "bot":
				_, err = client.AgentBots().Get(ctx, id)
			case "webhook":
				_, err = client.Webhooks().Get(ctx, id)
			case "custom attribute":
				_, err = client.CustomAttributes().Get(ctx, id)
			case "custom filter":
				_, err = client.CustomFilters().Get(ctx, id)
			default:
				err = fmt.Errorf("unsupported probe type %q", t)
			}

			results <- result{t: t, err: err}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var (
		matches     []string
		firstErr    error
		unsupported error
	)

	for r := range results {
		if r.err == nil {
			matches = append(matches, r.t)
			if len(matches) > 1 {
				// Cancel remaining probes to save time once ambiguous.
				cancel()
			}
			continue
		}
		if api.IsNotFoundError(r.err) || strings.Contains(strings.ToLower(r.err.Error()), "context canceled") {
			continue
		}
		// If we misconfigured probe types, surface the error clearly.
		if strings.Contains(r.err.Error(), "unsupported probe type") && unsupported == nil {
			unsupported = r.err
		}
		if firstErr == nil {
			firstErr = r.err
		}
	}

	if unsupported != nil {
		return nil, unsupported
	}
	if len(matches) > 0 {
		sort.Strings(matches)
		return matches, nil
	}
	return nil, firstErr
}
