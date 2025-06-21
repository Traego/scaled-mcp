package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	ServiceName    string
	ServiceVersion string
	ExporterType   string
	OTLPEndpoint   string
	Enabled        bool
}

func DefaultConfig() *Config {
	return &Config{
		ServiceName:    getEnvOrDefault("OTEL_SERVICE_NAME", "scaled-mcp-server"),
		ServiceVersion: getEnvOrDefault("OTEL_SERVICE_VERSION", "1.0.0"),
		ExporterType:   getEnvOrDefault("OTEL_EXPORTER_TYPE", "stdout"),
		OTLPEndpoint:   getEnvOrDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318"),
		Enabled:        getEnvOrDefault("OTEL_ENABLED", "true") == "true",
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func InitializeTracing(ctx context.Context, cfg *Config) (func(context.Context) error, error) {
	if !cfg.Enabled {
		slog.Info("OpenTelemetry tracing disabled")
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	var exporter sdktrace.SpanExporter
	switch cfg.ExporterType {
	case "otlp":
		exporter, err = otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
			otlptracehttp.WithInsecure(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}
	case "stdout":
		exporter, err = stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout exporter: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported exporter type: %s", cfg.ExporterType)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	slog.Info("OpenTelemetry tracing initialized", "service", cfg.ServiceName, "exporter", cfg.ExporterType)

	return tp.Shutdown, nil
}

func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

type TraceHandler interface {
	ExtractTraceId(r *http.Request) string
	SetTraceId(ctx context.Context, traceId string) context.Context
}

type OpenTelemetryTraceHandler struct {
	propagator propagation.TextMapPropagator
}

func NewOpenTelemetryTraceHandler() *OpenTelemetryTraceHandler {
	return &OpenTelemetryTraceHandler{
		propagator: propagation.TraceContext{},
	}
}

func (h *OpenTelemetryTraceHandler) ExtractTraceId(r *http.Request) string {
	ctx := h.propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		return spanCtx.TraceID().String()
	}

	traceIdHeader := r.Header.Get("X-Trace-Id")
	if traceIdHeader != "" {
		return traceIdHeader
	}

	return ""
}

func (h *OpenTelemetryTraceHandler) SetTraceId(ctx context.Context, traceId string) context.Context {
	return ctx
}
