package api

import (
	"context"
	"fmt"
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

// ListPortals lists all portals
func (c *Client) ListPortals(ctx context.Context) ([]Portal, error) {
	var result PortalListResponse
	err := c.Get(ctx, "/portals", &result)
	return result.Payload, err
}

// GetPortal gets a portal by slug
func (c *Client) GetPortal(ctx context.Context, portalSlug string) (*Portal, error) {
	var result Portal
	err := c.Get(ctx, fmt.Sprintf("/portals/%s", url.PathEscape(portalSlug)), &result)
	return &result, err
}

// CreatePortal creates a new portal
func (c *Client) CreatePortal(ctx context.Context, name, slug string) (*Portal, error) {
	body := map[string]any{
		"portal": map[string]any{
			"name": name,
			"slug": slug,
		},
	}

	var result Portal
	err := c.Post(ctx, "/portals", body, &result)
	return &result, err
}

// UpdatePortal updates a portal
func (c *Client) UpdatePortal(ctx context.Context, portalSlug string, name, slug string) (*Portal, error) {
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
	err := c.Patch(ctx, fmt.Sprintf("/portals/%s", url.PathEscape(portalSlug)), body, &result)
	return &result, err
}

// DeletePortal deletes a portal
func (c *Client) DeletePortal(ctx context.Context, portalSlug string) error {
	return c.Delete(ctx, fmt.Sprintf("/portals/%s", url.PathEscape(portalSlug)))
}

// ListPortalArticles lists articles in a portal
func (c *Client) ListPortalArticles(ctx context.Context, portalSlug string) ([]Article, error) {
	var result []Article
	err := c.Get(ctx, fmt.Sprintf("/portals/%s/articles", url.PathEscape(portalSlug)), &result)
	return result, err
}

// ListPortalCategories lists categories in a portal
func (c *Client) ListPortalCategories(ctx context.Context, portalSlug string) ([]Category, error) {
	var result []Category
	err := c.Get(ctx, fmt.Sprintf("/portals/%s/categories", url.PathEscape(portalSlug)), &result)
	return result, err
}

// GetArticle gets a specific article
func (c *Client) GetArticle(ctx context.Context, portalSlug string, articleID int) (*Article, error) {
	path := fmt.Sprintf("/portals/%s/articles/%d", url.PathEscape(portalSlug), articleID)
	var result Article
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateArticle creates a new article in a portal
func (c *Client) CreateArticle(ctx context.Context, portalSlug string, params map[string]any) (*Article, error) {
	path := fmt.Sprintf("/portals/%s/articles", url.PathEscape(portalSlug))
	var result Article
	if err := c.Post(ctx, path, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateArticle updates an article
func (c *Client) UpdateArticle(ctx context.Context, portalSlug string, articleID int, params map[string]any) (*Article, error) {
	path := fmt.Sprintf("/portals/%s/articles/%d", url.PathEscape(portalSlug), articleID)
	var result Article
	if err := c.Patch(ctx, path, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteArticle deletes an article
func (c *Client) DeleteArticle(ctx context.Context, portalSlug string, articleID int) error {
	path := fmt.Sprintf("/portals/%s/articles/%d", url.PathEscape(portalSlug), articleID)
	return c.Delete(ctx, path)
}

// GetCategory gets a specific category
func (c *Client) GetCategory(ctx context.Context, portalSlug string, categorySlug string) (*Category, error) {
	path := fmt.Sprintf("/portals/%s/categories/%s", url.PathEscape(portalSlug), url.PathEscape(categorySlug))
	var result Category
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateCategory creates a new category in a portal
func (c *Client) CreateCategory(ctx context.Context, portalSlug string, params map[string]any) (*Category, error) {
	path := fmt.Sprintf("/portals/%s/categories", url.PathEscape(portalSlug))
	var result Category
	if err := c.Post(ctx, path, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateCategory updates a category
func (c *Client) UpdateCategory(ctx context.Context, portalSlug, categorySlug string, params map[string]any) (*Category, error) {
	path := fmt.Sprintf("/portals/%s/categories/%s", url.PathEscape(portalSlug), url.PathEscape(categorySlug))
	var result Category
	if err := c.Patch(ctx, path, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteCategory deletes a category
func (c *Client) DeleteCategory(ctx context.Context, portalSlug, categorySlug string) error {
	path := fmt.Sprintf("/portals/%s/categories/%s", url.PathEscape(portalSlug), url.PathEscape(categorySlug))
	return c.Delete(ctx, path)
}
