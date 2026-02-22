package api

import (
	"context"
	"net/http"
)

// Get gets the account details.
func (s AccountService) Get(ctx context.Context) (*Account, error) {
	return getAccount(ctx, s)
}

func getAccount(ctx context.Context, r Requester) (*Account, error) {
	url := r.accountPath("")
	var result Account
	if err := r.do(ctx, http.MethodGet, url, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Update updates the account.
func (s AccountService) Update(ctx context.Context, name string) (*Account, error) {
	return updateAccount(ctx, s, name)
}

func updateAccount(ctx context.Context, r Requester, name string) (*Account, error) {
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}

	url := r.accountPath("")

	var result Account
	if err := r.do(ctx, http.MethodPatch, url, body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
