package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Article represents a help center article
type Article struct {
	ID         int    `json:"id"`
	PortalID   int    `json:"portal_id"`
	CategoryID int    `json:"category_id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Slug       string `json:"slug"`
	Status     string `json:"status"`
	Views      int    `json:"views"`
	AccountID  int    `json:"account_id"`
}

// Category represents a help center category
type Category struct {
	ID          int    `json:"id"`
	PortalID    int    `json:"portal_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	Position    int    `json:"position"`
	AccountID   int    `json:"account_id"`
}

// List lists all portals.
func (s PortalsService) List(ctx context.Context) ([]Portal, error) {
	return listPortals(ctx, s)
}

func listPortals(ctx context.Context, r Requester) ([]Portal, error) {
	var result PortalListResponse
	err := r.do(ctx, http.MethodGet, r.accountPath("/portals"), nil, &result)
	return result.Payload, err
}

// Get gets a portal by slug.
func (s PortalsService) Get(ctx context.Context, portalSlug string) (*Portal, error) {
	return getPortal(ctx, s, portalSlug)
}

func getPortal(ctx context.Context, r Requester, portalSlug string) (*Portal, error) {
	var result Portal
	path := fmt.Sprintf("/portals/%s", url.PathEscape(portalSlug))
	err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result)
	return &result, err
}

// Create creates a new portal.
func (s PortalsService) Create(ctx context.Context, name, slug string) (*Portal, error) {
	return createPortal(ctx, s, name, slug)
}

func createPortal(ctx context.Context, r Requester, name, slug string) (*Portal, error) {
	body := map[string]any{
		"portal": map[string]any{
			"name": name,
			"slug": slug,
		},
	}

	var result Portal
	err := r.do(ctx, http.MethodPost, r.accountPath("/portals"), body, &result)
	return &result, err
}

// Update updates a portal.
func (s PortalsService) Update(ctx context.Context, portalSlug string, name, slug string) (*Portal, error) {
	return updatePortal(ctx, s, portalSlug, name, slug)
}

func updatePortal(ctx context.Context, r Requester, portalSlug string, name, slug string) (*Portal, error) {
	portalParams := map[string]any{}
	if name != "" {
		portalParams["name"] = name
	}
	if slug != "" {
		portalParams["slug"] = slug
	}

	body := map[string]any{
		"portal": portalParams,
	}

	var result Portal
	path := fmt.Sprintf("/portals/%s", url.PathEscape(portalSlug))
	err := r.do(ctx, http.MethodPatch, r.accountPath(path), body, &result)
	return &result, err
}

// Delete deletes a portal.
func (s PortalsService) Delete(ctx context.Context, portalSlug string) error {
	return deletePortal(ctx, s, portalSlug)
}

func deletePortal(ctx context.Context, r Requester, portalSlug string) error {
	path := fmt.Sprintf("/portals/%s", url.PathEscape(portalSlug))
	return r.do(ctx, http.MethodDelete, r.accountPath(path), nil, nil)
}

// Articles lists articles in a portal.
func (s PortalsService) Articles(ctx context.Context, portalSlug string) ([]Article, error) {
	return listPortalArticles(ctx, s, portalSlug)
}

func listPortalArticles(ctx context.Context, r Requester, portalSlug string) ([]Article, error) {
	var result []Article
	path := fmt.Sprintf("/portals/%s/articles", url.PathEscape(portalSlug))
	err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result)
	return result, err
}

// SearchArticles searches articles in a portal by query string.
func (s PortalsService) SearchArticles(ctx context.Context, portalSlug, query string) ([]Article, error) {
	return searchPortalArticles(ctx, s, portalSlug, query)
}

func searchPortalArticles(ctx context.Context, r Requester, portalSlug, query string) ([]Article, error) {
	var result []Article
	path := fmt.Sprintf("/portals/%s/articles?query=%s", url.PathEscape(portalSlug), url.QueryEscape(query))
	err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result)
	return result, err
}

// Categories lists categories in a portal.
func (s PortalsService) Categories(ctx context.Context, portalSlug string) ([]Category, error) {
	return listPortalCategories(ctx, s, portalSlug)
}

func listPortalCategories(ctx context.Context, r Requester, portalSlug string) ([]Category, error) {
	var result []Category
	path := fmt.Sprintf("/portals/%s/categories", url.PathEscape(portalSlug))
	err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result)
	return result, err
}

// Article gets a specific article.
func (s PortalsService) Article(ctx context.Context, portalSlug string, articleID int) (*Article, error) {
	return getArticle(ctx, s, portalSlug, articleID)
}

func getArticle(ctx context.Context, r Requester, portalSlug string, articleID int) (*Article, error) {
	path := fmt.Sprintf("/portals/%s/articles/%d", url.PathEscape(portalSlug), articleID)
	var result Article
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateArticle creates a new article in a portal.
func (s PortalsService) CreateArticle(ctx context.Context, portalSlug string, params map[string]any) (*Article, error) {
	return createArticle(ctx, s, portalSlug, params)
}

func createArticle(ctx context.Context, r Requester, portalSlug string, params map[string]any) (*Article, error) {
	path := fmt.Sprintf("/portals/%s/articles", url.PathEscape(portalSlug))
	var result Article
	if err := r.do(ctx, http.MethodPost, r.accountPath(path), params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateArticle updates an article.
func (s PortalsService) UpdateArticle(ctx context.Context, portalSlug string, articleID int, params map[string]any) (*Article, error) {
	return updateArticle(ctx, s, portalSlug, articleID, params)
}

func updateArticle(ctx context.Context, r Requester, portalSlug string, articleID int, params map[string]any) (*Article, error) {
	path := fmt.Sprintf("/portals/%s/articles/%d", url.PathEscape(portalSlug), articleID)
	var result Article
	if err := r.do(ctx, http.MethodPatch, r.accountPath(path), params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteArticle deletes an article.
func (s PortalsService) DeleteArticle(ctx context.Context, portalSlug string, articleID int) error {
	return deleteArticle(ctx, s, portalSlug, articleID)
}

func deleteArticle(ctx context.Context, r Requester, portalSlug string, articleID int) error {
	path := fmt.Sprintf("/portals/%s/articles/%d", url.PathEscape(portalSlug), articleID)
	return r.do(ctx, http.MethodDelete, r.accountPath(path), nil, nil)
}

// Category gets a specific category.
func (s PortalsService) Category(ctx context.Context, portalSlug string, categorySlug string) (*Category, error) {
	return getCategory(ctx, s, portalSlug, categorySlug)
}

func getCategory(ctx context.Context, r Requester, portalSlug string, categorySlug string) (*Category, error) {
	path := fmt.Sprintf("/portals/%s/categories/%s", url.PathEscape(portalSlug), url.PathEscape(categorySlug))
	var result Category
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateCategory creates a new category in a portal.
func (s PortalsService) CreateCategory(ctx context.Context, portalSlug string, params map[string]any) (*Category, error) {
	return createCategory(ctx, s, portalSlug, params)
}

func createCategory(ctx context.Context, r Requester, portalSlug string, params map[string]any) (*Category, error) {
	path := fmt.Sprintf("/portals/%s/categories", url.PathEscape(portalSlug))
	var result Category
	if err := r.do(ctx, http.MethodPost, r.accountPath(path), params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateCategory updates a category.
func (s PortalsService) UpdateCategory(ctx context.Context, portalSlug, categorySlug string, params map[string]any) (*Category, error) {
	return updateCategory(ctx, s, portalSlug, categorySlug, params)
}

func updateCategory(ctx context.Context, r Requester, portalSlug, categorySlug string, params map[string]any) (*Category, error) {
	path := fmt.Sprintf("/portals/%s/categories/%s", url.PathEscape(portalSlug), url.PathEscape(categorySlug))
	var result Category
	if err := r.do(ctx, http.MethodPatch, r.accountPath(path), params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteCategory deletes a category.
func (s PortalsService) DeleteCategory(ctx context.Context, portalSlug, categorySlug string) error {
	return deleteCategory(ctx, s, portalSlug, categorySlug)
}

func deleteCategory(ctx context.Context, r Requester, portalSlug, categorySlug string) error {
	path := fmt.Sprintf("/portals/%s/categories/%s", url.PathEscape(portalSlug), url.PathEscape(categorySlug))
	return r.do(ctx, http.MethodDelete, r.accountPath(path), nil, nil)
}

// Archive archives a portal.
func (s PortalsService) Archive(ctx context.Context, portalSlug string) error {
	return archivePortal(ctx, s, portalSlug)
}

func archivePortal(ctx context.Context, r Requester, portalSlug string) error {
	path := fmt.Sprintf("/portals/%s/archive", url.PathEscape(portalSlug))
	return r.do(ctx, http.MethodPatch, r.accountPath(path), nil, nil)
}

// DeleteLogo removes the logo from a portal.
func (s PortalsService) DeleteLogo(ctx context.Context, portalSlug string) error {
	return deletePortalLogo(ctx, s, portalSlug)
}

func deletePortalLogo(ctx context.Context, r Requester, portalSlug string) error {
	path := fmt.Sprintf("/portals/%s/logo", url.PathEscape(portalSlug))
	return r.do(ctx, http.MethodDelete, r.accountPath(path), nil, nil)
}

// SendInstructions sends CNAME setup instructions for a portal.
func (s PortalsService) SendInstructions(ctx context.Context, portalSlug string) error {
	return sendPortalInstructions(ctx, s, portalSlug)
}

func sendPortalInstructions(ctx context.Context, r Requester, portalSlug string) error {
	path := fmt.Sprintf("/portals/%s/send_instructions", url.PathEscape(portalSlug))
	return r.do(ctx, http.MethodPost, r.accountPath(path), nil, nil)
}

// SSLStatus gets the SSL status for a portal.
func (s PortalsService) SSLStatus(ctx context.Context, portalSlug string) (map[string]any, error) {
	return getPortalSSLStatus(ctx, s, portalSlug)
}

func getPortalSSLStatus(ctx context.Context, r Requester, portalSlug string) (map[string]any, error) {
	path := fmt.Sprintf("/portals/%s/ssl_status", url.PathEscape(portalSlug))
	var result map[string]any
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ReorderArticles reorders articles in a portal.
func (s PortalsService) ReorderArticles(ctx context.Context, portalSlug string, articleIDs []int) error {
	return reorderArticles(ctx, s, portalSlug, articleIDs)
}

func reorderArticles(ctx context.Context, r Requester, portalSlug string, articleIDs []int) error {
	body := map[string][]int{"article_ids": articleIDs}
	path := fmt.Sprintf("/portals/%s/articles/reorder", url.PathEscape(portalSlug))
	return r.do(ctx, http.MethodPost, r.accountPath(path), body, nil)
}
