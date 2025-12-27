package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/auth"
	"github.com/chatwoot/chatwoot-cli/internal/config"
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
`),
		Example: strings.TrimSpace(`
  # Interactive browser-based login (default)
  chatwoot auth login

  # CLI-only login with flags
  chatwoot auth login --no-browser --url https://chatwoot.example.com --token YOUR_API_TOKEN --account-id 1
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			// If browser mode (default) and no flags provided, use browser setup
			if browser && url == "" && token == "" && accountID == 0 {
				return runBrowserSetup()
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
				BaseURL:   url,
				APIToken:  token,
				AccountID: accountID,
			}

			if err := config.SaveAccount(account); err != nil {
				return fmt.Errorf("failed to save credentials: %w", err)
			}

			fmt.Println("Authentication credentials saved successfully!")
			fmt.Printf("  Base URL: %s\n", url)
			fmt.Printf("  Account ID: %d\n", accountID)

			return nil
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "Chatwoot base URL (e.g. https://chatwoot.example.com)")
	cmd.Flags().StringVar(&token, "token", "", "API access token")
	cmd.Flags().IntVar(&accountID, "account-id", 0, "Account ID")
	cmd.Flags().BoolVar(&browser, "browser", true, "Use browser-based setup (default: true)")
	cmd.Flags().Lookup("browser").NoOptDefVal = "true"

	return cmd
}

// runBrowserSetup launches the browser-based authentication flow
func runBrowserSetup() error {
	fmt.Println("Opening browser for Chatwoot CLI setup...")
	fmt.Println("(Press Ctrl+C to cancel)")
	fmt.Println()

	// Create setup server
	server, err := auth.NewSetupServer()
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

	fmt.Println("Authentication credentials saved successfully!")
	fmt.Printf("  Base URL: %s\n", result.Account.BaseURL)
	fmt.Printf("  Account ID: %d\n", result.Account.AccountID)
	if result.Email != "" {
		fmt.Printf("  Email: %s\n", result.Email)
	}

	return nil
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
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			account, err := config.LoadAccount()
			if err != nil {
				if err == config.ErrNotConfigured {
					if isJSON(cmd) {
						return printJSON(map[string]any{
							"authenticated": false,
							"message":       "Not authenticated. Run 'chatwoot auth login' to configure credentials.",
						})
					}
					fmt.Println("Not authenticated.")
					fmt.Println("Run 'chatwoot auth login' to configure credentials.")
					return nil
				}
				return fmt.Errorf("failed to load credentials: %w", err)
			}

			if isJSON(cmd) {
				return printJSON(map[string]any{
					"authenticated": true,
					"base_url":      account.BaseURL,
					"account_id":    account.AccountID,
					"api_token":     maskToken(account.APIToken),
				})
			}

			fmt.Println("Authenticated")
			fmt.Printf("  Base URL: %s\n", account.BaseURL)
			fmt.Printf("  Account ID: %d\n", account.AccountID)
			fmt.Printf("  API Token: %s\n", maskToken(account.APIToken))

			return nil
		},
	}

	return cmd
}

// newAuthLogoutCmd creates the auth logout command
func newAuthLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove credentials from keychain",
		Long:  "Delete the stored authentication credentials from your OS keychain.",
		Example: strings.TrimSpace(`
  # Remove stored credentials
  chatwoot auth logout
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !config.HasAccount() {
				fmt.Println("No credentials found.")
				return nil
			}

			if err := config.DeleteAccount(); err != nil {
				return fmt.Errorf("failed to remove credentials: %w", err)
			}

			fmt.Println("Credentials removed successfully.")
			return nil
		},
	}

	return cmd
}

// maskToken masks an API token for display, showing only first and last 4 characters
func maskToken(token string) string {
	if len(token) < 8 {
		return strings.Repeat("*", len(token)) // Match actual length
	}
	return token[:4] + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
}
