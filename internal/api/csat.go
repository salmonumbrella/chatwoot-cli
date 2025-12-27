package api

import (
	"context"
	"fmt"
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

// ListCSATResponses retrieves CSAT survey responses with optional filters
func (c *Client) ListCSATResponses(ctx context.Context, params CSATListParams) (*CSATListResponse, error) {
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

	var result CSATListResponse
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetConversationCSAT retrieves CSAT for a specific conversation
func (c *Client) GetConversationCSAT(ctx context.Context, conversationID int) (*CSATResponse, error) {
	// The API doesn't have a direct endpoint for conversation CSAT,
	// so we filter the list by conversation (not ideal but works)
	path := fmt.Sprintf("/csat_survey_responses?conversation_id=%d", conversationID)

	var result CSATListResponse
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}

	if len(result.Payload) == 0 {
		return nil, nil
	}
	return &result.Payload[0], nil
}
