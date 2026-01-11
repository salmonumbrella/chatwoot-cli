package api

import (
	"context"
	"fmt"
	"net/http"
)

// ListApps lists available integration apps.
func (s IntegrationsService) ListApps(ctx context.Context) ([]Integration, error) {
	return listIntegrationApps(ctx, s)
}

func listIntegrationApps(ctx context.Context, r Requester) ([]Integration, error) {
	var result IntegrationAppsResponse
	if err := r.do(ctx, http.MethodGet, r.accountPath("/integrations/apps"), nil, &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// ListHooks lists all integration hooks by extracting them from the apps response.
func (s IntegrationsService) ListHooks(ctx context.Context) ([]IntegrationHook, error) {
	return listIntegrationHooks(ctx, s)
}

func listIntegrationHooks(ctx context.Context, r Requester) ([]IntegrationHook, error) {
	apps, err := listIntegrationApps(ctx, r)
	if err != nil {
		return nil, err
	}

	var hooks []IntegrationHook
	for _, app := range apps {
		hooks = append(hooks, app.Hooks...)
	}
	return hooks, nil
}

// CreateHook creates a new integration hook.
func (s IntegrationsService) CreateHook(ctx context.Context, appID string, inboxID int, settings map[string]any) (*IntegrationHook, error) {
	return createIntegrationHook(ctx, s, appID, inboxID, settings)
}

func createIntegrationHook(ctx context.Context, r Requester, appID string, inboxID int, settings map[string]any) (*IntegrationHook, error) {
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
	err := r.do(ctx, http.MethodPost, r.accountPath("/integrations/hooks"), body, &result)
	return &result, err
}

// UpdateHook updates an integration hook.
func (s IntegrationsService) UpdateHook(ctx context.Context, hookID int, settings map[string]any) (*IntegrationHook, error) {
	return updateIntegrationHook(ctx, s, hookID, settings)
}

func updateIntegrationHook(ctx context.Context, r Requester, hookID int, settings map[string]any) (*IntegrationHook, error) {
	body := map[string]any{}
	if settings != nil {
		body["settings"] = settings
	}

	var result IntegrationHook
	err := r.do(ctx, http.MethodPatch, r.accountPath(fmt.Sprintf("/integrations/hooks/%d", hookID)), body, &result)
	return &result, err
}

// DeleteHook deletes an integration hook.
func (s IntegrationsService) DeleteHook(ctx context.Context, hookID int) error {
	return deleteIntegrationHook(ctx, s, hookID)
}

func deleteIntegrationHook(ctx context.Context, r Requester, hookID int) error {
	return r.do(ctx, http.MethodDelete, r.accountPath(fmt.Sprintf("/integrations/hooks/%d", hookID)), nil, nil)
}
