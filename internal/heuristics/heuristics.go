// Package heuristics provides conversation analysis for agent assistance.
package heuristics

import (
	"fmt"
	"strings"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

// Analysis contains the results of conversation analysis.
type Analysis struct {
	Urgency       string   `json:"urgency"`        // "high", "medium", "low"
	Reasons       []string `json:"reasons"`        // reasons for urgency level
	SentimentHint string   `json:"sentiment_hint"` // "satisfied", "frustrated", "neutral", "unknown"
	Context       string   `json:"context"`        // context about the customer
}

// SuggestedAction represents a recommended action for a conversation.
type SuggestedAction struct {
	Action   string `json:"action"`   // action type: "reply", "assign", "resolve", "review_duplicates"
	Reason   string `json:"reason"`   // explanation of why this action is suggested
	Priority string `json:"priority"` // "high", "medium", "low"
}

// Urgency keywords by language
var (
	urgencyKeywordsEnglish = []string{"urgent", "asap", "immediately", "emergency"}
	urgencyKeywordsChinese = []string{"急", "趕", "盡快", "馬上", "緊急", "立即"}

	satisfactionKeywordsEnglish = []string{"thank you", "thanks", "solved", "resolved", "great", "perfect"}
	satisfactionKeywordsChinese = []string{"謝謝", "感謝", "解決了", "好的", "收到", "太好了", "完美"}

	frustrationKeywordsEnglish = []string{"angry", "upset", "frustrated", "disappointed", "waiting"}
	frustrationKeywordsChinese = []string{"生氣", "不滿", "失望", "等很久", "怎麼回事", "太慢"}

	questionIndicators = []string{"?", "？", "請問", "想問", "可以嗎", "怎麼"}
)

// AnalyzeConversation analyzes a conversation and returns insights.
func AnalyzeConversation(conv *api.Conversation, messages []api.Message, contactHistory []api.Conversation) *Analysis {
	analysis := &Analysis{
		Urgency:       "low",
		Reasons:       []string{},
		SentimentHint: "unknown",
		Context:       "",
	}

	if conv == nil {
		return analysis
	}

	// Get the last incoming message content
	var lastMessageContent string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].MessageType == api.MessageTypeIncoming {
			lastMessageContent = messages[i].Content
			break
		}
	}
	contentLower := strings.ToLower(lastMessageContent)

	// Analyze wait time (from last activity)
	now := time.Now()
	lastActivity := time.Unix(conv.LastActivityAt, 0)
	waitDuration := now.Sub(lastActivity)

	// Urgency based on wait time
	if conv.Unread > 0 {
		if waitDuration > 24*time.Hour {
			analysis.Urgency = "high"
			analysis.Reasons = append(analysis.Reasons, fmt.Sprintf("Customer has been waiting %s with no response", formatDuration(waitDuration)))
		} else if waitDuration > 4*time.Hour {
			analysis.Urgency = "medium"
			analysis.Reasons = append(analysis.Reasons, fmt.Sprintf("Customer has been waiting %s", formatDuration(waitDuration)))
		}
	}

	// Multiple unread messages increases urgency
	if conv.Unread > 2 && analysis.Urgency == "low" {
		analysis.Urgency = "medium"
		analysis.Reasons = append(analysis.Reasons, fmt.Sprintf("Customer sent %d messages without response", conv.Unread))
	}

	// Check for urgency keywords
	if containsAnyKeyword(contentLower, urgencyKeywordsEnglish) || containsAnyKeyword(lastMessageContent, urgencyKeywordsChinese) {
		analysis.Urgency = "high"
		analysis.Reasons = append(analysis.Reasons, "Message contains urgency indicators")
	}

	// Detect sentiment
	if len(messages) > 0 {
		analysis.SentimentHint = "neutral"
	}

	// Check for satisfaction
	if containsAnyKeyword(contentLower, satisfactionKeywordsEnglish) || containsAnyKeyword(lastMessageContent, satisfactionKeywordsChinese) {
		analysis.SentimentHint = "satisfied"
	}

	// Check for frustration (overrides satisfaction and raises urgency)
	if containsAnyKeyword(contentLower, frustrationKeywordsEnglish) || containsAnyKeyword(lastMessageContent, frustrationKeywordsChinese) {
		analysis.SentimentHint = "frustrated"
		analysis.Urgency = "high"
		analysis.Reasons = append(analysis.Reasons, "Customer appears frustrated")
	}

	// Check for questions
	if containsAnyKeyword(lastMessageContent, questionIndicators) {
		analysis.Reasons = append(analysis.Reasons, "Customer asked a question")
	}

	// Analyze contact history
	historyCount := len(contactHistory)
	if historyCount == 1 {
		analysis.Context = "New customer - first interaction"
	} else if historyCount >= 10 {
		analysis.Context = fmt.Sprintf("Repeat customer with %d previous conversations", historyCount)
	} else if historyCount > 1 {
		analysis.Context = fmt.Sprintf("Returning customer with %d previous conversations", historyCount)
	}

	return analysis
}

// SuggestActions suggests actions based on conversation state.
func SuggestActions(conv *api.Conversation, messages []api.Message, contactHistory []api.Conversation) []SuggestedAction {
	var actions []SuggestedAction

	if conv == nil {
		return actions
	}

	// Get the last incoming message content for sentiment detection
	var lastMessageContent string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].MessageType == api.MessageTypeIncoming {
			lastMessageContent = messages[i].Content
			break
		}
	}
	contentLower := strings.ToLower(lastMessageContent)

	// Calculate wait time
	now := time.Now()
	lastActivity := time.Unix(conv.LastActivityAt, 0)
	waitDuration := now.Sub(lastActivity)

	// Suggest reply for unread conversations
	if conv.Unread > 0 {
		priority := "low"
		reason := "Customer has unread messages"
		if waitDuration > 24*time.Hour {
			priority = "high"
			reason = fmt.Sprintf("Customer has been waiting %s", formatDuration(waitDuration))
		} else if waitDuration > 4*time.Hour {
			priority = "medium"
			reason = fmt.Sprintf("Customer has been waiting %s", formatDuration(waitDuration))
		}
		actions = append(actions, SuggestedAction{
			Action:   "reply",
			Reason:   reason,
			Priority: priority,
		})
	}

	// Suggest assignment if unassigned
	if conv.Meta != nil {
		if _, hasAssignee := conv.Meta["assignee"]; !hasAssignee {
			actions = append(actions, SuggestedAction{
				Action:   "assign",
				Reason:   "Conversation is not assigned to any agent",
				Priority: "medium",
			})
		}
	} else {
		// No meta means no assignee
		actions = append(actions, SuggestedAction{
			Action:   "assign",
			Reason:   "Conversation is not assigned to any agent",
			Priority: "medium",
		})
	}

	// Suggest resolve if customer seems satisfied
	if containsAnyKeyword(contentLower, satisfactionKeywordsEnglish) || containsAnyKeyword(lastMessageContent, satisfactionKeywordsChinese) {
		actions = append(actions, SuggestedAction{
			Action:   "resolve",
			Reason:   "Customer expressed satisfaction",
			Priority: "low",
		})
	}

	// Check for multiple open conversations (duplicates)
	openCount := 0
	for _, c := range contactHistory {
		if c.Status == "open" || c.Status == "pending" {
			openCount++
		}
	}
	if openCount > 1 {
		actions = append(actions, SuggestedAction{
			Action:   "review_duplicates",
			Reason:   fmt.Sprintf("Contact has %d open conversations", openCount),
			Priority: "medium",
		})
	}

	return actions
}

// formatDuration formats a duration in human-readable form.
func formatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}

// containsAnyKeyword checks if the text contains any of the keywords.
func containsAnyKeyword(text string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}
