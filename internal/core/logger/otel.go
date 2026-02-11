package logger

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type OTELLogger struct {
	logger   otellog.Logger
	provider *sdklog.LoggerProvider
}

func initializeOtelLogger(collectorEndpoint, serviceName string) (Logger, error) {
	ctx := context.Background()

	conn, err := grpc.NewClient(
		collectorEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	logExporter, err := otlploggrpc.New(ctx, otlploggrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create log exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	processor := sdklog.NewBatchProcessor(logExporter)
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(processor),
		sdklog.WithResource(res),
	)

	global.SetLoggerProvider(provider)

	return &OTELLogger{
		logger:   provider.Logger(serviceName),
		provider: provider,
	}, nil
}

func (l *OTELLogger) Log(ctx context.Context, entry LogEntry) {
	var logRecord otellog.Record
	logRecord.SetTimestamp(entry.Timestamp)
	logRecord.SetBody(otellog.StringValue(entry.Message))
	logRecord.SetSeverityText(string(entry.Level))

	switch entry.Level {
	case LogLevelDebug:
		logRecord.SetSeverity(otellog.SeverityDebug)
	case LogLevelInfo:
		logRecord.SetSeverity(otellog.SeverityInfo)
	case LogLevelWarn:
		logRecord.SetSeverity(otellog.SeverityWarn)
	case LogLevelError:
		logRecord.SetSeverity(otellog.SeverityError)
	case LogLevelFatal:
		logRecord.SetSeverity(otellog.SeverityFatal)
	}

	attrs := make([]otellog.KeyValue, 0, len(entry.Attributes))
	for key, value := range entry.Attributes {
		var kv otellog.KeyValue
		switch v := value.(type) {
		case string:
			kv = otellog.String(key, v)
		case int:
			kv = otellog.Int(key, v)
		case int64:
			kv = otellog.Int64(key, v)
		case float64:
			kv = otellog.Float64(key, v)
		case bool:
			kv = otellog.Bool(key, v)
		default:
			kv = otellog.String(key, fmt.Sprintf("%v", v))
		}
		attrs = append(attrs, kv)
	}

	if entry.Error != nil {
		attrs = append(attrs, otellog.String("error", entry.Error.Error()))
	}

	logRecord.AddAttributes(attrs...)
	l.logger.Emit(ctx, logRecord)
}

func (l *OTELLogger) Shutdown(ctx context.Context) error {
	return l.provider.Shutdown(ctx)
}
