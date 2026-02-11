package logger

import (
	"context"
	"log/slog"
	"os"
	"time"
)

type StdoutLogger struct {
	logger *slog.Logger
}

func initStdoutLogger(serviceName string) (Logger, error) {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})

	handlerWithAttrs := handler.WithAttrs([]slog.Attr{
		slog.String("service", serviceName),
	})

	return &StdoutLogger{
		logger: slog.New(handlerWithAttrs),
	}, nil
}

func (l *StdoutLogger) Log(ctx context.Context, entry LogEntry) {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	attrs := make([]any, 0, len(entry.Attributes)*2+2)
	// we skip this because it's not needed for stdout, the log become too verbose with it
	/*
			for key, value := range entry.Attributes {
				attrs = append(attrs, key, value)
			}


		if entry.Error != nil {
			attrs = append(attrs, "error", entry.Error.Error())
		}
	*/
	switch entry.Level {
	case LogLevelDebug:
		l.logger.DebugContext(ctx, entry.Message, attrs...)
	case LogLevelInfo:
		l.logger.InfoContext(ctx, entry.Message, attrs...)
	case LogLevelWarn:
		l.logger.WarnContext(ctx, entry.Message, attrs...)
	case LogLevelError:
		l.logger.ErrorContext(ctx, entry.Message, attrs...)
	case LogLevelFatal:
		l.logger.ErrorContext(ctx, entry.Message, attrs...)
		os.Exit(1)
	}
}

func (l *StdoutLogger) Shutdown(context.Context) error {
	return nil
}
