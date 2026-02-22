package api

import (
	"context"
	"net/http"
)

// Delete removes the Notion integration.
func (s NotionService) Delete(ctx context.Context) error {
	return deleteNotionIntegration(ctx, s)
}

func deleteNotionIntegration(ctx context.Context, r Requester) error {
	return r.do(ctx, http.MethodDelete, r.accountPath("/integrations/notion"), nil, nil)
}
