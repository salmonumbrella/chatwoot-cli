// internal/cmd/version.go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/chatwoot/chatwoot-cli/internal/update"
)

// version is set at build time via ldflags
var version = "dev"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("chatwoot-cli version %s\n", version)

			// Check for updates (non-blocking, fails silently)
			result := update.CheckForUpdate(cmd.Context(), version)
			if result != nil && result.UpdateAvailable {
				fmt.Fprintf(os.Stderr, "\nUpdate available: %s -> %s\n", result.CurrentVersion, result.LatestVersion)
				fmt.Fprintf(os.Stderr, "Download: %s\n", result.UpdateURL)
			}
		},
	}
}
