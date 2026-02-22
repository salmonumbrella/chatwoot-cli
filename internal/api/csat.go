package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// CSATListParams contains parameters for listing CSAT responses
type CSATListParams struct {
	Page      int
	Since     string // ISO date string
	Until     string // ISO date string
	InboxID   int
	TeamID    int
	Rating    string // comma-separated ratings like "1,2"
	Sort      string // created_at
	SortOrder string // asc or desc
}

// List retrieves CSAT survey responses with optional filters.
func (s CSATService) List(ctx context.Context, params CSATListParams) ([]CSATResponse, error) {
	return listCSATResponses(ctx, s, params)
}

func listCSATResponses(ctx context.Context, r Requester, params CSATListParams) ([]CSATResponse, error) {
	query := url.Values{}
	if params.Page > 0 {
		query.Set("page", strconv.Itoa(params.Page))
	}
	if params.Since != "" {
		query.Set("since", params.Since)
	}
	if params.Until != "" {
		query.Set("until", params.Until)
	}
	if params.InboxID > 0 {
		query.Set("inbox_id", strconv.Itoa(params.InboxID))
	}
	if params.TeamID > 0 {
		query.Set("team_id", strconv.Itoa(params.TeamID))
	}
	if params.Rating != "" {
		query.Set("rating", params.Rating)
	}
	if params.Sort != "" {
		query.Set("sort", params.Sort)
	}
	if params.SortOrder != "" {
		query.Set("sort_order", params.SortOrder)
	}

	path := "/csat_survey_responses"
	if len(query) > 0 {
		path = path + "?" + query.Encode()
	}

	body, err := r.doRaw(ctx, http.MethodGet, r.accountPath(path), nil)
	if err != nil {
		return nil, err
	}

	// Try bare array first (actual Chatwoot API response format)
	var responses []CSATResponse
	if err := json.Unmarshal(body, &responses); err == nil {
		return responses, nil
	}

	// Fall back to wrapped format for compatibility
	var wrapped CSATListResponse
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("unexpected API response format: %w", err)
	}
	return wrapped.Payload, nil
}

// Conversation retrieves CSAT for a specific conversation.
func (s CSATService) Conversation(ctx context.Context, conversationID int) (*CSATResponse, error) {
	return getConversationCSAT(ctx, s, conversationID)
}

func getConversationCSAT(ctx context.Context, r Requester, conversationID int) (*CSATResponse, error) {
	// The API doesn't have a direct endpoint for conversation CSAT,
	// so we filter the list by conversation (not ideal but works)
	path := fmt.Sprintf("/csat_survey_responses?conversation_id=%d", conversationID)

	body, err := r.doRaw(ctx, http.MethodGet, r.accountPath(path), nil)
	if err != nil {
		return nil, err
	}

	// Try bare array first (actual Chatwoot API response format)
	var responses []CSATResponse
	if err := json.Unmarshal(body, &responses); err == nil {
		if len(responses) == 0 {
			return nil, nil
		}
		return &responses[0], nil
	}

	// Fall back to wrapped format for compatibility
	var wrapped CSATListResponse
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("unexpected API response format: %w", err)
	}

	if len(wrapped.Payload) == 0 {
		return nil, nil
	}
	return &wrapped.Payload[0], nil
}
