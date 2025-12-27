package api

import (
	"context"
	"fmt"
)

// Campaigns API returns raw arrays/objects directly, unlike some other
// Chatwoot endpoints that wrap responses in {"payload": ...}.
// This was verified by testing against the actual API.

// ListCampaigns returns all campaigns for the account.
func (c *Client) ListCampaigns(ctx context.Context, page int) ([]Campaign, error) {
	path := "/campaigns"
	if page > 0 {
		path = fmt.Sprintf("%s?page=%d", path, page)
	}
	var campaigns []Campaign
	if err := c.Get(ctx, path, &campaigns); err != nil {
		return nil, err
	}
	return campaigns, nil
}

// GetCampaign returns a single campaign by ID.
func (c *Client) GetCampaign(ctx context.Context, id int) (*Campaign, error) {
	var campaign Campaign
	if err := c.Get(ctx, fmt.Sprintf("/campaigns/%d", id), &campaign); err != nil {
		return nil, err
	}
	return &campaign, nil
}

// CreateCampaignRequest contains fields for creating a campaign.
type CreateCampaignRequest struct {
	Title                          string             `json:"title"`
	Description                    string             `json:"description,omitempty"`
	Message                        string             `json:"message"`
	Enabled                        bool               `json:"enabled"`
	InboxID                        int                `json:"inbox_id"`
	SenderID                       int                `json:"sender_id,omitempty"`
	ScheduledAt                    int64              `json:"scheduled_at,omitempty"`
	TriggerOnlyDuringBusinessHours bool               `json:"trigger_only_during_business_hours"`
	Audience                       []CampaignAudience `json:"audience,omitempty"`
	TriggerRules                   map[string]any     `json:"trigger_rules,omitempty"`
}

// CreateCampaign creates a new campaign.
func (c *Client) CreateCampaign(ctx context.Context, req CreateCampaignRequest) (*Campaign, error) {
	var campaign Campaign
	if err := c.Post(ctx, "/campaigns", req, &campaign); err != nil {
		return nil, err
	}
	return &campaign, nil
}

// UpdateCampaignRequest contains fields for updating a campaign.
type UpdateCampaignRequest struct {
	Title                          string             `json:"title,omitempty"`
	Description                    string             `json:"description,omitempty"`
	Message                        string             `json:"message,omitempty"`
	Enabled                        *bool              `json:"enabled,omitempty"`
	SenderID                       int                `json:"sender_id,omitempty"`
	ScheduledAt                    int64              `json:"scheduled_at,omitempty"`
	TriggerOnlyDuringBusinessHours *bool              `json:"trigger_only_during_business_hours,omitempty"`
	Audience                       []CampaignAudience `json:"audience,omitempty"`
	TriggerRules                   map[string]any     `json:"trigger_rules,omitempty"`
}

// UpdateCampaign updates an existing campaign.
func (c *Client) UpdateCampaign(ctx context.Context, id int, req UpdateCampaignRequest) (*Campaign, error) {
	var campaign Campaign
	if err := c.Patch(ctx, fmt.Sprintf("/campaigns/%d", id), req, &campaign); err != nil {
		return nil, err
	}
	return &campaign, nil
}

// DeleteCampaign deletes a campaign by ID.
func (c *Client) DeleteCampaign(ctx context.Context, id int) error {
	return c.Delete(ctx, fmt.Sprintf("/campaigns/%d", id))
}
