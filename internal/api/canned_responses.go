package api

import (
	"context"
	"fmt"
)

// ListCannedResponses retrieves all canned responses for the account
func (c *Client) ListCannedResponses(ctx context.Context) ([]CannedResponse, error) {
	return listCannedResponses(ctx, c)
}

// List retrieves all canned responses for the account.
func (s CannedResponsesService) List(ctx context.Context) ([]CannedResponse, error) {
	return listCannedResponses(ctx, s)
}

func listCannedResponses(ctx context.Context, r Requester) ([]CannedResponse, error) {
	var responses []CannedResponse
	if err := r.do(ctx, "GET", r.accountPath("/canned_responses"), nil, &responses); err != nil {
		return nil, err
	}
	return responses, nil
}

// GetCannedResponse retrieves a single canned response by ID
// Note: Chatwoot API doesn't have a dedicated show endpoint for canned responses,
// so we fetch the list and filter client-side
func (c *Client) GetCannedResponse(ctx context.Context, id int) (*CannedResponse, error) {
	return getCannedResponse(ctx, c, id)
}

// Get retrieves a single canned response by ID.
func (s CannedResponsesService) Get(ctx context.Context, id int) (*CannedResponse, error) {
	return getCannedResponse(ctx, s, id)
}

func getCannedResponse(ctx context.Context, r Requester, id int) (*CannedResponse, error) {
	responses, err := listCannedResponses(ctx, r)
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
	return createCannedResponse(ctx, c, shortCode, content)
}

// Create creates a new canned response.
func (s CannedResponsesService) Create(ctx context.Context, shortCode, content string) (*CannedResponse, error) {
	return createCannedResponse(ctx, s, shortCode, content)
}

func createCannedResponse(ctx context.Context, r Requester, shortCode, content string) (*CannedResponse, error) {
	payload := map[string]any{
		"canned_response": map[string]string{
			"short_code": shortCode,
			"content":    content,
		},
	}
	var response CannedResponse
	if err := r.do(ctx, "POST", r.accountPath("/canned_responses"), payload, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// UpdateCannedResponse updates an existing canned response
func (c *Client) UpdateCannedResponse(ctx context.Context, id int, shortCode, content string) (*CannedResponse, error) {
	return updateCannedResponse(ctx, c, id, shortCode, content)
}

// Update updates an existing canned response.
func (s CannedResponsesService) Update(ctx context.Context, id int, shortCode, content string) (*CannedResponse, error) {
	return updateCannedResponse(ctx, s, id, shortCode, content)
}

func updateCannedResponse(ctx context.Context, r Requester, id int, shortCode, content string) (*CannedResponse, error) {
	payload := map[string]any{
		"canned_response": map[string]string{
			"short_code": shortCode,
			"content":    content,
		},
	}
	var response CannedResponse
	path := fmt.Sprintf("/canned_responses/%d", id)
	if err := r.do(ctx, "PATCH", r.accountPath(path), payload, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// DeleteCannedResponse deletes a canned response
func (c *Client) DeleteCannedResponse(ctx context.Context, id int) error {
	return deleteCannedResponse(ctx, c, id)
}

// Delete deletes a canned response.
func (s CannedResponsesService) Delete(ctx context.Context, id int) error {
	return deleteCannedResponse(ctx, s, id)
}

func deleteCannedResponse(ctx context.Context, r Requester, id int) error {
	path := fmt.Sprintf("/canned_responses/%d", id)
	return r.do(ctx, "DELETE", r.accountPath(path), nil, nil)
}
