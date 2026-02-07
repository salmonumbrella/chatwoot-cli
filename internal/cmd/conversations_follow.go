package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/actioncable"
	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newConversationsFollowCmd() *cobra.Command {
	var (
		incomingOnly bool
		tail         int
	)

	cmd := &cobra.Command{
		Use:   "follow <conversation-id|url>",
		Short: "Follow a conversation in real-time",
		Long: strings.TrimSpace(`
Follow a single conversation and print new messages as they arrive.

Connects directly to Chatwoot's real-time WebSocket (ActionCable) to receive
push notifications. No watch server or webhook setup required.
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

			// Fetch profile to get pubsub_token for WebSocket auth.
			profile, err := client.Profile().Get(ctx)
			if err != nil {
				return fmt.Errorf("failed to get profile (needed for WebSocket auth): %w", err)
			}
			if profile.PubsubToken == "" {
				return fmt.Errorf("profile has no pubsub_token; cannot connect to WebSocket")
			}

			cableURL := buildCableURL(client.BaseURL)
			channelID := actioncable.ChannelID{
				Channel:     "RoomChannel",
				PubsubToken: profile.PubsubToken,
				AccountID:   client.AccountID,
				UserID:      profile.ID,
			}

			// Reconnection loop with exponential backoff.
			backoff := 2 * time.Second
			maxBackoff := 30 * time.Second

			for {
				err := followViaWebSocket(ctx, cmd, cableURL, channelID, convID, incomingOnly, &lastSeenID)
				if ctx.Err() != nil {
					return nil
				}
				if !isJSON(cmd) {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "disconnected: %v, reconnecting in %s...\n", err, backoff)
				}
				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return nil
				}
				backoff = min(backoff*2, maxBackoff)
			}
		}),
	}

	cmd.Flags().BoolVar(&incomingOnly, "incoming-only", true, "Only show incoming (customer) messages")
	cmd.Flags().IntVar(&tail, "tail", 20, "Print the last N messages before following (0 to disable)")
	return cmd
}

// chatwootWSEvent is the outer envelope of a Chatwoot WebSocket event.
type chatwootWSEvent struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// followViaWebSocket connects to ActionCable, subscribes, and processes events
// until the connection drops or ctx is cancelled. Returns non-nil error on disconnect.
func followViaWebSocket(ctx context.Context, cmd *cobra.Command, cableURL string, channelID actioncable.ChannelID, convID int, incomingOnly bool, lastSeenID *int) error {
	conn, err := actioncable.Connect(ctx, cableURL)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.Subscribe(ctx, channelID); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	conn.StartPresence(ctx, 30*time.Second)

	events := conn.Listen(ctx)
	for ev := range events {
		if ev.Err != nil {
			return ev.Err
		}

		// Parse the outer Chatwoot event envelope.
		var wsEvent chatwootWSEvent
		if err := json.Unmarshal(ev.Data, &wsEvent); err != nil {
			continue // skip malformed events
		}

		if wsEvent.Event != "message.created" {
			continue
		}

		// Parse the message data.
		var msg api.Message
		if err := json.Unmarshal(wsEvent.Data, &msg); err != nil {
			continue
		}

		// Filter by conversation ID (WebSocket sends all account events).
		if msg.ConversationID != convID {
			continue
		}

		// Dedup by message ID.
		if lastSeenID != nil && msg.ID <= *lastSeenID {
			continue
		}
		if lastSeenID != nil {
			*lastSeenID = msg.ID
		}

		// Filter by message type if --incoming-only.
		if incomingOnly && msg.MessageType != api.MessageTypeIncoming {
			continue
		}

		if err := printFollowMessage(cmd, msg, "ws"); err != nil {
			return err
		}
	}

	return fmt.Errorf("event channel closed")
}

// buildCableURL converts a Chatwoot base URL to its ActionCable WebSocket URL.
func buildCableURL(baseURL string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL // fallback
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	u.Path = "/cable"
	u.RawQuery = ""
	return u.String()
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
