package executors

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/resources"
)

type ToolExecutor struct {
	serverInfo config.McpServerInfo
}

func NewToolExecutor(serverInfo config.McpServerInfo) *ToolExecutor {
	return &ToolExecutor{serverInfo: serverInfo}
}

// isToolMethod checks if the method is related to tools
func (t *ToolExecutor) CanHandleMethod(method string) bool {
	switch method {
	case "tools/list", "tools/get", "tools/call":
		return true
	default:
		return false
	}
}

// handleToolMethod handles tool-related methods
func (t *ToolExecutor) HandleMethod(ctx context.Context, method string, req *mcppb.JsonRpcRequest) (*mcppb.JsonRpcResponse, error) {
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

	// Check if tool registry is available
	if t.serverInfo.GetFeatureRegistry().ToolRegistry == nil {
		return nil, protocol.NewMethodNotFoundError(req.Method, req.Id)
	}

	// Parse the params
	var params map[string]interface{}
	if req.ParamsJson != "" {
		if err := json.Unmarshal([]byte(req.ParamsJson), &params); err != nil {
			return nil, protocol.NewInvalidParamsError("Invalid parameters: "+err.Error(), req.Id)
		}
	} else {
		params = make(map[string]interface{})
	}

	var result interface{}
	var err error

	switch req.Method {
	case "tools/list":
		result, err = t.handleListTools(ctx, params)
	case "tools/get":
		result, err = t.handleGetTool(ctx, params)
	case "tools/call":
		result, err = t.handleCallTool(ctx, params)
	default:
		return nil, protocol.NewMethodNotFoundError(req.Method, req.Id)
	}

	if err != nil {
		return nil, fmt.Errorf("error handling %s: %w"+req.Method, err)
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

// handleListTools handles a request to list tools
func (t *ToolExecutor) handleListTools(ctx context.Context, params map[string]interface{}) (resources.ToolListResult, error) {
	var cursor string

	// Extract cursor
	if cursorVal, ok := params["cursor"]; ok {
		if cursorStr, ok := cursorVal.(string); ok {
			cursor = cursorStr
		}
	}

	// Create options
	opts := resources.ToolListOptions{
		Cursor: cursor,
	}

	results := t.serverInfo.GetFeatureRegistry().ToolRegistry.ListTools(ctx, opts)
	// Call the registry
	return results, nil
}

// handleGetTool handles a request to get a specific tool
func (t *ToolExecutor) handleGetTool(ctx context.Context, params map[string]interface{}) (resources.Tool, error) {
	// Extract name
	nameVal, ok := params["name"]
	if !ok {
		return resources.Tool{}, fmt.Errorf("%w: tool name is required", resources.ErrInvalidParams)
	}

	name, ok := nameVal.(string)
	if !ok || name == "" {
		return resources.Tool{}, fmt.Errorf("%w: tool name must be a non-empty string", resources.ErrInvalidParams)
	}

	// Get the tool
	tool, found := t.serverInfo.GetFeatureRegistry().ToolRegistry.GetTool(ctx, name)
	if !found {
		return resources.Tool{}, fmt.Errorf("%w: tool '%s' not found", resources.ErrToolNotFound, name)
	}

	return tool, nil
}

// handleCallTool handles a request to invoke a tool
func (t *ToolExecutor) handleCallTool(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Extract name
	nameVal, ok := params["name"]
	if !ok {
		return nil, fmt.Errorf("%w: tool name is required", resources.ErrInvalidParams)
	}

	name, ok := nameVal.(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("%w: tool name must be a non-empty string", resources.ErrInvalidParams)
	}

	// Extract parameters
	var toolArgs map[string]interface{}

	if args, ok := params["arguments"]; ok {
		if p, ok := args.(map[string]interface{}); ok {
			toolArgs = p
		} else {
			return nil, fmt.Errorf("%w: parameters must be an object", resources.ErrInvalidParams)
		}
	} else {
		toolArgs = make(map[string]interface{})
	}

	// Invoke the tool
	return t.serverInfo.GetFeatureRegistry().ToolRegistry.CallTool(ctx, name, toolArgs)
}

var _ config.MethodHandler = (*ToolExecutor)(nil)
