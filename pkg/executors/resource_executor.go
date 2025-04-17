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

// ResourceExecutor handles resource-related methods in the MCP protocol
type ResourceExecutor struct {
	serverInfo config.McpServerInfo
}

// NewResourceExecutor creates a new resource executor
func NewResourceExecutor(serverInfo config.McpServerInfo) *ResourceExecutor {
	return &ResourceExecutor{serverInfo: serverInfo}
}

// CanHandleMethod checks if the method is related to resources
func (r *ResourceExecutor) CanHandleMethod(method string) bool {
	switch method {
	case "resources/list", "resources/read", "resources/subscribe", "resources/unsubscribe", "resources/templates/list":
		return true
	default:
		return false
	}
}

// HandleMethod handles resource-related methods
func (r *ResourceExecutor) HandleMethod(ctx context.Context, method string, req *mcppb.JsonRpcRequest) (*mcppb.JsonRpcResponse, error) {
	// Use the utility function to process the request
	response, params, err := ProcessRequest(method, req, r.serverInfo.GetFeatureRegistry().ResourceRegistry != nil)
	if err != nil {
		return nil, err
	}

	var result interface{}

	// Handle the method
	switch method {
	case "resources/list":
		result, err = r.handleListResources(ctx, params)
	case "resources/read":
		result, err = r.handleReadResource(ctx, params)
	case "resources/subscribe":
		result, err = r.handleSubscribeResource(ctx, params)
	case "resources/unsubscribe":
		result, err = r.handleUnsubscribeResource(ctx, params)
	case "resources/templates/list":
		result, err = r.handleListResourceTemplates(ctx, params)
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

// handleListResources handles a request to list resources
func (r *ResourceExecutor) handleListResources(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	var cursor string

	// Extract cursor
	if cursorVal, ok := params["cursor"]; ok {
		if cursorStr, ok := cursorVal.(string); ok {
			cursor = cursorStr
		}
	}

	// Create options
	opts := resources.ResourceListOptions{
		Cursor: cursor,
	}

	// Call the registry
	return r.serverInfo.GetFeatureRegistry().ResourceRegistry.ListResources(ctx, opts), nil
}

// handleReadResource handles a request to read a specific resource
func (r *ResourceExecutor) handleReadResource(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Extract URI
	uriVal, ok := params["uri"]
	if !ok {
		return nil, fmt.Errorf("%w: resource URI is required", resources.ErrInvalidParams)
	}

	uri, ok := uriVal.(string)
	if !ok || uri == "" {
		return nil, fmt.Errorf("%w: resource URI must be a non-empty string", resources.ErrInvalidParams)
	}

	// Read the resource
	contents, err := r.serverInfo.GetFeatureRegistry().ResourceRegistry.ReadResource(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("error reading resource: %w", err)
	}

	return contents, nil
}

// handleSubscribeResource handles a request to subscribe to a resource
func (r *ResourceExecutor) handleSubscribeResource(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Extract URI
	uriVal, ok := params["uri"]
	if !ok {
		return nil, fmt.Errorf("%w: resource URI is required", resources.ErrInvalidParams)
	}

	uri, ok := uriVal.(string)
	if !ok || uri == "" {
		return nil, fmt.Errorf("%w: resource URI must be a non-empty string", resources.ErrInvalidParams)
	}

	// Subscribe to the resource
	err := r.serverInfo.GetFeatureRegistry().ResourceRegistry.SubscribeResource(ctx, uri)
	if err != nil {
		if err == resources.ErrResourceNotFound {
			return nil, fmt.Errorf("resource not found: %s", uri)
		}
		return nil, fmt.Errorf("error subscribing to resource: %w", err)
	}

	// Return success
	return map[string]interface{}{
		"success": true,
	}, nil
}

// handleUnsubscribeResource handles a request to unsubscribe from a resource
func (r *ResourceExecutor) handleUnsubscribeResource(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Extract URI
	uriVal, ok := params["uri"]
	if !ok {
		return nil, fmt.Errorf("%w: resource URI is required", resources.ErrInvalidParams)
	}

	uri, ok := uriVal.(string)
	if !ok || uri == "" {
		return nil, fmt.Errorf("%w: resource URI must be a non-empty string", resources.ErrInvalidParams)
	}

	// Unsubscribe from the resource
	err := r.serverInfo.GetFeatureRegistry().ResourceRegistry.UnsubscribeResource(ctx, uri)
	if err != nil {
		if err == resources.ErrResourceNotFound {
			return nil, fmt.Errorf("resource not found: %s", uri)
		}
		return nil, fmt.Errorf("error unsubscribing from resource: %w", err)
	}

	// Return success
	return map[string]interface{}{
		"success": true,
	}, nil
}

// handleListResourceTemplates handles a request to list resource templates
func (r *ResourceExecutor) handleListResourceTemplates(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	var cursor string

	// Extract cursor
	if cursorVal, ok := params["cursor"]; ok {
		if cursorStr, ok := cursorVal.(string); ok {
			cursor = cursorStr
		}
	}

	// Create options
	opts := resources.ResourceTemplateListOptions{
		Cursor: cursor,
	}

	// Call the registry
	return r.serverInfo.GetFeatureRegistry().ResourceRegistry.ListResourceTemplates(ctx, opts), nil
}

// Ensure ResourceExecutor implements config.MethodHandler
var _ config.MethodHandler = (*ResourceExecutor)(nil)
