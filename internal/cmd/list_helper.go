package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
)

// ListResult represents the result of a paginated list operation
type ListResult[T any] struct {
	Items   []T
	HasMore bool
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
}

// NewListCommand creates a cobra command from ListConfig
func NewListCommand[T any](cfg ListConfig[T], getClient func(context.Context) (*api.Client, error)) *cobra.Command {
	var page int
	var pageSize int

	cmd := &cobra.Command{
		Use:     cfg.Use,
		Short:   cfg.Short,
		Long:    cfg.Long,
		Example: cfg.Example,
		RunE: func(cmd *cobra.Command, args []string) error {
			if pageSize < 10 {
				pageSize = 10
			}

			client, err := getClient(cmd.Context())
			if err != nil {
				return err
			}

			result, err := cfg.Fetch(cmd.Context(), client, page, pageSize)
			if err != nil {
				return err
			}

			f := outfmt.NewFormatter(cmd.Context(), os.Stdout, os.Stderr)

			if isJSON(cmd) {
				return f.Output(map[string]interface{}{
					"items":    result.Items,
					"has_more": result.HasMore,
				})
			}

			if len(result.Items) == 0 {
				f.Empty(cfg.EmptyMessage)
				return nil
			}

			f.StartTable(cfg.Headers)
			for _, item := range result.Items {
				f.Row(cfg.RowFunc(item)...)
			}
			if err := f.EndTable(); err != nil {
				return err
			}

			if result.HasMore {
				fmt.Fprintln(os.Stderr, "# More results available")
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&pageSize, "limit", 20, "Max results (min 10)")
	return cmd
}
