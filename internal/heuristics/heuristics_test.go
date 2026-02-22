package heuristics

import (
	"testing"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

func TestAnalyzeConversation_HighUrgencyLongWait(t *testing.T) {
	// Conversation with last activity >24h ago and unread messages
	now := time.Now()
	conv := &api.Conversation{
		ID:             1,
		Status:         "open",
		Unread:         1,
		LastActivityAt: now.Add(-25 * time.Hour).Unix(),
	}
	messages := []api.Message{
		{ID: 1, Content: "Hello, I need help", MessageType: api.MessageTypeIncoming},
	}

	analysis := AnalyzeConversation(conv, messages, nil)

	if analysis.Urgency != "high" {
		t.Errorf("expected urgency 'high', got '%s'", analysis.Urgency)
	}
	if !containsReason(analysis.Reasons, "waiting") {
		t.Errorf("expected reason about wait time, got %v", analysis.Reasons)
	}
}

func TestAnalyzeConversation_MediumUrgencyMultipleUnread(t *testing.T) {
	now := time.Now()
	conv := &api.Conversation{
		ID:             1,
		Status:         "open",
		Unread:         3,
		LastActivityAt: now.Add(-1 * time.Hour).Unix(),
	}
	messages := []api.Message{
		{ID: 1, Content: "Hello", MessageType: api.MessageTypeIncoming},
		{ID: 2, Content: "Are you there?", MessageType: api.MessageTypeIncoming},
		{ID: 3, Content: "Please respond", MessageType: api.MessageTypeIncoming},
	}

	analysis := AnalyzeConversation(conv, messages, nil)

	if analysis.Urgency != "medium" {
		t.Errorf("expected urgency at least 'medium' for 3 unread messages, got '%s'", analysis.Urgency)
	}
}

func TestAnalyzeConversation_SatisfactionKeywordsChinese(t *testing.T) {
	now := time.Now()
	conv := &api.Conversation{
		ID:             1,
		Status:         "open",
		Unread:         1,
		LastActivityAt: now.Unix(),
	}
	messages := []api.Message{
		{ID: 1, Content: "謝謝你的幫助", MessageType: api.MessageTypeIncoming},
	}

	analysis := AnalyzeConversation(conv, messages, nil)

	if analysis.SentimentHint != "satisfied" {
		t.Errorf("expected sentiment 'satisfied' for '謝謝', got '%s'", analysis.SentimentHint)
	}
}

func TestAnalyzeConversation_UrgencyKeywordsChinese(t *testing.T) {
	now := time.Now()
	conv := &api.Conversation{
		ID:             1,
		Status:         "open",
		Unread:         1,
		LastActivityAt: now.Unix(),
	}
	messages := []api.Message{
		{ID: 1, Content: "這很急，請盡快處理", MessageType: api.MessageTypeIncoming},
	}

	analysis := AnalyzeConversation(conv, messages, nil)

	if analysis.Urgency != "high" {
		t.Errorf("expected urgency 'high' for '急', got '%s'", analysis.Urgency)
	}
	if !containsReason(analysis.Reasons, "urgency indicator") {
		t.Errorf("expected reason about urgency indicators, got %v", analysis.Reasons)
	}
}

func TestAnalyzeConversation_FrustrationKeywords(t *testing.T) {
	now := time.Now()
	conv := &api.Conversation{
		ID:             1,
		Status:         "open",
		Unread:         1,
		LastActivityAt: now.Unix(),
	}
	messages := []api.Message{
		{ID: 1, Content: "I am very frustrated with this", MessageType: api.MessageTypeIncoming},
	}

	analysis := AnalyzeConversation(conv, messages, nil)

	if analysis.SentimentHint != "frustrated" {
		t.Errorf("expected sentiment 'frustrated', got '%s'", analysis.SentimentHint)
	}
	if analysis.Urgency != "high" {
		t.Errorf("expected urgency 'high' for frustrated customer, got '%s'", analysis.Urgency)
	}
}

func TestAnalyzeConversation_QuestionIndicators(t *testing.T) {
	now := time.Now()
	conv := &api.Conversation{
		ID:             1,
		Status:         "open",
		Unread:         1,
		LastActivityAt: now.Unix(),
	}
	messages := []api.Message{
		{ID: 1, Content: "請問這個怎麼操作？", MessageType: api.MessageTypeIncoming},
	}

	analysis := AnalyzeConversation(conv, messages, nil)

	if !containsReason(analysis.Reasons, "question") {
		t.Errorf("expected reason about question, got %v", analysis.Reasons)
	}
}

func TestAnalyzeConversation_ContactHistory(t *testing.T) {
	now := time.Now()
	conv := &api.Conversation{
		ID:             1,
		Status:         "open",
		Unread:         1,
		LastActivityAt: now.Unix(),
	}
	messages := []api.Message{
		{ID: 1, Content: "Hi there", MessageType: api.MessageTypeIncoming},
	}
	// 12 previous conversations
	history := make([]api.Conversation, 12)
	for i := range history {
		history[i] = api.Conversation{ID: i + 100}
	}

	analysis := AnalyzeConversation(conv, messages, history)

	if analysis.Context == "" {
		t.Error("expected context about repeat customer, got empty string")
	}
	if !containsSubstring(analysis.Context, "Repeat customer") {
		t.Errorf("expected context mentioning repeat customer, got '%s'", analysis.Context)
	}
}

func TestAnalyzeConversation_NewCustomer(t *testing.T) {
	now := time.Now()
	conv := &api.Conversation{
		ID:             1,
		Status:         "open",
		Unread:         1,
		LastActivityAt: now.Unix(),
	}
	messages := []api.Message{
		{ID: 1, Content: "First time here", MessageType: api.MessageTypeIncoming},
	}
	history := []api.Conversation{
		{ID: 1}, // Only current conversation
	}

	analysis := AnalyzeConversation(conv, messages, history)

	if !containsSubstring(analysis.Context, "New customer") {
		t.Errorf("expected context about new customer, got '%s'", analysis.Context)
	}
}

func TestAnalyzeConversation_NeutralSentiment(t *testing.T) {
	now := time.Now()
	conv := &api.Conversation{
		ID:             1,
		Status:         "open",
		Unread:         1,
		LastActivityAt: now.Unix(),
	}
	messages := []api.Message{
		{ID: 1, Content: "I would like to order something", MessageType: api.MessageTypeIncoming},
	}

	analysis := AnalyzeConversation(conv, messages, nil)

	if analysis.SentimentHint != "neutral" {
		t.Errorf("expected sentiment 'neutral' for neutral message, got '%s'", analysis.SentimentHint)
	}
}

func TestSuggestActions_UnreadConversation(t *testing.T) {
	now := time.Now()
	conv := &api.Conversation{
		ID:             1,
		Status:         "open",
		Unread:         2,
		LastActivityAt: now.Add(-5 * time.Hour).Unix(),
	}
	messages := []api.Message{
		{ID: 1, Content: "Help please", MessageType: api.MessageTypeIncoming},
	}

	actions := SuggestActions(conv, messages, nil)

	if len(actions) == 0 {
		t.Fatal("expected at least one suggested action")
	}
	hasReply := false
	for _, a := range actions {
		if a.Action == "reply" {
			hasReply = true
			if a.Priority != "medium" && a.Priority != "high" {
				t.Errorf("expected reply priority medium or high for 5h wait, got '%s'", a.Priority)
			}
		}
	}
	if !hasReply {
		t.Error("expected 'reply' action to be suggested")
	}
}

func TestSuggestActions_UnassignedConversation(t *testing.T) {
	now := time.Now()
	conv := &api.Conversation{
		ID:             1,
		Status:         "open",
		Unread:         1,
		LastActivityAt: now.Unix(),
		Meta:           map[string]any{}, // No assignee
	}
	messages := []api.Message{
		{ID: 1, Content: "Hello", MessageType: api.MessageTypeIncoming},
	}

	actions := SuggestActions(conv, messages, nil)

	hasAssign := false
	for _, a := range actions {
		if a.Action == "assign" {
			hasAssign = true
			if a.Priority != "medium" {
				t.Errorf("expected assign priority 'medium', got '%s'", a.Priority)
			}
		}
	}
	if !hasAssign {
		t.Error("expected 'assign' action to be suggested for unassigned conversation")
	}
}

func TestSuggestActions_SatisfiedCustomer(t *testing.T) {
	now := time.Now()
	conv := &api.Conversation{
		ID:             1,
		Status:         "open",
		Unread:         1,
		LastActivityAt: now.Unix(),
	}
	messages := []api.Message{
		{ID: 1, Content: "Thank you so much, problem solved!", MessageType: api.MessageTypeIncoming},
	}

	actions := SuggestActions(conv, messages, nil)

	hasResolve := false
	for _, a := range actions {
		if a.Action == "resolve" {
			hasResolve = true
			if a.Priority != "low" {
				t.Errorf("expected resolve priority 'low', got '%s'", a.Priority)
			}
		}
	}
	if !hasResolve {
		t.Error("expected 'resolve' action to be suggested for satisfied customer")
	}
}

func TestFormatDuration_Minutes(t *testing.T) {
	d := 45 * time.Minute
	result := formatDuration(d)
	if result != "45m" {
		t.Errorf("expected '45m', got '%s'", result)
	}
}

func TestFormatDuration_Hours(t *testing.T) {
	d := 3*time.Hour + 30*time.Minute
	result := formatDuration(d)
	if result != "3h 30m" {
		t.Errorf("expected '3h 30m', got '%s'", result)
	}
}

func TestFormatDuration_Days(t *testing.T) {
	d := 2*24*time.Hour + 5*time.Hour
	result := formatDuration(d)
	if result != "2d 5h" {
		t.Errorf("expected '2d 5h', got '%s'", result)
	}
}

// Helper functions for tests
func containsReason(reasons []string, substring string) bool {
	for _, r := range reasons {
		if containsSubstring(r, substring) {
			return true
		}
	}
	return false
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsCI(s, substr))
}

func containsCI(s, substr string) bool {
	// Case-insensitive contains
	sl := len(s)
	subl := len(substr)
	for i := 0; i <= sl-subl; i++ {
		if eqCI(s[i:i+subl], substr) {
			return true
		}
	}
	return false
}

func eqCI(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
