package resources

import (
	"context"
	"errors"
	"github.com/traego/scaled-mcp/pkg/protocol"
)

// Common errors
var (
	ErrToolNotFound  = errors.New("tool not found")
	ErrInvalidParams = errors.New("invalid parameters")
)

// ToolParameter represents a parameter for a tool
type ToolParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required"`
	Properties  interface{} `json:"properties,omitempty"`
}

// ToolHandler is a function that handles a tool invocation
type ToolHandler func(ctx context.Context, params map[string]interface{}) (interface{}, error)

// ToolRegistry defines the interface for a tool resources
type ToolRegistry interface {
	// GetTool returns a tool by name
	GetTool(ctx context.Context, name string) (protocol.Tool, error)

	// ListTools returns a paginated list of resources
	ListTools(ctx context.Context, opts protocol.ToolListOptions) (protocol.ToolListResult, error)

	// InvokeTool invokes a tool with the given parameters
	CallTool(ctx context.Context, name string, params map[string]interface{}) (interface{}, error)
}
