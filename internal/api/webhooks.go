package api

import (
	"context"
	"fmt"
)

// ListWebhooks retrieves all webhooks for the account
func (c *Client) ListWebhooks(ctx context.Context) ([]Webhook, error) {
	var result WebhookListResponse
	err := c.Get(ctx, "/webhooks", &result)
	if err != nil {
		return nil, err
	}
	return result.Payload.Webhooks, nil
}

// GetWebhook retrieves a single webhook by ID
// Note: The Chatwoot API doesn't have a show endpoint for webhooks,
// so we list all webhooks and filter by ID
func (c *Client) GetWebhook(ctx context.Context, id int) (*Webhook, error) {
	webhooks, err := c.ListWebhooks(ctx)
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

// CreateWebhook creates a new webhook
func (c *Client) CreateWebhook(ctx context.Context, url string, subscriptions []string) (*Webhook, error) {
	body := map[string]any{
		"url":           url,
		"subscriptions": subscriptions,
	}
	var result WebhookResponse
	err := c.Post(ctx, "/webhooks", body, &result)
	if err != nil {
		return nil, err
	}
	return &result.Payload.Webhook, nil
}

// UpdateWebhook updates an existing webhook
func (c *Client) UpdateWebhook(ctx context.Context, id int, url string, subscriptions []string) (*Webhook, error) {
	body := map[string]any{}
	if url != "" {
		body["url"] = url
	}
	if subscriptions != nil {
		body["subscriptions"] = subscriptions
	}

	var result WebhookResponse
	err := c.Patch(ctx, fmt.Sprintf("/webhooks/%d", id), body, &result)
	if err != nil {
		return nil, err
	}
	return &result.Payload.Webhook, nil
}

// DeleteWebhook deletes a webhook
func (c *Client) DeleteWebhook(ctx context.Context, id int) error {
	return c.Delete(ctx, fmt.Sprintf("/webhooks/%d", id))
}
