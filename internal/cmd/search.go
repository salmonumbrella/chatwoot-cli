package cmd

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

// bracketedNameRegex matches LINE Official Account message prefixes like "[Jack Su] Hello".
// LINE group messages are relayed through a single Chatwoot contact but prefix each
// message with the actual sender's name in brackets. This regex extracts that name
// for sender search matching.
var bracketedNameRegex = regexp.MustCompile(`^\s*\[([^\]]+)\]`)

// SnippetInfo contains a matching message snippet for a conversation
type SnippetInfo struct {
	MessageID int    `json:"message_id"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"created_at"`
}

// SenderMatch represents a person who messages through a contact/conversation
type SenderMatch struct {
	Name           string `json:"name"`
	ContactID      int    `json:"contact_id"`
	ContactName    string `json:"contact_name"`
	ConversationID int    `json:"conversation_id"`
	LastMessageAt  int64  `json:"last_message_at,omitempty"`
	MessageCount   int    `json:"message_count,omitempty"`
}

// UnifiedSearchResult represents a single search result of any type, sortable by activity
type UnifiedSearchResult struct {
	Type           string            `json:"type"`                   // "contact", "conversation", or "sender"
	ID             int               `json:"id"`                     // primary ID for the result
	Name           string            `json:"name"`                   // display name
	LastActivityAt int64             `json:"last_activity_at"`       // for sorting
	Contact        *api.Contact      `json:"contact,omitempty"`      // populated if type=contact
	Conversation   *api.Conversation `json:"conversation,omitempty"` // populated if type=conversation
	Sender         *SenderMatch      `json:"sender,omitempty"`       // populated if type=sender
}

// SearchResults represents the combined search results from multiple resource types
type SearchResults struct {
	Query         string                 `json:"query"`
	Results       []UnifiedSearchResult  `json:"results"`       // unified sorted list (never omitted, always [])
	Contacts      []api.Contact          `json:"contacts"`      // never omitted so jq filters don't fail on null
	Conversations []api.Conversation     `json:"conversations"` // never omitted so jq filters don't fail on null
	Senders       []SenderMatch          `json:"senders"`       // never omitted so jq filters don't fail on null
	Snippets      map[string]SnippetInfo `json:"snippets,omitempty"`
	Summary       map[string]int         `json:"summary"`
}

func newSearchCmd() *cobra.Command {
	var (
		types          []string
		limit          int
		selectOne      bool
		selectRaw      bool
		includeSnippet bool
		best           bool
		emit           string
		light          bool
	)

	cmd := &cobra.Command{
		Use:     "search <query>",
		Aliases: []string{"find", "s"},
		Short:   "Search across multiple resources",
		Long: `Search across contacts, conversations, and message senders in parallel.

By default searches contacts and conversations. Use --type to limit to specific
resource types or to add senders.

Note: Sender search (--type senders) is not included by default because it
requires scanning messages across many conversations. Use --type senders
explicitly when searching for people in shared channels like LINE groups.

The "senders" type finds people who message through shared channels (e.g., LINE
groups) where their name appears in messages but not as a top-level contact.
This is useful when searching for "Jack" finds the person who messages through
a contact named "Welgrow Support".

This command is optimized for agent workflows, enabling quick discovery
of relevant resources with a single query.`,
		Example: `  # Search for "john" across all supported types
  cw search john

  # Search only contacts
  cw search john --type contacts

  # Search only conversations
  cw search "support issue" --type conversations

  # Search message senders (finds people in shared channels)
  cw search "Jack" --type senders

  # Search multiple types explicitly
  cw search john --type contacts --type conversations

  # Limit results per type
  cw search john --limit 10

  # JSON output for scripting
  cw search john --output json

  # Select a result and emit a typed JSON wrapper
  cw search john --select --output json

  # Select a result and emit the raw JSON object
  cw search john --select --select-raw --output json`,
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

			// Default to all types if none specified
			searchTypes := types
			if len(searchTypes) == 0 {
				searchTypes = []string{"contacts", "conversations"}
			}

			// Normalize and validate types
			typeAliases := map[string]string{
				"contacts":      "contacts",
				"contact":       "contacts",
				"ct":            "contacts",
				"conversations": "conversations",
				"conversation":  "conversations",
				"cv":            "conversations",
				"senders":       "senders",
				"sender":        "senders",
				"s":             "senders",
			}
			for i, t := range searchTypes {
				if normalized, ok := typeAliases[strings.ToLower(t)]; ok {
					searchTypes[i] = normalized
				} else {
					return fmt.Errorf("invalid type %q: must be one of contacts (ct), conversations (cv), senders (s)", t)
				}
			}

			// Create result struct
			results := SearchResults{
				Query:         query,
				Results:       []UnifiedSearchResult{},
				Contacts:      []api.Contact{},
				Conversations: []api.Conversation{},
				Senders:       []SenderMatch{},
				Summary:       make(map[string]int),
			}

			if best && selectOne {
				return fmt.Errorf("--best and --select cannot be used together")
			}
			if best && selectRaw {
				return fmt.Errorf("--best and --select-raw cannot be used together")
			}
			if best {
				if emit == "" {
					emit = "json"
				}
				emit = strings.ToLower(strings.TrimSpace(emit))
				switch emit {
				case "json", "id", "url":
				default:
					return fmt.Errorf("invalid --emit %q: must be one of json, id, url", emit)
				}
			} else if emit != "" {
				return fmt.Errorf("--emit requires --best")
			}

			// Search in parallel
			var wg sync.WaitGroup
			var mu sync.Mutex
			var searchErr error

			ctx := cmdContext(cmd)

			for _, searchType := range searchTypes {
				wg.Add(1)
				go func(st string) {
					defer wg.Done()

					switch st {
					case "contacts":
						// Chatwoot contacts search has fixed page size of 15.
						// We fetch enough pages to satisfy the limit, then truncate client-side.
						var allContacts []api.Contact
						page := 1
						for {
							select {
							case <-ctx.Done():
								return
							default:
							}
							contacts, err := client.Contacts().Search(ctx, query, page)
							if err != nil {
								mu.Lock()
								if searchErr == nil {
									searchErr = fmt.Errorf("failed to search contacts: %w", err)
								}
								mu.Unlock()
								return
							}
							allContacts = append(allContacts, contacts.Payload...)
							// Stop if we have enough results or no more pages
							if limit > 0 && len(allContacts) >= limit {
								break
							}
							if len(contacts.Payload) == 0 || int(contacts.Meta.CurrentPage) >= int(contacts.Meta.TotalPages) {
								break
							}
							page++
						}
						mu.Lock()
						// Apply limit
						if limit > 0 && len(allContacts) > limit {
							results.Contacts = allContacts[:limit]
						} else if allContacts != nil {
							results.Contacts = allContacts
						}
						results.Summary["contacts"] = len(results.Contacts)
						mu.Unlock()

					case "conversations":
						// Use List API with query param instead of Search API
						// This searches message content, not just metadata
						var allConversations []api.Conversation
						page := 1
						for {
							select {
							case <-ctx.Done():
								return
							default:
							}
							params := api.ListConversationsParams{
								Query: query,
								Page:  page,
							}
							result, err := client.Conversations().List(ctx, params)
							if err != nil {
								mu.Lock()
								if searchErr == nil {
									searchErr = fmt.Errorf("failed to search conversations: %w", err)
								}
								mu.Unlock()
								return
							}
							allConversations = append(allConversations, result.Data.Payload...)
							// Stop if we have enough results or no more pages
							if limit > 0 && len(allConversations) >= limit {
								break
							}
							totalPages := int(result.Data.Meta.TotalPages)
							if totalPages == 0 || page >= totalPages {
								break
							}
							page++
						}
						mu.Lock()
						if limit > 0 && len(allConversations) > limit {
							results.Conversations = allConversations[:limit]
						} else if allConversations != nil {
							results.Conversations = allConversations
						}
						results.Summary["conversations"] = len(results.Conversations)
						mu.Unlock()

					case "senders":
						// Search message senders within recent conversations
						// This finds people who message through shared channels (e.g., LINE groups)
						queryLower := strings.ToLower(query)

						// Fetch recent conversations (all statuses, sorted by activity)
						var conversations []api.Conversation
						page := 1
						maxConversations := 100 // Scan up to 100 recent conversations
						for {
							select {
							case <-ctx.Done():
								return
							default:
							}
							params := api.ListConversationsParams{
								Status: "all",
								Page:   page,
							}
							result, err := client.Conversations().List(ctx, params)
							if err != nil {
								mu.Lock()
								if searchErr == nil {
									searchErr = fmt.Errorf("failed to list conversations for sender search: %w", err)
								}
								mu.Unlock()
								return
							}
							conversations = append(conversations, result.Data.Payload...)
							if len(conversations) >= maxConversations {
								conversations = conversations[:maxConversations]
								break
							}
							totalPages := int(result.Data.Meta.TotalPages)
							if totalPages == 0 || page >= totalPages {
								break
							}
							page++
						}

						// Track unique senders by name+conversation to avoid duplicates
						type senderKey struct {
							name   string
							convID int
						}
						seenSenders := make(map[senderKey]bool)
						var senderMatches []SenderMatch

						// Scan messages in each conversation for matching sender names
						for _, conv := range conversations {
							select {
							case <-ctx.Done():
								return
							default:
							}

							// Fetch first page of messages (most recent)
							messages, err := client.Messages().List(ctx, conv.ID)
							if err != nil {
								continue // Skip conversations we can't read
							}

							// Track senders in this conversation
							type senderInfo struct {
								name          string
								lastMessageAt int64
								messageCount  int
							}
							convSenders := make(map[string]*senderInfo)

							for _, msg := range messages {
								// Check sender name from Sender field
								if msg.Sender != nil && msg.Sender.Name != "" {
									senderName := msg.Sender.Name
									senderNameLower := strings.ToLower(senderName)

									if strings.Contains(senderNameLower, queryLower) {
										if info, exists := convSenders[senderName]; exists {
											info.messageCount++
											if msg.CreatedAt > info.lastMessageAt {
												info.lastMessageAt = msg.CreatedAt
											}
										} else {
											convSenders[senderName] = &senderInfo{
												name:          senderName,
												lastMessageAt: msg.CreatedAt,
												messageCount:  1,
											}
										}
									}
								}

								// Also check for LINE-style bracketed names in content: "[Jack Su] message..."
								if matches := bracketedNameRegex.FindStringSubmatch(msg.Content); len(matches) > 1 {
									bracketedName := matches[1]
									bracketedNameLower := strings.ToLower(bracketedName)

									if strings.Contains(bracketedNameLower, queryLower) {
										if info, exists := convSenders[bracketedName]; exists {
											info.messageCount++
											if msg.CreatedAt > info.lastMessageAt {
												info.lastMessageAt = msg.CreatedAt
											}
										} else {
											convSenders[bracketedName] = &senderInfo{
												name:          bracketedName,
												lastMessageAt: msg.CreatedAt,
												messageCount:  1,
											}
										}
									}
								}
							}

							// Get contact name from conversation metadata
							contactName := ""
							contactID := 0
							if conv.Meta != nil {
								if sender, ok := conv.Meta["sender"].(map[string]any); ok {
									if name, ok := sender["name"].(string); ok {
										contactName = name
									}
									if id, ok := sender["id"].(float64); ok {
										contactID = int(id)
									}
								}
							}

							// Add unique senders to results
							for _, info := range convSenders {
								key := senderKey{name: info.name, convID: conv.ID}
								if seenSenders[key] {
									continue
								}
								seenSenders[key] = true

								senderMatches = append(senderMatches, SenderMatch{
									Name:           info.name,
									ContactID:      contactID,
									ContactName:    contactName,
									ConversationID: conv.ID,
									LastMessageAt:  info.lastMessageAt,
									MessageCount:   info.messageCount,
								})

								if limit > 0 && len(senderMatches) >= limit {
									break
								}
							}

							if limit > 0 && len(senderMatches) >= limit {
								break
							}
						}

						mu.Lock()
						if senderMatches != nil {
							results.Senders = senderMatches
						}
						results.Summary["senders"] = len(senderMatches)
						mu.Unlock()
					}
				}(searchType)
			}

			wg.Wait()

			if searchErr != nil {
				return searchErr
			}

			// Fetch message snippets if requested
			if includeSnippet && len(results.Conversations) > 0 {
				results.Snippets = make(map[string]SnippetInfo)
				for _, conv := range results.Conversations {
					messages, err := client.Messages().List(ctx, conv.ID)
					if err != nil {
						// Skip conversations where we can't fetch messages
						continue
					}
					if snippet, found := extractSnippet(messages, query); found {
						results.Snippets[fmt.Sprintf("%d", conv.ID)] = snippet
					}
				}
			}

			// Build unified results list sorted by last_activity_at descending
			var unified []UnifiedSearchResult
			for i := range results.Contacts {
				c := &results.Contacts[i]
				lastActivity := int64(0)
				if c.LastActivityAt != nil {
					lastActivity = *c.LastActivityAt
				}
				unified = append(unified, UnifiedSearchResult{
					Type:           "contact",
					ID:             c.ID,
					Name:           c.Name,
					LastActivityAt: lastActivity,
					Contact:        c,
				})
			}
			for i := range results.Conversations {
				conv := &results.Conversations[i]
				name := ""
				if conv.Meta != nil {
					if sender, ok := conv.Meta["sender"].(map[string]any); ok {
						if n, ok := sender["name"].(string); ok {
							name = n
						}
					}
				}
				unified = append(unified, UnifiedSearchResult{
					Type:           "conversation",
					ID:             conv.ID,
					Name:           name,
					LastActivityAt: conv.LastActivityAt,
					Conversation:   conv,
				})
			}
			for i := range results.Senders {
				s := &results.Senders[i]
				unified = append(unified, UnifiedSearchResult{
					Type:           "sender",
					ID:             s.ConversationID,
					Name:           s.Name,
					LastActivityAt: s.LastMessageAt,
					Sender:         s,
				})
			}
			// Sort by last_activity_at descending (most recent first)
			sort.Slice(unified, func(i, j int) bool {
				return unified[i].LastActivityAt > unified[j].LastActivityAt
			})
			if unified != nil {
				results.Results = unified
			}

			if best {
				if len(results.Results) == 0 {
					return fmt.Errorf("no results found")
				}
				bestResult := results.Results[0]

				bestID := bestResult.ID
				bestURL := ""
				switch bestResult.Type {
				case "contact":
					u, _ := resourceURL("contacts", bestID)
					bestURL = u
				case "conversation", "sender":
					u, _ := resourceURL("conversations", bestID)
					bestURL = u
				}

				if emit == "url" && bestURL == "" {
					return fmt.Errorf("cannot emit URL: base URL/account not configured")
				}

				// Emit plain scalar values for easy command chaining.
				if !isJSON(cmd) && !isAgent(cmd) {
					switch emit {
					case "id":
						_, _ = fmt.Fprintln(cmd.OutOrStdout(), bestID)
						return nil
					case "url":
						_, _ = fmt.Fprintln(cmd.OutOrStdout(), bestURL)
						return nil
					}
				}

				if isAgent(cmd) {
					type agentBest struct {
						Type string `json:"type"`
						ID   int    `json:"id"`
						URL  string `json:"url,omitempty"`
						Item any    `json:"item,omitempty"`
					}
					payload := agentfmt.ItemEnvelope{
						Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()) + ".best",
						Item: agentBest{Type: bestResult.Type, ID: bestID, URL: bestURL},
					}
					// Attach a compact item payload for known types.
					switch bestResult.Type {
					case "contact":
						if bestResult.Contact != nil {
							item := agentfmt.ContactDetailFromContact(*bestResult.Contact)
							payload.Item = agentBest{Type: bestResult.Type, ID: bestID, URL: bestURL, Item: item}
						}
					case "conversation":
						if bestResult.Conversation != nil {
							item := agentfmt.ConversationDetailFromConversation(*bestResult.Conversation)
							item = resolveConversationDetail(ctx, client, item)
							payload.Item = agentBest{Type: bestResult.Type, ID: bestID, URL: bestURL, Item: item}
						}
					case "sender":
						if bestResult.Sender != nil {
							payload.Item = agentBest{Type: bestResult.Type, ID: bestID, URL: bestURL, Item: bestResult.Sender}
						}
					}
					return printJSON(cmd, payload)
				}

				// JSON output (non-agent): a stable wrapper for tool chaining.
				if isJSON(cmd) {
					out := map[string]any{
						"type": bestResult.Type,
						"id":   bestID,
					}
					if bestURL != "" {
						out["url"] = bestURL
					}
					switch bestResult.Type {
					case "contact":
						out["item"] = bestResult.Contact
					case "conversation":
						out["item"] = bestResult.Conversation
					case "sender":
						out["item"] = bestResult.Sender
					}
					// For --emit id/url in JSON mode, return a wrapper plus scalar field.
					switch emit {
					case "id":
						out = map[string]any{"id": bestID}
					case "url":
						out = map[string]any{"url": bestURL}
					}
					return printJSON(cmd, out)
				}

				// Text output default.
				switch bestResult.Type {
				case "contact":
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[contact] #%d %s\n", bestID, bestResult.Name)
				case "conversation":
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[conv] #%d %s\n", bestID, bestResult.Name)
				case "sender":
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[sender] %s (conv #%d)\n", bestResult.Name, bestID)
				}
				if bestURL != "" {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", bestURL)
				}
				return nil
			}

			if selectOne {
				if flags.NoInput || !isInteractive() {
					return fmt.Errorf("--select requires interactive input (omit --select or run in a terminal)")
				}

				type selection struct {
					contact      *api.Contact
					conversation *api.Conversation
				}

				var options []selectOption
				selections := make(map[int]selection)
				nextID := 1

				for _, c := range results.Contacts {
					label := c.Name
					if label == "" {
						label = fmt.Sprintf("Contact %d", c.ID)
					}
					if c.Email != "" {
						label = fmt.Sprintf("%s <%s>", label, c.Email)
					}
					options = append(options, selectOption{ID: nextID, Label: label})
					cCopy := c
					selections[nextID] = selection{contact: &cCopy}
					nextID++
				}
				for _, conv := range results.Conversations {
					label := fmt.Sprintf("Conversation %d (%s, inbox %d)", conv.ID, conv.Status, conv.InboxID)
					options = append(options, selectOption{ID: nextID, Label: label})
					convCopy := conv
					selections[nextID] = selection{conversation: &convCopy}
					nextID++
				}

				if len(options) == 0 {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No results to select")
					return nil
				}

				selectedID, ok, err := promptSelect(ctx, "Select result", options, true)
				if err != nil {
					return err
				}
				if !ok {
					return nil
				}
				chosen := selections[selectedID]
				if chosen.contact != nil {
					if isJSON(cmd) {
						if selectRaw {
							return printRawJSON(cmd, chosen.contact)
						}
						if isAgent(cmd) {
							kind := agentfmt.KindFromCommandPath(cmd.CommandPath()) + ".select"
							item := agentfmt.ContactDetailFromContact(*chosen.contact)
							return printJSON(cmd, agentfmt.ItemEnvelope{
								Kind: kind,
								Item: map[string]any{
									"type": "contact",
									"item": item,
								},
							})
						}
						return printJSON(cmd, map[string]any{
							"type": "contact",
							"item": chosen.contact,
						})
					}
					return printContactDetails(cmd.OutOrStdout(), chosen.contact)
				}
				if chosen.conversation != nil {
					if isJSON(cmd) {
						if selectRaw {
							return printRawJSON(cmd, chosen.conversation)
						}
						if isAgent(cmd) {
							kind := agentfmt.KindFromCommandPath(cmd.CommandPath()) + ".select"
							item := agentfmt.ConversationDetailFromConversation(*chosen.conversation)
							item = resolveConversationDetail(ctx, client, item)
							return printJSON(cmd, agentfmt.ItemEnvelope{
								Kind: kind,
								Item: map[string]any{
									"type": "conversation",
									"item": item,
								},
							})
						}
						return printJSON(cmd, map[string]any{
							"type": "conversation",
							"item": chosen.conversation,
						})
					}
					return printConversationDetails(cmd.OutOrStdout(), chosen.conversation)
				}
				return nil
			}

			if light {
				cmd.SetContext(outfmt.WithLight(cmd.Context(), true))
				return printRawJSON(cmd, buildLightSearchPayload(results))
			}

			if isAgent(cmd) {
				type agentSearchResult struct {
					Type           string                        `json:"type"`
					ID             int                           `json:"id,omitempty"`
					LastActivityAt int64                         `json:"last_activity_at,omitempty"`
					Contact        *agentfmt.ContactSummary      `json:"contact,omitempty"`
					Conversation   *agentfmt.ConversationSummary `json:"conversation,omitempty"`
					Sender         *SenderMatch                  `json:"sender,omitempty"`
				}

				resultsList := make([]agentSearchResult, 0, len(results.Contacts)+len(results.Conversations)+len(results.Senders))
				for _, contact := range results.Contacts {
					summary := agentfmt.ContactSummaryFromContact(contact)
					lastActivity := int64(0)
					if contact.LastActivityAt != nil {
						lastActivity = *contact.LastActivityAt
					}
					resultsList = append(resultsList, agentSearchResult{
						Type:           "contact",
						ID:             contact.ID,
						LastActivityAt: lastActivity,
						Contact:        &summary,
					})
				}
				convSummaries := make([]agentfmt.ConversationSummary, len(results.Conversations))
				for i, conv := range results.Conversations {
					convSummaries[i] = agentfmt.ConversationSummaryFromConversation(conv)
				}
				convSummaries = resolveConversationSummaries(ctx, client, convSummaries)
				for i, conv := range results.Conversations {
					summary := convSummaries[i]
					resultsList = append(resultsList, agentSearchResult{
						Type:           "conversation",
						ID:             conv.ID,
						LastActivityAt: conv.LastActivityAt,
						Conversation:   &summary,
					})
				}
				for _, sender := range results.Senders {
					senderCopy := sender
					resultsList = append(resultsList, agentSearchResult{
						Type:           "sender",
						ID:             sender.ConversationID,
						LastActivityAt: sender.LastMessageAt,
						Sender:         &senderCopy,
					})
				}

				// Sort by last_activity_at descending (most recent first)
				sort.Slice(resultsList, func(i, j int) bool {
					return resultsList[i].LastActivityAt > resultsList[j].LastActivityAt
				})

				payload := agentfmt.SearchEnvelope{
					Kind:    agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Query:   query,
					Results: resultsList,
					Summary: map[string]int{"contacts": len(results.Contacts), "conversations": len(results.Conversations), "senders": len(results.Senders)},
				}
				return printJSON(cmd, payload)
			}

			if isJSON(cmd) {
				return printJSON(cmd, results)
			}

			// Text output - unified sorted list
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Search results for %q (sorted by recent activity):\n\n", query)

			if len(results.Results) == 0 {
				searchedTypes := strings.Join(searchTypes, ", ")
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No results found in %s\n", searchedTypes)
			} else {
				for _, r := range results.Results {
					switch r.Type {
					case "contact":
						email := r.Contact.Email
						if email == "" {
							email = "-"
						}
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  [contact]  #%-6d %s <%s>\n", r.ID, r.Name, email)
					case "conversation":
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  [conv]     #%-6d %s [%s]\n", r.ID, r.Name, r.Conversation.Status)
					case "sender":
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  [sender]   %s → %s (conv #%d)\n", r.Name, r.Sender.ContactName, r.ID)
					}
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nTotal: %d contacts, %d conversations, %d senders\n",
					len(results.Contacts), len(results.Conversations), len(results.Senders))
			}

			return nil
		}),
	}

	cmd.Flags().StringArrayVarP(&types, "type", "t", nil, "Resource types to search (contacts, conversations, senders); repeatable")
	cmd.Flags().IntVarP(&limit, "limit", "l", 25, "Maximum results per type")
	cmd.Flags().BoolVar(&selectOne, "select", false, "Interactively select a single result")
	flagAlias(cmd.Flags(), "select", "sel")
	cmd.Flags().BoolVar(&selectRaw, "select-raw", false, "Emit raw selected object in JSON output (no wrapper)")
	flagAlias(cmd.Flags(), "select-raw", "sr")
	cmd.Flags().BoolVar(&includeSnippet, "include-snippet", false, "Include matching message snippet for conversations")
	flagAlias(cmd.Flags(), "include-snippet", "snippet")
	flagAlias(cmd.Flags(), "include-snippet", "sn")
	cmd.Flags().BoolVar(&best, "best", false, "Auto-select the best result (no interactive prompt)")
	flagAlias(cmd.Flags(), "best", "b")
	cmd.Flags().StringVarP(&emit, "emit", "E", "", "Output format with --best: json (default), id, or url")
	cmd.Flags().BoolVar(&light, "light", false, "Return minimal search payload for lookup")
	flagAlias(cmd.Flags(), "light", "li")

	return cmd
}

// extractSnippet searches messages for the query and returns a snippet with context.
// It finds the first message containing the query (case-insensitive) and extracts
// approximately 20 runes before and 50 runes after the match, adding "..." if truncated.
// Uses rune-based slicing to safely handle multi-byte UTF-8 characters.
func extractSnippet(messages []api.Message, query string) (SnippetInfo, bool) {
	queryLower := strings.ToLower(query)
	queryRuneLen := len([]rune(query))

	for _, msg := range messages {
		content := msg.Content
		contentLower := strings.ToLower(content)

		byteIdx := strings.Index(contentLower, queryLower)
		if byteIdx == -1 {
			continue
		}

		// Convert content to runes for safe slicing
		runes := []rune(content)
		runeCount := len(runes)

		// Convert byte index to rune index by counting runes in the prefix
		runeIdx := len([]rune(content[:byteIdx]))

		// Calculate snippet bounds (~20 runes before, ~50 runes after)
		const (
			contextBefore = 20
			contextAfter  = 50
		)

		start := runeIdx - contextBefore
		end := runeIdx + queryRuneLen + contextAfter

		// Adjust bounds to stay within content
		prefix := ""
		suffix := ""

		if start < 0 {
			start = 0
		} else {
			prefix = "..."
		}

		if end > runeCount {
			end = runeCount
		} else {
			suffix = "..."
		}

		snippet := prefix + string(runes[start:end]) + suffix

		return SnippetInfo{
			MessageID: msg.ID,
			Content:   snippet,
			CreatedAt: msg.CreatedAt,
		}, true
	}

	return SnippetInfo{}, false
}
