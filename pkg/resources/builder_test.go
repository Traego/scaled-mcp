package resources

import (
	"github.com/traego/scaled-mcp/pkg/protocol"
	"reflect"
	"testing"
)

func TestNewTool(t *testing.T) {
	toolName := "test-tool"
	builder := NewTool(toolName)

	if builder == nil {
		t.Fatal("NewTool returned nil")
	}

	tool := builder.Build()

	if tool.Name != toolName {
		t.Errorf("Expected tool name to be %q, got %q", toolName, tool.Name)
	}

	if tool.InputSchema.Type != "object" {
		t.Errorf("Expected input schema type to be 'object', got %q", tool.InputSchema.Type)
	}

	if len(tool.InputSchema.Properties) != 0 {
		t.Errorf("Expected empty properties, got %d properties", len(tool.InputSchema.Properties))
	}

	if len(tool.InputSchema.Required) != 0 {
		t.Errorf("Expected empty required fields, got %d required fields", len(tool.InputSchema.Required))
	}
}

func TestWithDescription(t *testing.T) {
	description := "Test tool description"
	tool := NewTool("test-tool").
		WithDescription(description).
		Build()

	if tool.Description != description {
		t.Errorf("Expected description to be %q, got %q", description, tool.Description)
	}
}

func TestWithInputs(t *testing.T) {
	testCases := []struct {
		name     string
		inputs   []ToolInput
		expected protocol.Tool
	}{
		{
			name:   "Empty inputs",
			inputs: []ToolInput{},
			expected: protocol.Tool{
				Name: "test-tool",
				InputSchema: protocol.InputSchema{
					Type:       "object",
					Properties: map[string]protocol.SchemaProperty{},
					Required:   []string{},
				},
			},
		},
		{
			name: "Single input",
			inputs: []ToolInput{
				{
					Name:        "param1",
					Type:        "string",
					Description: "Parameter 1",
					Required:    true,
				},
			},
			expected: protocol.Tool{
				Name: "test-tool",
				InputSchema: protocol.InputSchema{
					Type: "object",
					Properties: map[string]protocol.SchemaProperty{
						"param1": {
							Type:        "string",
							Description: "Parameter 1",
						},
					},
					Required: []string{"param1"},
				},
			},
		},
		{
			name: "Multiple inputs with defaults",
			inputs: []ToolInput{
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
					Required:    false,
					Default:     42,
				},
				{
					Name:        "param3",
					Type:        "boolean",
					Description: "Parameter 3",
					Required:    false,
					Default:     true,
				},
			},
			expected: protocol.Tool{
				Name: "test-tool",
				InputSchema: protocol.InputSchema{
					Type: "object",
					Properties: map[string]protocol.SchemaProperty{
						"param1": {
							Type:        "string",
							Description: "Parameter 1",
						},
						"param2": {
							Type:        "integer",
							Description: "Parameter 2",
							Default:     42,
						},
						"param3": {
							Type:        "boolean",
							Description: "Parameter 3",
							Default:     true,
						},
					},
					Required: []string{"param1"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set the name to match the expected tool
			tc.expected.Name = "test-tool"

			tool := NewTool("test-tool").
				WithInputs(tc.inputs).
				Build()

			// Check the properties
			if !reflect.DeepEqual(tool.InputSchema.Properties, tc.expected.InputSchema.Properties) {
				t.Errorf("Properties don't match\nExpected: %+v\nGot: %+v",
					tc.expected.InputSchema.Properties, tool.InputSchema.Properties)
			}

			// Check the required fields
			if !reflect.DeepEqual(tool.InputSchema.Required, tc.expected.InputSchema.Required) {
				t.Errorf("Required fields don't match\nExpected: %+v\nGot: %+v",
					tc.expected.InputSchema.Required, tool.InputSchema.Required)
			}
		})
	}
}

func TestWithString(t *testing.T) {
	paramName := "string-param"
	paramDesc := "String parameter description"

	tool := NewTool("test-tool").
		WithString(paramName).
		Description(paramDesc).
		Add().
		Build()

	// Check if the parameter was added correctly
	prop, exists := tool.InputSchema.Properties[paramName]
	if !exists {
		t.Fatalf("Parameter %q was not added to the tool", paramName)
	}

	if prop.Type != "string" {
		t.Errorf("Expected parameter type to be 'string', got %q", prop.Type)
	}

	if prop.Description != paramDesc {
		t.Errorf("Expected parameter description to be %q, got %q", paramDesc, prop.Description)
	}
}

func TestWithInteger(t *testing.T) {
	paramName := "int-param"
	paramDesc := "Integer parameter description"

	tool := NewTool("test-tool").
		WithInteger(paramName).
		Description(paramDesc).
		Add().
		Build()

	// Check if the parameter was added correctly
	prop, exists := tool.InputSchema.Properties[paramName]
	if !exists {
		t.Fatalf("Parameter %q was not added to the tool", paramName)
	}

	if prop.Type != "integer" {
		t.Errorf("Expected parameter type to be 'integer', got %q", prop.Type)
	}

	if prop.Description != paramDesc {
		t.Errorf("Expected parameter description to be %q, got %q", paramDesc, prop.Description)
	}
}

func TestWithBoolean(t *testing.T) {
	paramName := "bool-param"
	paramDesc := "Boolean parameter description"

	tool := NewTool("test-tool").
		WithBoolean(paramName).
		Description(paramDesc).
		Add().
		Build()

	// Check if the parameter was added correctly
	prop, exists := tool.InputSchema.Properties[paramName]
	if !exists {
		t.Fatalf("Parameter %q was not added to the tool", paramName)
	}

	if prop.Type != "boolean" {
		t.Errorf("Expected parameter type to be 'boolean', got %q", prop.Type)
	}

	if prop.Description != paramDesc {
		t.Errorf("Expected parameter description to be %q, got %q", paramDesc, prop.Description)
	}
}

func TestParameterBuilder_Required(t *testing.T) {
	paramName := "required-param"

	tool := NewTool("test-tool").
		WithString(paramName).
		Required().
		Add().
		Build()

	// Check if the parameter is in the required list
	found := false
	for _, req := range tool.InputSchema.Required {
		if req == paramName {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Parameter %q was not added to the required list", paramName)
	}
}

func TestParameterBuilder_Default(t *testing.T) {
	paramName := "default-param"
	defaultValue := "default-value"

	tool := NewTool("test-tool").
		WithString(paramName).
		Default(defaultValue).
		Add().
		Build()

	// Check if the parameter has the default value
	prop, exists := tool.InputSchema.Properties[paramName]
	if !exists {
		t.Fatalf("Parameter %q was not added to the tool", paramName)
	}

	if prop.Default != defaultValue {
		t.Errorf("Expected default value to be %q, got %v", defaultValue, prop.Default)
	}
}

func TestComplexToolDefinition(t *testing.T) {
	// Test a complex tool definition using both WithInputs and individual parameter methods
	tool := NewTool("complex-tool").
		WithDescription("A complex tool with multiple parameters").
		WithInputs([]ToolInput{
			{
				Name:        "input1",
				Type:        "string",
				Description: "First input parameter",
				Required:    true,
			},
			{
				Name:        "input2",
				Type:        "integer",
				Description: "Second input parameter",
				Default:     100,
			},
		}).
		WithString("input3").
		Description("Third input parameter").
		Add().
		WithBoolean("input4").
		Required().
		Description("Fourth input parameter").
		Add().
		Build()

	// Check the number of parameters
	if len(tool.InputSchema.Properties) != 4 {
		t.Errorf("Expected 4 parameters, got %d", len(tool.InputSchema.Properties))
	}

	// Check required parameters
	expectedRequired := []string{"input1", "input4"}
	if len(tool.InputSchema.Required) != len(expectedRequired) {
		t.Errorf("Expected %d required parameters, got %d",
			len(expectedRequired), len(tool.InputSchema.Required))
	}

	// Check each parameter exists
	params := []string{"input1", "input2", "input3", "input4"}
	for _, param := range params {
		if _, exists := tool.InputSchema.Properties[param]; !exists {
			t.Errorf("Parameter %q was not added to the tool", param)
		}
	}

	// Check types
	expectedTypes := map[string]string{
		"input1": "string",
		"input2": "integer",
		"input3": "string",
		"input4": "boolean",
	}

	for param, expectedType := range expectedTypes {
		if tool.InputSchema.Properties[param].Type != expectedType {
			t.Errorf("Parameter %q: expected type %q, got %q",
				param, expectedType, tool.InputSchema.Properties[param].Type)
		}
	}

	// Check default values
	if tool.InputSchema.Properties["input2"].Default != 100 {
		t.Errorf("Parameter input2: expected default value 100, got %v",
			tool.InputSchema.Properties["input2"].Default)
	}
}
