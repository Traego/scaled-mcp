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
	// Use the utility function to process the request
	response, params, err := ProcessRequest(method, req, t.serverInfo.GetFeatureRegistry().ToolRegistry != nil)
	if err != nil {
		return nil, err
	}

	var result interface{}

	// Handle the method
	switch method {
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

// handleListTools handles a request to list tools
func (t *ToolExecutor) handleListTools(ctx context.Context, params map[string]interface{}) (protocol.ToolListResult, error) {
	var cursor string

	// Extract cursor
	if cursorVal, ok := params["cursor"]; ok {
		if cursorStr, ok := cursorVal.(string); ok {
			cursor = cursorStr
		}
	}

	// Create options
	opts := protocol.ToolListOptions{
		Cursor: cursor,
	}

	results, err := t.serverInfo.GetFeatureRegistry().ToolRegistry.ListTools(ctx, opts)
	if err != nil {
		return protocol.ToolListResult{}, fmt.Errorf("error listing tools: %w", err)
	}

	return results, nil
}

// handleGetTool handles a request to get a specific tool
func (t *ToolExecutor) handleGetTool(ctx context.Context, params map[string]interface{}) (protocol.Tool, error) {
	// Extract name
	nameVal, ok := params["name"]
	if !ok {
		return protocol.Tool{}, fmt.Errorf("%w: tool name is required", resources.ErrInvalidParams)
	}

	name, ok := nameVal.(string)
	if !ok || name == "" {
		return protocol.Tool{}, fmt.Errorf("%w: tool name must be a non-empty string", resources.ErrInvalidParams)
	}

	// Get the tool
	tool, err := t.serverInfo.GetFeatureRegistry().ToolRegistry.GetTool(ctx, name)
	if err != nil {
		return protocol.Tool{}, fmt.Errorf("error getting tool %s: %w", name, err)
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
	results, err := t.serverInfo.GetFeatureRegistry().ToolRegistry.CallTool(ctx, name, toolArgs)
	if err != nil {
		// Create an error result
		errorContent := protocol.NewTextContent(fmt.Sprintf("Error calling %s: %v", name, err))
		return protocol.NewToolCallResult([]protocol.ToolCallContent{errorContent}, true), nil
	}

	// Convert the results to a ToolCallResult
	// If results is already a ToolCallResult, return it directly
	if toolCallResult, ok := results.(protocol.ToolCallResult); ok {
		return toolCallResult, nil
	}

	// If results is a string, create a text content item
	if strResult, ok := results.(string); ok {
		textContent := protocol.NewTextContent(strResult)
		return protocol.NewToolCallResult([]protocol.ToolCallContent{textContent}, false), nil
	}

	// For other result types, convert to JSON and create a text content item
	resultJSON, err := json.Marshal(results)
	if err != nil {
		errorContent := protocol.NewTextContent(fmt.Sprintf("Error marshaling result: %v", err))
		return protocol.NewToolCallResult([]protocol.ToolCallContent{errorContent}, true), nil
	}

	textContent := protocol.NewTextContent(string(resultJSON))
	return protocol.NewToolCallResult([]protocol.ToolCallContent{textContent}, false), nil
}

var _ config.MethodHandler = (*ToolExecutor)(nil)
