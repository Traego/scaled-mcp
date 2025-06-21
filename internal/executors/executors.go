package executors

import (
	"context"
	"fmt"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/telemetry"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TODO This actually wants to be pluggable, this is where we'd plug in new fancy stuff

type Executors struct {
	Tools        config.MethodHandler
	Prompts      config.MethodHandler
	Resources    config.MethodHandler
	Utilities    config.MethodHandler
	Experimental config.MethodHandler
}

func DefaultExecutors(serverInfo config.McpServerInfo, experimental config.MethodHandler) *Executors {
	return &Executors{
		Tools:        NewToolExecutor(serverInfo),
		Prompts:      NewPromptExecutor(serverInfo),
		Resources:    NewResourceExecutor(serverInfo),
		Utilities:    NewUtilitiesExecutor(serverInfo),
		Experimental: experimental,
	}
}

func (e *Executors) CanHandleMethod(method string) bool {
	if e.Tools != nil && e.Tools.CanHandleMethod(method) {
		return true
	} else if e.Prompts != nil && e.Prompts.CanHandleMethod(method) {
		return true
	} else if e.Resources != nil && e.Resources.CanHandleMethod(method) {
		return true
	} else if e.Utilities != nil && e.Utilities.CanHandleMethod(method) {
		return true
	} else if e.Experimental != nil && e.Experimental.CanHandleMethod(method) {
		return true
	}
	return false
}

func (e *Executors) HandleMethod(ctx context.Context, method string, req *mcppb.JsonRpcRequest) (*mcppb.JsonRpcResponse, error) {
	tracer := telemetry.GetTracer("scaled-mcp-server/executors")
	var reqIdStr string
	if req.Id != nil {
		switch id := req.Id.(type) {
		case *mcppb.JsonRpcRequest_StringId:
			reqIdStr = id.StringId
		case *mcppb.JsonRpcRequest_IntId:
			reqIdStr = fmt.Sprintf("%d", id.IntId)
		default:
			reqIdStr = "unknown"
		}
	}
	
	ctx, span := tracer.Start(ctx, "Executors.HandleMethod",
		trace.WithAttributes(
			attribute.String("mcp.method", method),
			attribute.String("mcp.request_id", reqIdStr),
		),
	)
	defer span.End()

	ms := strings.Split(method, "/")
	if len(ms) >= 2 {
		switch ms[0] {
		case "tools":
			if e.Tools.CanHandleMethod(method) {
				span.SetAttributes(attribute.String("mcp.executor_type", "tools"))
				return e.Tools.HandleMethod(ctx, method, req)
			}
		case "resources":
			if e.Resources.CanHandleMethod(method) {
				span.SetAttributes(attribute.String("mcp.executor_type", "resources"))
				return e.Resources.HandleMethod(ctx, method, req)
			}
		case "prompts":
			if e.Prompts.CanHandleMethod(method) {
				span.SetAttributes(attribute.String("mcp.executor_type", "prompts"))
				return e.Prompts.HandleMethod(ctx, method, req)
			}
		}
	}

	if e.Utilities != nil && e.Utilities.CanHandleMethod(method) {
		span.SetAttributes(attribute.String("mcp.executor_type", "utilities"))
		return e.Utilities.HandleMethod(ctx, method, req)
	}

	if e.Experimental.CanHandleMethod(method) {
		span.SetAttributes(attribute.String("mcp.executor_type", "experimental"))
		return e.Experimental.HandleMethod(ctx, method, req)
	}

	span.SetStatus(codes.Error, "Method not found")
	return nil, protocol.NewMethodNotFoundError(method, req.Id)
}
