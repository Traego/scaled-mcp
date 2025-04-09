package resources

import (
	"context"
	"errors"
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

// InputSchema represents the schema for tool inputs
type InputSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]SchemaProperty `json:"properties"`
	Required   []string                  `json:"required,omitempty"`
}

// SchemaProperty represents a property in an input schema
type SchemaProperty struct {
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Default     interface{} `json:"default,omitempty"`
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema InputSchema `json:"inputSchema,omitempty"`
}

// ToolHandler is a function that handles a tool invocation
type ToolHandler func(ctx context.Context, params map[string]interface{}) (interface{}, error)

// ToolListOptions provides pagination options for listing resources
type ToolListOptions struct {
	Cursor string // Cursor for pagination
}

// ToolListResult represents a paginated list of resources
type ToolListResult struct {
	Tools      []Tool `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"` // Cursor for the next page, empty if no more pages
}

// ToolRegistry defines the interface for a tool resources
type ToolRegistry interface {
	// GetTool returns a tool by name
	GetTool(ctx context.Context, name string) (Tool, bool)

	// ListTools returns a paginated list of resources
	ListTools(ctx context.Context, opts ToolListOptions) ToolListResult

	// InvokeTool invokes a tool with the given parameters
	CallTool(ctx context.Context, name string, params map[string]interface{}) (interface{}, error)
}
