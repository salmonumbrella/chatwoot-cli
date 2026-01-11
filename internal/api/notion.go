package api

import "context"

// Deprecated: Use client.Notion().Delete() instead.
func (c *Client) DeleteNotionIntegration(ctx context.Context) error {
	return deleteNotionIntegration(ctx, c)
}

// Delete removes the Notion integration.
func (s NotionService) Delete(ctx context.Context) error {
	return deleteNotionIntegration(ctx, s)
}

func deleteNotionIntegration(ctx context.Context, r Requester) error {
	return r.do(ctx, "DELETE", r.accountPath("/integrations/notion"), nil, nil)
}
