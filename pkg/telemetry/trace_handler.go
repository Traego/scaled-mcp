package telemetry

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/traego/scaled-mcp/pkg/utils"
)

type OpenTelemetryTraceHandlerWithUtils struct {
	propagator propagation.TextMapPropagator
}

func NewOpenTelemetryTraceHandlerWithUtils() *OpenTelemetryTraceHandlerWithUtils {
	return &OpenTelemetryTraceHandlerWithUtils{
		propagator: propagation.TraceContext{},
	}
}

func (h *OpenTelemetryTraceHandlerWithUtils) ExtractTraceId(r *http.Request) string {
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

func (h *OpenTelemetryTraceHandlerWithUtils) SetTraceId(ctx context.Context, traceId string) context.Context {
	return utils.SetTraceId(ctx, traceId)
}
