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
	return listAuditLogs(ctx, c, page)
}

// List lists audit logs.
func (s AuditLogsService) List(ctx context.Context, page int) (*AuditLogList, error) {
	return listAuditLogs(ctx, s, page)
}

func listAuditLogs(ctx context.Context, r Requester, page int) (*AuditLogList, error) {
	path := "/audit_logs"
	if page > 0 {
		path = fmt.Sprintf("%s?page=%d", path, page)
	}

	var result AuditLogList
	err := r.do(ctx, "GET", r.accountPath(path), nil, &result)
	return &result, err
}
