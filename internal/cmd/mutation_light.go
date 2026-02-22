package cmd

import "github.com/chatwoot/chatwoot-cli/internal/api"

type lightMessageMutationResult struct {
	ID        int    `json:"id"`
	MessageID int    `json:"mid,omitempty"`
	Status    string `json:"st,omitempty"`
}

type lightAssignResult struct {
	ID      int  `json:"id"`
	AgentID *int `json:"ag,omitempty"`
	TeamID  *int `json:"tm,omitempty"`
}

type lightBulkAssignResult struct {
	Success int                   `json:"ok"`
	Failed  int                   `json:"er,omitempty"`
	Results []lightBulkAssignItem `json:"rs,omitempty"`
}

type lightBulkAssignItem struct {
	ID      int     `json:"id"`
	AgentID *int    `json:"ag,omitempty"`
	TeamID  *int    `json:"tm,omitempty"`
	Error   *string `json:"er,omitempty"`
}

type lightToggleStatusResult struct {
	ID          int    `json:"id"`
	Status      string `json:"st"`
	SnoozedUnix *int64 `json:"su,omitempty"`
}

type lightTogglePriorityResult struct {
	ID       int    `json:"id"`
	Priority string `json:"pri"`
}

type lightBulkMutationSummary struct {
	Success int `json:"ok"`
	Total   int `json:"tot"`
	Failed  int `json:"er,omitempty"`
}

func buildLightMessageMutationResult(conversationID, messageID int, status string) lightMessageMutationResult {
	return lightMessageMutationResult{
		ID:        conversationID,
		MessageID: messageID,
		Status:    shortStatus(status),
	}
}

func buildLightAssignResult(conversationID int, agentID, teamID *int) lightAssignResult {
	result := lightAssignResult{ID: conversationID}
	if agentID != nil {
		v := *agentID
		result.AgentID = &v
	}
	if teamID != nil {
		v := *teamID
		result.TeamID = &v
	}
	return result
}

func buildLightBulkAssignResult(results []BulkResult, successCount, failCount int, agentID, teamID int) lightBulkAssignResult {
	out := lightBulkAssignResult{
		Success: successCount,
		Failed:  failCount,
	}
	if len(results) == 0 {
		out.Results = []lightBulkAssignItem{}
		return out
	}
	out.Results = make([]lightBulkAssignItem, 0, len(results))
	for _, r := range results {
		item := lightBulkAssignItem{ID: r.ID}
		if r.Success {
			if agentID > 0 {
				id := agentID
				item.AgentID = &id
			}
			if teamID > 0 {
				id := teamID
				item.TeamID = &id
			}
		} else if r.Error != nil {
			errText := r.Error.Error()
			item.Error = &errText
		}
		out.Results = append(out.Results, item)
	}
	return out
}

func buildLightToggleStatusResult(conversationID int, status string, snoozedUntil *int64) lightToggleStatusResult {
	result := lightToggleStatusResult{
		ID:     conversationID,
		Status: shortStatus(status),
	}
	if snoozedUntil != nil && *snoozedUntil > 0 {
		v := *snoozedUntil
		result.SnoozedUnix = &v
	}
	return result
}

func buildLightTogglePriorityResult(conversationID int, priority string) lightTogglePriorityResult {
	return lightTogglePriorityResult{
		ID:       conversationID,
		Priority: shortPriority(priority),
	}
}

func buildLightBulkMutationSummary(successCount, total int) lightBulkMutationSummary {
	summary := lightBulkMutationSummary{
		Success: successCount,
		Total:   total,
	}
	failed := total - successCount
	if failed > 0 {
		summary.Failed = failed
	}
	return summary
}

// buildAgentAssignResult constructs the compact agent-mode output for assign operations.
func buildAgentAssignResult(conv *api.Conversation) map[string]any {
	item := map[string]any{"id": conv.ID}
	if status := shortStatus(conv.Status); status != "" {
		item["st"] = status
	}
	if conv.AssigneeID != nil {
		item["ag"] = *conv.AssigneeID
	}
	if conv.TeamID != nil {
		item["tm"] = *conv.TeamID
	}
	return item
}
