package api

import (
	"context"
	"fmt"
)

// ListCannedResponses retrieves all canned responses for the account
func (c *Client) ListCannedResponses(ctx context.Context) ([]CannedResponse, error) {
	var responses []CannedResponse
	if err := c.Get(ctx, "/canned_responses", &responses); err != nil {
		return nil, err
	}
	return responses, nil
}

// GetCannedResponse retrieves a single canned response by ID
// Note: Chatwoot API doesn't have a dedicated show endpoint for canned responses,
// so we fetch the list and filter client-side
func (c *Client) GetCannedResponse(ctx context.Context, id int) (*CannedResponse, error) {
	responses, err := c.ListCannedResponses(ctx)
	if err != nil {
		return nil, err
	}

	for i := range responses {
		if responses[i].ID == id {
			return &responses[i], nil
		}
	}

	return nil, &APIError{
		StatusCode: 404,
		Body:       fmt.Sprintf("canned response with ID %d not found", id),
	}
}

// CreateCannedResponse creates a new canned response
func (c *Client) CreateCannedResponse(ctx context.Context, shortCode, content string) (*CannedResponse, error) {
	payload := map[string]any{
		"canned_response": map[string]string{
			"short_code": shortCode,
			"content":    content,
		},
	}
	var response CannedResponse
	if err := c.Post(ctx, "/canned_responses", payload, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// UpdateCannedResponse updates an existing canned response
func (c *Client) UpdateCannedResponse(ctx context.Context, id int, shortCode, content string) (*CannedResponse, error) {
	payload := map[string]any{
		"canned_response": map[string]string{
			"short_code": shortCode,
			"content":    content,
		},
	}
	var response CannedResponse
	path := fmt.Sprintf("/canned_responses/%d", id)
	if err := c.Patch(ctx, path, payload, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// DeleteCannedResponse deletes a canned response
func (c *Client) DeleteCannedResponse(ctx context.Context, id int) error {
	path := fmt.Sprintf("/canned_responses/%d", id)
	return c.Delete(ctx, path)
}
