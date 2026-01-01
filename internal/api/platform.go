package api

import (
	"context"
	"fmt"
)

// PlatformAccount represents a platform account
// Fields are kept minimal to avoid tight coupling to API changes.
type PlatformAccount struct {
	ID     int            `json:"id"`
	Name   string         `json:"name"`
	Locale string         `json:"locale,omitempty"`
	Domain string         `json:"domain,omitempty"`
	Status string         `json:"status,omitempty"`
	Meta   map[string]any `json:"meta,omitempty"`
}

// CreatePlatformAccountRequest represents a request to create an account
// See Chatwoot Platform API docs for optional fields.
type CreatePlatformAccountRequest struct {
	Name             string         `json:"name"`
	Locale           string         `json:"locale,omitempty"`
	Domain           string         `json:"domain,omitempty"`
	SupportEmail     string         `json:"support_email,omitempty"`
	Status           string         `json:"status,omitempty"`
	CustomAttributes map[string]any `json:"custom_attributes,omitempty"`
	Limits           map[string]any `json:"limits,omitempty"`
}

// CreatePlatformAccount creates a new account via platform API
func (c *Client) CreatePlatformAccount(ctx context.Context, req CreatePlatformAccountRequest) (*PlatformAccount, error) {
	var result PlatformAccount
	if err := c.do(ctx, "POST", c.platformPath("/accounts"), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPlatformAccount retrieves an account by ID via platform API
func (c *Client) GetPlatformAccount(ctx context.Context, accountID int) (*PlatformAccount, error) {
	var result PlatformAccount
	if err := c.do(ctx, "GET", c.platformPath(fmt.Sprintf("/accounts/%d", accountID)), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeletePlatformAccount deletes an account by ID via platform API
func (c *Client) DeletePlatformAccount(ctx context.Context, accountID int) error {
	return c.do(ctx, "DELETE", c.platformPath(fmt.Sprintf("/accounts/%d", accountID)), nil, nil)
}

// UpdatePlatformAccountRequest represents a request to update an account
type UpdatePlatformAccountRequest struct {
	Name   string `json:"name,omitempty"`
	Locale string `json:"locale,omitempty"`
	Domain string `json:"domain,omitempty"`
	Status string `json:"status,omitempty"`
}

// UpdatePlatformAccount updates an account via platform API
func (c *Client) UpdatePlatformAccount(ctx context.Context, accountID int, req UpdatePlatformAccountRequest) (*PlatformAccount, error) {
	var result PlatformAccount
	if err := c.do(ctx, "PATCH", c.platformPath(fmt.Sprintf("/accounts/%d", accountID)), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PlatformUser represents a platform user
// Fields are kept minimal to avoid tight coupling to API changes.
type PlatformUser struct {
	ID               int            `json:"id"`
	Name             string         `json:"name"`
	DisplayName      string         `json:"display_name,omitempty"`
	Email            string         `json:"email"`
	CustomAttributes map[string]any `json:"custom_attributes,omitempty"`
}

// CreatePlatformUserRequest represents a request to create a user
// See Chatwoot Platform API docs for optional fields.
type CreatePlatformUserRequest struct {
	Name             string         `json:"name"`
	DisplayName      string         `json:"display_name,omitempty"`
	Email            string         `json:"email"`
	Password         string         `json:"password"`
	CustomAttributes map[string]any `json:"custom_attributes,omitempty"`
}

// UpdatePlatformUserRequest represents a request to update a user
// Fields are optional.
type UpdatePlatformUserRequest struct {
	Name             string         `json:"name,omitempty"`
	DisplayName      string         `json:"display_name,omitempty"`
	Email            string         `json:"email,omitempty"`
	Password         string         `json:"password,omitempty"`
	CustomAttributes map[string]any `json:"custom_attributes,omitempty"`
}

// CreatePlatformUser creates a new user via platform API
func (c *Client) CreatePlatformUser(ctx context.Context, req CreatePlatformUserRequest) (*PlatformUser, error) {
	var result PlatformUser
	if err := c.do(ctx, "POST", c.platformPath("/users"), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPlatformUser retrieves a user by ID via platform API
func (c *Client) GetPlatformUser(ctx context.Context, userID int) (*PlatformUser, error) {
	var result PlatformUser
	if err := c.do(ctx, "GET", c.platformPath(fmt.Sprintf("/users/%d", userID)), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdatePlatformUser updates a user via platform API
func (c *Client) UpdatePlatformUser(ctx context.Context, userID int, req UpdatePlatformUserRequest) (*PlatformUser, error) {
	var result PlatformUser
	if err := c.do(ctx, "PATCH", c.platformPath(fmt.Sprintf("/users/%d", userID)), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeletePlatformUser deletes a user by ID via platform API
func (c *Client) DeletePlatformUser(ctx context.Context, userID int) error {
	return c.do(ctx, "DELETE", c.platformPath(fmt.Sprintf("/users/%d", userID)), nil, nil)
}

// PlatformAccountUser represents a user membership in an account
// Fields are kept minimal to avoid tight coupling to API changes.
type PlatformAccountUser struct {
	ID        int    `json:"id"`
	AccountID int    `json:"account_id"`
	UserID    int    `json:"user_id"`
	Role      string `json:"role"`
}

// CreatePlatformAccountUserRequest represents a request to add a user to an account
// See Chatwoot Platform API docs for role values.
type CreatePlatformAccountUserRequest struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
}

// ListPlatformAccountUsers lists account users via platform API
func (c *Client) ListPlatformAccountUsers(ctx context.Context, accountID int) ([]PlatformAccountUser, error) {
	var result []PlatformAccountUser
	if err := c.do(ctx, "GET", c.platformPath(fmt.Sprintf("/accounts/%d/account_users", accountID)), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreatePlatformAccountUser creates an account user via platform API
func (c *Client) CreatePlatformAccountUser(ctx context.Context, accountID int, req CreatePlatformAccountUserRequest) (*PlatformAccountUser, error) {
	var result PlatformAccountUser
	if err := c.do(ctx, "POST", c.platformPath(fmt.Sprintf("/accounts/%d/account_users", accountID)), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeletePlatformAccountUser removes a user from an account via platform API
func (c *Client) DeletePlatformAccountUser(ctx context.Context, accountID int, userID int) error {
	path := fmt.Sprintf("/accounts/%d/account_users", accountID)
	if userID > 0 {
		path = fmt.Sprintf("%s?user_id=%d", path, userID)
	}
	return c.do(ctx, "DELETE", c.platformPath(path), nil, nil)
}
