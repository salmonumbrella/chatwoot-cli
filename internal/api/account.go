package api

import (
	"context"
	"fmt"
	"net/http"
)

// GetAccount gets the account details
func (c *Client) GetAccount(ctx context.Context) (*Account, error) {
	url := fmt.Sprintf("%s/api/v1/accounts/%d", c.BaseURL, c.AccountID)

	var result Account
	if err := c.do(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateAccount updates the account
func (c *Client) UpdateAccount(ctx context.Context, name string) (*Account, error) {
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}

	url := fmt.Sprintf("%s/api/v1/accounts/%d", c.BaseURL, c.AccountID)

	var result Account
	if err := c.do(ctx, http.MethodPatch, url, body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
