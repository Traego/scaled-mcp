package executors

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/traego/scaled-mcp/scaled-mcp-server/pkg/config"
	"github.com/traego/scaled-mcp/scaled-mcp-server/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/scaled-mcp-server/pkg/protocol"
	"github.com/traego/scaled-mcp/scaled-mcp-server/pkg/resources"
	"log/slog"
)

// PromptExecutor handles prompt-related methods in the MCP protocol
type PromptExecutor struct {
	serverInfo config.McpServerInfo
}

// NewPromptExecutor creates a new prompt executor
func NewPromptExecutor(serverInfo config.McpServerInfo) *PromptExecutor {
	return &PromptExecutor{serverInfo: serverInfo}
}

// CanHandleMethod checks if the method is related to prompts
func (p *PromptExecutor) CanHandleMethod(method string) bool {
	switch method {
	case "prompts/list", "prompts/get":
		return true
	default:
		return false
	}
}

// HandleMethod handles prompt-related methods
func (p *PromptExecutor) HandleMethod(ctx context.Context, method string, req *mcppb.JsonRpcRequest) (*mcppb.JsonRpcResponse, error) {
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

	// Check if prompt registry is available
	if p.serverInfo.GetFeatureRegistry().PromptRegistry == nil {
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
	case "prompts/list":
		result, err = p.handleListPrompts(ctx, params)
	case "prompts/get":
		result, err = p.handleGetPrompt(ctx, params)
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

// handleListPrompts handles a request to list prompts
func (p *PromptExecutor) handleListPrompts(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	var cursor string

	// Extract cursor
	if cursorVal, ok := params["cursor"]; ok {
		if cursorStr, ok := cursorVal.(string); ok {
			cursor = cursorStr
		}
	}

	// Create options
	opts := resources.PromptListOptions{
		Cursor: cursor,
	}

	// Call the registry
	result := p.serverInfo.GetFeatureRegistry().PromptRegistry.ListPrompts(ctx, opts)

	return result, nil
}

// handleGetPrompt handles a request to get a specific prompt
func (p *PromptExecutor) handleGetPrompt(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Extract name
	nameVal, ok := params["name"]
	if !ok {
		return nil, fmt.Errorf("%w: prompt name is required", resources.ErrInvalidParams)
	}

	name, ok := nameVal.(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("%w: prompt name must be a non-empty string", resources.ErrInvalidParams)
	}

	// Extract arguments if provided
	var arguments map[string]string
	if argsVal, ok := params["arguments"]; ok {
		if argsMap, ok := argsVal.(map[string]interface{}); ok {
			arguments = make(map[string]string)
			for k, v := range argsMap {
				if strVal, ok := v.(string); ok {
					arguments[k] = strVal
				} else {
					slog.Warn("Non-string argument value ignored", "key", k, "value", v)
				}
			}
		}
	}

	// Get the prompt
	prompt, found := p.serverInfo.GetFeatureRegistry().PromptRegistry.GetPrompt(ctx, name)
	if !found {
		return nil, fmt.Errorf("%w: prompt '%s' not found", resources.ErrPromptNotFound, name)
	}

	// If arguments were provided, process the prompt template
	if len(arguments) > 0 {
		messages, err := p.serverInfo.GetFeatureRegistry().PromptRegistry.ProcessPrompt(ctx, name, arguments)
		if err != nil {
			return nil, fmt.Errorf("error processing prompt template: %w", err)
		}

		return map[string]interface{}{
			"messages":    messages,
			"description": prompt.Description,
		}, nil
	}

	// Return the prompt definition
	return map[string]interface{}{
		"messages":    prompt.Messages,
		"description": prompt.Description,
	}, nil
}

// Ensure PromptExecutor implements config.MethodHandler
var _ config.MethodHandler = (*PromptExecutor)(nil)
