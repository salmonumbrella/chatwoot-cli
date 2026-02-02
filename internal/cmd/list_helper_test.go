package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/iocontext"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
)

type testItem struct {
	ID   int
	Name string
}

func TestListConfig_NewListCommand(t *testing.T) {
	cfg := ListConfig[testItem]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID", "NAME"},
		RowFunc: func(item testItem) []string {
			return []string{fmt.Sprintf("%d", item.ID), item.Name}
		},
		EmptyMessage: "No items found",
		Fetch: func(ctx context.Context, client *api.Client, page, pageSize int) (ListResult[testItem], error) {
			return ListResult[testItem]{
				Items:   []testItem{{ID: 1, Name: "test"}},
				HasMore: false,
			}, nil
		},
	}

	cmd := NewListCommand(cfg, func(ctx context.Context) (*api.Client, error) {
		return nil, nil
	})

	if cmd.Use != "list" {
		t.Errorf("expected Use='list', got %s", cmd.Use)
	}
}

func TestListCommand_UsesContextErrOut(t *testing.T) {
	cfg := ListConfig[testItem]{
		Use:          "list",
		Short:        "List items",
		Headers:      []string{"ID", "NAME"},
		RowFunc:      func(item testItem) []string { return []string{fmt.Sprintf("%d", item.ID), item.Name} },
		EmptyMessage: "No items found",
		Fetch: func(ctx context.Context, client *api.Client, page, pageSize int) (ListResult[testItem], error) {
			return ListResult[testItem]{Items: nil, HasMore: false}, nil
		},
	}

	cmd := NewListCommand(cfg, func(ctx context.Context) (*api.Client, error) { return nil, nil })

	var errBuf bytes.Buffer
	ctx := outfmt.WithMode(context.Background(), outfmt.Text)
	ctx = iocontext.WithIO(ctx, &iocontext.IO{Out: ioDiscard{}, ErrOut: &errBuf, In: nil})
	cmd.SetContext(ctx)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(errBuf.String(), "No items found") {
		t.Fatalf("expected empty message in ErrOut, got %q", errBuf.String())
	}
}

func TestListCommand_JSONOutput(t *testing.T) {
	cfg := ListConfig[testItem]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID", "NAME"},
		RowFunc: func(item testItem) []string { return []string{fmt.Sprintf("%d", item.ID), item.Name} },
		Fetch: func(ctx context.Context, client *api.Client, page, pageSize int) (ListResult[testItem], error) {
			return ListResult[testItem]{Items: []testItem{{ID: 1, Name: "test"}}, HasMore: true}, nil
		},
	}

	cmd := NewListCommand(cfg, func(ctx context.Context) (*api.Client, error) { return nil, nil })

	var out bytes.Buffer
	ctx := outfmt.WithMode(context.Background(), outfmt.JSON)
	ctx = iocontext.WithIO(ctx, &iocontext.IO{Out: &out, ErrOut: ioDiscard{}, In: nil})
	cmd.SetContext(ctx)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if _, ok := payload["items"]; !ok {
		t.Fatalf("expected JSON output to contain items, got %v", payload)
	}
	if _, ok := payload["has_more"]; !ok {
		t.Fatalf("expected JSON output to contain has_more, got %v", payload)
	}
	meta, ok := payload["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected JSON output to contain meta, got %v", payload)
	}
	if _, ok := meta["page"]; !ok {
		t.Fatalf("expected meta to contain page, got %v", meta)
	}
	if _, ok := meta["page_size"]; !ok {
		t.Fatalf("expected meta to contain page_size, got %v", meta)
	}
	if _, ok := meta["pages_fetched"]; !ok {
		t.Fatalf("expected meta to contain pages_fetched, got %v", meta)
	}
	if _, ok := meta["total_items"]; !ok {
		t.Fatalf("expected meta to contain total_items, got %v", meta)
	}
}

func TestListCommand_JSONOutput_IncludesRateLimitMeta(t *testing.T) {
	cfg := ListConfig[testItem]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID", "NAME"},
		RowFunc: func(item testItem) []string { return []string{fmt.Sprintf("%d", item.ID), item.Name} },
		Fetch: func(ctx context.Context, client *api.Client, page, pageSize int) (ListResult[testItem], error) {
			return ListResult[testItem]{Items: []testItem{{ID: 1, Name: "test"}}, HasMore: false}, nil
		},
	}

	client := api.New("https://example.com", "", 1)
	limit := 100
	remaining := 42
	resetAt := time.Date(2026, 2, 2, 15, 30, 0, 0, time.UTC)
	client.SetRateLimitInfo(&api.RateLimitInfo{
		Limit:     &limit,
		Remaining: &remaining,
		ResetAt:   &resetAt,
	})

	cmd := NewListCommand(cfg, func(ctx context.Context) (*api.Client, error) { return client, nil })

	var out bytes.Buffer
	ctx := outfmt.WithMode(context.Background(), outfmt.JSON)
	ctx = iocontext.WithIO(ctx, &iocontext.IO{Out: &out, ErrOut: ioDiscard{}, In: nil})
	cmd.SetContext(ctx)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	meta, ok := payload["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected meta in payload, got %v", payload)
	}
	rateLimit, ok := meta["rate_limit"].(map[string]any)
	if !ok {
		t.Fatalf("expected rate_limit metadata, got %v", meta)
	}
	if rateLimit["limit"] != float64(limit) {
		t.Fatalf("expected limit %d, got %v", limit, rateLimit["limit"])
	}
	if rateLimit["remaining"] != float64(remaining) {
		t.Fatalf("expected remaining %d, got %v", remaining, rateLimit["remaining"])
	}
	if rateLimit["reset_at"] != resetAt.Format(time.RFC3339) {
		t.Fatalf("expected reset_at %s, got %v", resetAt.Format(time.RFC3339), rateLimit["reset_at"])
	}
}

func TestListCommand_JSONOutput_DisablePaginationUsesItemCount(t *testing.T) {
	cfg := ListConfig[testItem]{
		Use:               "list",
		Short:             "List items",
		DisablePagination: true,
		Headers:           []string{"ID", "NAME"},
		RowFunc:           func(item testItem) []string { return []string{fmt.Sprintf("%d", item.ID), item.Name} },
		Fetch: func(ctx context.Context, client *api.Client, page, pageSize int) (ListResult[testItem], error) {
			return ListResult[testItem]{Items: []testItem{{ID: 1, Name: "test"}, {ID: 2, Name: "two"}}, HasMore: false}, nil
		},
	}

	cmd := NewListCommand(cfg, func(ctx context.Context) (*api.Client, error) { return nil, nil })

	var out bytes.Buffer
	ctx := outfmt.WithMode(context.Background(), outfmt.JSON)
	ctx = iocontext.WithIO(ctx, &iocontext.IO{Out: &out, ErrOut: ioDiscard{}, In: nil})
	cmd.SetContext(ctx)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	meta, ok := payload["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected JSON output to contain meta, got %v", payload)
	}
	if meta["page_size"] != float64(2) {
		t.Fatalf("expected page_size to match item count, got %v", meta["page_size"])
	}
}

func TestListCommand_JSONLOutput(t *testing.T) {
	cfg := ListConfig[testItem]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID", "NAME"},
		RowFunc: func(item testItem) []string { return []string{fmt.Sprintf("%d", item.ID), item.Name} },
		Fetch: func(ctx context.Context, client *api.Client, page, pageSize int) (ListResult[testItem], error) {
			return ListResult[testItem]{Items: []testItem{{ID: 1, Name: "one"}, {ID: 2, Name: "two"}}, HasMore: false}, nil
		},
	}

	cmd := NewListCommand(cfg, func(ctx context.Context) (*api.Client, error) { return nil, nil })

	var out bytes.Buffer
	ctx := outfmt.WithMode(context.Background(), outfmt.JSONL)
	ctx = iocontext.WithIO(ctx, &iocontext.IO{Out: &out, ErrOut: ioDiscard{}, In: nil})
	cmd.SetContext(ctx)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 JSONL lines, got %d: %q", len(lines), out.String())
	}
	var item1 testItem
	if err := json.Unmarshal([]byte(lines[0]), &item1); err != nil {
		t.Fatalf("failed to parse first JSONL line: %v", err)
	}
	if item1.ID != 1 {
		t.Fatalf("expected first item id 1, got %v", item1.ID)
	}
}

func TestListCommand_AllPagesFetchesMultiplePages(t *testing.T) {
	var calls []int
	cfg := ListConfig[testItem]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID", "NAME"},
		RowFunc: func(item testItem) []string { return []string{fmt.Sprintf("%d", item.ID), item.Name} },
		Fetch: func(ctx context.Context, client *api.Client, page, pageSize int) (ListResult[testItem], error) {
			calls = append(calls, page)
			switch page {
			case 1:
				return ListResult[testItem]{Items: []testItem{{ID: 1, Name: "first"}}, HasMore: true}, nil
			case 2:
				return ListResult[testItem]{Items: []testItem{{ID: 2, Name: "second"}}, HasMore: false}, nil
			default:
				return ListResult[testItem]{Items: nil, HasMore: false}, nil
			}
		},
	}

	cmd := NewListCommand(cfg, func(ctx context.Context) (*api.Client, error) { return nil, nil })
	_ = cmd.Flags().Set("all", "true")

	ctx := outfmt.WithMode(context.Background(), outfmt.JSON)
	ctx = iocontext.WithIO(ctx, &iocontext.IO{Out: ioDiscard{}, ErrOut: ioDiscard{}, In: nil})
	cmd.SetContext(ctx)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 2 || calls[0] != 1 || calls[1] != 2 {
		t.Fatalf("expected fetch to be called for pages 1 and 2, got %v", calls)
	}
}

func TestListCommand_AllPagesRespectsMaxPages(t *testing.T) {
	cfg := ListConfig[testItem]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID", "NAME"},
		RowFunc: func(item testItem) []string { return []string{fmt.Sprintf("%d", item.ID), item.Name} },
		Fetch: func(ctx context.Context, client *api.Client, page, pageSize int) (ListResult[testItem], error) {
			return ListResult[testItem]{Items: []testItem{{ID: page, Name: "item"}}, HasMore: true}, nil
		},
	}

	cmd := NewListCommand(cfg, func(ctx context.Context) (*api.Client, error) { return nil, nil })
	_ = cmd.Flags().Set("all", "true")
	_ = cmd.Flags().Set("max-pages", "1")

	ctx := outfmt.WithMode(context.Background(), outfmt.JSON)
	ctx = iocontext.WithIO(ctx, &iocontext.IO{Out: ioDiscard{}, ErrOut: ioDiscard{}, In: nil})
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "safety limit reached") {
		t.Fatalf("expected safety limit error, got %v", err)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
