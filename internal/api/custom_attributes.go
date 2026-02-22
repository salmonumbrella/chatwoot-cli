package api

import (
	"context"
	"fmt"
	"net/http"
)

// translateModelToAPIValue converts human-readable model names to API integer values.
// Chatwoot API expects: 0 for conversation_attribute, 1 for contact_attribute
func translateModelToAPIValue(model string) int {
	switch model {
	case "contact", "contact_attribute":
		return 1
	case "conversation", "conversation_attribute":
		return 0
	default:
		return -1 // invalid model
	}
}

// translateModelToQueryParam converts human-readable model names to query parameter strings.
func translateModelToQueryParam(model string) string {
	switch model {
	case "contact", "contact_attribute":
		return "1"
	case "conversation", "conversation_attribute":
		return "0"
	default:
		return model
	}
}

// List retrieves all custom attribute definitions for a model.
func (s CustomAttributesService) List(ctx context.Context, model string) ([]CustomAttribute, error) {
	return listCustomAttributes(ctx, s, model)
}

func listCustomAttributes(ctx context.Context, r Requester, model string) ([]CustomAttribute, error) {
	path := "/custom_attribute_definitions"
	if model != "" {
		apiModel := translateModelToQueryParam(model)
		path = fmt.Sprintf("/custom_attribute_definitions?attribute_model=%s", apiModel)
	}

	var attrs []CustomAttribute
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &attrs); err != nil {
		return nil, err
	}
	return attrs, nil
}

// Get retrieves a single custom attribute by ID.
func (s CustomAttributesService) Get(ctx context.Context, id int) (*CustomAttribute, error) {
	return getCustomAttribute(ctx, s, id)
}

func getCustomAttribute(ctx context.Context, r Requester, id int) (*CustomAttribute, error) {
	path := fmt.Sprintf("/custom_attribute_definitions/%d", id)
	var attr CustomAttribute
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &attr); err != nil {
		return nil, err
	}
	return &attr, nil
}

// Create creates a new custom attribute definition.
func (s CustomAttributesService) Create(ctx context.Context, name, key, model, attrType string) (*CustomAttribute, error) {
	return createCustomAttribute(ctx, s, name, key, model, attrType)
}

func createCustomAttribute(ctx context.Context, r Requester, name, key, model, attrType string) (*CustomAttribute, error) {
	body := map[string]any{
		"attribute_display_name": name,
		"attribute_key":          key,
		"attribute_model":        translateModelToAPIValue(model),
		"attribute_display_type": attrType,
	}

	var attr CustomAttribute
	if err := r.do(ctx, http.MethodPost, r.accountPath("/custom_attribute_definitions"), body, &attr); err != nil {
		return nil, err
	}
	return &attr, nil
}

// Update updates an existing custom attribute definition.
func (s CustomAttributesService) Update(ctx context.Context, id int, name string) (*CustomAttribute, error) {
	return updateCustomAttribute(ctx, s, id, name)
}

func updateCustomAttribute(ctx context.Context, r Requester, id int, name string) (*CustomAttribute, error) {
	body := map[string]any{
		"attribute_display_name": name,
	}

	path := fmt.Sprintf("/custom_attribute_definitions/%d", id)
	var attr CustomAttribute
	if err := r.do(ctx, http.MethodPatch, r.accountPath(path), body, &attr); err != nil {
		return nil, err
	}
	return &attr, nil
}

// Delete deletes a custom attribute definition.
func (s CustomAttributesService) Delete(ctx context.Context, id int) error {
	return deleteCustomAttribute(ctx, s, id)
}

func deleteCustomAttribute(ctx context.Context, r Requester, id int) error {
	path := fmt.Sprintf("/custom_attribute_definitions/%d", id)
	return r.do(ctx, http.MethodDelete, r.accountPath(path), nil, nil)
}
