package cmd

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/spf13/cobra"
)

func newWatchSetupCmd() *cobra.Command {
	var (
		hookBaseURL   string
		hookToken     string
		subscriptions []string
		webhookID     int
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Create/update the Chatwoot webhook for push watch",
		Long: strings.TrimSpace(`
Creates (or updates) a Chatwoot webhook subscription that points at your watch receiver.

Chatwoot webhooks do not allow custom headers, so the receiver token is passed via
the webhook URL query string (?token=...).
`),
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			base := strings.TrimSpace(hookBaseURL)
			if base == "" {
				return fmt.Errorf("--url is required")
			}
			tok := strings.TrimSpace(hookToken)
			if tok == "" {
				tok = strings.TrimSpace(os.Getenv("CHATWOOT_WATCH_HOOK_TOKEN"))
			}
			if tok == "" {
				return fmt.Errorf("--token is required (or set CHATWOOT_WATCH_HOOK_TOKEN)")
			}
			if len(subscriptions) == 0 {
				subscriptions = []string{"message_created"}
			}

			fullURL, err := withTokenQuery(base, tok)
			if err != nil {
				return err
			}

			existing, err := client.Webhooks().List(cmdContext(cmd))
			if err != nil {
				return err
			}

			normalize := func(raw string) string {
				u, err := url.Parse(raw)
				if err != nil {
					return raw
				}
				q := u.Query()
				q.Del("token")
				u.RawQuery = q.Encode()
				return u.String()
			}
			fullNorm := normalize(fullURL)

			sort.Strings(subscriptions)

			var target *api.Webhook
			if webhookID > 0 {
				for i := range existing {
					if existing[i].ID == webhookID {
						target = &existing[i]
						break
					}
				}
				if target == nil {
					return fmt.Errorf("webhook %d not found", webhookID)
				}
			} else {
				for i := range existing {
					if normalize(existing[i].URL) == fullNorm {
						target = &existing[i]
						break
					}
				}
			}

			if target == nil {
				wh, err := client.Webhooks().Create(cmdContext(cmd), fullURL, subscriptions)
				if err != nil {
					return err
				}
				if isJSON(cmd) {
					return printRawJSON(cmd, map[string]any{
						"action":  "created",
						"webhook": wh,
					})
				}
				printAction(cmd, "Created", "webhook", wh.ID, "")
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "URL: %s\n", wh.URL)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Subscriptions: %s\n", strings.Join(wh.Subscriptions, ", "))
				return nil
			}

			needsUpdate := strings.TrimSpace(target.URL) != strings.TrimSpace(fullURL) ||
				!sameStringSet(target.Subscriptions, subscriptions)

			if !needsUpdate {
				if isJSON(cmd) {
					return printRawJSON(cmd, map[string]any{
						"action":  "noop",
						"webhook": target,
					})
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Webhook already configured: %d\n", target.ID)
				return nil
			}

			wh, err := client.Webhooks().Update(cmdContext(cmd), target.ID, fullURL, subscriptions)
			if err != nil {
				return err
			}
			if isJSON(cmd) {
				return printRawJSON(cmd, map[string]any{
					"action":  "updated",
					"webhook": wh,
				})
			}
			printAction(cmd, "Updated", "webhook", wh.ID, "")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "URL: %s\n", wh.URL)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Subscriptions: %s\n", strings.Join(wh.Subscriptions, ", "))
			return nil
		}),
	}

	cmd.Flags().StringVar(&hookBaseURL, "url", "", "Receiver webhook URL (e.g. https://chatwoot.example.com/hooks/chatwoot)")
	cmd.Flags().StringVar(&hookToken, "token", "", "Shared token appended as ?token=... (or env CHATWOOT_WATCH_HOOK_TOKEN)")
	cmd.Flags().StringSliceVar(&subscriptions, "subscriptions", []string{"message_created"}, "Webhook subscriptions (repeatable)")
	cmd.Flags().IntVar(&webhookID, "id", 0, "Webhook ID to update (optional)")

	return cmd
}

func withTokenQuery(rawBase string, token string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(rawBase))
	if err != nil {
		return "", fmt.Errorf("invalid --url: %w", err)
	}
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func sameStringSet(a, b []string) bool {
	na := normalizeStringSet(a)
	nb := normalizeStringSet(b)
	if len(na) != len(nb) {
		return false
	}
	for i := range na {
		if na[i] != nb[i] {
			return false
		}
	}
	return true
}

func normalizeStringSet(v []string) []string {
	seen := make(map[string]struct{}, len(v))
	out := make([]string, 0, len(v))
	for _, s := range v {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
