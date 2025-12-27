package api

import (
	"context"
	"fmt"
)

// AuditLogList is a paginated list of audit logs
type AuditLogList struct {
	Payload []AuditLog     `json:"payload"`
	Meta    PaginationMeta `json:"meta"`
}

// ListAuditLogs lists audit logs
func (c *Client) ListAuditLogs(ctx context.Context, page int) (*AuditLogList, error) {
	path := "/audit_logs"
	if page > 0 {
		path = fmt.Sprintf("%s?page=%d", path, page)
	}

	var result AuditLogList
	err := c.Get(ctx, path, &result)
	return &result, err
}
