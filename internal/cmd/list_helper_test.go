package cmd

import (
	"context"
	"fmt"
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/api"
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
