package api

import (
	"context"
	"fmt"
	"net/http"
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

// Auth authenticates with Shopify using OAuth code.
func (s ShopifyService) Auth(ctx context.Context, shopDomain, code string) error {
	return shopifyAuth(ctx, s, shopDomain, code)
}

func shopifyAuth(ctx context.Context, r Requester, shopDomain, code string) error {
	body := map[string]string{
		"shop": shopDomain,
		"code": code,
	}
	return r.do(ctx, http.MethodPost, r.accountPath("/integrations/shopify/auth"), body, nil)
}

// ListOrders retrieves Shopify orders for a contact.
func (s ShopifyService) ListOrders(ctx context.Context, contactID int) ([]ShopifyOrder, error) {
	return listShopifyOrders(ctx, s, contactID)
}

func listShopifyOrders(ctx context.Context, r Requester, contactID int) ([]ShopifyOrder, error) {
	path := fmt.Sprintf("/integrations/shopify/orders?contact_id=%d", contactID)
	var result []ShopifyOrder
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Delete removes the Shopify integration.
func (s ShopifyService) Delete(ctx context.Context) error {
	return deleteShopifyIntegration(ctx, s)
}

func deleteShopifyIntegration(ctx context.Context, r Requester) error {
	return r.do(ctx, http.MethodDelete, r.accountPath("/integrations/shopify"), nil, nil)
}
