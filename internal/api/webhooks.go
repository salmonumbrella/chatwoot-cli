package api

import (
	"context"
	"fmt"
	"net/http"
)

// List retrieves all webhooks for the account.
func (s WebhooksService) List(ctx context.Context) ([]Webhook, error) {
	return listWebhooks(ctx, s)
}

func listWebhooks(ctx context.Context, r Requester) ([]Webhook, error) {
	var result WebhookListResponse
	if err := r.do(ctx, http.MethodGet, r.accountPath("/webhooks"), nil, &result); err != nil {
		return nil, err
	}
	return result.Payload.Webhooks, nil
}

// Get retrieves a single webhook by ID.
func (s WebhooksService) Get(ctx context.Context, id int) (*Webhook, error) {
	return getWebhook(ctx, s, id)
}

func getWebhook(ctx context.Context, r Requester, id int) (*Webhook, error) {
	webhooks, err := listWebhooks(ctx, r)
	if err != nil {
		return nil, err
	}

	for _, wh := range webhooks {
		if wh.ID == id {
			return &wh, nil
		}
	}

	return nil, &APIError{
		StatusCode: 404,
		Body:       fmt.Sprintf("webhook with ID %d not found", id),
	}
}

// Create creates a new webhook.
func (s WebhooksService) Create(ctx context.Context, url string, subscriptions []string) (*Webhook, error) {
	return createWebhook(ctx, s, url, subscriptions)
}

func createWebhook(ctx context.Context, r Requester, url string, subscriptions []string) (*Webhook, error) {
	body := map[string]any{
		"url":           url,
		"subscriptions": subscriptions,
	}
	var result WebhookResponse
	if err := r.do(ctx, http.MethodPost, r.accountPath("/webhooks"), body, &result); err != nil {
		return nil, err
	}
	return &result.Payload.Webhook, nil
}

// Update updates an existing webhook.
func (s WebhooksService) Update(ctx context.Context, id int, url string, subscriptions []string) (*Webhook, error) {
	return updateWebhook(ctx, s, id, url, subscriptions)
}

func updateWebhook(ctx context.Context, r Requester, id int, url string, subscriptions []string) (*Webhook, error) {
	body := map[string]any{}
	if url != "" {
		body["url"] = url
	}
	if subscriptions != nil {
		body["subscriptions"] = subscriptions
	}

	var result WebhookResponse
	if err := r.do(ctx, http.MethodPatch, r.accountPath(fmt.Sprintf("/webhooks/%d", id)), body, &result); err != nil {
		return nil, err
	}
	return &result.Payload.Webhook, nil
}

// Delete deletes a webhook.
func (s WebhooksService) Delete(ctx context.Context, id int) error {
	return deleteWebhook(ctx, s, id)
}

func deleteWebhook(ctx context.Context, r Requester, id int) error {
	return r.do(ctx, http.MethodDelete, r.accountPath(fmt.Sprintf("/webhooks/%d", id)), nil, nil)
}
