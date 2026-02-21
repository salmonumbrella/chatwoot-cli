package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newContactsBulkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bulk",
		Aliases: []string{"bk"},
		Short:   "Bulk operations on contacts",
		Long:    "Perform bulk operations on multiple contacts at once",
	}

	cmd.AddCommand(newContactsBulkAddLabelCmd())
	cmd.AddCommand(newContactsBulkRemoveLabelCmd())

	return cmd
}

func newContactsBulkAddLabelCmd() *cobra.Command {
	var (
		contactIDs  string
		labels      string
		concurrency int
		progress    bool
		noProgress  bool
	)

	cmd := &cobra.Command{
		Use:     "add-label",
		Aliases: []string{"add"},
		Short:   "Add labels to multiple contacts",
		Long:    "Add one or more labels to multiple contacts at once",
		Example: strings.TrimSpace(`
  # Add a single label to multiple contacts
  cw contacts bulk add-label --ids 1,2,3 --labels important

  # Add multiple labels to multiple contacts
  cw contacts bulk add-label --ids 1,2,3 --labels important,vip

  # Control concurrency (default: 5)
  cw contacts bulk add-label --ids 1,2,3 --labels vip --concurrency 10
`),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			ids, err := ParseResourceIDListFlag(contactIDs, "contact")
			if err != nil {
				return fmt.Errorf("invalid contact IDs: %w", err)
			}

			labelList, err := ParseStringListFlag(labels)
			if err != nil {
				return fmt.Errorf("invalid labels: %w", err)
			}
			if len(labelList) == 0 {
				return fmt.Errorf("no valid labels provided")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			results := runBulkOperation(
				ctx,
				ids,
				int64(concurrency),
				bulkProgressEnabled(cmd, progress, noProgress),
				cmd.ErrOrStderr(),
				func(ctx context.Context, id int) (any, error) {
					_, err := client.Contacts().AddLabels(ctx, id, labelList)
					if err != nil {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Failed to add labels to contact %d: %v\n", id, err)
						return nil, err
					}
					return nil, nil
				},
			)

			successCount, failCount := countResults(results)

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{
					"success_count": successCount,
					"fail_count":    failCount,
				})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added labels to %d contacts (%d failed)\n", successCount, failCount)
			return nil
		}),
	}

	cmd.Flags().StringVar(&contactIDs, "ids", "", "Contact IDs (CSV, whitespace, JSON array; or @- / @path) (required)")
	cmd.Flags().StringVar(&labels, "labels", "", "Labels to add (CSV, whitespace, JSON array; or @- / @path) (required)")
	cmd.Flags().IntVar(&concurrency, "concurrency", DefaultConcurrency, "Max concurrent operations")
	cmd.Flags().BoolVar(&progress, "progress", true, "Show progress while running")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "Disable progress output")
	flagAlias(cmd.Flags(), "concurrency", "cc")
	flagAlias(cmd.Flags(), "ids", "id")
	flagAlias(cmd.Flags(), "labels", "lb")
	flagAlias(cmd.Flags(), "progress", "prg")
	flagAlias(cmd.Flags(), "no-progress", "npr")
	_ = cmd.MarkFlagRequired("ids")
	_ = cmd.MarkFlagRequired("labels")

	return cmd
}

func newContactsBulkRemoveLabelCmd() *cobra.Command {
	var (
		contactIDs  string
		labels      string
		concurrency int
		progress    bool
		noProgress  bool
	)

	cmd := &cobra.Command{
		Use:     "remove-label",
		Aliases: []string{"rl"},
		Short:   "Remove labels from multiple contacts",
		Long: `Remove one or more labels from multiple contacts at once.

For each contact, this command fetches current labels, removes the specified
labels, and updates the contact with the remaining labels.`,
		Example: strings.TrimSpace(`
  # Remove a single label from multiple contacts
  cw contacts bulk remove-label --ids 1,2,3 --labels spam

  # Remove multiple labels from multiple contacts
  cw contacts bulk remove-label --ids 1,2,3 --labels spam,inactive

  # Control concurrency (default: 5)
  cw contacts bulk remove-label --ids 1,2,3 --labels spam --concurrency 10
`),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			ids, err := ParseResourceIDListFlag(contactIDs, "contact")
			if err != nil {
				return fmt.Errorf("invalid contact IDs: %w", err)
			}

			labelsToRemove := make(map[string]bool)
			labelList, err := ParseStringListFlag(labels)
			if err != nil {
				return fmt.Errorf("invalid labels: %w", err)
			}
			for _, l := range labelList {
				labelsToRemove[l] = true
			}
			if len(labelsToRemove) == 0 {
				return fmt.Errorf("no valid labels provided")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			results := runBulkOperation(
				ctx,
				ids,
				int64(concurrency),
				bulkProgressEnabled(cmd, progress, noProgress),
				cmd.ErrOrStderr(),
				func(ctx context.Context, id int) (any, error) {
					// Get current labels
					currentLabels, err := client.Contacts().Labels(ctx, id)
					if err != nil {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Failed to get labels for contact %d: %v\n", id, err)
						return nil, err
					}

					// Filter out labels to remove
					var remainingLabels []string
					for _, label := range currentLabels {
						if !labelsToRemove[label] {
							remainingLabels = append(remainingLabels, label)
						}
					}

					// Update with remaining labels (API replaces all labels)
					_, err = client.Contacts().AddLabels(ctx, id, remainingLabels)
					if err != nil {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Failed to update labels for contact %d: %v\n", id, err)
						return nil, err
					}
					return nil, nil
				},
			)

			successCount, failCount := countResults(results)

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{
					"success_count": successCount,
					"fail_count":    failCount,
				})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed labels from %d contacts (%d failed)\n", successCount, failCount)
			return nil
		}),
	}

	cmd.Flags().StringVar(&contactIDs, "ids", "", "Contact IDs (CSV, whitespace, JSON array; or @- / @path) (required)")
	cmd.Flags().StringVar(&labels, "labels", "", "Labels to remove (CSV, whitespace, JSON array; or @- / @path) (required)")
	cmd.Flags().IntVar(&concurrency, "concurrency", DefaultConcurrency, "Max concurrent operations")
	cmd.Flags().BoolVar(&progress, "progress", true, "Show progress while running")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "Disable progress output")
	flagAlias(cmd.Flags(), "concurrency", "cc")
	flagAlias(cmd.Flags(), "ids", "id")
	flagAlias(cmd.Flags(), "labels", "lb")
	flagAlias(cmd.Flags(), "progress", "prg")
	flagAlias(cmd.Flags(), "no-progress", "npr")
	_ = cmd.MarkFlagRequired("ids")
	_ = cmd.MarkFlagRequired("labels")

	return cmd
}
