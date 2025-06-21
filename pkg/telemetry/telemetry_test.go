package telemetry

import (
	"context"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if cfg.ServiceName != "scaled-mcp-server" {
		t.Errorf("Expected service name 'scaled-mcp-server', got %s", cfg.ServiceName)
	}

	if cfg.ExporterType != "stdout" {
		t.Errorf("Expected exporter type 'stdout', got %s", cfg.ExporterType)
	}

	if !cfg.Enabled {
		t.Error("Expected telemetry to be enabled by default")
	}
}

func TestInitializeTracingDisabled(t *testing.T) {
	cfg := &Config{
		Enabled: false,
	}

	ctx := context.Background()
	shutdown, err := InitializeTracing(ctx, cfg)
	if err != nil {
		t.Fatalf("InitializeTracing failed: %v", err)
	}

	if shutdown == nil {
		t.Fatal("Expected shutdown function, got nil")
	}

	err = shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestInitializeTracingStdout(t *testing.T) {
	cfg := &Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		ExporterType:   "stdout",
		Enabled:        true,
	}

	ctx := context.Background()
	shutdown, err := InitializeTracing(ctx, cfg)
	if err != nil {
		t.Fatalf("InitializeTracing failed: %v", err)
	}

	if shutdown == nil {
		t.Fatal("Expected shutdown function, got nil")
	}

	tracer := GetTracer("test")
	if tracer == nil {
		t.Fatal("Expected tracer, got nil")
	}

	_, span := tracer.Start(ctx, "test-span")
	span.End()

	err = shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestOpenTelemetryTraceHandler(t *testing.T) {
	handler := NewOpenTelemetryTraceHandler()
	if handler == nil {
		t.Fatal("NewOpenTelemetryTraceHandler returned nil")
	}

	ctx := context.Background()
	traceId := "test-trace-id"

	newCtx := handler.SetTraceId(ctx, traceId)
	if newCtx == nil {
		t.Fatal("SetTraceId returned nil context")
	}
}

func TestOpenTelemetryTraceHandlerWithUtils(t *testing.T) {
	handler := NewOpenTelemetryTraceHandlerWithUtils()
	if handler == nil {
		t.Fatal("NewOpenTelemetryTraceHandlerWithUtils returned nil")
	}

	ctx := context.Background()
	traceId := "test-trace-id"

	newCtx := handler.SetTraceId(ctx, traceId)
	if newCtx == nil {
		t.Fatal("SetTraceId returned nil context")
	}
}

func TestGetTracer(t *testing.T) {
	cfg := &Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		ExporterType:   "stdout",
		Enabled:        true,
	}
	
	ctx := context.Background()
	shutdown, err := InitializeTracing(ctx, cfg)
	if err != nil {
		t.Fatalf("InitializeTracing failed: %v", err)
	}
	defer shutdown(ctx)
	
	tracer := GetTracer("test-tracer")
	if tracer == nil {
		t.Fatal("GetTracer returned nil")
	}
	
	_, span := tracer.Start(ctx, "test-operation")
	if span == nil {
		t.Fatal("Start returned nil span")
	}
	
	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		t.Error("Expected valid span context")
	}
	
	if spanCtx.TraceID().String() == "00000000000000000000000000000000" {
		t.Error("Expected non-zero trace ID")
	}
	
	span.End()
}
