package api

import (
	"context"
	"fmt"
	"net/http"
)

// Get gets the current user's profile.
func (s ProfileService) Get(ctx context.Context) (*Profile, error) {
	url := fmt.Sprintf("%s/api/v1/profile", s.BaseURL)

	var result Profile
	if err := s.do(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
