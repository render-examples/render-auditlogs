package processor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/renderinc/render-auditlogs/pkg/auditlogs"
	"github.com/renderinc/render-auditlogs/pkg/aws"
	"github.com/renderinc/render-auditlogs/pkg/processor"
	"github.com/renderinc/render-auditlogs/pkg/render"
	"github.com/renderinc/render-auditlogs/pkg/testhelpers"
)

type mockUploader struct {
	lastCheckpoint *aws.Checkpoint
	s3Error        error
	numUploads     int
}

func (m *mockUploader) LoadCheckpoint(ctx context.Context, logType auditlogs.LogType, id string) (*aws.Checkpoint, error) {
	return m.lastCheckpoint, m.s3Error
}

func (m *mockUploader) SaveCheckpoint(ctx context.Context, cp *aws.Checkpoint, logType auditlogs.LogType, id string) error {
	m.lastCheckpoint = cp
	return m.s3Error
}

func (m *mockUploader) UploadAuditLogs(ctx context.Context, logType auditlogs.LogType, id string, data []render.AuditLogEntry) (string, error) {
	if m.s3Error != nil {
		return "", m.s3Error
	}

	m.numUploads++
	return "s3://bucket/key", nil
}

type mockAuditLogService struct {
	auditLogs   []render.AuditLogEntry
	logType     auditlogs.LogType
	renderError error
}

func (m *mockAuditLogService) Get(id string, cursor string, limit int) ([]render.AuditLogEntry, error) {
	if m.renderError != nil {
		return nil, m.renderError
	}

	finalIndex := len(m.auditLogs)

	// returns cursor onwards with a max of 1000
	for i, auditLog := range m.auditLogs {
		if auditLog.Cursor == cursor {
			return m.auditLogs[i+1 : min(i+1001, finalIndex)], nil
		}
	}

	return m.auditLogs[0:min(finalIndex, 1000)], nil
}

func (m *mockAuditLogService) Type() auditlogs.LogType {
	return m.logType
}

func today() time.Time {
	return time.Date(
		time.Now().Year(),
		time.Now().Month(),
		time.Now().Day(),
		0, 0, 0, 0, time.UTC)
}

func TestProcess(t *testing.T) {
	t.Parallel()
	t.Run("EmptyLogs", func(t *testing.T) {

		ctx := t.Context()
		uploader := &mockUploader{
			lastCheckpoint: &aws.Checkpoint{LastCursor: "cursor-123"},
		}
		service := &mockAuditLogService{
			auditLogs: testhelpers.CreateTestAuditLogs(0, today()),
		}

		lp := processor.NewLogProcessor(uploader, service)

		err := lp.Process(ctx, "workspace-123")
		require.NoError(t, err)
		require.Zero(t, uploader.numUploads)
	})

	t.Run("LogsWithinSameDay", func(t *testing.T) {
		uploader := &mockUploader{
			lastCheckpoint: &aws.Checkpoint{LastCursor: "0"},
		}

		logs := testhelpers.CreateTestAuditLogs(3, today())

		service := &mockAuditLogService{
			auditLogs: logs,
			logType:   auditlogs.WorkspaceAuditLog,
		}

		lp := processor.NewLogProcessor(uploader, service)
		ctx := t.Context()

		err := lp.Process(ctx, "workspace-123")
		require.NoError(t, err)

		require.Equal(t, 1, uploader.numUploads)
		require.Equal(t, logs[2].Cursor, uploader.lastCheckpoint.LastCursor)
	})

	t.Run("MultiplePages", func(t *testing.T) {
		uploader := &mockUploader{
			lastCheckpoint: &aws.Checkpoint{LastCursor: "0"},
		}

		logs := testhelpers.CreateTestAuditLogs(1005, today())

		service := &mockAuditLogService{
			auditLogs: logs,
			logType:   auditlogs.WorkspaceAuditLog,
		}

		lp := processor.NewLogProcessor(uploader, service)
		ctx := t.Context()

		err := lp.Process(ctx, "workspace-123")
		require.NoError(t, err)

		require.Equal(t, 2, uploader.numUploads)
		require.Equal(t, logs[1004].Cursor, uploader.lastCheckpoint.LastCursor)
	})

	t.Run("MultipleDays", func(t *testing.T) {
		uploader := &mockUploader{
			lastCheckpoint: &aws.Checkpoint{LastCursor: "0"},
		}

		logs := append(
			testhelpers.CreateTestAuditLogs(3, today().AddDate(0, 0, -1)),
			testhelpers.CreateTestAuditLogs(3, today())...,
		)

		service := &mockAuditLogService{
			auditLogs: logs,
			logType:   auditlogs.WorkspaceAuditLog,
		}

		lp := processor.NewLogProcessor(uploader, service)
		ctx := t.Context()

		err := lp.Process(ctx, "workspace-123")
		require.NoError(t, err)

		require.Equal(t, 2, uploader.numUploads)
		require.Equal(t, logs[5].Cursor, uploader.lastCheckpoint.LastCursor)
	})

	t.Run("ErrorFetchingAuditLogs", func(t *testing.T) {
		uploader := &mockUploader{
			lastCheckpoint: &aws.Checkpoint{LastCursor: "0"},
		}

		logs := testhelpers.CreateTestAuditLogs(3, today())

		service := &mockAuditLogService{
			auditLogs:   logs,
			logType:     auditlogs.WorkspaceAuditLog,
			renderError: errors.New("cannot get logs"),
		}

		lp := processor.NewLogProcessor(uploader, service)
		ctx := t.Context()

		err := lp.Process(ctx, "workspace-123")
		require.Error(t, err)

		require.Equal(t, 0, uploader.numUploads)
		require.Equal(t, "0", uploader.lastCheckpoint.LastCursor)
	})

	t.Run("ErrorUploading", func(t *testing.T) {
		uploader := &mockUploader{
			lastCheckpoint: &aws.Checkpoint{LastCursor: "0"},

			s3Error: errors.New("cannot access s3"),
		}

		logs := testhelpers.CreateTestAuditLogs(3, today())

		service := &mockAuditLogService{
			auditLogs: logs,
			logType:   auditlogs.WorkspaceAuditLog,
		}

		lp := processor.NewLogProcessor(uploader, service)
		ctx := t.Context()

		err := lp.Process(ctx, "workspace-123")
		require.Error(t, err)

	})

	t.Run("NoCheckpoint", func(t *testing.T) {
		uploader := &mockUploader{
			lastCheckpoint: nil,
		}

		logs := testhelpers.CreateTestAuditLogs(3, today())

		service := &mockAuditLogService{
			auditLogs: logs,
			logType:   auditlogs.WorkspaceAuditLog,
		}

		lp := processor.NewLogProcessor(uploader, service)
		ctx := t.Context()

		err := lp.Process(ctx, "workspace-123")
		require.NoError(t, err)

		require.Equal(t, 1, uploader.numUploads)
		require.Equal(t, logs[2].Cursor, uploader.lastCheckpoint.LastCursor)
	})
}
