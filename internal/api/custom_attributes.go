package api

import (
	"context"
	"fmt"
)

// translateModelToAPIValue converts human-readable model names to API values.
// Chatwoot API expects: 0 for conversation_attribute, 1 for contact_attribute
func translateModelToAPIValue(model string) string {
	switch model {
	case "contact", "contact_attribute":
		return "1"
	case "conversation", "conversation_attribute":
		return "0"
	default:
		return model
	}
}

// ListCustomAttributes retrieves all custom attribute definitions for a model
func (c *Client) ListCustomAttributes(ctx context.Context, model string) ([]CustomAttribute, error) {
	path := "/custom_attribute_definitions"
	if model != "" {
		apiModel := translateModelToAPIValue(model)
		path = fmt.Sprintf("/custom_attribute_definitions?attribute_model=%s", apiModel)
	}

	var attrs []CustomAttribute
	if err := c.Get(ctx, path, &attrs); err != nil {
		return nil, err
	}
	return attrs, nil
}

// GetCustomAttribute retrieves a single custom attribute by ID
func (c *Client) GetCustomAttribute(ctx context.Context, id int) (*CustomAttribute, error) {
	path := fmt.Sprintf("/custom_attribute_definitions/%d", id)
	var attr CustomAttribute
	if err := c.Get(ctx, path, &attr); err != nil {
		return nil, err
	}
	return &attr, nil
}

// CreateCustomAttribute creates a new custom attribute definition
func (c *Client) CreateCustomAttribute(ctx context.Context, name, key, model, attrType string) (*CustomAttribute, error) {
	body := map[string]any{
		"attribute_display_name": name,
		"attribute_key":          key,
		"attribute_model":        model,
		"attribute_display_type": attrType,
	}

	var attr CustomAttribute
	if err := c.Post(ctx, "/custom_attribute_definitions", body, &attr); err != nil {
		return nil, err
	}
	return &attr, nil
}

// UpdateCustomAttribute updates an existing custom attribute definition
func (c *Client) UpdateCustomAttribute(ctx context.Context, id int, name string) (*CustomAttribute, error) {
	body := map[string]any{
		"attribute_display_name": name,
	}

	path := fmt.Sprintf("/custom_attribute_definitions/%d", id)
	var attr CustomAttribute
	if err := c.Patch(ctx, path, body, &attr); err != nil {
		return nil, err
	}
	return &attr, nil
}

// DeleteCustomAttribute deletes a custom attribute definition
func (c *Client) DeleteCustomAttribute(ctx context.Context, id int) error {
	path := fmt.Sprintf("/custom_attribute_definitions/%d", id)
	return c.Delete(ctx, path)
}
