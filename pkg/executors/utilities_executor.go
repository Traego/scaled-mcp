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
	// Use the utility function to process the request - utilities don't require any specific registry
	response, _, err := ProcessRequest(method, req, true)
	if err != nil {
		return nil, err
	}

	var result interface{}

	// Handle the method
	switch method {
	case "ping":
		result, err = u.handlePing(ctx)
	default:
		return nil, protocol.NewMethodNotFoundError(method, req.Id)
	}

	if err != nil {
		return nil, fmt.Errorf("error handling %s: %w", method, err)
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
