package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/iocontext"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
)

// ListResult represents the result of a paginated list operation
type ListResult[T any] struct {
	Items   []T
	HasMore bool
}

// ListSummary describes the rendered list output.
type ListSummary struct {
	Page         int
	PageSize     int
	PagesFetched int
	TotalItems   int
	HasMore      bool
	All          bool
}

// ListConfig defines how a list command behaves
type ListConfig[T any] struct {
	Use          string
	Short        string
	Long         string
	Example      string
	Fetch        func(ctx context.Context, client *api.Client, page, pageSize int) (ListResult[T], error)
	Headers      []string
	RowFunc      func(T) []string
	EmptyMessage string
	// DisablePagination prevents adding page/limit flags for list commands without pagination.
	DisablePagination bool
	// DisableLimit prevents adding the --limit flag (useful when the API doesn't support page size).
	DisableLimit bool
	// DefaultPage overrides the default page flag value (defaults to 1).
	DefaultPage int
	// DefaultLimit overrides the default limit flag value (defaults to 20).
	DefaultLimit int
	// MinLimit overrides the minimum limit value (defaults to 10).
	MinLimit int
	// DefaultMaxPages overrides the default max-pages flag value (defaults to 100).
	DefaultMaxPages int
	// AfterOutput runs after table output (text mode only).
	AfterOutput func(cmd *cobra.Command, summary ListSummary) error
	// AgentTransform overrides agent-mode item transformation.
	AgentTransform func(ctx context.Context, client *api.Client, items []T) (any, error)
	// JSONTransform overrides JSON-mode item transformation.
	JSONTransform func(ctx context.Context, client *api.Client, items []T) (any, error)
	// ForceJSON allows command-specific flags to force JSON output semantics.
	// This is evaluated after flags are parsed and before rendering.
	ForceJSON func(cmd *cobra.Command) bool
	// StripPagination removes has_more/meta from JSON/agent output payloads.
	StripPagination bool
	// ForceJSONUnwrapItems outputs the raw items array/object instead of an
	// {"items": ...} envelope when ForceJSON is active.
	ForceJSONUnwrapItems bool
}

func writeJSONLItem(w io.Writer, item any, query, tmpl string, light bool) error {
	if query != "" {
		var (
			filtered any
			err      error
		)
		if light {
			filtered, err = outfmt.ApplyQueryLiteral(item, query)
		} else {
			filtered, err = outfmt.ApplyQuery(item, query)
		}
		if err != nil {
			return err
		}
		item = filtered
	}

	if tmpl != "" {
		if err := outfmt.WriteTemplate(w, item, tmpl); err != nil {
			return err
		}
		_, err := fmt.Fprintln(w)
		return err
	}

	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	_, err = w.Write([]byte("\n"))
	return err
}

// NewListCommand creates a cobra command from ListConfig
func NewListCommand[T any](cfg ListConfig[T], getClient func(context.Context) (*api.Client, error)) *cobra.Command {
	var page int
	var pageSize int
	var all bool
	var maxPages int

	defaultPage := cfg.DefaultPage
	if defaultPage == 0 {
		defaultPage = 1
	}
	defaultLimit := cfg.DefaultLimit
	if defaultLimit == 0 {
		defaultLimit = 20
	}
	minLimit := cfg.MinLimit
	if minLimit == 0 {
		minLimit = 10
	}
	defaultMaxPages := cfg.DefaultMaxPages
	if defaultMaxPages == 0 {
		defaultMaxPages = 100
	}

	cmd := &cobra.Command{
		Use:     cfg.Use,
		Short:   cfg.Short,
		Long:    cfg.Long,
		Example: cfg.Example,
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if !cfg.DisablePagination {
				if page < 1 {
					return fmt.Errorf("page must be >= 1")
				}
				if pageSize < minLimit {
					pageSize = minLimit
				}
				if all && maxPages < 1 {
					return fmt.Errorf("max-pages must be >= 1")
				}
			} else {
				page = 1
				pageSize = defaultLimit
			}

			ctx := cmd.Context()
			mode := outfmt.ModeFromContext(ctx)
			forceJSON := cfg.ForceJSON != nil && cfg.ForceJSON(cmd)
			if forceJSON {
				// ForceJSON paths (for example --light) should always preserve literal
				// short-key jq behavior, even when mode is already JSON.
				ctx = outfmt.WithLight(ctx, true)
				if mode == outfmt.Text {
					ctx = outfmt.WithMode(ctx, outfmt.JSON)
					mode = outfmt.JSON
				}
				cmd.SetContext(ctx)
			}

			client, err := getClient(ctx)
			if err != nil {
				return err
			}

			ioStreams := iocontext.GetIO(ctx)
			f := outfmt.NewFormatter(ctx, ioStreams.Out, ioStreams.ErrOut)

			if cfg.DisablePagination || !all {
				result, err := cfg.Fetch(ctx, client, page, pageSize)
				if err != nil {
					return err
				}

				if mode == outfmt.JSONL {
					query := outfmt.GetQuery(ctx)
					tmpl := outfmt.GetTemplate(ctx)
					light := outfmt.IsLight(ctx)
					for _, item := range result.Items {
						if err := writeJSONLItem(ioStreams.Out, item, query, tmpl, light); err != nil {
							return err
						}
					}
					return nil
				}

				if mode == outfmt.JSON || mode == outfmt.Agent {
					summaryPageSize := pageSize
					if cfg.DisablePagination {
						summaryPageSize = len(result.Items)
					}
					items := result.Items
					if items == nil {
						items = make([]T, 0)
					}
					meta := map[string]any{
						"page":          page,
						"page_size":     summaryPageSize,
						"pages_fetched": 1,
						"total_items":   len(items),
						"all":           false,
					}
					payload := map[string]interface{}{
						"items": items,
					}
					if !cfg.StripPagination {
						addRateLimitMeta(meta, client)
						payload["has_more"] = result.HasMore
						payload["meta"] = meta
					}
					if mode == outfmt.Agent {
						payload["kind"] = agentfmt.KindFromCommandPath(cmd.CommandPath())
						if cfg.AgentTransform != nil {
							agentItems, err := cfg.AgentTransform(ctx, client, items)
							if err != nil {
								return err
							}
							payload["items"] = agentItems
						} else {
							payload["items"] = agentfmt.TransformListItems(items)
						}
					} else if cfg.JSONTransform != nil {
						jsonItems, err := cfg.JSONTransform(ctx, client, items)
						if err != nil {
							return err
						}
						payload["items"] = jsonItems
					}
					// When ForceJSON is active (e.g. --light), optionally strip
					// pagination metadata and/or unwrap items for raw JSON output.
					if forceJSON {
						if !cfg.StripPagination {
							delete(payload, "has_more")
							delete(payload, "meta")
						}
					}
					if cfg.ForceJSONUnwrapItems && (forceJSON || outfmt.IsLight(ctx)) {
						raw, err := json.Marshal(payload["items"])
						if err != nil {
							return err
						}
						return f.Output(json.RawMessage(raw))
					}
					return f.Output(payload)
				}

				if len(result.Items) == 0 {
					if cfg.EmptyMessage != "" {
						f.Empty(cfg.EmptyMessage)
					}
					return nil
				}

				f.StartTable(cfg.Headers)
				for _, item := range result.Items {
					f.Row(cfg.RowFunc(item)...)
				}
				if err := f.EndTable(); err != nil {
					return err
				}

				if cfg.AfterOutput != nil {
					if err := cfg.AfterOutput(cmd, ListSummary{
						Page:         page,
						PageSize:     pageSize,
						PagesFetched: 1,
						TotalItems:   len(result.Items),
						HasMore:      result.HasMore,
						All:          false,
					}); err != nil {
						return err
					}
				}

				if result.HasMore {
					_, _ = fmt.Fprintln(ioStreams.ErrOut, "# More results available")
				}
				return nil
			}

			if mode == outfmt.JSONL {
				query := outfmt.GetQuery(ctx)
				tmpl := outfmt.GetTemplate(ctx)
				light := outfmt.IsLight(ctx)
				currentPage := page
				pagesFetched := 0
				totalItems := 0

				for {
					if maxPages > 0 && pagesFetched >= maxPages {
						return fmt.Errorf("safety limit reached: fetched %d pages (%d items). Use --max-pages to increase the limit", maxPages, totalItems)
					}

					result, err := cfg.Fetch(ctx, client, currentPage, pageSize)
					if err != nil {
						return err
					}
					if len(result.Items) == 0 {
						break
					}
					for _, item := range result.Items {
						if err := writeJSONLItem(ioStreams.Out, item, query, tmpl, light); err != nil {
							return err
						}
						totalItems++
					}
					pagesFetched++

					if !result.HasMore {
						break
					}

					currentPage++
				}
				return nil
			}

			if mode == outfmt.Text {
				currentPage := page
				pagesFetched := 0
				totalItems := 0
				started := false

				for {
					if maxPages > 0 && pagesFetched >= maxPages {
						return fmt.Errorf("safety limit reached: fetched %d pages (%d items). Use --max-pages to increase the limit", maxPages, totalItems)
					}

					if currentPage > page && !flags.Quiet && !flags.Silent {
						_, _ = fmt.Fprintf(ioStreams.ErrOut, "Fetching page %d...\n", currentPage) //nolint:errcheck
					}

					result, err := cfg.Fetch(ctx, client, currentPage, pageSize)
					if err != nil {
						return err
					}
					if len(result.Items) == 0 {
						break
					}
					if !started {
						f.StartTable(cfg.Headers)
						started = true
					}
					for _, item := range result.Items {
						f.Row(cfg.RowFunc(item)...)
						totalItems++
					}
					pagesFetched++

					if !result.HasMore {
						break
					}

					currentPage++
				}

				if !started {
					if cfg.EmptyMessage != "" {
						f.Empty(cfg.EmptyMessage)
					}
					return nil
				}

				if err := f.EndTable(); err != nil {
					return err
				}

				if cfg.AfterOutput != nil {
					return cfg.AfterOutput(cmd, ListSummary{
						Page:         page,
						PageSize:     pageSize,
						PagesFetched: pagesFetched,
						TotalItems:   totalItems,
						HasMore:      false,
						All:          true,
					})
				}
				return nil
			}

			allItems := make([]T, 0)
			currentPage := page
			pagesFetched := 0

			for {
				if maxPages > 0 && pagesFetched >= maxPages {
					return fmt.Errorf("safety limit reached: fetched %d pages (%d items). Use --max-pages to increase the limit", maxPages, len(allItems))
				}

				result, err := cfg.Fetch(ctx, client, currentPage, pageSize)
				if err != nil {
					return err
				}
				if len(result.Items) == 0 {
					break
				}
				allItems = append(allItems, result.Items...)
				pagesFetched++

				if !result.HasMore {
					break
				}

				currentPage++
			}

			summaryPageSize := pageSize
			if cfg.DisablePagination {
				summaryPageSize = len(allItems)
			}
			meta := map[string]any{
				"page":          page,
				"page_size":     summaryPageSize,
				"pages_fetched": pagesFetched,
				"total_items":   len(allItems),
				"all":           true,
			}
			payload := map[string]interface{}{
				"items": allItems,
			}
			if !cfg.StripPagination {
				addRateLimitMeta(meta, client)
				payload["has_more"] = false
				payload["meta"] = meta
			}
			if mode == outfmt.Agent {
				payload["kind"] = agentfmt.KindFromCommandPath(cmd.CommandPath())
				if cfg.AgentTransform != nil {
					agentItems, err := cfg.AgentTransform(ctx, client, allItems)
					if err != nil {
						return err
					}
					payload["items"] = agentItems
				} else {
					payload["items"] = agentfmt.TransformListItems(allItems)
				}
			} else if cfg.JSONTransform != nil {
				jsonItems, err := cfg.JSONTransform(ctx, client, allItems)
				if err != nil {
					return err
				}
				payload["items"] = jsonItems
			}
			// When ForceJSON is active (e.g. --light), optionally strip
			// pagination metadata and/or unwrap items for raw JSON output.
			if forceJSON {
				if !cfg.StripPagination {
					delete(payload, "has_more")
					delete(payload, "meta")
				}
			}
			if cfg.ForceJSONUnwrapItems && (forceJSON || outfmt.IsLight(ctx)) {
				raw, err := json.Marshal(payload["items"])
				if err != nil {
					return err
				}
				return f.Output(json.RawMessage(raw))
			}
			return f.Output(payload)
		}),
	}

	if !cfg.DisablePagination {
		cmd.Flags().IntVarP(&page, "page", "p", defaultPage, "Page number")
		if !cfg.DisableLimit {
			cmd.Flags().IntVarP(&pageSize, "limit", "l", defaultLimit, fmt.Sprintf("Max results (min %d)", minLimit))
		} else {
			pageSize = defaultLimit
		}
		cmd.Flags().BoolVarP(&all, "all", "a", false, "Fetch all pages")
		cmd.Flags().IntVarP(&maxPages, "max-pages", "M", defaultMaxPages, "Maximum number of pages to fetch when using --all")
		flagAlias(cmd.Flags(), "max-pages", "mp")
	} else {
		page = 1
		pageSize = defaultLimit
	}
	return cmd
}

func addRateLimitMeta(meta map[string]any, client *api.Client) {
	if meta == nil || client == nil {
		return
	}
	info := client.LastRateLimit()
	if info == nil {
		return
	}
	if rateMeta := info.Meta(); rateMeta != nil {
		meta["rate_limit"] = rateMeta
	}
}
