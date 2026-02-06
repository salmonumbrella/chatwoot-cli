package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/spf13/cobra"
)

func newWatchStatusCmd() *cobra.Command {
	var (
		hookBaseURL string
		token       string
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show watch webhook status in Chatwoot",
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			webhooks, err := client.Webhooks().List(cmdContext(cmd))
			if err != nil {
				return err
			}

			base := strings.TrimSpace(hookBaseURL)
			if base == "" {
				base = strings.TrimSpace(os.Getenv("CHATWOOT_WATCH_HOOK_URL"))
			}

			if base == "" {
				// No filter: just dump.
				if isJSON(cmd) {
					return printRawJSON(cmd, map[string]any{
						"webhooks": webhooks,
					})
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Webhooks: %d\n", len(webhooks))
				for _, wh := range webhooks {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %d: %s (%s)\n", wh.ID, wh.URL, strings.Join(wh.Subscriptions, ", "))
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Tip: pass --url to filter to the watch receiver webhook.")
				return nil
			}

			tok := strings.TrimSpace(token)
			if tok == "" {
				tok = strings.TrimSpace(os.Getenv("CHATWOOT_WATCH_HOOK_TOKEN"))
			}

			wantFull := base
			var wantNorm string
			if tok != "" {
				u, err := withTokenQuery(base, tok)
				if err == nil {
					wantFull = u
				}
			}
			wantNorm = normalizeWebhookURL(wantFull)

			var matches []api.Webhook
			for _, wh := range webhooks {
				if normalizeWebhookURL(wh.URL) == wantNorm {
					matches = append(matches, wh)
				}
			}

			if isJSON(cmd) {
				return printRawJSON(cmd, map[string]any{
					"url":      wantFull,
					"matches":  matches,
					"webhooks": webhooks,
				})
			}

			if len(matches) == 0 {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No matching webhook found for: %s\n", wantFull)
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Run: chatwoot watch setup --url ... --token ...")
				return nil
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Matching webhooks for %s:\n", wantFull)
			for _, wh := range matches {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %d: %s (%s)\n", wh.ID, wh.URL, strings.Join(wh.Subscriptions, ", "))
			}
			return nil
		}),
	}

	cmd.Flags().StringVar(&hookBaseURL, "url", "", "Receiver webhook URL (e.g. https://chatwoot.example.com/hooks/chatwoot)")
	cmd.Flags().StringVar(&token, "token", "", "Receiver shared token (optional; env CHATWOOT_WATCH_HOOK_TOKEN)")
	return cmd
}

func normalizeWebhookURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return raw
	}
	q := u.Query()
	q.Del("token")
	u.RawQuery = q.Encode()
	return u.String()
}
