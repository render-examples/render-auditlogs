package auditlogs

import (
	"fmt"

	"github.com/renderinc/render-auditlogs/pkg/render"
)

type LogType string

const (
	auditLogsEndpoint = "/audit-logs"

	WorkspaceAuditLog    LogType = "workspace"
	OrganizationAuditLog LogType = "organization"
)

type Service interface {
	Get(id string, cursor string, limit int) ([]render.AuditLogEntry, error)
	Type() LogType
}

func NewWorkspaceSvc(client RenderClient) Service {
	return &WorkspaceSvc{client}
}

type WorkspaceSvc struct {
	client RenderClient
}

type RenderClient interface {
	GetAuditLogs(endpoint string, cursor string, limit int) ([]render.AuditLogEntry, error)
}

func (w *WorkspaceSvc) Get(id string, cursor string, limit int) ([]render.AuditLogEntry, error) {
	endpoint := fmt.Sprintf("/owners/%s%s", id, auditLogsEndpoint)

	return w.client.GetAuditLogs(endpoint, cursor, limit)
}

func (w *WorkspaceSvc) Type() LogType {
	return WorkspaceAuditLog
}

func NewOrganizationSvc(client *render.Client) Service {
	return &OrganizationSvc{client}
}

type OrganizationSvc struct {
	client RenderClient
}

func (o *OrganizationSvc) Get(id string, cursor string, limit int) ([]render.AuditLogEntry, error) {
	endpoint := fmt.Sprintf("/organizations/%s%s", id, auditLogsEndpoint)

	return o.client.GetAuditLogs(endpoint, cursor, limit)
}

func (o *OrganizationSvc) Type() LogType {
	return OrganizationAuditLog
}
