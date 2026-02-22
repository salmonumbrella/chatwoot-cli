package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func newFollowTestCmd(mode outfmt.Mode) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.SetContext(outfmt.WithMode(context.Background(), mode))
	return cmd, out, errOut
}

type stubSnapshotClient struct {
	conv      *api.Conversation
	convErr   error
	contact   *api.Contact
	msgs      []api.Message
	labels    []string
	labelsErr error
}

func (s stubSnapshotClient) GetConversation(_ context.Context, _ int) (*api.Conversation, error) {
	if s.convErr != nil {
		return nil, s.convErr
	}
	return s.conv, nil
}

func (s stubSnapshotClient) GetContact(_ context.Context, _ int) (*api.Contact, error) {
	if s.contact == nil {
		return nil, fmt.Errorf("missing contact")
	}
	return s.contact, nil
}

func (s stubSnapshotClient) ListMessages(_ context.Context, _ int, _ int, _ int) ([]api.Message, error) {
	return s.msgs, nil
}

func (s stubSnapshotClient) ListLabels(_ context.Context, _ int) ([]string, error) {
	if s.labelsErr != nil {
		return nil, s.labelsErr
	}
	return s.labels, nil
}

func TestBuildCableURL(t *testing.T) {
	if got := buildCableURL("https://app.example.com/path?q=1"); got != "wss://app.example.com/cable" {
		t.Fatalf("https cable url = %q", got)
	}
	if got := buildCableURL("http://localhost:3000"); got != "ws://localhost:3000/cable" {
		t.Fatalf("http cable url = %q", got)
	}
	if got := buildCableURL("://bad-url"); got != "://bad-url" {
		t.Fatalf("invalid URL fallback = %q", got)
	}
}

func TestPrintFollowMessageModes(t *testing.T) {
	msg := api.Message{
		ID:             7,
		ConversationID: 44,
		Content:        "hello world",
		MessageType:    api.MessageTypeIncoming,
		CreatedAt:      1700000000,
		Sender:         &api.MessageSender{Name: "Customer"},
	}
	raw := json.RawMessage(`{"event":"message.created"}`)

	t.Run("text", func(t *testing.T) {
		cmd, out, _ := newFollowTestCmd(outfmt.Text)
		if err := printFollowMessage(cmd, msg, "ws"); err != nil {
			t.Fatalf("printFollowMessage text error: %v", err)
		}
		if !strings.Contains(out.String(), "Customer (incoming)") {
			t.Fatalf("unexpected text output: %q", out.String())
		}
	})

	t.Run("json includes raw", func(t *testing.T) {
		cmd, out, _ := newFollowTestCmd(outfmt.JSON)
		if err := printFollowMessageWithRaw(cmd, nil, "message.created", msg, "ws", raw, true); err != nil {
			t.Fatalf("printFollowMessageWithRaw json error: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode output: %v", err)
		}
		if payload["event"] != "message.created" {
			t.Fatalf("unexpected event: %#v", payload)
		}
		if payload["raw"] == nil {
			t.Fatalf("expected raw payload: %#v", payload)
		}
	})

	t.Run("agent mode", func(t *testing.T) {
		cmd, out, _ := newFollowTestCmd(outfmt.Agent)
		if err := printFollowMessageWithRaw(cmd, nil, "message.created", msg, "ws", nil, false); err != nil {
			t.Fatalf("printFollowMessageWithRaw agent error: %v", err)
		}
		if !strings.Contains(out.String(), `"kind": "conversations.follow"`) {
			t.Fatalf("unexpected agent output: %s", out.String())
		}
		if !strings.Contains(out.String(), `"item"`) {
			t.Fatalf("expected item summary in agent output: %s", out.String())
		}
	})
}

func TestPrintFollowMessageBatchAndEventText(t *testing.T) {
	cmd, out, _ := newFollowTestCmd(outfmt.Text)
	messages := []followMsg{
		{msg: api.Message{ConversationID: 1, Content: "first", MessageType: api.MessageTypeIncoming, CreatedAt: 1700000001, Sender: &api.MessageSender{Name: "Customer"}}},
		{msg: api.Message{ConversationID: 1, Content: "second", MessageType: api.MessageTypeIncoming, CreatedAt: 1700000002, Sender: &api.MessageSender{Name: "Customer"}}},
	}
	if err := printFollowMessageBatch(cmd, nil, messages, "ws", false); err != nil {
		t.Fatalf("printFollowMessageBatch error: %v", err)
	}
	if !strings.Contains(out.String(), "first") || !strings.Contains(out.String(), "second") {
		t.Fatalf("batch output missing content: %q", out.String())
	}

	out.Reset()
	evt := chatwootWSEvent{Event: "conversation.typing_on", Data: json.RawMessage(`{"conversation":{"id":1},"user":{"name":"Alice"}}`)}
	if err := printFollowEvent(cmd, nil, evt, "ws", nil, false); err != nil {
		t.Fatalf("printFollowEvent text error: %v", err)
	}
	if !strings.Contains(out.String(), "Alice is typing in #1") {
		t.Fatalf("unexpected event output: %q", out.String())
	}
}

func TestDedupeAndEventSummaryHelpers(t *testing.T) {
	if got := dedupeStrings([]string{" a", "a", "", "b", "b"}); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("dedupeStrings unexpected result: %#v", got)
	}

	id, contact, inbox := conversationCreatedSummary(json.RawMessage(`{"id":10,"inbox_id":2,"contact":{"name":"Jane"}}`))
	if id != 10 || contact != "Jane" || inbox != 2 {
		t.Fatalf("conversationCreatedSummary mismatch: %d %q %d", id, contact, inbox)
	}

	id, assignee, assigned := assigneeChangedSummary(json.RawMessage(`{"id":11,"assignee":{"id":7,"name":"Agent"}}`))
	if id != 11 || assignee != "Agent" || !assigned {
		t.Fatalf("assigneeChangedSummary assigned mismatch: %d %q %v", id, assignee, assigned)
	}
	id, _, assigned = assigneeChangedSummary(json.RawMessage(`{"id":11,"assignee":null}`))
	if id != 11 || assigned {
		t.Fatalf("assigneeChangedSummary unassigned mismatch: %d %v", id, assigned)
	}

	convID, user := typingSummary(json.RawMessage(`{"conversation":{"id":55},"user":{"name":"Typer"}}`))
	if convID != 55 || user != "Typer" {
		t.Fatalf("typingSummary mismatch: %d %q", convID, user)
	}

	if anyToInt(" 9 ") != 9 || anyToInt(json.Number("12")) != 12 || anyToInt(3.0) != 3 || anyToInt("x") != 0 {
		t.Fatalf("anyToInt conversions failed")
	}

	if got := conversationIDFromEvent("ignored", json.RawMessage(`{"conversation":{"id":42}}`)); got != 42 {
		t.Fatalf("conversationIDFromEvent = %d, want 42", got)
	}
}

func TestFollowCursorPersistenceAndWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cursor.json")

	// Missing file should not error.
	cur, err := loadFollowCursor(path)
	if err != nil {
		t.Fatalf("loadFollowCursor missing file error: %v", err)
	}
	if cur.LastSeenMessageID != 0 {
		t.Fatalf("expected empty cursor on missing file, got %#v", cur)
	}

	if err := saveFollowCursor(path, followCursor{BaseURL: "https://chatwoot.example.com", AccountID: 1, LastSeenMessageID: 10}); err != nil {
		t.Fatalf("saveFollowCursor error: %v", err)
	}

	loaded, err := loadFollowCursor(path)
	if err != nil {
		t.Fatalf("loadFollowCursor after save error: %v", err)
	}
	if loaded.Version != 1 || loaded.LastSeenMessageID != 10 || loaded.UpdatedAt == "" {
		t.Fatalf("unexpected loaded cursor: %#v", loaded)
	}

	w, err := newFollowCursorWriter(path, "https://chatwoot.example.com", 1, 10, time.Hour)
	if err != nil {
		t.Fatalf("newFollowCursorWriter error: %v", err)
	}
	w.Update(9)
	if w.LastSeenID != 10 {
		t.Fatalf("Update should ignore lower IDs, got %d", w.LastSeenID)
	}
	w.Update(12)
	if w.LastSeenID != 12 {
		t.Fatalf("Update did not set LastSeenID, got %d", w.LastSeenID)
	}
	if w.LastFlushed != 12 {
		t.Fatalf("expected immediate first flush, got LastFlushed=%d", w.LastFlushed)
	}

	w.LastFlushAt = time.Now()
	w.Update(13)
	if w.LastFlushed != 12 {
		t.Fatalf("expected throttled flush, got LastFlushed=%d", w.LastFlushed)
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush error: %v", err)
	}
	if w.LastFlushed != 13 {
		t.Fatalf("expected flushed ID 13, got %d", w.LastFlushed)
	}

	badPath := filepath.Join(t.TempDir(), "bad.json")
	if err := osWriteFile(badPath, []byte("not-json")); err != nil {
		t.Fatalf("write bad cursor: %v", err)
	}
	if _, err := loadFollowCursor(badPath); err == nil {
		t.Fatal("expected parse error for invalid cursor JSON")
	}
}

func osWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

func TestConvMetaFilteringAndEvents(t *testing.T) {
	m := &convMeta{}
	if m.HasLabel("vip") {
		t.Fatal("nil labels should not match")
	}
	m.SetLabels([]string{"vip", "", "vip", "sales"})
	if !m.HasLabel("vip") || !m.HasLabel("sales") {
		t.Fatalf("expected labels set, got %#v", m.Labels)
	}

	priority := "high"
	assigneeID := 7
	conv := api.Conversation{ID: 99, InboxID: 3, Status: "open", Priority: &priority, AssigneeID: &assigneeID, ContactID: 55, Labels: []string{"vip"}}
	m.ApplyConversation(conv)
	if !m.Hydrated || m.ID != 99 || m.Priority != "high" {
		t.Fatalf("ApplyConversation mismatch: %#v", m)
	}

	filters := followFilters{InboxID: 3, Status: "open", AssigneeID: 7, Labels: []string{"vip"}, Priority: "high", ContactID: 55}
	if !filters.matchMeta(m) {
		t.Fatalf("expected filters to match meta: %#v", m)
	}

	m.ApplyEvent("conversation.status_changed", json.RawMessage(`{"id":99,"status":"resolved"}`))
	if m.Status != "resolved" {
		t.Fatalf("status not updated: %#v", m)
	}
	m.ApplyEvent("assignee.changed", json.RawMessage(`{"id":99,"assignee":null}`))
	if m.AssigneeID != nil {
		t.Fatalf("expected assignee cleared: %#v", m)
	}
	m.ApplyEvent("label.added", json.RawMessage(`{"label":"urgent"}`))
	if !m.HasLabel("urgent") {
		t.Fatalf("label.added not applied: %#v", m.Labels)
	}
	m.ApplyEvent("label.removed", json.RawMessage(`{"label":"urgent"}`))
	if m.HasLabel("urgent") {
		t.Fatalf("label.removed not applied: %#v", m.Labels)
	}
	m.ApplyEvent("conversation.updated", json.RawMessage(`{"id":99,"inbox_id":9,"status":"pending","priority":"low","contact_id":77,"labels":["new"]}`))
	if m.InboxID != 9 || m.Status != "pending" || m.Priority != "low" || m.ContactID != 77 || !m.HasLabel("new") {
		t.Fatalf("conversation.updated fallback not applied: %#v", m)
	}
}

func TestFetchConversationMeta(t *testing.T) {
	conv := &api.Conversation{ID: 42, InboxID: 2, Status: "open"}
	meta, err := fetchConversationMeta(context.Background(), stubSnapshotClient{conv: conv, labels: []string{"vip"}}, 42, []string{"vip"})
	if err != nil {
		t.Fatalf("fetchConversationMeta error: %v", err)
	}
	if meta == nil || meta.ID != 42 || !meta.HasLabel("vip") {
		t.Fatalf("unexpected meta: %#v", meta)
	}

	meta, err = fetchConversationMeta(context.Background(), nil, 42, nil)
	if err != nil || meta != nil {
		t.Fatalf("expected nil meta with nil client, got meta=%#v err=%v", meta, err)
	}

	_, err = fetchConversationMeta(context.Background(), stubSnapshotClient{convErr: fmt.Errorf("boom")}, 42, nil)
	if err == nil {
		t.Fatal("expected fetchConversationMeta error")
	}
}

func TestFollowAPISnapshotClientMethods(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/accounts/1/conversations/123":
			_, _ = w.Write([]byte(`{"id":123,"inbox_id":1,"status":"open","created_at":1700000000}`))
		case r.URL.Path == "/api/v1/accounts/1/contacts/55":
			_, _ = w.Write([]byte(`{"payload":{"id":55,"name":"Contact","created_at":1700000000}}`))
		case r.URL.Path == "/api/v1/accounts/1/conversations/123/messages" && r.URL.Query().Get("before") == "":
			_, _ = w.Write([]byte(`{"payload":[{"id":2,"conversation_id":123,"content":"hi","message_type":0,"private":false,"created_at":1700000000},{"id":1,"conversation_id":123,"content":"yo","message_type":0,"private":false,"created_at":1700000001}]}`))
		case r.URL.Path == "/api/v1/accounts/1/conversations/123/messages" && r.URL.Query().Get("before") == "1":
			_, _ = w.Write([]byte(`{"payload":[]}`))
		case r.URL.Path == "/api/v1/accounts/1/conversations/123/labels":
			_, _ = w.Write([]byte(`{"payload":["vip"]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CHATWOOT_TESTING", "1")
	client := followAPISnapshotClient{client: api.New(server.URL, "token", 1)}

	conv, err := client.GetConversation(context.Background(), 123)
	if err != nil || conv.ID != 123 {
		t.Fatalf("GetConversation mismatch: conv=%#v err=%v", conv, err)
	}
	contact, err := client.GetContact(context.Background(), 55)
	if err != nil || contact.ID != 55 {
		t.Fatalf("GetContact mismatch: contact=%#v err=%v", contact, err)
	}
	msgs, err := client.ListMessages(context.Background(), 123, 2, 10)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("ListMessages mismatch: msgs=%#v err=%v", msgs, err)
	}
	labels, err := client.ListLabels(context.Background(), 123)
	if err != nil || len(labels) != 1 || labels[0] != "vip" {
		t.Fatalf("ListLabels mismatch: labels=%#v err=%v", labels, err)
	}
}

func TestFollowExecHookAndEmitStreamRecord(t *testing.T) {
	cmd, out, errOut := newFollowTestCmd(outfmt.JSON)

	if hook := newFollowExecHook(cmd, "", 0, false); hook != nil {
		t.Fatal("expected nil hook for empty command")
	}

	hook := newFollowExecHook(cmd, "cat >/dev/null", 0, false)
	if hook == nil {
		t.Fatal("expected hook")
	}
	if hook.timeout != 30*time.Second {
		t.Fatalf("default timeout = %s, want 30s", hook.timeout)
	}
	if err := hook.Run(map[string]any{"ok": true}); err != nil {
		t.Fatalf("hook.Run success path error: %v", err)
	}

	nonFatal := newFollowExecHook(cmd, "exit 2", time.Second, false)
	if err := emitStreamRecord(cmd, nonFatal, map[string]any{"hello": "world"}); err != nil {
		t.Fatalf("emitStreamRecord non-fatal should not fail: %v", err)
	}
	if !strings.Contains(errOut.String(), "exec hook error:") {
		t.Fatalf("expected non-fatal exec error log, got %q", errOut.String())
	}
	if !strings.Contains(out.String(), `"hello": "world"`) {
		t.Fatalf("expected stream JSON output, got %q", out.String())
	}

	fatal := newFollowExecHook(cmd, "exit 2", time.Second, true)
	if err := emitStreamRecord(cmd, fatal, map[string]any{"x": 1}); err == nil {
		t.Fatal("expected fatal exec error")
	}
}

func TestEmitConversationSnapshot(t *testing.T) {
	conv := &api.Conversation{ID: 8, InboxID: 2, Status: "open", ContactID: 9, CreatedAt: 1700000000}
	contact := &api.Contact{ID: 9, Name: "Customer", CreatedAt: 1700000000}
	msgs := []api.Message{{ID: 2, ConversationID: 8, Content: "newer", MessageType: api.MessageTypeIncoming, CreatedAt: 1700000002}, {ID: 1, ConversationID: 8, Content: "older", MessageType: api.MessageTypeIncoming, CreatedAt: 1700000001}}

	t.Run("json snapshot", func(t *testing.T) {
		cmd, out, _ := newFollowTestCmd(outfmt.JSON)
		err := emitConversationSnapshot(context.Background(), cmd, nil, stubSnapshotClient{conv: conv, contact: contact, msgs: msgs, labels: []string{"vip"}}, 8, 10)
		if err != nil {
			t.Fatalf("emitConversationSnapshot error: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode snapshot output: %v\n%s", err, out.String())
		}
		if payload["event"] != "conversation.snapshot" {
			t.Fatalf("unexpected event payload: %#v", payload)
		}
	})

	t.Run("agent snapshot error event", func(t *testing.T) {
		cmd, out, _ := newFollowTestCmd(outfmt.Agent)
		err := emitConversationSnapshot(context.Background(), cmd, nil, stubSnapshotClient{convErr: fmt.Errorf("boom")}, 8, 10)
		if err != nil {
			t.Fatalf("emitConversationSnapshot error emission failed: %v", err)
		}
		if !strings.Contains(out.String(), "conversation.snapshot_error") {
			t.Fatalf("expected snapshot_error output, got %s", out.String())
		}
	})

	t.Run("text mode silent", func(t *testing.T) {
		cmd, out, _ := newFollowTestCmd(outfmt.Text)
		err := emitConversationSnapshot(context.Background(), cmd, nil, stubSnapshotClient{conv: conv, contact: contact, msgs: msgs}, 8, 10)
		if err != nil {
			t.Fatalf("text snapshot should not error: %v", err)
		}
		if strings.TrimSpace(out.String()) != "" {
			t.Fatalf("expected silent text snapshot, got %q", out.String())
		}
	})
}

func TestFollowEmitterBehavior(t *testing.T) {
	t.Run("direct mode executes immediately", func(t *testing.T) {
		cmd, _, _ := newFollowTestCmd(outfmt.Text)
		e := newFollowEmitter(cmd, 0, false)
		called := false
		if err := e.Emit(func() error {
			called = true
			return nil
		}); err != nil {
			t.Fatalf("Emit error: %v", err)
		}
		if !called {
			t.Fatal("expected immediate execution in direct mode")
		}
		if err := e.CloseAndDrain(); err != nil {
			t.Fatalf("CloseAndDrain error: %v", err)
		}
	})

	t.Run("queue mode stores write error", func(t *testing.T) {
		cmd, _, _ := newFollowTestCmd(outfmt.Text)
		e := newFollowEmitter(cmd, 1, false)
		if err := e.Emit(func() error { return fmt.Errorf("boom") }); err != nil {
			t.Fatalf("Emit enqueue error: %v", err)
		}
		if err := e.CloseAndDrain(); err == nil || !strings.Contains(err.Error(), "boom") {
			t.Fatalf("expected CloseAndDrain boom error, got %v", err)
		}
	})

	t.Run("drop when full reports drops", func(t *testing.T) {
		cmd, _, errOut := newFollowTestCmd(outfmt.Text)
		e := newFollowEmitter(cmd, 1, true)
		started := make(chan struct{})
		release := make(chan struct{})

		if err := e.Emit(func() error {
			close(started)
			<-release
			return nil
		}); err != nil {
			t.Fatalf("first emit error: %v", err)
		}
		<-started
		if err := e.Emit(func() error {
			<-release
			return nil
		}); err != nil {
			t.Fatalf("second emit error: %v", err)
		}
		if err := e.Emit(nil); err != nil {
			t.Fatalf("third emit error: %v", err)
		}

		e.MaybeReportDrops()
		if !strings.Contains(errOut.String(), "dropped 1 events") {
			t.Fatalf("expected drop report, got %q", errOut.String())
		}

		close(release)
		if err := e.CloseAndDrain(); err != nil {
			t.Fatalf("CloseAndDrain error: %v", err)
		}
	})
}
