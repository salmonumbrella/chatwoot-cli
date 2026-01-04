package cmd

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

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

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
