package logger

import (
	"context"
	"time"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
	LogLevelFatal LogLevel = "FATAL"
)

type attributes = map[string]any

type LogEntry struct {
	Level      LogLevel
	Message    string
	Attributes attributes
	Error      error
	Timestamp  time.Time
}

type Logger interface {
	Log(ctx context.Context, entry LogEntry)
	Shutdown(ctx context.Context) error
}

var globalLogger Logger = &noopLogger{}

func newLogEntry(level LogLevel, message string, err error, attrs attributes) LogEntry {
	return LogEntry{
		Level:      level,
		Message:    message,
		Attributes: attrs,
		Error:      err,
		Timestamp:  time.Now(),
	}
}

func Debug(ctx context.Context, message string, attrs attributes) {
	globalLogger.Log(ctx, newLogEntry(LogLevelDebug, message, nil, attrs))
}

func Info(ctx context.Context, message string, attrs attributes) {
	globalLogger.Log(ctx, newLogEntry(LogLevelInfo, message, nil, attrs))
}

func Warn(ctx context.Context, message string, attrs attributes) {
	globalLogger.Log(ctx, newLogEntry(LogLevelWarn, message, nil, attrs))
}

func Error(ctx context.Context, message string, err error, attrs attributes) {
	globalLogger.Log(ctx, newLogEntry(LogLevelError, message, err, attrs))
}

func Fatal(ctx context.Context, message string, err error, attrs attributes) {
	globalLogger.Log(ctx, newLogEntry(LogLevelFatal, message, err, attrs))
}

func Log(ctx context.Context, entry LogEntry) {
	globalLogger.Log(ctx, entry)
}

func Shutdown(ctx context.Context) error {
	return globalLogger.Shutdown(ctx)
}

func Initialize(collectorEndpoint, serviceName string, isProduction bool) error {
	var (
		l   Logger
		err error
	)

	if isProduction {
		l, err = initializeOtelLogger(collectorEndpoint, serviceName)
	} else {
		l, err = initStdoutLogger(serviceName)
	}

	if err != nil {
		return err
	}

	globalLogger = l
	return nil
}
