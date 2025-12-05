package logger

import (
	"context"
	"log/slog"
	"os"
)

type contextKey string

const loggerKey contextKey = "logger"

func New(ctx context.Context) (context.Context, *slog.Logger) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	return withLogger(ctx, l), l
}

func withLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves the logger from the context.
// If no logger is found, it returns the default logger.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// With returns a new context with the logger updated with the given attributes
func With(ctx context.Context, args ...any) (context.Context, *slog.Logger) {
	l := FromContext(ctx).With(args...)

	return withLogger(ctx, l), l
}
