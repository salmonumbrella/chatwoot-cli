package api

import (
	"context"
	"fmt"
	"net/http"
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

// GetResponse retrieves a survey response by conversation UUID.
func (s SurveyService) GetResponse(ctx context.Context, conversationUUID string) (*SurveyResponse, error) {
	var result SurveyResponse
	path := s.surveyPath(fmt.Sprintf("/responses/%s", conversationUUID))
	if err := s.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
