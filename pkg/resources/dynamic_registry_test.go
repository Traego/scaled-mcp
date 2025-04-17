package resources

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"testing"
)

// MockToolProvider implements the ToolProvider interface for testing
type MockToolProvider struct {
	tools map[string]protocol.Tool
}

func NewMockToolProvider() *MockToolProvider {
	return &MockToolProvider{
		tools: make(map[string]protocol.Tool),
	}
}

func (p *MockToolProvider) GetTool(ctx context.Context, name string) (protocol.Tool, error) {
	tool, ok := p.tools[name]
	if !ok {
		return protocol.Tool{}, ErrToolNotFound
	}
	return tool, nil
}

func (p *MockToolProvider) ListTools(ctx context.Context, cursor string) (protocol.ToolListResult, error) {
	// Simple implementation that returns all tools
	tools := make([]protocol.Tool, 0, len(p.tools))
	for _, tool := range p.tools {
		tools = append(tools, tool)
	}

	return protocol.ToolListResult{
		Tools:      tools,
		NextCursor: "",
	}, nil
}

func (p *MockToolProvider) HandleToolInvocation(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
	tool, found := p.tools[name]
	if !found {
		return nil, ErrToolNotFound
	}

	// Validate required parameters
	for _, paramName := range tool.InputSchema.Required {
		if _, exists := params[paramName]; !exists {
			return nil, fmt.Errorf("%w: missing required parameter %s", ErrInvalidParams, paramName)
		}
	}

	// Echo back the parameters for testing
	return params, nil
}

func (p *MockToolProvider) AddTool(tool protocol.Tool) {
	p.tools[tool.Name] = tool
}

func TestDynamicToolRegistry(t *testing.T) {
	// Create a mock tool provider
	provider := NewMockToolProvider()

	// Create tools with the new WithInputs method
	weatherTool := NewTool("weather").
		WithDescription("Get weather information").
		WithInputs([]ToolInput{
			{
				Name:        "location",
				Type:        "string",
				Description: "Location to get weather for",
				Required:    true,
			},
			{
				Name:        "units",
				Type:        "string",
				Description: "Temperature units",
				Default:     "celsius",
			},
		}).
		Build()

	calculatorTool := NewTool("calculator").
		WithDescription("Perform calculations").
		WithInputs([]ToolInput{
			{
				Name:        "operation",
				Type:        "string",
				Description: "Operation to perform",
				Required:    true,
			},
			{
				Name:        "a",
				Type:        "integer",
				Description: "First operand",
				Required:    true,
			},
			{
				Name:        "b",
				Type:        "integer",
				Description: "Second operand",
				Required:    true,
			},
		}).
		Build()

	// Add tools to the provider
	provider.AddTool(weatherTool)
	provider.AddTool(calculatorTool)

	// Create a dynamic tool registry with the provider
	registry := NewDynamicToolRegistry(provider)

	// Test GetTool
	t.Run("GetTool", func(t *testing.T) {
		ctx := context.Background()

		// Get an existing tool
		gotTool, err := registry.GetTool(ctx, "weather")
		require.NoError(t, err)

		if gotTool.Name != weatherTool.Name {
			t.Errorf("Expected tool name %q, got %q", weatherTool.Name, gotTool.Name)
		}

		if gotTool.Description != weatherTool.Description {
			t.Errorf("Expected tool description %q, got %q", weatherTool.Description, gotTool.Description)
		}

		// Get a non-existent tool
		_, err = registry.GetTool(ctx, "non-existent-tool")
		assert.ErrorIs(t, err, ErrToolNotFound)
	})

	// Test ListTools
	t.Run("ListTools", func(t *testing.T) {
		ctx := context.Background()

		result, err := registry.ListTools(ctx, protocol.ToolListOptions{})
		require.NoError(t, err)

		if len(result.Tools) != 2 {
			t.Fatalf("Expected 2 tools, got %d", len(result.Tools))
		}

		// Check if both tools are in the result
		toolNames := make(map[string]bool)
		for _, tool := range result.Tools {
			toolNames[tool.Name] = true
		}

		if !toolNames["weather"] {
			t.Error("Tool 'weather' not found in list result")
		}

		if !toolNames["calculator"] {
			t.Error("Tool 'calculator' not found in list result")
		}
	})

	// Test CallTool
	t.Run("CallTool", func(t *testing.T) {
		ctx := context.Background()

		// Call weather tool with valid parameters
		weatherParams := map[string]interface{}{
			"location": "New York",
			"units":    "fahrenheit",
		}

		result, err := registry.CallTool(ctx, "weather", weatherParams)
		if err != nil {
			t.Fatalf("Failed to call weather tool: %v", err)
		}

		// Our mock provider echoes back the parameters
		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map result, got %T", result)
		}

		if resultMap["location"] != "New York" {
			t.Errorf("Expected location %q, got %q", "New York", resultMap["location"])
		}

		if resultMap["units"] != "fahrenheit" {
			t.Errorf("Expected units %q, got %q", "fahrenheit", resultMap["units"])
		}

		// Call calculator tool with valid parameters
		calcParams := map[string]interface{}{
			"operation": "add",
			"a":         float64(5),
			"b":         float64(3),
		}

		result, err = registry.CallTool(ctx, "calculator", calcParams)
		if err != nil {
			t.Fatalf("Failed to call calculator tool: %v", err)
		}

		resultMap, ok = result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map result, got %T", result)
		}

		if resultMap["operation"] != "add" {
			t.Errorf("Expected operation %q, got %q", "add", resultMap["operation"])
		}

		// Call with missing required parameter
		_, err = registry.CallTool(ctx, "calculator", map[string]interface{}{
			"operation": "add",
			"a":         float64(5),
		})
		if err == nil {
			t.Error("Expected error for missing required parameter")
		}

		// Call non-existent tool
		_, err = registry.CallTool(ctx, "non-existent-tool", weatherParams)
		if err == nil {
			t.Error("Expected error for non-existent tool")
		}
	})
}

func TestDynamicToolRegistry_WithNilProvider(t *testing.T) {
	// Test creating a registry with a nil provider
	registry := NewDynamicToolRegistry(nil)

	ctx := context.Background()

	// GetTool should return not found
	_, err := registry.GetTool(ctx, "any-tool")
	assert.Error(t, err, ErrToolNotFound)

	// ListTools should return empty list
	result, err := registry.ListTools(ctx, protocol.ToolListOptions{})
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}
	if len(result.Tools) != 0 {
		t.Errorf("Expected empty tools list, got %d tools", len(result.Tools))
	}

	// CallTool should return error
	_, err = registry.CallTool(ctx, "any-tool", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error from CallTool with nil provider")
	}
}
