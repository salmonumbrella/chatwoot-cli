package cmd

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	redis "github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
)

func newWatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch server utilities (webhook receiver + SSE)",
	}
	cmd.AddCommand(newWatchServeCmd())
	cmd.AddCommand(newWatchSetupCmd())
	cmd.AddCommand(newWatchStatusCmd())
	return cmd
}

type watchServeConfig struct {
	Bind       string
	Port       int
	HookPath   string
	WatchPath  string
	HookToken  string
	BackendURL string
	AccountID  int

	// Limits
	MaxHookBodyBytes int64

	// Persistence (optional)
	RedisURL     string
	RedisPrefix  string
	BufferSize   int
	BufferTTL    time.Duration
	RedisTimeout time.Duration
}

func newWatchServeCmd() *cobra.Command {
	cfg := watchServeConfig{
		Bind:             "127.0.0.1",
		Port:             8789,
		HookPath:         "/hooks/chatwoot",
		WatchPath:        "/watch",
		MaxHookBodyBytes: 1024 * 1024,
		RedisPrefix:      "chatwoot-watch",
		BufferSize:       500,
		BufferTTL:        2 * time.Hour,
		RedisTimeout:     5 * time.Second,
	}

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run a webhook receiver + SSE server for agent-friendly push",
		Long: strings.TrimSpace(`
Runs an HTTP server that:
- Receives Chatwoot webhooks (POST) at --hook-path (default: /hooks/chatwoot)
- Streams events to agents over SSE at --watch-path (default: /watch)

This enables near-instant "push" UX for agents without polling and without WebSockets.
`),
		RunE: RunE(func(cmd *cobra.Command, _ []string) error {
			if !strings.HasPrefix(cfg.HookPath, "/") {
				return fmt.Errorf("--hook-path must start with '/'")
			}
			if !strings.HasPrefix(cfg.WatchPath, "/") {
				return fmt.Errorf("--watch-path must start with '/'")
			}
			if strings.TrimSpace(cfg.HookToken) == "" && !isLoopbackHost(cfg.Bind) {
				return fmt.Errorf("--hook-token is required when binding non-loopback")
			}
			if strings.TrimSpace(cfg.BackendURL) == "" {
				// For server-side use we can still authorize by calling profile/conversation endpoints.
				// Default to CHATWOOT_BASE_URL if present.
				cfg.BackendURL = strings.TrimSpace(os.Getenv("CHATWOOT_BASE_URL"))
			}
			if strings.TrimSpace(cfg.BackendURL) == "" {
				return fmt.Errorf("--backend-url is required (or set CHATWOOT_BASE_URL)")
			}
			if cfg.AccountID <= 0 {
				if v := strings.TrimSpace(os.Getenv("CHATWOOT_ACCOUNT_ID")); v != "" {
					if id, err := strconv.Atoi(v); err == nil {
						cfg.AccountID = id
					}
				}
			}
			if cfg.AccountID <= 0 {
				return fmt.Errorf("--account-id is required (or set CHATWOOT_ACCOUNT_ID)")
			}

			ctx, stop := signal.NotifyContext(cmdContext(cmd), os.Interrupt, syscall.SIGTERM)
			defer stop()

			s := newWatchServer(cfg)
			s.Start(ctx)
			mux := http.NewServeMux()
			mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok\n"))
			})
			mux.HandleFunc("/", s.serveHTTP)

			addr := net.JoinHostPort(cfg.Bind, strconv.Itoa(cfg.Port))
			srv := &http.Server{
				Addr:              addr,
				Handler:           mux,
				ReadHeaderTimeout: 10 * time.Second,
			}

			errCh := make(chan error, 1)
			go func() {
				errCh <- srv.ListenAndServe()
			}()

			select {
			case <-ctx.Done():
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				_ = srv.Shutdown(shutdownCtx)
				return nil
			case err := <-errCh:
				if errors.Is(err, http.ErrServerClosed) {
					return nil
				}
				return err
			}
		}),
	}

	cmd.Flags().StringVar(&cfg.Bind, "bind", cfg.Bind, "Bind address")
	cmd.Flags().IntVar(&cfg.Port, "port", cfg.Port, "Listen port")
	cmd.Flags().StringVar(&cfg.HookPath, "hook-path", cfg.HookPath, "Webhook receiver path")
	cmd.Flags().StringVar(&cfg.WatchPath, "watch-path", cfg.WatchPath, "SSE base path")
	cmd.Flags().StringVar(&cfg.HookToken, "hook-token", strings.TrimSpace(os.Getenv("CHATWOOT_WATCH_HOOK_TOKEN")), "Shared token for webhook receiver auth (query: ?token=..., header: X-Chatwoot-Token)")
	cmd.Flags().StringVar(&cfg.BackendURL, "backend-url", "", "Chatwoot backend URL for authorization checks (defaults to CHATWOOT_BASE_URL)")
	cmd.Flags().IntVar(&cfg.AccountID, "account-id", 0, "Chatwoot account ID (defaults to CHATWOOT_ACCOUNT_ID)")
	cmd.Flags().StringVar(&cfg.RedisURL, "redis-url", firstNonEmptyEnv("CHATWOOT_WATCH_REDIS_URL", "REDIS_URL"), "Redis URL for durable replay/dedupe (optional)")
	cmd.Flags().StringVar(&cfg.RedisPrefix, "redis-prefix", cfg.RedisPrefix, "Redis key prefix (only used when --redis-url is set)")
	cmd.Flags().IntVar(&cfg.BufferSize, "buffer-size", cfg.BufferSize, "Max buffered events per conversation (replay window)")
	cmd.Flags().DurationVar(&cfg.BufferTTL, "buffer-ttl", cfg.BufferTTL, "How long to retain buffered events (Redis)")

	return cmd
}

type watchEvent struct {
	Type           string         `json:"type"`
	ConversationID int            `json:"conversation_id"`
	MessageID      int            `json:"message_id"`
	MessageType    string         `json:"message_type,omitempty"` // "incoming"/"outgoing"/"activity"/...
	Private        bool           `json:"private,omitempty"`
	Content        string         `json:"content,omitempty"`
	ContentType    string         `json:"content_type,omitempty"`
	CreatedAt      string         `json:"created_at,omitempty"`
	Sender         map[string]any `json:"sender,omitempty"`
	Contact        map[string]any `json:"contact,omitempty"`
	RawEvent       string         `json:"raw_event,omitempty"`
}

type watchServer struct {
	cfg watchServeConfig

	broker *watchBroker

	dedupe *idDedupe
	store  watchStore

	authMu    sync.Mutex
	authCache map[string]time.Time // token|conversationID -> expiresAt
}

func newWatchServer(cfg watchServeConfig) *watchServer {
	var store watchStore
	if strings.TrimSpace(cfg.RedisURL) != "" {
		rdb := redis.NewClient(&redis.Options{Addr: redisAddrFromURL(cfg.RedisURL)})
		// Prefer parsing full URL when possible.
		if opt, err := redis.ParseURL(cfg.RedisURL); err == nil {
			opt.ReadTimeout = cfg.RedisTimeout
			opt.WriteTimeout = cfg.RedisTimeout
			opt.DialTimeout = cfg.RedisTimeout
			rdb = redis.NewClient(opt)
		}
		store = &redisWatchStore{
			rdb:        rdb,
			prefix:     cfg.RedisPrefix,
			accountID:  cfg.AccountID,
			bufferSize: cfg.BufferSize,
			ttl:        cfg.BufferTTL,
			instanceID: newWatchInstanceID(),
		}
	}

	return &watchServer{
		cfg:       cfg,
		broker:    newWatchBroker(),
		dedupe:    newIDDedupe(10*time.Minute, 50000),
		store:     store,
		authCache: make(map[string]time.Time),
	}
}

func (s *watchServer) Start(ctx context.Context) {
	if s.store == nil {
		return
	}
	if starter, ok := s.store.(interface {
		Start(ctx context.Context, onEvent func(watchEvent))
	}); ok {
		starter.Start(ctx, func(ev watchEvent) {
			// Extra safety: pubsub redeliveries shouldn't fan out duplicates.
			if ev.MessageID > 0 && s.dedupe.Seen(ev.MessageID) {
				return
			}
			s.broker.Publish(ev.ConversationID, ev)
		})
	}
}

func (s *watchServer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if pathMatches(s.cfg.HookPath, r.URL.Path) {
		s.handleHook(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, s.cfg.WatchPath+"/") || r.URL.Path == s.cfg.WatchPath {
		s.handleWatch(w, r)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (s *watchServer) handleHook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(s.cfg.HookToken) != "" && !s.hookAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, s.cfg.MaxHookBodyBytes))
	_ = r.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var p chatwootWebhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Only care about message_created for now.
	if strings.TrimSpace(p.Event) != "message_created" {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	convID := p.Conversation.ID
	if convID == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	msgID := p.ID.Int()
	if msgID == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	ev := watchEvent{
		Type:           "message_created",
		ConversationID: convID,
		MessageID:      msgID,
		MessageType:    p.MessageType.String(),
		Private:        p.Private,
		Content:        p.Content,
		ContentType:    p.ContentType,
		CreatedAt:      p.CreatedAt.String(),
		Sender:         p.Sender,
		Contact:        p.Contact,
		RawEvent:       p.Event,
	}

	if s.store != nil {
		dup, err := s.store.Put(r.Context(), convID, ev)
		if err == nil && dup {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		// On store errors, we still publish live and rely on client-side dedupe + polling fallback.
	} else {
		if s.dedupe.Seen(msgID) {
			w.WriteHeader(http.StatusAccepted)
			return
		}
	}

	s.broker.Publish(convID, ev)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (s *watchServer) hookAuthorized(r *http.Request) bool {
	token := strings.TrimSpace(r.Header.Get("X-Chatwoot-Token"))
	if token == "" {
		token = strings.TrimSpace(r.Header.Get("X-Webhook-Token"))
	}
	if token == "" {
		token = strings.TrimSpace(r.URL.Query().Get("token"))
	}
	if token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.HookToken)) == 1
}

func (s *watchServer) handleWatch(w http.ResponseWriter, r *http.Request) {
	// SSE endpoints only.
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// /watch/conversations/{id}
	prefix := strings.TrimSuffix(s.cfg.WatchPath, "/") + "/conversations/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, prefix)
	idStr, _, _ := strings.Cut(rest, "/")
	convID, err := strconv.Atoi(idStr)
	if err != nil || convID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	token := s.extractAPIToken(r)
	if token == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	ok, authErr := s.authorizeTokenForConversation(r.Context(), token, convID)
	if authErr != nil || !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sinceID := parseSinceID(r)
	sub := s.broker.Subscribe(convID)
	defer s.broker.Unsubscribe(convID, sub)

	// Initial comment to open the stream.
	_, _ = io.WriteString(w, ":ok\n\n")
	flusher.Flush()

	// Replay any buffered events missed since the last seen message ID.
	if s.store != nil && sinceID > 0 {
		replay, err := s.store.Replay(r.Context(), convID, sinceID, s.cfg.BufferSize)
		if err == nil {
			for _, ev := range replay {
				b, err := json.Marshal(ev)
				if err != nil {
					continue
				}
				writeSSE(w, ev.Type, ev.MessageID, b)
			}
		}
	}
	flusher.Flush()

	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-keepalive.C:
			_, _ = io.WriteString(w, ":keepalive\n\n")
			flusher.Flush()
		case ev, ok := <-sub:
			if !ok {
				return
			}
			b, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			writeSSE(w, ev.Type, ev.MessageID, b)
			flusher.Flush()
		}
	}
}

func (s *watchServer) extractAPIToken(r *http.Request) string {
	// Chatwoot API token header used by this CLI.
	if tok := strings.TrimSpace(r.Header.Get("api_access_token")); tok != "" {
		return tok
	}
	// Common alternative.
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

func (s *watchServer) authorizeTokenForConversation(ctx context.Context, token string, convID int) (bool, error) {
	// Small cache to avoid hammering the backend on reconnect storms.
	cacheKey := token + "|" + strconv.Itoa(convID)
	s.authMu.Lock()
	if exp, ok := s.authCache[cacheKey]; ok && time.Now().Before(exp) {
		s.authMu.Unlock()
		return true, nil
	}
	s.authMu.Unlock()

	client := api.New(s.cfg.BackendURL, token, s.cfg.AccountID)
	_, err := client.Conversations().Get(ctx, convID)
	if err != nil {
		return false, err
	}

	s.authMu.Lock()
	s.authCache[cacheKey] = time.Now().Add(5 * time.Minute)
	s.authMu.Unlock()
	return true, nil
}

func writeSSE(w io.Writer, event string, id int, data []byte) {
	// Minimal SSE framing; handle embedded newlines by splitting into multiple data lines.
	if id > 0 {
		_, _ = fmt.Fprintf(w, "id: %d\n", id)
	}
	_, _ = fmt.Fprintf(w, "event: %s\n", event)
	for _, line := range strings.Split(string(data), "\n") {
		_, _ = fmt.Fprintf(w, "data: %s\n", line)
	}
	_, _ = io.WriteString(w, "\n")
}

func parseSinceID(r *http.Request) int {
	// Standard SSE reconnection mechanism.
	if v := strings.TrimSpace(r.Header.Get("Last-Event-ID")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("since")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 0
}

func pathMatches(prefix, path string) bool {
	prefix = strings.TrimSuffix(prefix, "/")
	if prefix == "" {
		return path == "" || path == "/"
	}
	if path == prefix || path == prefix+"/" {
		return true
	}
	return strings.HasPrefix(path, prefix+"/")
}

func isLoopbackHost(bind string) bool {
	host := strings.TrimSpace(bind)
	if host == "" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return strings.EqualFold(host, "localhost")
}

type watchBroker struct {
	mu   sync.RWMutex
	subs map[int]map[chan watchEvent]struct{}
}

func newWatchBroker() *watchBroker {
	return &watchBroker{
		subs: make(map[int]map[chan watchEvent]struct{}),
	}
}

func (b *watchBroker) Subscribe(conversationID int) chan watchEvent {
	ch := make(chan watchEvent, 64)
	b.mu.Lock()
	m := b.subs[conversationID]
	if m == nil {
		m = make(map[chan watchEvent]struct{})
		b.subs[conversationID] = m
	}
	m[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *watchBroker) Unsubscribe(conversationID int, ch chan watchEvent) {
	b.mu.Lock()
	if m := b.subs[conversationID]; m != nil {
		delete(m, ch)
		if len(m) == 0 {
			delete(b.subs, conversationID)
		}
	}
	b.mu.Unlock()
	close(ch)
}

func (b *watchBroker) Publish(conversationID int, ev watchEvent) {
	b.mu.Lock()
	m := b.subs[conversationID]
	for ch := range m {
		select {
		case ch <- ev:
		default:
			// Drop if subscriber is slow; SSE client can reconnect.
		}
	}
	b.mu.Unlock()
}

type idDedupe struct {
	mu      sync.Mutex
	seen    map[int]time.Time
	ttl     time.Duration
	maxSize int
}

func newIDDedupe(ttl time.Duration, maxSize int) *idDedupe {
	return &idDedupe{
		seen:    make(map[int]time.Time),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

func (d *idDedupe) Seen(id int) bool {
	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()

	// Opportunistic GC.
	if len(d.seen) > d.maxSize {
		for k, t := range d.seen {
			if now.Sub(t) > d.ttl {
				delete(d.seen, k)
			}
		}
		// If still too big, drop random-ish entries.
		for k := range d.seen {
			if len(d.seen) <= d.maxSize {
				break
			}
			delete(d.seen, k)
		}
	}

	if t, ok := d.seen[id]; ok {
		if now.Sub(t) <= d.ttl {
			return true
		}
	}
	d.seen[id] = now
	return false
}

// chatwootWebhookPayload is a minimal schema for message_created webhooks.
// We keep it flexible because Chatwoot webhook payloads can evolve.
type chatwootWebhookPayload struct {
	Event        string         `json:"event"`
	ID           flexInt        `json:"id"`
	Content      string         `json:"content"`
	ContentType  string         `json:"content_type"`
	MessageType  flexString     `json:"message_type"`
	Private      bool           `json:"private"`
	CreatedAt    flexString     `json:"created_at"`
	Sender       map[string]any `json:"sender"`
	Contact      map[string]any `json:"contact"`
	Conversation struct {
		ID int `json:"id"`
	} `json:"conversation"`
}

type flexInt struct{ raw any }

func (f *flexInt) UnmarshalJSON(data []byte) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	f.raw = v
	return nil
}

func (f flexInt) Int() int {
	switch v := f.raw.(type) {
	case float64:
		return int(v)
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(v))
		return i
	default:
		return 0
	}
}

type flexString struct{ raw any }

func (f *flexString) UnmarshalJSON(data []byte) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	f.raw = v
	return nil
}

func (f flexString) String() string {
	switch v := f.raw.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		// Chatwoot API uses ints for message type; map common ones.
		switch int(v) {
		case api.MessageTypeIncoming:
			return "incoming"
		case api.MessageTypeOutgoing:
			return "outgoing"
		case api.MessageTypeActivity:
			return "activity"
		case api.MessageTypeTemplate:
			return "template"
		default:
			return ""
		}
	default:
		return ""
	}
}

func firstNonEmptyEnv(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

type watchStore interface {
	// Put stores the event; returns dup=true if it was already stored.
	Put(ctx context.Context, conversationID int, ev watchEvent) (dup bool, err error)
	Replay(ctx context.Context, conversationID int, sinceMessageID int, limit int) ([]watchEvent, error)
}

type redisWatchStore struct {
	rdb        *redis.Client
	prefix     string
	accountID  int
	bufferSize int
	ttl        time.Duration
	instanceID string
	pubsub     *redis.PubSub
	startOnce  sync.Once
}

func (s *redisWatchStore) streamKey(conversationID int) string {
	return fmt.Sprintf("%s:acct:%d:conv:%d:stream", s.prefix, s.accountID, conversationID)
}

func (s *redisWatchStore) pubsubChannel() string {
	return fmt.Sprintf("%s:acct:%d:pubsub", s.prefix, s.accountID)
}

type watchPubSubMessage struct {
	InstanceID string     `json:"instance_id"`
	Event      watchEvent `json:"event"`
}

func (s *redisWatchStore) Put(ctx context.Context, conversationID int, ev watchEvent) (bool, error) {
	if ev.MessageID <= 0 {
		return false, nil
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return false, err
	}
	key := s.streamKey(conversationID)
	entryID := fmt.Sprintf("%d-0", ev.MessageID)

	// Use messageID-based stream entry IDs for:
	// - dedupe across webhook retries and across multiple receiver instances
	// - efficient replay using Last-Event-ID
	args := &redis.XAddArgs{
		Stream: key,
		ID:     entryID,
		Approx: true,
		Values: map[string]any{
			"ev": string(b),
		},
	}
	if s.bufferSize > 0 {
		args.MaxLen = int64(s.bufferSize)
	}
	_, err = s.rdb.XAdd(ctx, args).Result()
	if err != nil {
		// Duplicate IDs (webhook retry / multi-instance) show up as XADD error.
		// Treat them as duplicates.
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "already exists") || strings.Contains(msg, "equal or smaller") {
			return true, nil
		}
		return false, err
	}

	if s.ttl > 0 {
		_ = s.rdb.Expire(ctx, key, s.ttl).Err()
	}

	// Fan out to other receiver instances (best effort).
	_ = s.publish(ctx, ev)
	return false, nil
}

func (s *redisWatchStore) Replay(ctx context.Context, conversationID int, sinceMessageID int, limit int) ([]watchEvent, error) {
	if sinceMessageID <= 0 || limit <= 0 {
		return nil, nil
	}
	key := s.streamKey(conversationID)
	start := fmt.Sprintf("(%d-0", sinceMessageID)

	vals, err := s.rdb.XRangeN(ctx, key, start, "+", int64(limit)).Result()
	if err != nil {
		return nil, err
	}
	out := make([]watchEvent, 0, len(vals))
	for _, v := range vals {
		var ev watchEvent
		raw, ok := v.Values["ev"]
		if !ok {
			continue
		}
		rawStr, ok := raw.(string)
		if !ok {
			// go-redis sometimes returns []byte depending on decoder; handle both.
			if bb, ok := raw.([]byte); ok {
				rawStr = string(bb)
			} else {
				continue
			}
		}
		if err := json.Unmarshal([]byte(rawStr), &ev); err != nil {
			continue
		}
		out = append(out, ev)
	}
	return out, nil
}

func (s *redisWatchStore) Start(ctx context.Context, onEvent func(watchEvent)) {
	if s.rdb == nil {
		return
	}
	s.startOnce.Do(func() {
		s.pubsub = s.rdb.Subscribe(ctx, s.pubsubChannel())
		ch := s.pubsub.Channel()
		go func() {
			defer func() {
				_ = s.pubsub.Close()
			}()
			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-ch:
					if !ok || msg == nil {
						return
					}
					var m watchPubSubMessage
					if err := json.Unmarshal([]byte(msg.Payload), &m); err != nil {
						continue
					}
					if m.InstanceID != "" && subtle.ConstantTimeCompare([]byte(m.InstanceID), []byte(s.instanceID)) == 1 {
						continue
					}
					if m.Event.ConversationID <= 0 || m.Event.MessageID <= 0 {
						continue
					}
					onEvent(m.Event)
				}
			}
		}()
	})
}

func (s *redisWatchStore) publish(ctx context.Context, ev watchEvent) error {
	if s.rdb == nil {
		return nil
	}
	b, err := json.Marshal(watchPubSubMessage{
		InstanceID: s.instanceID,
		Event:      ev,
	})
	if err != nil {
		return err
	}
	return s.rdb.Publish(ctx, s.pubsubChannel(), string(b)).Err()
}

func redisAddrFromURL(u string) string {
	// Best-effort fallback when ParseURL fails; ParseURL handles redis:// URIs.
	u = strings.TrimSpace(u)
	u = strings.TrimPrefix(u, "redis://")
	u = strings.TrimPrefix(u, "rediss://")
	if at := strings.LastIndexByte(u, '@'); at >= 0 {
		u = u[at+1:]
	}
	if slash := strings.IndexByte(u, '/'); slash >= 0 {
		u = u[:slash]
	}
	return u
}

func newWatchInstanceID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
