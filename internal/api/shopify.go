package api

import (
	"context"
	"fmt"
)

// ShopifyOrder represents a Shopify order
type ShopifyOrder struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Email             string `json:"email"`
	TotalPrice        string `json:"total_price"`
	Currency          string `json:"currency"`
	FinancialStatus   string `json:"financial_status"`
	FulfillmentStatus string `json:"fulfillment_status,omitempty"`
	CreatedAt         string `json:"created_at"`
}

// Deprecated: Use client.Shopify().Auth() instead.
func (c *Client) ShopifyAuth(ctx context.Context, shopDomain, code string) error {
	return shopifyAuth(ctx, c, shopDomain, code)
}

// Auth authenticates with Shopify using OAuth code.
func (s ShopifyService) Auth(ctx context.Context, shopDomain, code string) error {
	return shopifyAuth(ctx, s, shopDomain, code)
}

func shopifyAuth(ctx context.Context, r Requester, shopDomain, code string) error {
	body := map[string]string{
		"shop": shopDomain,
		"code": code,
	}
	return r.do(ctx, "POST", r.accountPath("/integrations/shopify/auth"), body, nil)
}

// Deprecated: Use client.Shopify().ListOrders() instead.
func (c *Client) ListShopifyOrders(ctx context.Context, contactID int) ([]ShopifyOrder, error) {
	return listShopifyOrders(ctx, c, contactID)
}

// ListOrders retrieves Shopify orders for a contact.
func (s ShopifyService) ListOrders(ctx context.Context, contactID int) ([]ShopifyOrder, error) {
	return listShopifyOrders(ctx, s, contactID)
}

func listShopifyOrders(ctx context.Context, r Requester, contactID int) ([]ShopifyOrder, error) {
	path := fmt.Sprintf("/integrations/shopify/orders?contact_id=%d", contactID)
	var result []ShopifyOrder
	if err := r.do(ctx, "GET", r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Deprecated: Use client.Shopify().Delete() instead.
func (c *Client) DeleteShopifyIntegration(ctx context.Context) error {
	return deleteShopifyIntegration(ctx, c)
}

// Delete removes the Shopify integration.
func (s ShopifyService) Delete(ctx context.Context) error {
	return deleteShopifyIntegration(ctx, s)
}

func deleteShopifyIntegration(ctx context.Context, r Requester) error {
	return r.do(ctx, "DELETE", r.accountPath("/integrations/shopify"), nil, nil)
}
