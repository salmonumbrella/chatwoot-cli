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
	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newConversationsFollowCmd() *cobra.Command {
	var (
		incomingOnly bool
		tail         int
		followAll    bool
		events       []string
		showTyping   bool
		debounce     time.Duration
		includeRaw   bool
	)

	cmd := &cobra.Command{
		Use:   "follow [conversation-id|url]",
		Short: "Follow a conversation in real-time",
		Long: strings.TrimSpace(`
Follow conversations and print new messages as they arrive.

Connects directly to Chatwoot's real-time WebSocket (ActionCable) to receive
push notifications. No watch server or webhook setup required.

By default, follows a single conversation by ID. Use --all to follow
all conversations on the account.
`),
		Args: cobra.MaximumNArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if followAll && cmd.Flags().Changed("tail") && tail != 0 {
				return fmt.Errorf("--tail requires a single conversation")
			}

			// In --all mode, typing indicators are useful by default unless the user
			// explicitly configured --events.
			if followAll && !cmd.Flags().Changed("events") && !showTyping {
				events = append(events, "conversation.typing_on", "conversation.typing_off")
			}

			// Apply typing convenience flag by adding typing events.
			if showTyping {
				events = append(events, "conversation.typing_on", "conversation.typing_off")
			}
			events = dedupeStrings(events)

			allowedEvents := make(map[string]struct{}, len(events))
			for _, e := range events {
				e = strings.TrimSpace(e)
				if e == "" {
					continue
				}
				allowedEvents[e] = struct{}{}
			}

			convID := 0
			if followAll {
				convID = 0
				// Tail only makes sense for a single conversation.
				tail = 0
			} else {
				if len(args) == 0 {
					return fmt.Errorf("missing conversation id (or use --all)")
				}

				parsedID, err := parseIDOrURL(args[0], "conversation")
				if err != nil {
					// allow plain numeric without URL parsing rules
					if id, idErr := strconv.Atoi(strings.TrimSpace(args[0])); idErr == nil && id > 0 {
						parsedID = id
					} else {
						return err
					}
				}
				convID = parsedID
			}

			ctx, stop := signal.NotifyContext(cmdContext(cmd), os.Interrupt, syscall.SIGTERM)
			defer stop()

			client, err := getClient()
			if err != nil {
				return err
			}

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
				if convID == 0 {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Following all conversations (press Ctrl+C to stop)...\n")
				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Following conversation %d (press Ctrl+C to stop)...\n", convID)
				}
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
				err := followViaWebSocket(ctx, cmd, cableURL, channelID, convID, incomingOnly, &lastSeenID, allowedEvents, debounce, includeRaw)
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
	cmd.Flags().BoolVar(&followAll, "all", false, "Follow all conversations (no conversation ID required)")
	cmd.Flags().StringSliceVar(&events, "events", []string{"message.created"}, "Event types to show (message.created,conversation.created,conversation.status_changed,assignee.changed)")
	cmd.Flags().BoolVar(&showTyping, "typing", false, "Show typing indicators")
	cmd.Flags().DurationVar(&debounce, "debounce", 0, "Batch rapid messages from same conversation (e.g., 2s)")
	cmd.Flags().BoolVar(&includeRaw, "raw", false, "Include raw WebSocket payload (JSON/agent modes only)")
	return cmd
}

// chatwootWSEvent is the outer envelope of a Chatwoot WebSocket event.
type chatwootWSEvent struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// followViaWebSocket connects to ActionCable, subscribes, and processes events
// until the connection drops or ctx is cancelled. Returns non-nil error on disconnect.
func followViaWebSocket(ctx context.Context, cmd *cobra.Command, cableURL string, channelID actioncable.ChannelID, convID int, incomingOnly bool, lastSeenID *int, allowedEvents map[string]struct{}, debounce time.Duration, includeRaw bool) error {
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
	type debounceBuf struct {
		timer    *time.Timer
		messages []followMsg
	}

	// Debounce state: keyed by conversation_id.
	debounced := make(map[int]*debounceBuf)
	flushCh := make(chan int, 4096)
	done := make(chan struct{})
	defer close(done)

	flushConv := func(id int) error {
		buf, ok := debounced[id]
		if !ok || buf == nil || len(buf.messages) == 0 {
			return nil
		}

		msgs := buf.messages
		buf.messages = nil
		if buf.timer != nil {
			buf.timer.Stop()
			buf.timer = nil
		}
		delete(debounced, id)

		return printFollowMessageBatch(cmd, msgs, "ws", includeRaw)
	}

	flushAll := func() error {
		// Deterministic flush order for tests/logging.
		ids := make([]int, 0, len(debounced))
		for id := range debounced {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		for _, id := range ids {
			if err := flushConv(id); err != nil {
				return err
			}
		}
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			_ = flushAll()
			return nil
		case id := <-flushCh:
			if err := flushConv(id); err != nil {
				return err
			}
		case ev, ok := <-events:
			if !ok {
				_ = flushAll()
				if ctx.Err() != nil {
					return nil
				}
				return fmt.Errorf("event channel closed")
			}
			if ev.Err != nil {
				_ = flushAll()
				if ctx.Err() != nil {
					return nil
				}
				return ev.Err
			}

			rawEnvelope := json.RawMessage(ev.Data)

			// Parse the outer Chatwoot event envelope.
			var wsEvent chatwootWSEvent
			if err := json.Unmarshal(ev.Data, &wsEvent); err != nil {
				continue // skip malformed events
			}

			// Filter by event type.
			if allowedEvents != nil {
				if _, ok := allowedEvents[wsEvent.Event]; !ok {
					continue
				}
			}

			// message.created has a strongly-typed payload.
			if wsEvent.Event == "message.created" {
				var msg api.Message
				if err := json.Unmarshal(wsEvent.Data, &msg); err != nil {
					continue
				}

				// Filter by conversation ID (WebSocket sends all account events).
				if convID != 0 && msg.ConversationID != convID {
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

				// Debounce (batch) rapid messages per conversation.
				if debounce <= 0 {
					if err := printFollowMessageWithRaw(cmd, msg, "ws", rawEnvelope, includeRaw); err != nil {
						return err
					}
					continue
				}

				id := msg.ConversationID
				buf := debounced[id]
				if buf == nil {
					buf = &debounceBuf{}
					debounced[id] = buf
				}
				buf.messages = append(buf.messages, followMsg{msg: msg, raw: rawEnvelope})
				if buf.timer == nil {
					// Flush after debounce duration from the first message in the batch.
					buf.timer = time.AfterFunc(debounce, func() {
						select {
						case flushCh <- id:
						case <-done:
						}
					})
				}
				continue
			}

			// For non-message events, try to filter by conversation id if following a single conversation.
			if convID != 0 {
				if id := conversationIDFromEvent(wsEvent.Event, wsEvent.Data); id != 0 && id != convID {
					continue
				}
			}

			if err := printFollowEvent(cmd, wsEvent, "ws", rawEnvelope, includeRaw); err != nil {
				return err
			}
		}
	}

	// unreachable
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

type followMsg struct {
	msg api.Message
	raw json.RawMessage
}

func printFollowMessage(cmd *cobra.Command, m api.Message, source string) error {
	return printFollowMessageWithRaw(cmd, m, source, nil, false)
}

func printFollowMessageWithRaw(cmd *cobra.Command, m api.Message, source string, rawEnvelope json.RawMessage, includeRaw bool) error {
	if isAgent(cmd) {
		summary := agentfmt.MessageSummaryFromMessage(m)
		out := map[string]any{
			"kind":   "conversations.follow",
			"event":  "message.created",
			"source": source,
			"item":   summary,
		}
		if includeRaw && len(rawEnvelope) > 0 {
			out["raw"] = rawEnvelope
		}
		return writeStreamJSON(cmd, out)
	}
	if isJSON(cmd) {
		// Emit as JSON lines.
		out := map[string]any{
			"source": source,
			"type":   "message",
			"event":  "message.created",
			"item":   m,
		}
		if includeRaw && len(rawEnvelope) > 0 {
			out["raw"] = rawEnvelope
		}
		return writeStreamJSON(cmd, out)
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

func printFollowMessageBatch(cmd *cobra.Command, messages []followMsg, source string, includeRaw bool) error {
	if len(messages) == 0 {
		return nil
	}
	if len(messages) == 1 {
		return printFollowMessageWithRaw(cmd, messages[0].msg, source, messages[0].raw, includeRaw)
	}

	convID := messages[0].msg.ConversationID

	if isAgent(cmd) {
		rawItems := make([]json.RawMessage, 0, len(messages))
		msgs := make([]api.Message, 0, len(messages))
		for _, m := range messages {
			msgs = append(msgs, m.msg)
			if len(m.raw) > 0 {
				rawItems = append(rawItems, m.raw)
			} else {
				rawItems = append(rawItems, nil)
			}
		}
		out := map[string]any{
			"kind":            "conversations.follow",
			"event":           "message.batch",
			"source":          source,
			"conversation_id": convID,
			"items":           agentfmt.MessageSummaries(msgs),
		}
		if includeRaw {
			out["raw_items"] = rawItems
		}
		return writeStreamJSON(cmd, out)
	}
	if isJSON(cmd) {
		rawItems := make([]json.RawMessage, 0, len(messages))
		msgs := make([]api.Message, 0, len(messages))
		for _, m := range messages {
			msgs = append(msgs, m.msg)
			if len(m.raw) > 0 {
				rawItems = append(rawItems, m.raw)
			} else {
				rawItems = append(rawItems, nil)
			}
		}
		out := map[string]any{
			"type":            "message_batch",
			"event":           "message.batch",
			"source":          source,
			"conversation_id": convID,
			"items":           msgs,
		}
		if includeRaw {
			out["raw_items"] = rawItems
		}
		return writeStreamJSON(cmd, out)
	}

	// Text output: batch as a single entry with concatenated content.
	first := messages[0].msg
	last := messages[len(messages)-1].msg
	ts := time.Unix(last.CreatedAt, 0).Format("15:04:05")
	sender := "-"
	if first.Sender != nil && strings.TrimSpace(first.Sender.Name) != "" {
		sender = first.Sender.Name
	}
	kind := first.MessageTypeName()
	privacy := ""
	if first.Private {
		privacy = " [private]"
	}

	var parts []string
	for _, m := range messages {
		parts = append(parts, strings.TrimSpace(m.msg.Content))
	}
	content := strings.TrimSpace(strings.Join(parts, "\n"))
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s (%s)%s: %s\n", ts, sender, kind, privacy, content)
	return err
}

func printFollowEvent(cmd *cobra.Command, wsEvent chatwootWSEvent, source string, rawEnvelope json.RawMessage, includeRaw bool) error {
	if isAgent(cmd) {
		out := map[string]any{
			"kind":   "conversations.follow",
			"event":  wsEvent.Event,
			"source": source,
			"data":   json.RawMessage(wsEvent.Data),
		}
		if includeRaw && len(rawEnvelope) > 0 {
			out["raw"] = rawEnvelope
		}
		return writeStreamJSON(cmd, out)
	}
	if isJSON(cmd) {
		out := map[string]any{
			"type":   "event",
			"event":  wsEvent.Event,
			"source": source,
			"data":   json.RawMessage(wsEvent.Data),
		}
		if includeRaw && len(rawEnvelope) > 0 {
			out["raw"] = rawEnvelope
		}
		return writeStreamJSON(cmd, out)
	}

	// Text output: attempt to format known event types.
	ts := time.Now().Format("15:04:05")
	switch wsEvent.Event {
	case "conversation.created":
		id, contactName, inboxID := conversationCreatedSummary(wsEvent.Data)
		if id == 0 {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", ts, wsEvent.Event)
			return err
		}
		msg := fmt.Sprintf("New conversation #%d", id)
		if strings.TrimSpace(contactName) != "" {
			msg += fmt.Sprintf(" from %s", contactName)
		}
		if inboxID != 0 {
			msg += fmt.Sprintf(" (inbox: %d)", inboxID)
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", ts, msg)
		return err
	case "conversation.status_changed":
		id, status := conversationStatusChangedSummary(wsEvent.Data)
		if id == 0 {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", ts, wsEvent.Event)
			return err
		}
		if strings.TrimSpace(status) == "" {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] Conversation #%d status changed\n", ts, id)
			return err
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] Conversation #%d status changed to %s\n", ts, id, status)
		return err
	case "assignee.changed":
		id, assigneeName, assigned := assigneeChangedSummary(wsEvent.Data)
		if id == 0 {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", ts, wsEvent.Event)
			return err
		}
		if assigned {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] Conversation #%d assigned to %s\n", ts, id, assigneeName)
			return err
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] Conversation #%d unassigned\n", ts, id)
		return err
	case "conversation.typing_on", "conversation.typing_off":
		id, userName := typingSummary(wsEvent.Data)
		if id == 0 {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", ts, wsEvent.Event)
			return err
		}
		name := strings.TrimSpace(userName)
		if name == "" {
			name = "Someone"
		}
		if wsEvent.Event == "conversation.typing_on" {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s is typing in #%d...\n", ts, name, id)
			return err
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s stopped typing in #%d\n", ts, name, id)
		return err
	default:
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", ts, wsEvent.Event)
		return err
	}
}

func writeStreamJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	// JSONL is for streaming/piping: emit a single JSON object per line.
	if outfmt.IsJSONL(cmd.Context()) {
		return enc.Encode(v)
	}
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func dedupeStrings(in []string) []string {
	if len(in) == 0 {
		return in
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
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
	return out
}

func conversationIDFromEvent(_ string, data json.RawMessage) int {
	// Best-effort extraction across event types:
	// - { "id": 123 }
	// - { "conversation_id": 123 }
	// - { "conversation": { "id": 123 } }
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return 0
	}
	if id := anyToInt(m["id"]); id > 0 {
		return id
	}
	if id := anyToInt(m["conversation_id"]); id > 0 {
		return id
	}
	if conv, ok := m["conversation"].(map[string]any); ok {
		if id := anyToInt(conv["id"]); id > 0 {
			return id
		}
	}
	return 0
}

func conversationCreatedSummary(data json.RawMessage) (id int, contactName string, inboxID int) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return 0, "", 0
	}
	id = anyToInt(m["id"])
	inboxID = anyToInt(m["inbox_id"])
	if contact, ok := m["contact"].(map[string]any); ok {
		if name, ok := contact["name"].(string); ok {
			contactName = name
		}
	}
	return id, contactName, inboxID
}

func conversationStatusChangedSummary(data json.RawMessage) (id int, status string) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return 0, ""
	}
	id = anyToInt(m["id"])
	if s, ok := m["status"].(string); ok {
		status = s
	}
	return id, status
}

func assigneeChangedSummary(data json.RawMessage) (id int, assigneeName string, assigned bool) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return 0, "", false
	}
	id = anyToInt(m["id"])
	assignee, ok := m["assignee"].(map[string]any)
	if !ok || assignee == nil {
		return id, "", false
	}
	if name, ok := assignee["name"].(string); ok && strings.TrimSpace(name) != "" {
		return id, name, true
	}
	// Assignee exists but has no name; treat as assigned.
	return id, "", true
}

func typingSummary(data json.RawMessage) (conversationID int, userName string) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return 0, ""
	}
	if conv, ok := m["conversation"].(map[string]any); ok {
		conversationID = anyToInt(conv["id"])
	}
	if user, ok := m["user"].(map[string]any); ok {
		if name, ok := user["name"].(string); ok {
			userName = name
		}
	}
	return conversationID, userName
}

func anyToInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(n))
		return i
	default:
		return 0
	}
}
