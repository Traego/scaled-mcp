package utils

import "context"

func SetTraceId(ctx context.Context, traceId string) context.Context {
	return context.WithValue(ctx, TraceIdCtx, traceId)
}

func GetTraceId(ctx context.Context) string {
	if traceId, ok := ctx.Value(TraceIdCtx).(string); ok {
		return traceId
	}
	return ""
}
