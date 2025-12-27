package api

import (
	"context"
	"fmt"
)

// ListCustomFilters retrieves all custom filters for a filter type
func (c *Client) ListCustomFilters(ctx context.Context, filterType string) ([]CustomFilter, error) {
	path := "/custom_filters"
	if filterType != "" {
		path = fmt.Sprintf("/custom_filters?filter_type=%s", filterType)
	}

	var filters []CustomFilter
	if err := c.Get(ctx, path, &filters); err != nil {
		return nil, err
	}
	return filters, nil
}

// GetCustomFilter retrieves a single custom filter by ID
func (c *Client) GetCustomFilter(ctx context.Context, id int) (*CustomFilter, error) {
	path := fmt.Sprintf("/custom_filters/%d", id)
	var filter CustomFilter
	if err := c.Get(ctx, path, &filter); err != nil {
		return nil, err
	}
	return &filter, nil
}

// CreateCustomFilter creates a new custom filter
func (c *Client) CreateCustomFilter(ctx context.Context, name, filterType string, query map[string]any) (*CustomFilter, error) {
	body := map[string]any{
		"name":        name,
		"filter_type": filterType,
		"query":       query,
	}

	var filter CustomFilter
	if err := c.Post(ctx, "/custom_filters", body, &filter); err != nil {
		return nil, err
	}
	return &filter, nil
}

// UpdateCustomFilter updates an existing custom filter
func (c *Client) UpdateCustomFilter(ctx context.Context, id int, name string, query map[string]any) (*CustomFilter, error) {
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if query != nil {
		body["query"] = query
	}

	path := fmt.Sprintf("/custom_filters/%d", id)
	var filter CustomFilter
	if err := c.Patch(ctx, path, body, &filter); err != nil {
		return nil, err
	}
	return &filter, nil
}

// DeleteCustomFilter deletes a custom filter
func (c *Client) DeleteCustomFilter(ctx context.Context, id int) error {
	path := fmt.Sprintf("/custom_filters/%d", id)
	return c.Delete(ctx, path)
}
