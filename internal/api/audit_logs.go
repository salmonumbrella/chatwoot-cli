package api

import (
	"context"
	"fmt"
	"net/http"
)

// AuditLogList is a paginated list of audit logs
type AuditLogList struct {
	Payload []AuditLog     `json:"payload"`
	Meta    PaginationMeta `json:"meta"`
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
	err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result)
	return &result, err
}
