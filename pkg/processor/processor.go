package processor

import (
	"context"
	"fmt"
	"time"

	"github.com/renderinc/render-auditlogs/pkg/auditlogs"
	"github.com/renderinc/render-auditlogs/pkg/aws"
	"github.com/renderinc/render-auditlogs/pkg/logger"
	"github.com/renderinc/render-auditlogs/pkg/render"
)

const (
	pageSize int = 1000
)

type Uploader interface {
	LoadCheckpoint(ctx context.Context, logType auditlogs.LogType, id string) (*aws.Checkpoint, error)
	SaveCheckpoint(ctx context.Context, cp *aws.Checkpoint, logType auditlogs.LogType, id string) error
	UploadAuditLogs(ctx context.Context, logType auditlogs.LogType, id string, data []render.AuditLogEntry) (string, error)
}

type LogProcessor struct {
	uploader    Uploader
	auditLogSvc auditlogs.Service
}

func NewLogProcessor(uploader Uploader, auditLogSvc auditlogs.Service) *LogProcessor {
	return &LogProcessor{
		uploader:    uploader,
		auditLogSvc: auditLogSvc,
	}
}

func (lp *LogProcessor) Process(ctx context.Context, id string) error {
	l := logger.FromContext(ctx)

	cursor := ""

	checkpoint, err := lp.getLastCheckpoint(ctx, id)
	if err != nil {
		return err
	}

	if checkpoint != nil {
		cursor = checkpoint.LastCursor
	}

	var finalAuditLog *render.AuditLogEntry

	for {
		lastAuditLog, err := lp.processPage(ctx, id, cursor)
		if err != nil {
			return fmt.Errorf("error processing workspace page: %w", err)
		}

		if lastAuditLog == nil {
			break
		}

		cursor = lastAuditLog.Cursor
		finalAuditLog = lastAuditLog
	}

	l.Info("final cursor processed", "finalAuditLog", finalAuditLog)

	if finalAuditLog != nil {
		newCheckpoint := &aws.Checkpoint{
			LastCursor:    finalAuditLog.Cursor,
			LastTimestamp: finalAuditLog.AuditLog.Timestamp,
		}
		return lp.updateLastCheckpoint(ctx, id, newCheckpoint)
	}

	return nil
}

func (lp *LogProcessor) getLastCheckpoint(ctx context.Context, id string) (*aws.Checkpoint, error) {
	// Load checkpoint from S3
	checkpoint, err := lp.uploader.LoadCheckpoint(ctx, lp.auditLogSvc.Type(), id)
	if err != nil {
		return nil, err
	}

	return checkpoint, nil
}

func (lp *LogProcessor) updateLastCheckpoint(ctx context.Context, id string, cp *aws.Checkpoint) error {
	logger.FromContext(ctx).Info("updating checkpoint")
	if err := lp.uploader.SaveCheckpoint(ctx, cp, lp.auditLogSvc.Type(), id); err != nil {
		return fmt.Errorf("error saving checkpoint: %w", err)
	}
	return nil
}

func (lp *LogProcessor) processPage(ctx context.Context, id string, cursor string) (*render.AuditLogEntry, error) {
	l := logger.FromContext(ctx)

	// Fetch audit logs
	auditLogs, err := lp.auditLogSvc.Get(
		id,
		cursor,
		pageSize)
	if err != nil {
		return nil, fmt.Errorf("error fetching audit logs %w", err)
	}

	l.Info("found audit log entries", "count", len(auditLogs))

	if len(auditLogs) == 0 {
		return nil, nil
	}

	cursorDay := time.Date(
		auditLogs[0].AuditLog.Timestamp.Year(),
		auditLogs[0].AuditLog.Timestamp.Month(),
		auditLogs[0].AuditLog.Timestamp.Day(),
		0, 0, 0, 0, time.UTC,
	)

	windowStart := 0

	for i, auditLog := range auditLogs {
		if auditLog.AuditLog.Timestamp.After(cursorDay.Add(24 * time.Hour)) {
			l.Info("upload", "start", windowStart, "end", i)

			s3URI, err := lp.uploader.UploadAuditLogs(
				ctx,
				lp.auditLogSvc.Type(),
				id,
				auditLogs[windowStart:i],
			)
			if err != nil {
				l.Error("error uploading to S3", "error", err)
				return nil, err
			}
			l.Info("audit logs uploaded", "s3URI", s3URI)

			windowStart = i
			cursorDay = cursorDay.Add(time.Hour * 24)
		}
	}

	if len(auditLogs[windowStart:]) > 0 {
		l.Info("upload", "start", windowStart, "end", len(auditLogs))

		s3URI, err := lp.uploader.UploadAuditLogs(
			ctx,
			lp.auditLogSvc.Type(),
			id,
			auditLogs[windowStart:],
		)

		if err != nil {
			l.Error("error uploading to S3:", "error", err)
			return nil, err
		}
		l.Info("audit logs uploaded to", "s3URI", s3URI)

	}

	return &auditLogs[len(auditLogs)-1], nil
}
