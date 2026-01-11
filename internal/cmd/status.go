package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/chatwoot/chatwoot-cli/internal/config"
)

// StatusInfo holds configuration and authentication status information
type StatusInfo struct {
	Authenticated bool   `json:"authenticated"`
	BaseURL       string `json:"base_url,omitempty"`
	AccountID     int    `json:"account_id,omitempty"`
	TokenPreview  string `json:"token_preview,omitempty"`
	Profile       string `json:"profile,omitempty"`
	CLIVersion    string `json:"cli_version"`
	GoVersion     string `json:"go_version"`
	Platform      string `json:"platform"`
	ConfigSource  string `json:"config_source,omitempty"`
}

// getConfigSource determines where credentials are loaded from
func getConfigSource() string {
	if os.Getenv("CHATWOOT_BASE_URL") != "" &&
		os.Getenv("CHATWOOT_API_TOKEN") != "" &&
		os.Getenv("CHATWOOT_ACCOUNT_ID") != "" {
		return "environment"
	}
	if os.Getenv("CHATWOOT_PROFILE") != "" {
		return "environment (profile)"
	}
	return "keychain"
}

func newStatusCmd() *cobra.Command {
	var checkOnly bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current configuration and authentication status",
		Long: `Display the current CLI configuration including authentication status,
base URL, account ID, and other useful information.

This command is useful for agents and scripts to verify configuration
before making API calls.`,
		Example: `  # Show current status
  chatwoot status

  # Show status as JSON
  chatwoot status --output json

  # Check if authenticated (exits with code 1 if not)
  chatwoot status --check`,
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			info := StatusInfo{
				CLIVersion: version,
				GoVersion:  runtime.Version(),
				Platform:   runtime.GOOS + "/" + runtime.GOARCH,
			}

			// Try to load account credentials
			account, err := config.LoadAccount()
			if err == nil {
				info.Authenticated = true
				info.BaseURL = account.BaseURL
				info.AccountID = account.AccountID
				info.TokenPreview = maskToken(account.APIToken)
				info.ConfigSource = getConfigSource()

				// Get current profile name (only relevant for keychain source)
				if info.ConfigSource == "keychain" {
					if profile, err := config.CurrentProfile(); err == nil {
						info.Profile = profile
					}
				}
			}

			// If --check flag is set, just exit with appropriate code
			if checkOnly {
				if !info.Authenticated {
					return fmt.Errorf("not authenticated - run 'chatwoot auth login' first")
				}
				if !isJSON(cmd) {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "authenticated")
				} else {
					return printJSON(cmd, info)
				}
				return nil
			}

			if isJSON(cmd) {
				return printJSON(cmd, info)
			}

			// Text output
			w := newTabWriterFromCmd(cmd)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintln(w, "CLI STATUS")
			_, _ = fmt.Fprintln(w, strings.Repeat("-", 40))

			if info.Authenticated {
				_, _ = fmt.Fprintf(w, "Authenticated:\t%s\n", green("yes"))
				_, _ = fmt.Fprintf(w, "Base URL:\t%s\n", info.BaseURL)
				_, _ = fmt.Fprintf(w, "Account ID:\t%d\n", info.AccountID)
				_, _ = fmt.Fprintf(w, "Token:\t%s\n", info.TokenPreview)
				_, _ = fmt.Fprintf(w, "Config Source:\t%s\n", info.ConfigSource)
				if info.Profile != "" {
					_, _ = fmt.Fprintf(w, "Profile:\t%s\n", info.Profile)
				}
			} else {
				_, _ = fmt.Fprintf(w, "Authenticated:\t%s\n", red("no"))
				_, _ = fmt.Fprintf(w, "Hint:\tRun 'chatwoot auth login' to authenticate\n")
			}

			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintf(w, "CLI Version:\t%s\n", info.CLIVersion)
			_, _ = fmt.Fprintf(w, "Go Version:\t%s\n", info.GoVersion)
			_, _ = fmt.Fprintf(w, "Platform:\t%s\n", info.Platform)

			return nil
		}),
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "Exit with code 1 if not authenticated")

	return cmd
}
