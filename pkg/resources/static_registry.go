package resources

import (
	"context"
	"fmt"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"log/slog"
	"reflect"
	"sort"
	"sync"
)

// StaticToolRegistry is a resources that holds a fixed set of resources
type StaticToolRegistry struct {
	mu       sync.RWMutex
	tools    map[string]protocol.Tool
	handlers map[string]ToolHandler
}

// NewStaticToolRegistry creates a new static tool resources
func NewStaticToolRegistry() *StaticToolRegistry {
	return &StaticToolRegistry{
		tools:    make(map[string]protocol.Tool),
		handlers: make(map[string]ToolHandler),
	}
}

// RegisterTool registers a tool with the resources
func (r *StaticToolRegistry) RegisterTool(tool protocol.Tool, handler ToolHandler) error {
	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if a tool with this name already exists
	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool with name %q already exists", tool.Name)
	}

	r.tools[tool.Name] = tool
	r.handlers[tool.Name] = handler

	slog.Info("Registered tool", "name", tool.Name)
	return nil
}

// RegisterStructToolWithHandler registers a tool with a typed handler function
func (r *StaticToolRegistry) RegisterStructToolWithHandler(name, description string, structType reflect.Type, handlerFunc interface{}) error {
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	schema, err := GenerateSchemaFromStruct(structType)
	if err != nil {
		return fmt.Errorf("failed to generate schema from struct: %w", err)
	}

	tool := protocol.Tool{
		Name:        name,
		Description: description,
		InputSchema: schema,
	}

	handlerValue := reflect.ValueOf(handlerFunc)
	handlerType := handlerValue.Type()

	if handlerType.Kind() != reflect.Func {
		return fmt.Errorf("handler must be a function")
	}

	if handlerType.NumIn() != 2 || handlerType.NumOut() != 2 {
		return fmt.Errorf("handler must have signature func(context.Context, *StructType) (interface{}, error)")
	}

	wrappedHandler := func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		inputPtr := reflect.New(structType)
		if err := UnmarshalParams(params, inputPtr.Interface()); err != nil {
			return nil, fmt.Errorf("%w: failed to unmarshal parameters: %v", ErrInvalidParams, err)
		}

		args := []reflect.Value{
			reflect.ValueOf(ctx),
			inputPtr,
		}

		results := handlerValue.Call(args)

		var err error
		if !results[1].IsNil() {
			err = results[1].Interface().(error)
		}

		return results[0].Interface(), err
	}

	return r.RegisterTool(tool, wrappedHandler)
}



// GetTool returns a tool by name
func (r *StaticToolRegistry) GetTool(ctx context.Context, name string) (protocol.Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, ok := r.tools[name]
	if !ok {
		return protocol.Tool{}, ErrToolNotFound
	}
	return tool, nil
}

// ListTools returns a paginated list of resources
func (r *StaticToolRegistry) ListTools(ctx context.Context, opts protocol.ToolListOptions) (protocol.ToolListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get all tool names and sort them for consistent pagination
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)

	// Find the starting position based on cursor
	startPos := 0
	if opts.Cursor != "" {
		for i, name := range names {
			if name == opts.Cursor {
				startPos = i + 1 // Start after the cursor
				break
			}
		}
	}

	// Calculate end position
	endPos := startPos + 20
	if endPos > len(names) {
		endPos = len(names)
	}

	// Extract the resources for this page
	result := protocol.ToolListResult{
		Tools: make([]protocol.Tool, 0),
	}

	// No resources or cursor beyond the end
	if startPos >= len(names) {
		return result, nil
	}

	// Get the resources for this page
	result.Tools = make([]protocol.Tool, 0, endPos-startPos)
	for i := startPos; i < endPos; i++ {
		result.Tools = append(result.Tools, r.tools[names[i]])
	}

	// Set next cursor if there are more resources
	if endPos < len(names) {
		result.NextCursor = names[endPos-1]
	}

	return result, nil
}

// InvokeTool invokes a tool with the given parameters
func (r *StaticToolRegistry) CallTool(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
	r.mu.RLock()
	tool, toolExists := r.tools[name]
	handler, handlerExists := r.handlers[name]
	r.mu.RUnlock()

	if !toolExists || !handlerExists {
		return nil, ErrToolNotFound
	}

	// Validate required parameters
	if len(tool.InputSchema.Required) > 0 {
		for _, requiredParam := range tool.InputSchema.Required {
			if _, exists := params[requiredParam]; !exists {
				return nil, fmt.Errorf("%w: missing required parameter %s", ErrInvalidParams, requiredParam)
			}
		}
	}

	// Apply default values for missing optional parameters
	for paramName, propSchema := range tool.InputSchema.Properties {
		if propSchema.Default != nil {
			if _, exists := params[paramName]; !exists {
				params[paramName] = propSchema.Default
			}
		}
	}

	return handler(ctx, params)
}

// SetToolHandler sets a handler for an existing tool
func (r *StaticToolRegistry) SetToolHandler(name string, handler ToolHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("%w: tool %s not found", ErrToolNotFound, name)
	}

	r.handlers[name] = handler
	slog.Info("Set handler for tool", "name", name)
	return nil
}

// Ensure StaticToolRegistry implements ToolRegistry
var _ ToolRegistry = (*StaticToolRegistry)(nil)
