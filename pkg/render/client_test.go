package render_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/renderinc/render-auditlogs/pkg/render"
)

var workspaceLogs = []render.AuditLogEntry{
	{
		Cursor: "cursor-1",
		AuditLog: render.AuditLog{
			ID:        "aud-1",
			Timestamp: time.Now(),
			Event:     "LoginEvent",
			Status:    "success",
			Actor: render.Actor{
				Type:  "user",
				Email: "test@example.com",
				ID:    "user-123",
			},
			Metadata: map[string]string{
				"login_method": "saml",
			},
		},
	},
	{
		Cursor: "cursor-2",
		AuditLog: render.AuditLog{
			ID:        "aud-2",
			Timestamp: time.Now(),
			Event:     "ViewEnvVarValuesEvent",
			Status:    "success",
			Actor: render.Actor{
				Type:  "user",
				Email: "test@example.com",
				ID:    "user-123",
			},
			Metadata: map[string]string{
				"environment_variable": "var",
				"service":              "crn-123",
			},
		},
	},
}

func TestClient_GetAuditLogs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		require.Equal(t, "50", r.URL.Query().Get("limit"))
		require.Equal(t, "forward", r.URL.Query().Get("direction"))

		if r.URL.Query().Get("cursor") == "cursor-2" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]render.AuditLogEntry{})
			return
		}

		if r.URL.Path == "/owners/workspace-123/audit-logs" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(workspaceLogs)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]render.AuditLogEntry{})
	}))
	defer server.Close()

	t.Run("successful request with audit logs", func(t *testing.T) {
		client := render.NewClient(server.URL, "test-api-key")

		logs, err := client.GetAuditLogs("/owners/workspace-123/audit-logs", "", 50)
		require.NoError(t, err)
		require.Len(t, logs, 2)
		require.Equal(t, "cursor-1", logs[0].Cursor)
		require.Equal(t, "LoginEvent", logs[0].AuditLog.Event)
	})

	t.Run("successful request with no logs", func(t *testing.T) {
		client := render.NewClient(server.URL, "test-api-key")

		logs, err := client.GetAuditLogs("/owners/workspace-123/audit-logs", "cursor-2", 50)
		require.NoError(t, err)
		require.Len(t, logs, 0)
	})

	t.Run("invalid api key", func(t *testing.T) {
		client := render.NewClient(server.URL, "test-api-key-invalid")

		logs, err := client.GetAuditLogs("/owners/workspace-123/audit-logs", "cursor-2", 50)
		require.Error(t, err)
		require.Len(t, logs, 0)
	})
}
