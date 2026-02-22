package api

import (
	"context"
	"fmt"
	"net/http"
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

// List retrieves all labels for the account.
func (s LabelsService) List(ctx context.Context) ([]Label, error) {
	return listLabels(ctx, s)
}

func listLabels(ctx context.Context, r Requester) ([]Label, error) {
	var result LabelListResponse
	if err := r.do(ctx, http.MethodGet, r.accountPath("/labels"), nil, &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// Get retrieves a specific label by ID.
func (s LabelsService) Get(ctx context.Context, id int) (*Label, error) {
	return getLabel(ctx, s, id)
}

func getLabel(ctx context.Context, r Requester, id int) (*Label, error) {
	path := fmt.Sprintf("/labels/%d", id)
	var result Label
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Create creates a new label.
func (s LabelsService) Create(ctx context.Context, title, description, color string, showOnSidebar bool) (*Label, error) {
	return createLabel(ctx, s, title, description, color, showOnSidebar)
}

func createLabel(ctx context.Context, r Requester, title, description, color string, showOnSidebar bool) (*Label, error) {
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
	if err := r.do(ctx, http.MethodPost, r.accountPath("/labels"), body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Update updates an existing label.
func (s LabelsService) Update(ctx context.Context, id int, title, description, color string, showOnSidebar *bool) (*Label, error) {
	return updateLabel(ctx, s, id, title, description, color, showOnSidebar)
}

func updateLabel(ctx context.Context, r Requester, id int, title, description, color string, showOnSidebar *bool) (*Label, error) {
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
	if err := r.do(ctx, http.MethodPatch, r.accountPath(path), body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes a label.
func (s LabelsService) Delete(ctx context.Context, id int) error {
	return deleteLabel(ctx, s, id)
}

func deleteLabel(ctx context.Context, r Requester, id int) error {
	path := fmt.Sprintf("/labels/%d", id)
	return r.do(ctx, http.MethodDelete, r.accountPath(path), nil, nil)
}
