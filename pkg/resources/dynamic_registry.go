package resources

import (
	"context"
	"fmt"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"log/slog"
)

// ToolProvider defines the interface for dynamically providing resources
type ToolProvider interface {
	// GetTool returns a tool by name
	GetTool(ctx context.Context, name string) (protocol.Tool, error)

	// ListTools returns a list of available resources
	ListTools(ctx context.Context, cursor string) (protocol.ToolListResult, error)

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
func (r *DynamicToolRegistry) GetTool(ctx context.Context, name string) (protocol.Tool, error) {
	if r.provider == nil {
		return protocol.Tool{}, ErrToolNotFound
	}

	tool, err := r.provider.GetTool(ctx, name)
	if err != nil {
		return protocol.Tool{}, fmt.Errorf("problem getting tool: %w", err)
	}

	return tool, nil
}

// ListTools returns a paginated list of resources
func (r *DynamicToolRegistry) ListTools(ctx context.Context, opts protocol.ToolListOptions) (protocol.ToolListResult, error) {
	if r.provider == nil {
		slog.Debug("Tool provider is nil")
		return protocol.ToolListResult{
			Tools:      []protocol.Tool{},
			NextCursor: "",
		}, nil
	}

	tools, err := r.provider.ListTools(ctx, opts.Cursor)
	if err != nil {
		return protocol.ToolListResult{}, fmt.Errorf("problem listing tools: %w", err)
	}

	return tools, nil
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
