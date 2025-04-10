package resources

import (
	"context"
	"errors"
	"testing"
)

func TestStaticToolRegistry(t *testing.T) {
	registry := NewStaticToolRegistry()

	// Test registering a tool
	toolName := "test-tool"
	tool := NewTool(toolName).
		WithDescription("Test tool").
		WithInputs([]ToolInput{
			{
				Name:        "param1",
				Type:        "string",
				Description: "Parameter 1",
				Required:    true,
			},
			{
				Name:        "param2",
				Type:        "integer",
				Description: "Parameter 2",
				Default:     42,
			},
		}).
		Build()

	// Create a handler function
	handlerFunc := func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		param1, ok := params["param1"].(string)
		if !ok {
			return nil, errors.New("param1 must be a string")
		}

		param2Val, ok := params["param2"].(float64)
		if !ok {
			param2Val = 42 // Default
		}

		return map[string]interface{}{
			"result": param1 + "-" + string(rune(int(param2Val))),
		}, nil
	}

	// Register the tool
	err := registry.RegisterTool(tool, handlerFunc)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Test GetTool
	t.Run("GetTool", func(t *testing.T) {
		ctx := context.Background()

		// Get an existing tool
		gotTool, exists := registry.GetTool(ctx, toolName)
		if !exists {
			t.Fatalf("Tool %q not found", toolName)
		}

		if gotTool.Name != tool.Name {
			t.Errorf("Expected tool name %q, got %q", tool.Name, gotTool.Name)
		}

		if gotTool.Description != tool.Description {
			t.Errorf("Expected tool description %q, got %q", tool.Description, gotTool.Description)
		}

		// Get a non-existent tool
		_, exists = registry.GetTool(ctx, "non-existent-tool")
		if exists {
			t.Error("Non-existent tool should not exist")
		}
	})

	// Test ListTools
	t.Run("ListTools", func(t *testing.T) {
		ctx := context.Background()

		result := registry.ListTools(ctx, ToolListOptions{})

		if len(result.Tools) != 1 {
			t.Fatalf("Expected 1 tool, got %d", len(result.Tools))
		}

		if result.Tools[0].Name != toolName {
			t.Errorf("Expected tool name %q, got %q", toolName, result.Tools[0].Name)
		}

		// Empty next cursor for a single page
		if result.NextCursor != "" {
			t.Errorf("Expected empty next cursor, got %q", result.NextCursor)
		}
	})

	// Test CallTool
	t.Run("CallTool", func(t *testing.T) {
		ctx := context.Background()

		// Call with valid parameters
		params := map[string]interface{}{
			"param1": "test",
			"param2": float64(65), // ASCII 'A'
		}

		result, err := registry.CallTool(ctx, toolName, params)
		if err != nil {
			t.Fatalf("Failed to call tool: %v", err)
		}

		// Check the result
		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map result, got %T", result)
		}

		expectedResult := "test-A"
		if resultMap["result"] != expectedResult {
			t.Errorf("Expected result %q, got %q", expectedResult, resultMap["result"])
		}

		// Call with missing required parameter
		_, err = registry.CallTool(ctx, toolName, map[string]interface{}{
			"param2": float64(65),
		})
		if err == nil {
			t.Error("Expected error for missing required parameter")
		}

		// Call non-existent tool
		_, err = registry.CallTool(ctx, "non-existent-tool", params)
		if err == nil {
			t.Error("Expected error for non-existent tool")
		}
	})

	// Test registering a duplicate tool
	err = registry.RegisterTool(tool, handlerFunc)
	if err == nil {
		t.Error("Expected error when registering duplicate tool")
	}
}

func TestStaticToolRegistry_SetToolHandler(t *testing.T) {
	registry := NewStaticToolRegistry()

	// Register a tool without a handler
	toolName := "test-tool"
	tool := NewTool(toolName).
		WithDescription("Test tool").
		WithInputs([]ToolInput{
			{
				Name:        "param",
				Type:        "string",
				Description: "Parameter",
				Required:    true,
			},
		}).
		Build()

	err := registry.RegisterTool(tool, nil)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Set the handler after registration
	handlerFunc := func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		param, ok := params["param"].(string)
		if !ok {
			return nil, errors.New("param must be a string")
		}

		return map[string]interface{}{
			"echo": param,
		}, nil
	}

	err = registry.SetToolHandler(toolName, handlerFunc)
	if err != nil {
		t.Fatalf("Failed to set tool handler: %v", err)
	}

	// Test the handler
	ctx := context.Background()
	params := map[string]interface{}{
		"param": "hello",
	}

	result, err := registry.CallTool(ctx, toolName, params)
	if err != nil {
		t.Fatalf("Failed to call tool: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["echo"] != "hello" {
		t.Errorf("Expected echo %q, got %q", "hello", resultMap["echo"])
	}

	// Test setting handler for non-existent tool
	err = registry.SetToolHandler("non-existent-tool", handlerFunc)
	if err == nil {
		t.Error("Expected error when setting handler for non-existent tool")
	}
}
