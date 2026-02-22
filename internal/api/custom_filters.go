package api

import (
	"context"
	"fmt"
	"net/http"
)

// List retrieves all custom filters for a filter type.
func (s CustomFiltersService) List(ctx context.Context, filterType string) ([]CustomFilter, error) {
	return listCustomFilters(ctx, s, filterType)
}

func listCustomFilters(ctx context.Context, r Requester, filterType string) ([]CustomFilter, error) {
	path := "/custom_filters"
	if filterType != "" {
		path = fmt.Sprintf("/custom_filters?filter_type=%s", filterType)
	}

	var filters []CustomFilter
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &filters); err != nil {
		return nil, err
	}
	return filters, nil
}

// Get retrieves a single custom filter by ID.
func (s CustomFiltersService) Get(ctx context.Context, id int) (*CustomFilter, error) {
	return getCustomFilter(ctx, s, id)
}

func getCustomFilter(ctx context.Context, r Requester, id int) (*CustomFilter, error) {
	path := fmt.Sprintf("/custom_filters/%d", id)
	var filter CustomFilter
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &filter); err != nil {
		return nil, err
	}
	return &filter, nil
}

// Create creates a new custom filter.
func (s CustomFiltersService) Create(ctx context.Context, name, filterType string, query map[string]any) (*CustomFilter, error) {
	return createCustomFilter(ctx, s, name, filterType, query)
}

func createCustomFilter(ctx context.Context, r Requester, name, filterType string, query map[string]any) (*CustomFilter, error) {
	body := map[string]any{
		"name":        name,
		"filter_type": filterType,
		"query":       query,
	}

	var filter CustomFilter
	if err := r.do(ctx, http.MethodPost, r.accountPath("/custom_filters"), body, &filter); err != nil {
		return nil, err
	}
	return &filter, nil
}

// Update updates an existing custom filter.
func (s CustomFiltersService) Update(ctx context.Context, id int, name string, query map[string]any) (*CustomFilter, error) {
	return updateCustomFilter(ctx, s, id, name, query)
}

func updateCustomFilter(ctx context.Context, r Requester, id int, name string, query map[string]any) (*CustomFilter, error) {
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if query != nil {
		body["query"] = query
	}

	path := fmt.Sprintf("/custom_filters/%d", id)
	var filter CustomFilter
	if err := r.do(ctx, http.MethodPatch, r.accountPath(path), body, &filter); err != nil {
		return nil, err
	}
	return &filter, nil
}

// Delete deletes a custom filter.
func (s CustomFiltersService) Delete(ctx context.Context, id int) error {
	return deleteCustomFilter(ctx, s, id)
}

func deleteCustomFilter(ctx context.Context, r Requester, id int) error {
	path := fmt.Sprintf("/custom_filters/%d", id)
	return r.do(ctx, http.MethodDelete, r.accountPath(path), nil, nil)
}
