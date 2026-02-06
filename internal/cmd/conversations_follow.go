package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newConversationsFollowCmd() *cobra.Command {
	var (
		incomingOnly  bool
		tail          int
		pollFallback  bool
		pollIntervalS int
	)

	cmd := &cobra.Command{
		Use:   "follow <conversation-id|url>",
		Short: "Follow a conversation (push via SSE)",
		Long: strings.TrimSpace(`
Follow a single conversation and print new messages as they arrive.

This uses the server-side watch receiver (chatwoot watch serve) and streams events
via Server-Sent Events (SSE). This is push-based (no polling) and does not use WebSockets.
`),
		Args: cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			convID, err := parseIDOrURL(args[0], "conversation")
			if err != nil {
				// allow plain numeric without URL parsing rules
				if id, idErr := strconv.Atoi(strings.TrimSpace(args[0])); idErr == nil && id > 0 {
					convID = id
				} else {
					return err
				}
			}

			ctx, stop := signal.NotifyContext(cmdContext(cmd), os.Interrupt, syscall.SIGTERM)
			defer stop()

			var lastSeenID int
			if tail > 0 {
				msgs, err := client.Messages().ListWithLimit(ctx, convID, tail, 10)
				if err == nil && len(msgs) > 0 {
					// Print oldest -> newest.
					sort.Slice(msgs, func(i, j int) bool { return msgs[i].ID < msgs[j].ID })
					for _, m := range msgs {
						if m.ID > lastSeenID {
							lastSeenID = m.ID
						}
						if incomingOnly && m.MessageType != api.MessageTypeIncoming {
							continue
						}
						if err := printFollowMessage(cmd, m, "history"); err != nil {
							return err
						}
					}
				}
			}

			if !isJSON(cmd) {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Following conversation %d (press Ctrl+C to stop)...\n", convID)
			}

			watchURL := strings.TrimSuffix(client.BaseURL, "/") + fmt.Sprintf("/watch/conversations/%d", convID)
			return followConversation(ctx, cmd, client, watchURL, convID, incomingOnly, pollFallback, time.Duration(pollIntervalS)*time.Second, &lastSeenID)
		}),
	}

	cmd.Flags().BoolVar(&incomingOnly, "incoming-only", true, "Only show incoming (customer) messages")
	cmd.Flags().IntVar(&tail, "tail", 20, "Print the last N messages before following (0 to disable)")
	cmd.Flags().BoolVar(&pollFallback, "poll-fallback", true, "Fallback to polling if SSE disconnects")
	cmd.Flags().IntVar(&pollIntervalS, "poll-interval", 3, "Polling interval in seconds when falling back")
	return cmd
}

func followConversation(ctx context.Context, cmd *cobra.Command, client *api.Client, watchURL string, convID int, incomingOnly bool, pollFallback bool, pollInterval time.Duration, lastSeenID *int) error {
	sseClient := &http.Client{Timeout: 0}

	trySSE := func(connectTimeout time.Duration) (*http.Response, error) {
		connectCtx := ctx
		var cancel func()
		if connectTimeout > 0 {
			connectCtx, cancel = context.WithTimeout(ctx, connectTimeout)
			defer cancel()
		}

		req, err := http.NewRequestWithContext(connectCtx, http.MethodGet, watchURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("api_access_token", client.APIToken)
		if lastSeenID != nil && *lastSeenID > 0 {
			req.Header.Set("Last-Event-ID", strconv.Itoa(*lastSeenID))
		}
		return sseClient.Do(req)
	}

	for {
		resp, err := trySSE(5 * time.Second)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if !pollFallback {
				return err
			}
			if err := pollUntilSSE(ctx, cmd, client, watchURL, convID, incomingOnly, pollInterval, lastSeenID, trySSE); err != nil {
				return err
			}
			continue
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := ioReadAllLimit(resp.Body, 64*1024)
			_ = resp.Body.Close()
			msg := fmt.Sprintf("watch stream failed (%s): %s", resp.Status, strings.TrimSpace(string(body)))
			if !pollFallback {
				return fmt.Errorf("%s", msg)
			}
			if err := pollUntilSSE(ctx, cmd, client, watchURL, convID, incomingOnly, pollInterval, lastSeenID, trySSE); err != nil {
				return err
			}
			continue
		}

		err = readSSE(ctx, cmd, resp.Body, incomingOnly, lastSeenID)
		_ = resp.Body.Close()
		if err == nil || ctx.Err() != nil {
			return nil
		}
		if !pollFallback {
			return err
		}
		if err := pollUntilSSE(ctx, cmd, client, watchURL, convID, incomingOnly, pollInterval, lastSeenID, trySSE); err != nil {
			return err
		}
	}
}

func pollUntilSSE(ctx context.Context, cmd *cobra.Command, client *api.Client, watchURL string, convID int, incomingOnly bool, pollInterval time.Duration, lastSeenID *int, trySSE func(time.Duration) (*http.Response, error)) error {
	if pollInterval <= 0 {
		pollInterval = 3 * time.Second
	}
	if !isJSON(cmd) {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "watch: SSE disconnected; polling every %s...\n", pollInterval)
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		if ctx.Err() != nil {
			return nil
		}

		// Poll once.
		_ = pollOnce(ctx, cmd, client, convID, incomingOnly, lastSeenID)

		// Try reconnect quickly.
		resp, err := trySSE(2 * time.Second)
		if err == nil && resp != nil {
			if resp.StatusCode == http.StatusOK {
				if !isJSON(cmd) {
					_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "watch: SSE reconnected")
				}
				err = readSSE(ctx, cmd, resp.Body, incomingOnly, lastSeenID)
				_ = resp.Body.Close()
				if err == nil || ctx.Err() != nil {
					return nil
				}
				// If it drops again, continue polling.
			} else {
				_ = resp.Body.Close()
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func printFollowMessage(cmd *cobra.Command, m api.Message, source string) error {
	if isJSON(cmd) {
		// Emit as JSON lines.
		return outfmt.WriteJSON(cmd.OutOrStdout(), map[string]any{
			"source": source,
			"type":   "message",
			"item":   m,
		})
	}
	ts := time.Unix(m.CreatedAt, 0).Format("15:04:05")
	sender := "-"
	if m.Sender != nil && strings.TrimSpace(m.Sender.Name) != "" {
		sender = m.Sender.Name
	}
	kind := m.MessageTypeName()
	privacy := ""
	if m.Private {
		privacy = " [private]"
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s (%s)%s: %s\n", ts, sender, kind, privacy, strings.TrimSpace(m.Content))
	return err
}

func printFollowEvent(cmd *cobra.Command, ev watchEvent, eventName string) error {
	if isJSON(cmd) {
		return outfmt.WriteJSON(cmd.OutOrStdout(), ev)
	}

	ts := ""
	if strings.TrimSpace(ev.CreatedAt) != "" {
		ts = ev.CreatedAt
	} else {
		ts = time.Now().Format(time.RFC3339)
	}

	sender := "-"
	if ev.Sender != nil {
		if name, ok := ev.Sender["name"].(string); ok && strings.TrimSpace(name) != "" {
			sender = name
		}
	}
	privacy := ""
	if ev.Private {
		privacy = " [private]"
	}
	kind := ev.MessageType
	// Today we only push message_created, but keep eventName available for future event types.
	if strings.TrimSpace(eventName) != "" && eventName != "message_created" {
		kind = kind + ":" + strings.TrimSpace(eventName)
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s (%s)%s: %s\n", ts, sender, kind, privacy, strings.TrimSpace(ev.Content))
	return err
}

func ioReadAllLimit(r io.Reader, limit int64) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r, limit))
}

func readSSE(ctx context.Context, cmd *cobra.Command, r io.Reader, incomingOnly bool, lastSeenID *int) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var eventName string
	var dataLines []string
	for sc.Scan() {
		if ctx.Err() != nil {
			return nil
		}
		line := sc.Text()
		if line == "" {
			if len(dataLines) == 0 {
				eventName = ""
				continue
			}
			raw := strings.Join(dataLines, "\n")
			dataLines = dataLines[:0]

			var ev watchEvent
			if err := json.Unmarshal([]byte(raw), &ev); err != nil {
				continue
			}
			if lastSeenID != nil && ev.MessageID > *lastSeenID {
				*lastSeenID = ev.MessageID
			} else if lastSeenID != nil && ev.MessageID <= *lastSeenID {
				eventName = ""
				continue
			}

			if incomingOnly && strings.ToLower(ev.MessageType) != "incoming" {
				eventName = ""
				continue
			}
			if err := printFollowEvent(cmd, ev, eventName); err != nil {
				return err
			}
			eventName = ""
			continue
		}

		if strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			continue
		}
	}
	return sc.Err()
}

func pollOnce(ctx context.Context, cmd *cobra.Command, client *api.Client, convID int, incomingOnly bool, lastSeenID *int) error {
	// Fetch a recent window; 50 should be enough for typical “agent waiting for reply”.
	msgs, err := client.Messages().ListWithLimit(ctx, convID, 50, 10)
	if err != nil {
		if !isJSON(cmd) {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "watch: poll error: %v\n", err)
		}
		return err
	}
	if len(msgs) == 0 {
		return nil
	}

	sort.Slice(msgs, func(i, j int) bool { return msgs[i].ID < msgs[j].ID })
	for _, m := range msgs {
		if lastSeenID != nil && m.ID <= *lastSeenID {
			continue
		}
		if lastSeenID != nil && m.ID > *lastSeenID {
			*lastSeenID = m.ID
		}
		if incomingOnly && m.MessageType != api.MessageTypeIncoming {
			continue
		}
		if err := printFollowMessage(cmd, m, "poll"); err != nil {
			return err
		}
	}
	return nil
}
