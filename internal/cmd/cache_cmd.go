package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chatwoot/chatwoot-cli/internal/cache"
	"github.com/spf13/cobra"
)

func newCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cache",
		Aliases: []string{"ch"},
		Short:   "Manage the local cache",
	}

	cmd.AddCommand(newCacheClearCmd())
	cmd.AddCommand(newCachePathCmd())
	return cmd
}

func newCacheClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Clear all cached data",
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			dir := resolveCacheDir()
			if dir == "" {
				return fmt.Errorf("could not determine cache directory")
			}
			cache.ClearAll(dir)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Cache cleared: %s\n", dir)
			return nil
		}),
	}
}

func newCachePathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Show the cache directory path",
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			dir := resolveCacheDir()
			if dir == "" {
				return fmt.Errorf("could not determine cache directory")
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), dir)

			entries, err := os.ReadDir(dir)
			if err != nil {
				return nil // directory might not exist yet
			}
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				info, err := e.Info()
				if err != nil {
					continue
				}
				name := e.Name()
				if filepath.Ext(name) != ".json" {
					continue
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s (%d bytes)\n", name, info.Size())
			}
			return nil
		}),
	}
}
