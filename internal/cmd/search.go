package cmd

import (
	"fmt"
	"strings"
	"sync"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/spf13/cobra"
)

// SearchResults represents the combined search results from multiple resource types
type SearchResults struct {
	Query         string             `json:"query"`
	Contacts      []api.Contact      `json:"contacts,omitempty"`
	Conversations []api.Conversation `json:"conversations,omitempty"`
	Summary       map[string]int     `json:"summary"`
}

func newSearchCmd() *cobra.Command {
	var (
		types []string
		limit int
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search across multiple resources",
		Long: `Search across contacts and conversations in parallel.

By default searches both contacts and conversations. Use --type to limit
to specific resource types.

This command is optimized for agent workflows, enabling quick discovery
of relevant resources with a single query.`,
		Example: `  # Search for "john" across all supported types
  chatwoot search john

  # Search only contacts
  chatwoot search john --type contacts

  # Search only conversations
  chatwoot search "support issue" --type conversations

  # Search multiple types explicitly
  chatwoot search john --type contacts --type conversations

  # Limit results per type
  chatwoot search john --limit 10

  # JSON output for scripting
  chatwoot search john --output json`,
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

			// Validate types
			validTypes := map[string]bool{
				"contacts":      true,
				"conversations": true,
			}
			for _, t := range searchTypes {
				if !validTypes[t] {
					return fmt.Errorf("invalid type %q: must be one of contacts, conversations", t)
				}
			}

			// Create result struct
			results := SearchResults{
				Query:   query,
				Summary: make(map[string]int),
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
							contacts, err := client.SearchContacts(ctx, query, page)
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
						} else {
							results.Contacts = allContacts
						}
						results.Summary["contacts"] = len(results.Contacts)
						mu.Unlock()

					case "conversations":
						// Chatwoot conversations search has fixed page size of 25.
						// We fetch enough pages to satisfy the limit, then truncate client-side.
						var allConversations []api.Conversation
						page := 1
						for {
							conversations, err := client.SearchConversations(ctx, query, page)
							if err != nil {
								mu.Lock()
								if searchErr == nil {
									searchErr = fmt.Errorf("failed to search conversations: %w", err)
								}
								mu.Unlock()
								return
							}
							allConversations = append(allConversations, conversations.Data.Payload...)
							// Stop if we have enough results or no more pages
							if limit > 0 && len(allConversations) >= limit {
								break
							}
							if len(conversations.Data.Payload) == 0 || int(conversations.Data.Meta.CurrentPage) >= int(conversations.Data.Meta.TotalPages) {
								break
							}
							page++
						}
						mu.Lock()
						// Apply limit
						if limit > 0 && len(allConversations) > limit {
							results.Conversations = allConversations[:limit]
						} else {
							results.Conversations = allConversations
						}
						results.Summary["conversations"] = len(results.Conversations)
						mu.Unlock()
					}
				}(searchType)
			}

			wg.Wait()

			if searchErr != nil {
				return searchErr
			}

			if isJSON(cmd) {
				return printJSON(cmd, results)
			}

			// Text output
			fmt.Printf("Search results for %q:\n\n", query)

			if len(results.Contacts) > 0 {
				fmt.Printf("Contacts (%d):\n", len(results.Contacts))
				for _, c := range results.Contacts {
					email := c.Email
					if email == "" {
						email = "-"
					}
					fmt.Printf("  #%-6d %s <%s>\n", c.ID, c.Name, email)
				}
				fmt.Println()
			}

			if len(results.Conversations) > 0 {
				fmt.Printf("Conversations (%d):\n", len(results.Conversations))
				for _, conv := range results.Conversations {
					fmt.Printf("  #%-6d [%s] inbox:%d\n", conv.ID, conv.Status, conv.InboxID)
				}
				fmt.Println()
			}

			// Show empty message if no results
			total := len(results.Contacts) + len(results.Conversations)
			if total == 0 {
				searchedTypes := strings.Join(searchTypes, ", ")
				fmt.Printf("No results found in %s\n", searchedTypes)
			}

			return nil
		}),
	}

	cmd.Flags().StringArrayVar(&types, "type", nil, "Resource types to search (contacts, conversations); repeatable")
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum results per type")

	return cmd
}
