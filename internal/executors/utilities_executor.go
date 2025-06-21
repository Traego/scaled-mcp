package executors

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// UtilitiesExecutor handles utility methods in the MCP protocol
type UtilitiesExecutor struct {
	serverInfo config.McpServerInfo
}

// NewUtilitiesExecutor creates a new utilities executor
func NewUtilitiesExecutor(serverInfo config.McpServerInfo) *UtilitiesExecutor {
	return &UtilitiesExecutor{serverInfo: serverInfo}
}

// CanHandleMethod checks if the method is related to utilities
func (u *UtilitiesExecutor) CanHandleMethod(method string) bool {
	switch method {
	case "ping":
		return true
	default:
		return false
	}
}

// HandleMethod handles utility-related methods
func (u *UtilitiesExecutor) HandleMethod(ctx context.Context, method string, req *mcppb.JsonRpcRequest) (*mcppb.JsonRpcResponse, error) {
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
	
	ctx, span := tracer.Start(ctx, "UtilitiesExecutor.HandleMethod",
		trace.WithAttributes(
			attribute.String("mcp.method", method),
			attribute.String("mcp.request_id", reqIdStr),
		),
	)
	defer span.End()

	// Use the utility function to process the request - utilities don't require any specific registry
	response, _, err := ProcessRequest(method, req, true)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to process request")
		return nil, err
	}

	var result interface{}

	// Handle the method
	switch method {
	case "ping":
		span.SetAttributes(attribute.String("mcp.utility_operation", "ping"))
		result, err = u.handlePing(ctx)
	default:
		span.SetStatus(codes.Error, "Method not found")
		return nil, protocol.NewMethodNotFoundError(method, req.Id)
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Utility operation failed")
		return nil, fmt.Errorf("error handling %s: %w", method, err)
	}

	// Marshal the result
	resultJSON, err := json.Marshal(result)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to marshal result")
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	response.Response = &mcppb.JsonRpcResponse_ResultJson{
		ResultJson: string(resultJSON),
	}

	return response, nil
}

// handlePing handles a ping request
func (u *UtilitiesExecutor) handlePing(ctx context.Context) (interface{}, error) {
	// Return an empty object as per the protocol specification
	return map[string]interface{}{}, nil
}

// Ensure UtilitiesExecutor implements config.MethodHandler
var _ config.MethodHandler = (*UtilitiesExecutor)(nil)
