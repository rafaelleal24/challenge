package logger

import "context"

type noopLogger struct{}

func (n *noopLogger) Log(context.Context, LogEntry)  {}
func (n *noopLogger) Shutdown(context.Context) error { return nil }
