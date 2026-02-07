// internal/cmd/version.go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chatwoot/chatwoot-cli/internal/update"
)

// version is set at build time via ldflags
var version = "dev"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "chatwoot-cli version %s\n", version)

			// Check for updates (non-blocking, fails silently)
			result := update.CheckForUpdate(cmd.Context(), version)
			if result != nil && result.UpdateAvailable {
				errOut := cmd.ErrOrStderr()
				_, _ = fmt.Fprintf(errOut, "\nUpdate available: %s -> %s\n", result.CurrentVersion, result.LatestVersion) //nolint:errcheck
				_, _ = fmt.Fprintf(errOut, "Download: %s\n", result.UpdateURL)                                            //nolint:errcheck
			}
		},
	}
}
