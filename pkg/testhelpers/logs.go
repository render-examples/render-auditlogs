package testhelpers

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/renderinc/render-auditlogs/pkg/render"
)

func CreateTestAuditLogs(num int, date time.Time) []render.AuditLogEntry {
	auditLogs := []render.AuditLogEntry{}

	for i := range num {
		id := uuid.New().String()

		auditLogs = append(auditLogs,
			render.AuditLogEntry{
				Cursor: id,
				AuditLog: render.AuditLog{
					ID:        fmt.Sprintf("aud-%s", id),
					Timestamp: date.Add(time.Duration(i) * time.Minute),
					Event:     "LoginEvent",
					Status:    "success",
					Actor: render.Actor{
						Type:  "user",
						Email: "test@example.com",
						ID:    "user-1",
					},
				},
			})
	}

	return auditLogs

}
