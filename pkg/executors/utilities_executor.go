package executors

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
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
	// Create base response
	response := &mcppb.JsonRpcResponse{
		Jsonrpc: "2.0",
	}

	// Copy the ID from the request
	switch id := req.Id.(type) {
	case *mcppb.JsonRpcRequest_IntId:
		response.Id = &mcppb.JsonRpcResponse_IntId{IntId: id.IntId}
	case *mcppb.JsonRpcRequest_StringId:
		response.Id = &mcppb.JsonRpcResponse_StringId{StringId: id.StringId}
	}

	var result interface{}
	var err error

	switch req.Method {
	case "ping":
		result, err = u.handlePing(ctx)
	default:
		return nil, protocol.NewMethodNotFoundError(req.Method, req.Id)
	}

	if err != nil {
		return nil, fmt.Errorf("error handling %s: %w", req.Method, err)
	}

	// Marshal the result
	resultJSON, err := json.Marshal(result)
	if err != nil {
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
