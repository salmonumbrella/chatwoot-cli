package api

import (
	"context"
	"fmt"
	"net/http"
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

// CreateAccount creates a new account via platform API.
func (s PlatformService) CreateAccount(ctx context.Context, req CreatePlatformAccountRequest) (*PlatformAccount, error) {
	return createPlatformAccount(ctx, s, req)
}

func createPlatformAccount(ctx context.Context, r Requester, req CreatePlatformAccountRequest) (*PlatformAccount, error) {
	var result PlatformAccount
	if err := r.do(ctx, http.MethodPost, r.platformPath("/accounts"), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAccount retrieves an account by ID via platform API.
func (s PlatformService) GetAccount(ctx context.Context, accountID int) (*PlatformAccount, error) {
	return getPlatformAccount(ctx, s, accountID)
}

func getPlatformAccount(ctx context.Context, r Requester, accountID int) (*PlatformAccount, error) {
	var result PlatformAccount
	if err := r.do(ctx, http.MethodGet, r.platformPath(fmt.Sprintf("/accounts/%d", accountID)), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteAccount deletes an account by ID via platform API.
func (s PlatformService) DeleteAccount(ctx context.Context, accountID int) error {
	return deletePlatformAccount(ctx, s, accountID)
}

func deletePlatformAccount(ctx context.Context, r Requester, accountID int) error {
	return r.do(ctx, http.MethodDelete, r.platformPath(fmt.Sprintf("/accounts/%d", accountID)), nil, nil)
}

// UpdatePlatformAccountRequest represents a request to update an account
type UpdatePlatformAccountRequest struct {
	Name   string `json:"name,omitempty"`
	Locale string `json:"locale,omitempty"`
	Domain string `json:"domain,omitempty"`
	Status string `json:"status,omitempty"`
}

// UpdateAccount updates an account via platform API.
func (s PlatformService) UpdateAccount(ctx context.Context, accountID int, req UpdatePlatformAccountRequest) (*PlatformAccount, error) {
	return updatePlatformAccount(ctx, s, accountID, req)
}

func updatePlatformAccount(ctx context.Context, r Requester, accountID int, req UpdatePlatformAccountRequest) (*PlatformAccount, error) {
	var result PlatformAccount
	if err := r.do(ctx, http.MethodPatch, r.platformPath(fmt.Sprintf("/accounts/%d", accountID)), req, &result); err != nil {
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

// CreateUser creates a new user via platform API.
func (s PlatformService) CreateUser(ctx context.Context, req CreatePlatformUserRequest) (*PlatformUser, error) {
	return createPlatformUser(ctx, s, req)
}

func createPlatformUser(ctx context.Context, r Requester, req CreatePlatformUserRequest) (*PlatformUser, error) {
	var result PlatformUser
	if err := r.do(ctx, http.MethodPost, r.platformPath("/users"), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetUser retrieves a user by ID via platform API.
func (s PlatformService) GetUser(ctx context.Context, userID int) (*PlatformUser, error) {
	return getPlatformUser(ctx, s, userID)
}

func getPlatformUser(ctx context.Context, r Requester, userID int) (*PlatformUser, error) {
	var result PlatformUser
	if err := r.do(ctx, http.MethodGet, r.platformPath(fmt.Sprintf("/users/%d", userID)), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateUser updates a user via platform API.
func (s PlatformService) UpdateUser(ctx context.Context, userID int, req UpdatePlatformUserRequest) (*PlatformUser, error) {
	return updatePlatformUser(ctx, s, userID, req)
}

func updatePlatformUser(ctx context.Context, r Requester, userID int, req UpdatePlatformUserRequest) (*PlatformUser, error) {
	var result PlatformUser
	if err := r.do(ctx, http.MethodPatch, r.platformPath(fmt.Sprintf("/users/%d", userID)), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteUser deletes a user by ID via platform API.
func (s PlatformService) DeleteUser(ctx context.Context, userID int) error {
	return deletePlatformUser(ctx, s, userID)
}

func deletePlatformUser(ctx context.Context, r Requester, userID int) error {
	return r.do(ctx, http.MethodDelete, r.platformPath(fmt.Sprintf("/users/%d", userID)), nil, nil)
}

// PlatformUserLogin represents the SSO login response
type PlatformUserLogin struct {
	URL string `json:"url"`
}

// GetUserLogin gets the SSO login URL for a user.
func (s PlatformService) GetUserLogin(ctx context.Context, userID int) (*PlatformUserLogin, error) {
	return getPlatformUserLogin(ctx, s, userID)
}

func getPlatformUserLogin(ctx context.Context, r Requester, userID int) (*PlatformUserLogin, error) {
	var result PlatformUserLogin
	if err := r.do(ctx, http.MethodGet, r.platformPath(fmt.Sprintf("/users/%d/login", userID)), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
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

// ListAccountUsers lists account users via platform API.
func (s PlatformService) ListAccountUsers(ctx context.Context, accountID int) ([]PlatformAccountUser, error) {
	return listPlatformAccountUsers(ctx, s, accountID)
}

func listPlatformAccountUsers(ctx context.Context, r Requester, accountID int) ([]PlatformAccountUser, error) {
	var result []PlatformAccountUser
	if err := r.do(ctx, http.MethodGet, r.platformPath(fmt.Sprintf("/accounts/%d/account_users", accountID)), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateAccountUser creates an account user via platform API.
func (s PlatformService) CreateAccountUser(ctx context.Context, accountID int, req CreatePlatformAccountUserRequest) (*PlatformAccountUser, error) {
	return createPlatformAccountUser(ctx, s, accountID, req)
}

func createPlatformAccountUser(ctx context.Context, r Requester, accountID int, req CreatePlatformAccountUserRequest) (*PlatformAccountUser, error) {
	var result PlatformAccountUser
	if err := r.do(ctx, http.MethodPost, r.platformPath(fmt.Sprintf("/accounts/%d/account_users", accountID)), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteAccountUser removes a user from an account via platform API.
func (s PlatformService) DeleteAccountUser(ctx context.Context, accountID int, userID int) error {
	return deletePlatformAccountUser(ctx, s, accountID, userID)
}

func deletePlatformAccountUser(ctx context.Context, r Requester, accountID int, userID int) error {
	path := fmt.Sprintf("/accounts/%d/account_users", accountID)
	if userID > 0 {
		path = fmt.Sprintf("%s?user_id=%d", path, userID)
	}
	return r.do(ctx, http.MethodDelete, r.platformPath(path), nil, nil)
}
