package api

import (
	"context"
	"fmt"
)

// SurveyResponse represents a CSAT survey response from the survey API
type SurveyResponse struct {
	ConversationID  int    `json:"conversation_id"`
	Rating          int    `json:"rating"`
	Message         string `json:"message,omitempty"`
	FeedbackMessage string `json:"feedback_message,omitempty"`
	ContactID       int    `json:"contact_id,omitempty"`
	AssignedAgentID int    `json:"assigned_agent_id,omitempty"`
}

// surveyPath returns the path for survey API (not account-scoped)
func (c *Client) surveyPath(path string) string {
	return fmt.Sprintf("%s/survey%s", c.BaseURL, path)
}

// GetSurveyResponse retrieves a survey response by conversation UUID
func (c *Client) GetSurveyResponse(ctx context.Context, conversationUUID string) (*SurveyResponse, error) {
	var result SurveyResponse
	path := c.surveyPath(fmt.Sprintf("/responses/%s", conversationUUID))
	if err := c.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetResponse retrieves a survey response by conversation UUID.
func (s SurveyService) GetResponse(ctx context.Context, conversationUUID string) (*SurveyResponse, error) {
	return s.GetSurveyResponse(ctx, conversationUUID)
}
