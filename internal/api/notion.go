package api

import "context"

// DeleteNotionIntegration removes the Notion integration
func (c *Client) DeleteNotionIntegration(ctx context.Context) error {
	return c.Delete(ctx, "/integrations/notion")
}
