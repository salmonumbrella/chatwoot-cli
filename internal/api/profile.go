package api

import (
	"context"
	"fmt"
	"net/http"
)

// GetProfile gets the current user's profile
func (c *Client) GetProfile(ctx context.Context) (*Profile, error) {
	url := fmt.Sprintf("%s/api/v1/profile", c.BaseURL)

	var result Profile
	if err := c.do(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Get gets the current user's profile.
func (s ProfileService) Get(ctx context.Context) (*Profile, error) {
	return s.GetProfile(ctx)
}
