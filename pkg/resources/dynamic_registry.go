package resources

import (
	"context"
	"log/slog"
)

// ToolProvider defines the interface for dynamically providing resources
type ToolProvider interface {
	// GetTool returns a tool by name
	GetTool(ctx context.Context, name string) (Tool, bool)

	// ListTools returns a list of available resources
	ListTools(ctx context.Context, cursor string) ([]Tool, string)

	// HandleToolInvocation handles a tool invocation
	HandleToolInvocation(ctx context.Context, name string, params map[string]interface{}) (interface{}, error)
}

// DynamicToolRegistry is a resources that dynamically provides resources through a provider
type DynamicToolRegistry struct {
	provider ToolProvider
}

// NewDynamicToolRegistry creates a new dynamic tool resources
func NewDynamicToolRegistry(provider ToolProvider) *DynamicToolRegistry {
	return &DynamicToolRegistry{
		provider: provider,
	}
}

// GetTool returns a tool by name
func (r *DynamicToolRegistry) GetTool(ctx context.Context, name string) (Tool, bool) {
	if r.provider == nil {
		slog.Debug("Tool provider is nil")
		return Tool{}, false
	}
	
	tool, found := r.provider.GetTool(ctx, name)
	if !found {
		slog.Debug("Tool not found", "name", name)
	}
	return tool, found
}

// ListTools returns a paginated list of resources
func (r *DynamicToolRegistry) ListTools(ctx context.Context, opts ToolListOptions) ToolListResult {
	if r.provider == nil {
		slog.Debug("Tool provider is nil")
		return ToolListResult{
			Tools:      []Tool{},
			NextCursor: "",
		}
	}

	tools, nextCursor := r.provider.ListTools(ctx, opts.Cursor)

	return ToolListResult{
		Tools:      tools,
		NextCursor: nextCursor,
	}
}

// InvokeTool invokes a tool with the given parameters
func (r *DynamicToolRegistry) CallTool(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
	if r.provider == nil {
		slog.Debug("Tool provider is nil")
		return nil, ErrToolNotFound
	}
	
	return r.provider.HandleToolInvocation(ctx, name, params)
}

// Ensure DynamicToolRegistry implements ToolRegistry
var _ ToolRegistry = (*DynamicToolRegistry)(nil)
