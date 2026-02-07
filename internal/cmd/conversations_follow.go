package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
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
		incomingOnly   bool
		tail           int
		followAll      bool
		events         []string
		showTyping     bool
		debounce       time.Duration
		includeRaw     bool
		withContext    bool
		contextMsgs    int
		cursorFile     string
		sinceID        int
		sinceTime      string
		filterInbox    int
		filterStatus   string
		filterAgent    int
		filterLabels   []string
		filterPrio     string
		filterContact  int
		onlyUnassigned bool
		excludePrivate bool
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

			allowAllEvents := false
			for _, e := range events {
				if e == "all" || e == "*" {
					allowAllEvents = true
					break
				}
			}

			contextEnabled := withContext
			if isAgent(cmd) && !cmd.Flags().Changed("context") {
				contextEnabled = true
			}
			if contextMsgs <= 0 {
				contextMsgs = 10
			}

			if filterStatus != "" {
				if err := validateStatus(filterStatus); err != nil {
					return err
				}
			}
			if filterPrio != "" {
				if err := validatePriority(filterPrio); err != nil {
					return err
				}
			}

			filters := followFilters{
				InboxID:        filterInbox,
				Status:         filterStatus,
				AssigneeID:     filterAgent,
				Labels:         dedupeStrings(filterLabels),
				Priority:       filterPrio,
				ContactID:      filterContact,
				OnlyUnassigned: onlyUnassigned,
				ExcludePrivate: excludePrivate,
			}

			var allowedEvents map[string]struct{}
			if !allowAllEvents {
				allowedEvents = make(map[string]struct{}, len(events))
				for _, e := range events {
					e = strings.TrimSpace(e)
					if e == "" || e == "all" || e == "*" {
						continue
					}
					allowedEvents[e] = struct{}{}
				}
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

			snapshotClient := followAPISnapshotClient{client: client}

			sinceT, err := parseSinceTime(sinceTime, time.Now())
			if err != nil {
				return err
			}
			minCreatedAt := int64(0)
			if !sinceT.IsZero() {
				minCreatedAt = sinceT.Unix()
			}

			lastSeenID := 0
			if sinceID > 0 {
				lastSeenID = sinceID
			}
			if cursorFile != "" && sinceID <= 0 {
				cur, err := loadFollowCursor(cursorFile)
				if err != nil {
					return err
				}
				// Ignore cursors from other accounts.
				if cur.LastSeenMessageID > 0 && (cur.AccountID == 0 || cur.AccountID == client.AccountID) && (cur.BaseURL == "" || cur.BaseURL == client.BaseURL) {
					lastSeenID = max(lastSeenID, cur.LastSeenMessageID)
				}
			}

			var cw *followCursorWriter
			if cursorFile != "" {
				w, err := newFollowCursorWriter(cursorFile, client.BaseURL, client.AccountID, lastSeenID, 1*time.Second)
				if err != nil {
					return err
				}
				cw = w
				defer func() { _ = cw.Flush() }()
			}

			// If we're tailing a single conversation and context is enabled, emit the
			// snapshot once up front so the agent has full state before history/ws.
			if contextEnabled && convID != 0 && tail > 0 {
				if err := emitConversationSnapshot(ctx, cmd, snapshotClient, convID, contextMsgs); err != nil {
					return err
				}
			}

			if tail > 0 {
				// For a single conversation, we can apply meta-based filters once.
				var meta *convMeta
				if convID != 0 && filters.metaFiltersEnabled() {
					// Best-effort; if meta can't be fetched, fall back to no meta filtering.
					meta, _ = fetchConversationMeta(ctx, snapshotClient, convID, filters.Labels)
				}

				msgs, err := client.Messages().ListWithLimit(ctx, convID, tail, 10)
				if err == nil && len(msgs) > 0 {
					// Print oldest -> newest.
					sort.Slice(msgs, func(i, j int) bool { return msgs[i].ID < msgs[j].ID })
					for _, m := range msgs {
						if minCreatedAt > 0 && m.CreatedAt < minCreatedAt {
							continue
						}
						if m.ID <= lastSeenID {
							continue
						}
						if filters.ExcludePrivate && m.Private {
							continue
						}
						if incomingOnly && m.MessageType != api.MessageTypeIncoming {
							continue
						}
						if filters.metaFiltersEnabled() && meta != nil && !filters.matchMeta(meta) {
							continue
						}
						if err := printFollowMessage(cmd, m, "history"); err != nil {
							return err
						}
						lastSeenID = m.ID
						if cw != nil {
							cw.Update(lastSeenID)
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
				onLastSeen := func(id int) {
					if cw != nil {
						cw.Update(id)
					}
				}
				err := followViaWebSocket(ctx, cmd, snapshotClient, contextEnabled, contextMsgs, minCreatedAt, onLastSeen, filters, cableURL, channelID, convID, incomingOnly, &lastSeenID, allowedEvents, debounce, includeRaw)
				if ctx.Err() != nil {
					return nil
				}
				if cw != nil {
					_ = cw.Flush()
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
	cmd.Flags().StringSliceVar(&events, "events", []string{"message.created"}, "Event types to show (or 'all'): message.created,message.updated,conversation.created,conversation.updated,conversation.status_changed,assignee.changed,label.added,label.removed,conversation.typing_on,conversation.typing_off")
	cmd.Flags().BoolVar(&showTyping, "typing", false, "Show typing indicators")
	cmd.Flags().DurationVar(&debounce, "debounce", 0, "Batch rapid messages from same conversation (e.g., 2s)")
	cmd.Flags().BoolVar(&includeRaw, "raw", false, "Include raw WebSocket payload (JSON/agent modes only)")
	cmd.Flags().BoolVar(&withContext, "context", false, "Emit a conversation snapshot on the first event per conversation (default true in agent mode)")
	cmd.Flags().IntVar(&contextMsgs, "context-messages", 10, "Number of recent messages to include in conversation snapshots")
	cmd.Flags().StringVar(&cursorFile, "cursor-file", "", "Persist last seen message id to a file for resume/restart")
	cmd.Flags().IntVar(&sinceID, "since-id", 0, "Skip messages with id <= this value (useful for resume)")
	cmd.Flags().StringVar(&sinceTime, "since-time", "", "Skip messages created before this time (RFC3339, unix seconds, or duration like 24h)")
	cmd.Flags().IntVar(&filterInbox, "inbox", 0, "Only show events for conversations in this inbox ID")
	cmd.Flags().StringVar(&filterStatus, "status", "", "Only show events for conversations with this status (open|resolved|pending|snoozed)")
	cmd.Flags().IntVar(&filterAgent, "assignee", 0, "Only show events for conversations assigned to this agent ID")
	cmd.Flags().StringSliceVar(&filterLabels, "label", nil, "Only show events for conversations that have all of these labels")
	cmd.Flags().StringVar(&filterPrio, "priority", "", "Only show events for conversations with this priority (urgent|high|medium|low|none)")
	cmd.Flags().IntVar(&filterContact, "contact", 0, "Only show events for conversations with this contact ID")
	cmd.Flags().BoolVar(&onlyUnassigned, "only-unassigned", false, "Only show events for conversations with no assignee")
	cmd.Flags().BoolVar(&excludePrivate, "exclude-private", false, "Exclude private messages")
	return cmd
}

// chatwootWSEvent is the outer envelope of a Chatwoot WebSocket event.
type chatwootWSEvent struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

type followSnapshotClient interface {
	GetConversation(ctx context.Context, id int) (*api.Conversation, error)
	GetContact(ctx context.Context, id int) (*api.Contact, error)
	ListMessages(ctx context.Context, conversationID, limit, maxPages int) ([]api.Message, error)
	ListLabels(ctx context.Context, conversationID int) ([]string, error)
}

type followAPISnapshotClient struct {
	client *api.Client
}

func (c followAPISnapshotClient) GetConversation(ctx context.Context, id int) (*api.Conversation, error) {
	return c.client.Conversations().Get(ctx, id)
}

func (c followAPISnapshotClient) GetContact(ctx context.Context, id int) (*api.Contact, error) {
	return c.client.Contacts().Get(ctx, id)
}

func (c followAPISnapshotClient) ListMessages(ctx context.Context, conversationID, limit, maxPages int) ([]api.Message, error) {
	return c.client.Messages().ListWithLimit(ctx, conversationID, limit, maxPages)
}

func (c followAPISnapshotClient) ListLabels(ctx context.Context, conversationID int) ([]string, error) {
	return c.client.Conversations().Labels(ctx, conversationID)
}

// followViaWebSocket connects to ActionCable, subscribes, and processes events
// until the connection drops or ctx is cancelled. Returns non-nil error on disconnect.
func followViaWebSocket(ctx context.Context, cmd *cobra.Command, snapshotClient followSnapshotClient, contextEnabled bool, contextMsgs int, minCreatedAt int64, onLastSeen func(int), filters followFilters, cableURL string, channelID actioncable.ChannelID, convID int, incomingOnly bool, lastSeenID *int, allowedEvents map[string]struct{}, debounce time.Duration, includeRaw bool) error {
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

	snapshotted := make(map[int]bool)
	convCache := make(map[int]*convMeta)

	ensureMeta := func(conversationID int) (*convMeta, error) {
		if conversationID <= 0 || snapshotClient == nil {
			return nil, nil
		}
		if m := convCache[conversationID]; m != nil && m.Hydrated {
			return m, nil
		}
		m, err := fetchConversationMeta(ctx, snapshotClient, conversationID, filters.Labels)
		if err != nil {
			return nil, err
		}
		if m != nil {
			convCache[conversationID] = m
		}
		return m, nil
	}

	updateMetaFromEvent := func(wsEvent chatwootWSEvent) {
		id := conversationIDFromEvent(wsEvent.Event, wsEvent.Data)
		if id <= 0 {
			return
		}
		m := convCache[id]
		if m == nil {
			m = &convMeta{ID: id}
			convCache[id] = m
		}
		m.ApplyEvent(wsEvent.Event, wsEvent.Data)
	}

	maybeSnapshot := func(conversationID int) error {
		if !contextEnabled {
			return nil
		}
		if conversationID <= 0 {
			return nil
		}
		if snapshotted[conversationID] {
			return nil
		}
		snapshotted[conversationID] = true
		if snapshotClient == nil {
			return nil
		}
		return emitConversationSnapshot(ctx, cmd, snapshotClient, conversationID, contextMsgs)
	}

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

		if err := maybeSnapshot(id); err != nil {
			return err
		}
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
			if wsEvent.Event == "message.created" || wsEvent.Event == "message.updated" {
				var msg api.Message
				if err := json.Unmarshal(wsEvent.Data, &msg); err != nil {
					continue
				}

				// Filter by conversation ID (WebSocket sends all account events).
				if convID != 0 && msg.ConversationID != convID {
					continue
				}

				// Filter by created_at threshold (useful for resume).
				if minCreatedAt > 0 && msg.CreatedAt < minCreatedAt {
					continue
				}

				// Dedup by message ID.
				if lastSeenID != nil && msg.ID <= *lastSeenID {
					continue
				}
				if lastSeenID != nil {
					*lastSeenID = msg.ID
					if onLastSeen != nil {
						onLastSeen(*lastSeenID)
					}
				}

				// Filter by message type if --incoming-only.
				if incomingOnly && msg.MessageType != api.MessageTypeIncoming {
					continue
				}
				if filters.ExcludePrivate && msg.Private {
					continue
				}

				if filters.metaFiltersEnabled() {
					meta, _ := ensureMeta(msg.ConversationID)
					if meta == nil || !filters.matchMeta(meta) {
						continue
					}
				}

				// Debounce (batch) rapid messages per conversation.
				if debounce <= 0 || wsEvent.Event != "message.created" {
					if err := maybeSnapshot(msg.ConversationID); err != nil {
						return err
					}
					if err := printFollowMessageWithRaw(cmd, wsEvent.Event, msg, "ws", rawEnvelope, includeRaw); err != nil {
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
			eventConvID := 0
			if convID != 0 {
				eventConvID = conversationIDFromEvent(wsEvent.Event, wsEvent.Data)
				if eventConvID != 0 && eventConvID != convID {
					continue
				}
			} else {
				eventConvID = conversationIDFromEvent(wsEvent.Event, wsEvent.Data)
			}

			updateMetaFromEvent(wsEvent)
			if filters.metaFiltersEnabled() {
				if eventConvID <= 0 {
					continue
				}
				meta, _ := ensureMeta(eventConvID)
				if meta == nil || !filters.matchMeta(meta) {
					continue
				}
			}

			if err := maybeSnapshot(eventConvID); err != nil {
				return err
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
	return printFollowMessageWithRaw(cmd, "message.created", m, source, nil, false)
}

func printFollowMessageWithRaw(cmd *cobra.Command, event string, m api.Message, source string, rawEnvelope json.RawMessage, includeRaw bool) error {
	if isAgent(cmd) {
		summary := agentfmt.MessageSummaryFromMessage(m)
		out := map[string]any{
			"kind":            "conversations.follow",
			"event":           event,
			"source":          source,
			"conversation_id": m.ConversationID,
			"ts":              time.Unix(m.CreatedAt, 0).UTC().Format(time.RFC3339Nano),
			"item":            summary,
		}
		if includeRaw && len(rawEnvelope) > 0 {
			out["raw"] = rawEnvelope
		}
		return writeStreamJSON(cmd, out)
	}
	if isJSON(cmd) {
		// Emit as JSON lines.
		out := map[string]any{
			"kind":            "conversations.follow",
			"source":          source,
			"type":            "message",
			"event":           event,
			"conversation_id": m.ConversationID,
			"ts":              time.Unix(m.CreatedAt, 0).UTC().Format(time.RFC3339Nano),
			"item":            m,
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
		return printFollowMessageWithRaw(cmd, "message.created", messages[0].msg, source, messages[0].raw, includeRaw)
	}

	convID := messages[0].msg.ConversationID
	ts := time.Unix(messages[len(messages)-1].msg.CreatedAt, 0).UTC().Format(time.RFC3339Nano)

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
			"ts":              ts,
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
			"kind":            "conversations.follow",
			"type":            "message_batch",
			"event":           "message.batch",
			"source":          source,
			"conversation_id": convID,
			"ts":              ts,
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
	tsHuman := time.Unix(last.CreatedAt, 0).Format("15:04:05")
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
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s (%s)%s: %s\n", tsHuman, sender, kind, privacy, content)
	return err
}

func printFollowEvent(cmd *cobra.Command, wsEvent chatwootWSEvent, source string, rawEnvelope json.RawMessage, includeRaw bool) error {
	convID := conversationIDFromEvent(wsEvent.Event, wsEvent.Data)
	ts := time.Now().UTC().Format(time.RFC3339Nano)

	if isAgent(cmd) {
		out := map[string]any{
			"kind":            "conversations.follow",
			"event":           wsEvent.Event,
			"source":          source,
			"conversation_id": convID,
			"ts":              ts,
			"data":            json.RawMessage(wsEvent.Data),
		}
		if includeRaw && len(rawEnvelope) > 0 {
			out["raw"] = rawEnvelope
		}
		return writeStreamJSON(cmd, out)
	}
	if isJSON(cmd) {
		out := map[string]any{
			"kind":            "conversations.follow",
			"type":            "event",
			"event":           wsEvent.Event,
			"source":          source,
			"conversation_id": convID,
			"ts":              ts,
			"data":            json.RawMessage(wsEvent.Data),
		}
		if includeRaw && len(rawEnvelope) > 0 {
			out["raw"] = rawEnvelope
		}
		return writeStreamJSON(cmd, out)
	}

	// Text output: attempt to format known event types.
	tsHuman := time.Now().Format("15:04:05")
	switch wsEvent.Event {
	case "conversation.created":
		id, contactName, inboxID := conversationCreatedSummary(wsEvent.Data)
		if id == 0 {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", tsHuman, wsEvent.Event)
			return err
		}
		msg := fmt.Sprintf("New conversation #%d", id)
		if strings.TrimSpace(contactName) != "" {
			msg += fmt.Sprintf(" from %s", contactName)
		}
		if inboxID != 0 {
			msg += fmt.Sprintf(" (inbox: %d)", inboxID)
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", tsHuman, msg)
		return err
	case "conversation.status_changed":
		id, status := conversationStatusChangedSummary(wsEvent.Data)
		if id == 0 {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", tsHuman, wsEvent.Event)
			return err
		}
		if strings.TrimSpace(status) == "" {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] Conversation #%d status changed\n", tsHuman, id)
			return err
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] Conversation #%d status changed to %s\n", tsHuman, id, status)
		return err
	case "assignee.changed":
		id, assigneeName, assigned := assigneeChangedSummary(wsEvent.Data)
		if id == 0 {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", tsHuman, wsEvent.Event)
			return err
		}
		if assigned {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] Conversation #%d assigned to %s\n", tsHuman, id, assigneeName)
			return err
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] Conversation #%d unassigned\n", tsHuman, id)
		return err
	case "conversation.typing_on", "conversation.typing_off":
		id, userName := typingSummary(wsEvent.Data)
		if id == 0 {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", tsHuman, wsEvent.Event)
			return err
		}
		name := strings.TrimSpace(userName)
		if name == "" {
			name = "Someone"
		}
		if wsEvent.Event == "conversation.typing_on" {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s is typing in #%d...\n", tsHuman, name, id)
			return err
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s stopped typing in #%d\n", tsHuman, name, id)
		return err
	default:
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", tsHuman, wsEvent.Event)
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

func emitConversationSnapshot(ctx context.Context, cmd *cobra.Command, snapshotClient followSnapshotClient, conversationID int, maxMessages int) error {
	if snapshotClient == nil || conversationID <= 0 {
		return nil
	}

	// Keep snapshot fetches bounded so a slow API doesn't block the stream forever.
	snapCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conv, err := snapshotClient.GetConversation(snapCtx, conversationID)
	if err != nil {
		if isJSON(cmd) || isAgent(cmd) {
			return writeStreamJSON(cmd, map[string]any{
				"kind":            "conversations.follow",
				"event":           "conversation.snapshot_error",
				"source":          "api",
				"conversation_id": conversationID,
				"ts":              time.Now().UTC().Format(time.RFC3339Nano),
				"error":           err.Error(),
			})
		}
		return nil
	}

	// Best-effort: attach labels (the show endpoint doesn't always include them).
	if labels, err := snapshotClient.ListLabels(snapCtx, conversationID); err == nil && len(labels) > 0 {
		conv.Labels = labels
	}

	var contact *api.Contact
	if conv.ContactID > 0 {
		if c, err := snapshotClient.GetContact(snapCtx, conv.ContactID); err == nil {
			contact = c
		}
	}

	var msgs []api.Message
	if maxMessages > 0 {
		if m, err := snapshotClient.ListMessages(snapCtx, conversationID, maxMessages, 10); err == nil && len(m) > 0 {
			// Oldest -> newest.
			sort.Slice(m, func(i, j int) bool { return m[i].ID < m[j].ID })
			msgs = m
		}
	}

	if isAgent(cmd) {
		payload := map[string]any{
			"kind":            "conversations.follow",
			"event":           "conversation.snapshot",
			"source":          "api",
			"conversation_id": conversationID,
			"ts":              time.Now().UTC().Format(time.RFC3339Nano),
			"conversation":    agentfmt.ConversationDetailFromConversation(*conv),
			"messages":        agentfmt.MessageSummaries(msgs),
		}
		if contact != nil {
			payload["contact"] = agentfmt.ContactDetailFromContact(*contact)
		}
		return writeStreamJSON(cmd, payload)
	}

	if isJSON(cmd) {
		payload := map[string]any{
			"kind":            "conversations.follow",
			"event":           "conversation.snapshot",
			"source":          "api",
			"conversation_id": conversationID,
			"ts":              time.Now().UTC().Format(time.RFC3339Nano),
			"conversation":    conv,
			"messages":        msgs,
		}
		if contact != nil {
			payload["contact"] = contact
		}
		return writeStreamJSON(cmd, payload)
	}

	// Text mode: keep snapshots silent by default.
	return nil
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

type followCursor struct {
	Version           int    `json:"version"`
	BaseURL           string `json:"base_url,omitempty"`
	AccountID         int    `json:"account_id,omitempty"`
	LastSeenMessageID int    `json:"last_seen_message_id"`
	UpdatedAt         string `json:"updated_at,omitempty"`
}

func loadFollowCursor(path string) (followCursor, error) {
	var cur followCursor
	if strings.TrimSpace(path) == "" {
		return cur, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cur, nil
		}
		return cur, fmt.Errorf("read cursor file: %w", err)
	}
	if err := json.Unmarshal(b, &cur); err != nil {
		return cur, fmt.Errorf("parse cursor file: %w", err)
	}
	return cur, nil
}

func saveFollowCursor(path string, cur followCursor) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create cursor dir: %w", err)
		}
	}

	cur.Version = 1
	cur.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)

	tmp, err := os.CreateTemp(dir, ".chatwoot-follow-cursor-*")
	if err != nil {
		return fmt.Errorf("create cursor temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cur); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write cursor temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close cursor temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace cursor file: %w", err)
	}
	return nil
}

type followCursorWriter struct {
	Path        string
	BaseURL     string
	AccountID   int
	MinInterval time.Duration

	LastSeenID  int
	LastFlushed int
	LastFlushAt time.Time
}

func newFollowCursorWriter(path, baseURL string, accountID int, initialLastSeen int, minInterval time.Duration) (*followCursorWriter, error) {
	w := &followCursorWriter{
		Path:        path,
		BaseURL:     baseURL,
		AccountID:   accountID,
		MinInterval: minInterval,
		LastSeenID:  initialLastSeen,
	}
	return w, nil
}

func (w *followCursorWriter) Update(lastSeenID int) {
	if w == nil || w.Path == "" {
		return
	}
	if lastSeenID <= w.LastSeenID {
		return
	}
	w.LastSeenID = lastSeenID
	if w.MinInterval <= 0 || w.LastFlushAt.IsZero() || time.Since(w.LastFlushAt) >= w.MinInterval {
		_ = w.Flush()
	}
}

func (w *followCursorWriter) Flush() error {
	if w == nil || w.Path == "" {
		return nil
	}
	if w.LastSeenID <= 0 || w.LastSeenID == w.LastFlushed {
		return nil
	}
	cur := followCursor{
		BaseURL:           w.BaseURL,
		AccountID:         w.AccountID,
		LastSeenMessageID: w.LastSeenID,
	}
	if err := saveFollowCursor(w.Path, cur); err != nil {
		return err
	}
	w.LastFlushed = w.LastSeenID
	w.LastFlushAt = time.Now()
	return nil
}

func parseSinceTime(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}

	// Allow durations as "look back" values (e.g. 24h).
	if d, err := time.ParseDuration(s); err == nil {
		if d < 0 {
			d = -d
		}
		return now.Add(-d), nil
	}

	// Allow unix seconds (or unix milliseconds) as integers.
	if i, err := strconv.ParseInt(s, 10, 64); err == nil && i > 0 {
		// Heuristic: > 1e12 is almost certainly milliseconds.
		if i > 1_000_000_000_000 {
			return time.UnixMilli(i), nil
		}
		return time.Unix(i, 0), nil
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid --since-time %q (use RFC3339, unix seconds, or duration like 24h)", s)
}

type followFilters struct {
	InboxID        int
	Status         string
	AssigneeID     int
	Labels         []string
	Priority       string
	ContactID      int
	OnlyUnassigned bool
	ExcludePrivate bool
}

func (f followFilters) metaFiltersEnabled() bool {
	return f.InboxID > 0 ||
		f.Status != "" ||
		f.AssigneeID > 0 ||
		len(f.Labels) > 0 ||
		f.Priority != "" ||
		f.ContactID > 0 ||
		f.OnlyUnassigned
}

func (f followFilters) matchMeta(m *convMeta) bool {
	if m == nil {
		return false
	}
	if f.InboxID > 0 && m.InboxID != f.InboxID {
		return false
	}
	if f.Status != "" && strings.TrimSpace(m.Status) != f.Status {
		return false
	}
	if f.Priority != "" && strings.TrimSpace(m.Priority) != f.Priority {
		return false
	}
	if f.ContactID > 0 && m.ContactID != f.ContactID {
		return false
	}
	if f.AssigneeID > 0 {
		if m.AssigneeID == nil || *m.AssigneeID != f.AssigneeID {
			return false
		}
	}
	if f.OnlyUnassigned {
		if m.AssigneeID != nil && *m.AssigneeID != 0 {
			return false
		}
	}
	if len(f.Labels) > 0 {
		for _, lbl := range f.Labels {
			if lbl == "" {
				continue
			}
			if !m.HasLabel(lbl) {
				return false
			}
		}
	}
	return true
}

type convMeta struct {
	ID         int
	InboxID    int
	Status     string
	Priority   string
	AssigneeID *int
	ContactID  int
	Labels     map[string]bool
	Hydrated   bool
}

func (m *convMeta) HasLabel(label string) bool {
	if m == nil || m.Labels == nil {
		return false
	}
	return m.Labels[label]
}

func (m *convMeta) SetLabels(labels []string) {
	if m == nil {
		return
	}
	if len(labels) == 0 {
		m.Labels = nil
		return
	}
	set := make(map[string]bool, len(labels))
	for _, l := range labels {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		set[l] = true
	}
	m.Labels = set
}

func (m *convMeta) ApplyConversation(conv api.Conversation) {
	if m == nil {
		return
	}
	m.ID = conv.ID
	m.InboxID = conv.InboxID
	m.Status = conv.Status
	if conv.Priority != nil {
		m.Priority = *conv.Priority
	} else {
		m.Priority = "none"
	}
	m.AssigneeID = conv.AssigneeID
	m.ContactID = conv.ContactID
	if len(conv.Labels) > 0 {
		m.SetLabels(conv.Labels)
	}
	m.Hydrated = true
}

func (m *convMeta) ApplyEvent(event string, data json.RawMessage) {
	if m == nil {
		return
	}
	switch event {
	case "conversation.status_changed":
		_, status := conversationStatusChangedSummary(data)
		if strings.TrimSpace(status) != "" {
			m.Status = status
		}
	case "assignee.changed":
		var payload map[string]any
		if err := json.Unmarshal(data, &payload); err != nil {
			return
		}
		assigneeAny, ok := payload["assignee"]
		if !ok || assigneeAny == nil {
			m.AssigneeID = nil
			return
		}
		assignee, ok := assigneeAny.(map[string]any)
		if !ok {
			return
		}
		id := anyToInt(assignee["id"])
		if id > 0 {
			m.AssigneeID = &id
		}
	case "label.added":
		var payload map[string]any
		if err := json.Unmarshal(data, &payload); err != nil {
			return
		}
		if lbl, ok := payload["label"].(string); ok && strings.TrimSpace(lbl) != "" {
			if m.Labels == nil {
				m.Labels = make(map[string]bool)
			}
			m.Labels[strings.TrimSpace(lbl)] = true
		}
	case "label.removed":
		var payload map[string]any
		if err := json.Unmarshal(data, &payload); err != nil {
			return
		}
		if lbl, ok := payload["label"].(string); ok && strings.TrimSpace(lbl) != "" {
			if m.Labels != nil {
				delete(m.Labels, strings.TrimSpace(lbl))
			}
		}
	case "conversation.created", "conversation.updated":
		var conv api.Conversation
		if err := json.Unmarshal(data, &conv); err == nil && conv.ID > 0 {
			m.ApplyConversation(conv)
			return
		}
		// Best-effort fallback to partial fields.
		var payload map[string]any
		if err := json.Unmarshal(data, &payload); err != nil {
			return
		}
		if inboxID := anyToInt(payload["inbox_id"]); inboxID > 0 {
			m.InboxID = inboxID
		}
		if status, ok := payload["status"].(string); ok && strings.TrimSpace(status) != "" {
			m.Status = status
		}
		if prio, ok := payload["priority"].(string); ok && strings.TrimSpace(prio) != "" {
			m.Priority = prio
		}
		if contactID := anyToInt(payload["contact_id"]); contactID > 0 {
			m.ContactID = contactID
		}
		if labelsAny, ok := payload["labels"].([]any); ok && len(labelsAny) > 0 {
			var labels []string
			for _, v := range labelsAny {
				if s, ok := v.(string); ok {
					labels = append(labels, s)
				}
			}
			m.SetLabels(labels)
		}
	}
}

func fetchConversationMeta(ctx context.Context, snapshotClient followSnapshotClient, conversationID int, wantLabels []string) (*convMeta, error) {
	if snapshotClient == nil || conversationID <= 0 {
		return nil, nil
	}
	metaCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conv, err := snapshotClient.GetConversation(metaCtx, conversationID)
	if err != nil {
		return nil, err
	}
	m := &convMeta{ID: conversationID}
	m.ApplyConversation(*conv)

	// If label filters are active, fetch the label set explicitly to ensure correctness.
	if len(wantLabels) > 0 {
		if labels, err := snapshotClient.ListLabels(metaCtx, conversationID); err == nil {
			m.SetLabels(labels)
		}
	}
	return m, nil
}
