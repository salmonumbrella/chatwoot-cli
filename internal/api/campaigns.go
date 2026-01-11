package api

import (
	"context"
	"fmt"
	"net/http"
)

// Campaigns API returns raw arrays/objects directly, unlike some other
// Chatwoot endpoints that wrap responses in {"payload": ...}.
// This was verified by testing against the actual API.

// List returns all campaigns for the account.
func (s CampaignsService) List(ctx context.Context, page int) ([]Campaign, error) {
	return listCampaigns(ctx, s, page)
}

func listCampaigns(ctx context.Context, r Requester, page int) ([]Campaign, error) {
	path := "/campaigns"
	if page > 0 {
		path = fmt.Sprintf("%s?page=%d", path, page)
	}
	var campaigns []Campaign
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &campaigns); err != nil {
		return nil, err
	}
	return campaigns, nil
}

// Get returns a single campaign by ID.
func (s CampaignsService) Get(ctx context.Context, id int) (*Campaign, error) {
	return getCampaign(ctx, s, id)
}

func getCampaign(ctx context.Context, r Requester, id int) (*Campaign, error) {
	var campaign Campaign
	if err := r.do(ctx, http.MethodGet, r.accountPath(fmt.Sprintf("/campaigns/%d", id)), nil, &campaign); err != nil {
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

// Create creates a new campaign.
func (s CampaignsService) Create(ctx context.Context, req CreateCampaignRequest) (*Campaign, error) {
	return createCampaign(ctx, s, req)
}

func createCampaign(ctx context.Context, r Requester, req CreateCampaignRequest) (*Campaign, error) {
	var campaign Campaign
	if err := r.do(ctx, http.MethodPost, r.accountPath("/campaigns"), req, &campaign); err != nil {
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

// Update updates an existing campaign.
func (s CampaignsService) Update(ctx context.Context, id int, req UpdateCampaignRequest) (*Campaign, error) {
	return updateCampaign(ctx, s, id, req)
}

func updateCampaign(ctx context.Context, r Requester, id int, req UpdateCampaignRequest) (*Campaign, error) {
	var campaign Campaign
	if err := r.do(ctx, http.MethodPatch, r.accountPath(fmt.Sprintf("/campaigns/%d", id)), req, &campaign); err != nil {
		return nil, err
	}
	return &campaign, nil
}

// Delete deletes a campaign by ID.
func (s CampaignsService) Delete(ctx context.Context, id int) error {
	return deleteCampaign(ctx, s, id)
}

func deleteCampaign(ctx context.Context, r Requester, id int) error {
	return r.do(ctx, http.MethodDelete, r.accountPath(fmt.Sprintf("/campaigns/%d", id)), nil, nil)
}
