package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/auth"
	"github.com/chatwoot/chatwoot-cli/internal/config"
	"github.com/chatwoot/chatwoot-cli/internal/skill"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

// newAuthCmd returns the auth command with subcommands
func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication credentials",
		Long:  "Configure and manage Chatwoot API authentication credentials stored securely in your OS keychain.",
	}

	cmd.AddCommand(newAuthLoginCmd())
	cmd.AddCommand(newAuthStatusCmd())
	cmd.AddCommand(newAuthLogoutCmd())

	return cmd
}

// newAuthLoginCmd creates the auth login command
func newAuthLoginCmd() *cobra.Command {
	var (
		url       string
		token     string
		accountID int
		browser   bool
		profile   string
		platform  string
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate via browser",
		Long: strings.TrimSpace(`
Save Chatwoot authentication credentials securely to your OS keychain.

By default, opens a browser window for easy setup. Use --no-browser for CLI-only setup.

You'll need:
- Base URL: Your Chatwoot instance URL (e.g. https://chatwoot.example.com)
- API Token: Generate from Settings > Profile Settings > Access Token
- Account ID: Found in your Chatwoot URL (e.g. /app/accounts/1)

Optional:
- Profile: Save multiple accounts and switch between them
- Platform Token: For platform API operations (self-hosted/managed)
`),
		Example: strings.TrimSpace(`
  # Interactive browser-based login (default)
  chatwoot auth login

  # CLI-only login with flags
  chatwoot auth login --no-browser --url https://chatwoot.example.com --token YOUR_API_TOKEN --account-id 1

  # Save to a named profile with a platform token
  chatwoot auth login --no-browser --url https://chatwoot.example.com --token YOUR_API_TOKEN --account-id 1 --profile staging --platform-token PLATFORM_TOKEN
`),
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			// If browser mode (default) and no flags provided, use browser setup
			if browser && url == "" && token == "" && accountID == 0 {
				return runBrowserSetup(cmd.OutOrStdout(), profile)
			}

			// CLI mode: validate required flags
			if url == "" {
				return fmt.Errorf("--url is required (or use browser mode without --no-browser)")
			}
			if token == "" {
				return fmt.Errorf("--token is required (or use browser mode without --no-browser)")
			}
			if accountID <= 0 {
				return fmt.Errorf("--account-id must be a positive integer (or use browser mode without --no-browser)")
			}

			// Normalize URL (remove trailing slash)
			url = strings.TrimSuffix(url, "/")

			// Validate URL to prevent SSRF attacks
			if err := validation.ValidateChatwootURL(url); err != nil {
				return fmt.Errorf("invalid URL: %w", err)
			}

			// Save to keychain
			account := config.Account{
				BaseURL:       url,
				APIToken:      token,
				AccountID:     accountID,
				PlatformToken: platform,
			}

			if err := config.SaveProfile(profile, account); err != nil {
				return fmt.Errorf("failed to save credentials: %w", err)
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Authentication credentials saved successfully!")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Base URL: %s\n", url)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Account ID: %d\n", accountID)
			if profile != "" && profile != "default" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Profile: %s\n", profile)
			}

			// Generate workspace skill
			generateWorkspaceSkill(cmd.Context(), cmd.OutOrStdout(), account)

			return nil
		}),
	}

	cmd.Flags().StringVar(&url, "url", "", "Chatwoot base URL (e.g. https://chatwoot.example.com)")
	cmd.Flags().StringVar(&token, "token", "", "API access token")
	cmd.Flags().IntVar(&accountID, "account-id", 0, "Account ID")
	cmd.Flags().BoolVar(&browser, "browser", true, "Use browser-based setup (default: true)")
	cmd.Flags().StringVar(&profile, "profile", "default", "Profile name to save credentials under")
	cmd.Flags().StringVar(&platform, "platform-token", "", "Platform API token (optional)")
	cmd.Flags().Lookup("browser").NoOptDefVal = "true"

	return cmd
}

// runBrowserSetup launches the browser-based authentication flow
func runBrowserSetup(out io.Writer, profile string) error {
	_, _ = fmt.Fprintln(out, "Opening browser for Chatwoot CLI setup...")
	_, _ = fmt.Fprintln(out, "(Press Ctrl+C to cancel)")
	_, _ = fmt.Fprintln(out)

	// Create setup server
	server, err := auth.NewSetupServer(profile)
	if err != nil {
		return fmt.Errorf("failed to create setup server: %w", err)
	}

	// Create context with 5-minute timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Start server and wait for result
	result, err := server.Start(ctx)
	if err != nil {
		if err == context.DeadlineExceeded {
			return fmt.Errorf("setup timed out after 5 minutes")
		}
		return fmt.Errorf("setup failed: %w", err)
	}

	if result.Error != nil {
		return result.Error
	}

	_, _ = fmt.Fprintln(out, "Authentication credentials saved successfully!")
	_, _ = fmt.Fprintf(out, "  Base URL: %s\n", result.Account.BaseURL)
	_, _ = fmt.Fprintf(out, "  Account ID: %d\n", result.Account.AccountID)
	if result.Email != "" {
		_, _ = fmt.Fprintf(out, "  Email: %s\n", result.Email)
	}

	// Generate workspace skill
	generateWorkspaceSkill(ctx, out, result.Account)

	return nil
}

// generateWorkspaceSkill creates a Claude skill file with workspace context.
// Errors are non-fatal and just logged as warnings.
func generateWorkspaceSkill(ctx context.Context, out io.Writer, account config.Account) {
	_, _ = fmt.Fprintln(out, "Generating workspace skill...")

	client := api.New(account.BaseURL, account.APIToken, account.AccountID)
	if err := skill.GenerateWorkspaceSkill(ctx, client, account.BaseURL); err != nil {
		_, _ = fmt.Fprintf(out, "Warning: failed to generate workspace skill: %v\n", err)
		return
	}

	skillPath, _ := skill.SkillPath()
	_, _ = fmt.Fprintf(out, "Generated %s\n", skillPath)
}

// newAuthStatusCmd creates the auth status command
func newAuthStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current authentication configuration",
		Long:  "Display the currently saved authentication configuration (API token is masked for security).",
		Example: strings.TrimSpace(`
  # Check authentication status
  chatwoot auth status

  # JSON output for scripting
  chatwoot auth status --json
`),
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			envBaseURL := strings.TrimSpace(os.Getenv("CHATWOOT_BASE_URL"))
			envToken := strings.TrimSpace(os.Getenv("CHATWOOT_API_TOKEN"))
			envAccountID := strings.TrimSpace(os.Getenv("CHATWOOT_ACCOUNT_ID"))
			usingEnv := envBaseURL != "" || envToken != "" || envAccountID != ""

			account, err := config.LoadAccount()
			if err != nil {
				if err == config.ErrNotConfigured {
					if isJSON(cmd) {
						return printJSON(cmd, map[string]any{
							"authenticated": false,
							"message":       "Not authenticated. Run 'chatwoot auth login' to configure credentials.",
						})
					}
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Not authenticated.")
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Run 'chatwoot auth login' to configure credentials.")
					return nil
				}
				return fmt.Errorf("failed to load credentials: %w", err)
			}

			var profile string
			if !usingEnv {
				if current, err := config.CurrentProfile(); err == nil {
					profile = current
				}
			}

			if isJSON(cmd) {
				payload := map[string]any{
					"authenticated":  true,
					"base_url":       account.BaseURL,
					"account_id":     account.AccountID,
					"api_token":      maskToken(account.APIToken),
					"platform_token": maskToken(account.PlatformToken),
					"source":         map[bool]string{true: "env", false: "keychain"}[usingEnv],
				}
				if profile != "" {
					payload["profile"] = profile
				}
				return printJSON(cmd, payload)
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Authenticated")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Base URL: %s\n", account.BaseURL)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Account ID: %d\n", account.AccountID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  API Token: %s\n", maskToken(account.APIToken))
			if account.PlatformToken != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Platform Token: %s\n", maskToken(account.PlatformToken))
			}
			if profile != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Profile: %s\n", profile)
			}
			if usingEnv {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "  Source: env")
			}

			return nil
		}),
	}

	return cmd
}

// newAuthLogoutCmd creates the auth logout command
func newAuthLogoutCmd() *cobra.Command {
	var profile string

	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove credentials from keychain",
		Long:  "Delete the stored authentication credentials from your OS keychain.",
		Example: strings.TrimSpace(`
  # Remove stored credentials
  chatwoot auth logout
`),
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			if profile == "" {
				current, err := config.CurrentProfile()
				if err == nil {
					profile = current
				}
			}

			if profile == "" && !config.HasAccount() {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No credentials found.")
				return nil
			}

			if err := config.DeleteProfile(profile); err != nil {
				return fmt.Errorf("failed to remove credentials: %w", err)
			}

			if profile == "" {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Credentials removed successfully.")
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Profile %s removed successfully.\n", profile)
			}
			return nil
		}),
	}

	cmd.Flags().StringVar(&profile, "profile", "", "Profile name to remove (defaults to current)")

	return cmd
}

// maskToken masks an API token for display, showing only first and last 4 characters
func maskToken(token string) string {
	if len(token) < 8 {
		return strings.Repeat("*", len(token)) // Match actual length
	}
	return token[:4] + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
}
