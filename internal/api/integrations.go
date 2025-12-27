package api

import (
	"context"
	"fmt"
)

// ListIntegrationApps lists available integration apps
func (c *Client) ListIntegrationApps(ctx context.Context) ([]Integration, error) {
	var result IntegrationAppsResponse
	if err := c.Get(ctx, "/integrations/apps", &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// ListIntegrationHooks lists integration hooks
// Note: The Chatwoot API does not have a dedicated list endpoint for hooks.
// Hooks are returned nested within the apps response from ListIntegrationApps.
// This method will return a 404 error.
func (c *Client) ListIntegrationHooks(ctx context.Context) ([]IntegrationHook, error) {
	var result []IntegrationHook
	err := c.Get(ctx, "/integrations/hooks", &result)
	return result, err
}

// CreateIntegrationHook creates a new integration hook
func (c *Client) CreateIntegrationHook(ctx context.Context, appID string, inboxID int, settings map[string]any) (*IntegrationHook, error) {
	body := map[string]any{
		"app_id": appID,
	}
	if inboxID > 0 {
		body["inbox_id"] = inboxID
	}
	if settings != nil {
		body["settings"] = settings
	}

	var result IntegrationHook
	err := c.Post(ctx, "/integrations/hooks", body, &result)
	return &result, err
}

// UpdateIntegrationHook updates an integration hook
func (c *Client) UpdateIntegrationHook(ctx context.Context, hookID int, settings map[string]any) (*IntegrationHook, error) {
	body := map[string]any{}
	if settings != nil {
		body["settings"] = settings
	}

	var result IntegrationHook
	err := c.Patch(ctx, fmt.Sprintf("/integrations/hooks/%d", hookID), body, &result)
	return &result, err
}

// DeleteIntegrationHook deletes an integration hook
func (c *Client) DeleteIntegrationHook(ctx context.Context, hookID int) error {
	return c.Delete(ctx, fmt.Sprintf("/integrations/hooks/%d", hookID))
}
