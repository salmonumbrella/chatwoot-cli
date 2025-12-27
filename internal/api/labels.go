package api

import (
	"context"
	"fmt"
)

// Label represents an account-level label
type Label struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	Description   string `json:"description,omitempty"`
	Color         string `json:"color,omitempty"`
	ShowOnSidebar bool   `json:"show_on_sidebar"`
}

// LabelListResponse wraps the labels list response
type LabelListResponse struct {
	Payload []Label `json:"payload"`
}

// ListLabels retrieves all labels for the account
func (c *Client) ListLabels(ctx context.Context) ([]Label, error) {
	var result LabelListResponse
	if err := c.Get(ctx, "/labels", &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// GetLabel retrieves a specific label by ID
func (c *Client) GetLabel(ctx context.Context, id int) (*Label, error) {
	path := fmt.Sprintf("/labels/%d", id)
	var result Label
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateLabel creates a new label
func (c *Client) CreateLabel(ctx context.Context, title, description, color string, showOnSidebar bool) (*Label, error) {
	body := map[string]any{
		"title":           title,
		"show_on_sidebar": showOnSidebar,
	}
	if description != "" {
		body["description"] = description
	}
	if color != "" {
		body["color"] = color
	}

	var result Label
	if err := c.Post(ctx, "/labels", body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateLabel updates an existing label
func (c *Client) UpdateLabel(ctx context.Context, id int, title, description, color string, showOnSidebar *bool) (*Label, error) {
	body := map[string]any{}
	if title != "" {
		body["title"] = title
	}
	if description != "" {
		body["description"] = description
	}
	if color != "" {
		body["color"] = color
	}
	if showOnSidebar != nil {
		body["show_on_sidebar"] = *showOnSidebar
	}

	path := fmt.Sprintf("/labels/%d", id)
	var result Label
	if err := c.Patch(ctx, path, body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteLabel deletes a label
func (c *Client) DeleteLabel(ctx context.Context, id int) error {
	path := fmt.Sprintf("/labels/%d", id)
	return c.Delete(ctx, path)
}
