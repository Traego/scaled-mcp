package resources

import (
	"context"
	"reflect"
	"testing"

	"github.com/traego/scaled-mcp/pkg/protocol"
)

type SimpleStruct struct {
	Name     string `mcp:"name,The name field,required"`
	Age      int    `mcp:"age,The age field"`
	Optional string `mcp:"optional,An optional field,default=test"`
}

type ComplexStruct struct {
	Operation string  `mcp:"operation,The operation to perform,required"`
	A         float64 `mcp:"a,First operand,required"`
	B         float64 `mcp:"b,Second operand,required"`
	Precision *int    `mcp:"precision,Number of decimal places"`
}

type IgnoredFieldStruct struct {
	Included string `mcp:"included,This field is included"`
	Ignored  string `mcp:"-"`
}

func TestGenerateSchemaFromStruct(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected protocol.InputSchema
	}{
		{
			name:  "Simple struct",
			input: SimpleStruct{},
			expected: protocol.InputSchema{
				Type: "object",
				Properties: map[string]protocol.SchemaProperty{
					"name": {
						Type:        "string",
						Description: "The name field",
					},
					"age": {
						Type:        "integer",
						Description: "The age field",
					},
					"optional": {
						Type:        "string",
						Description: "An optional field",
						Default:     "test",
					},
				},
				Required: []string{"name"},
			},
		},
		{
			name:  "Complex struct",
			input: ComplexStruct{},
			expected: protocol.InputSchema{
				Type: "object",
				Properties: map[string]protocol.SchemaProperty{
					"operation": {
						Type:        "string",
						Description: "The operation to perform",
					},
					"a": {
						Type:        "number",
						Description: "First operand",
					},
					"b": {
						Type:        "number",
						Description: "Second operand",
					},
					"precision": {
						Type:        "integer",
						Description: "Number of decimal places",
					},
				},
				Required: []string{"operation", "a", "b"},
			},
		},
		{
			name:  "Ignored fields",
			input: IgnoredFieldStruct{},
			expected: protocol.InputSchema{
				Type: "object",
				Properties: map[string]protocol.SchemaProperty{
					"included": {
						Type:        "string",
						Description: "This field is included",
					},
				},
				Required: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structType := reflect.TypeOf(tt.input)
			schema, err := GenerateSchemaFromStruct(structType)
			if err != nil {
				t.Fatalf("GenerateSchemaFromStruct() error = %v", err)
			}

			if schema.Type != tt.expected.Type {
				t.Errorf("Type = %v, want %v", schema.Type, tt.expected.Type)
			}

			if len(schema.Properties) != len(tt.expected.Properties) {
				t.Errorf("Properties count = %v, want %v", len(schema.Properties), len(tt.expected.Properties))
			}

			for name, expectedProp := range tt.expected.Properties {
				actualProp, exists := schema.Properties[name]
				if !exists {
					t.Errorf("Property %s not found", name)
					continue
				}

				if actualProp.Type != expectedProp.Type {
					t.Errorf("Property %s type = %v, want %v", name, actualProp.Type, expectedProp.Type)
				}

				if actualProp.Description != expectedProp.Description {
					t.Errorf("Property %s description = %v, want %v", name, actualProp.Description, expectedProp.Description)
				}

				if actualProp.Default != expectedProp.Default {
					t.Errorf("Property %s default = %v, want %v", name, actualProp.Default, expectedProp.Default)
				}
			}

			if len(schema.Required) != len(tt.expected.Required) {
				t.Errorf("Required count = %v, want %v", len(schema.Required), len(tt.expected.Required))
			}

			requiredMap := make(map[string]bool)
			for _, req := range schema.Required {
				requiredMap[req] = true
			}

			for _, expectedReq := range tt.expected.Required {
				if !requiredMap[expectedReq] {
					t.Errorf("Required field %s not found", expectedReq)
				}
			}
		})
	}
}

func TestUnmarshalParams(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]interface{}
		target   interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name: "Simple struct unmarshaling",
			params: map[string]interface{}{
				"name": "John",
				"age":  float64(30),
			},
			target: &SimpleStruct{},
			expected: &SimpleStruct{
				Name: "John",
				Age:  30,
			},
			wantErr: false,
		},
		{
			name: "Complex struct unmarshaling",
			params: map[string]interface{}{
				"operation": "add",
				"a":         float64(10.5),
				"b":         float64(20.3),
			},
			target: &ComplexStruct{},
			expected: &ComplexStruct{
				Operation: "add",
				A:         10.5,
				B:         20.3,
			},
			wantErr: false,
		},
		{
			name: "Missing optional fields",
			params: map[string]interface{}{
				"name": "Jane",
			},
			target: &SimpleStruct{},
			expected: &SimpleStruct{
				Name: "Jane",
				Age:  0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UnmarshalParams(tt.params, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				targetValue := reflect.ValueOf(tt.target).Elem()
				expectedValue := reflect.ValueOf(tt.expected).Elem()

				if !reflect.DeepEqual(targetValue.Interface(), expectedValue.Interface()) {
					t.Errorf("UnmarshalParams() result = %+v, want %+v", targetValue.Interface(), expectedValue.Interface())
				}
			}
		})
	}
}

func TestRegisterStructToolWithHandler(t *testing.T) {
	registry := NewStaticToolRegistry()

	handler := func(ctx context.Context, input *SimpleStruct) (interface{}, error) {
		return map[string]interface{}{
			"greeting": "Hello, " + input.Name,
			"age":      input.Age,
		}, nil
	}

	structType := reflect.TypeOf(SimpleStruct{})
	err := registry.RegisterStructToolWithHandler("greeting", "Generate a greeting", structType, handler)
	if err != nil {
		t.Fatalf("RegisterStructToolWithHandler() error = %v", err)
	}

	tool, err := registry.GetTool(context.Background(), "greeting")
	if err != nil {
		t.Fatalf("GetTool() error = %v", err)
	}

	if tool.Name != "greeting" {
		t.Errorf("Tool name = %v, want %v", tool.Name, "greeting")
	}

	if tool.Description != "Generate a greeting" {
		t.Errorf("Tool description = %v, want %v", tool.Description, "Generate a greeting")
	}

	if tool.InputSchema.Type != "object" {
		t.Errorf("InputSchema type = %v, want %v", tool.InputSchema.Type, "object")
	}

	nameProperty, exists := tool.InputSchema.Properties["name"]
	if !exists {
		t.Error("Name property not found in schema")
	} else {
		if nameProperty.Type != "string" {
			t.Errorf("Name property type = %v, want %v", nameProperty.Type, "string")
		}
		if nameProperty.Description != "The name field" {
			t.Errorf("Name property description = %v, want %v", nameProperty.Description, "The name field")
		}
	}

	params := map[string]interface{}{
		"name": "Alice",
		"age":  float64(25),
	}

	result, err := registry.CallTool(context.Background(), "greeting", params)
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Result is not a map[string]interface{}")
	}

	if resultMap["greeting"] != "Hello, Alice" {
		t.Errorf("Result greeting = %v, want %v", resultMap["greeting"], "Hello, Alice")
	}

	if resultMap["age"] != 25 {
		t.Errorf("Result age = %v, want %v", resultMap["age"], 25)
	}
}

func TestStructHelpers(t *testing.T) {
	registry := NewStaticToolRegistry()

	handler := func(ctx context.Context, input *SimpleStruct) (interface{}, error) {
		return map[string]interface{}{
			"message": "Hello " + input.Name,
		}, nil
	}

	err := RegisterStructTool(registry, "test-helper", "Test helper function", handler)
	if err != nil {
		t.Fatalf("RegisterStructTool() error = %v", err)
	}

	tool, err := registry.GetTool(context.Background(), "test-helper")
	if err != nil {
		t.Fatalf("GetTool() error = %v", err)
	}

	if tool.Name != "test-helper" {
		t.Errorf("Tool name = %v, want %v", tool.Name, "test-helper")
	}

	MustRegisterStructTool(registry, "must-test", "Must test function", handler)

	mustTool, err := registry.GetTool(context.Background(), "must-test")
	if err != nil {
		t.Fatalf("GetTool() for must tool error = %v", err)
	}

	if mustTool.Name != "must-test" {
		t.Errorf("Must tool name = %v, want %v", mustTool.Name, "must-test")
	}
}

func TestDefaultValueParsing(t *testing.T) {
	type DefaultStruct struct {
		StringField  string  `mcp:"str,String field,default=hello"`
		IntField     int     `mcp:"int,Int field,default=42"`
		FloatField   float64 `mcp:"float,Float field,default=3.14"`
		BoolField    bool    `mcp:"bool,Bool field,default=true"`
		UintField    uint    `mcp:"uint,Uint field,default=100"`
		InvalidField string  `mcp:"invalid,Invalid field,default=invalid_int"`
	}

	structType := reflect.TypeOf(DefaultStruct{})
	schema, err := GenerateSchemaFromStruct(structType)
	if err != nil {
		t.Fatalf("GenerateSchemaFromStruct() error = %v", err)
	}

	tests := []struct {
		fieldName string
		expected  interface{}
	}{
		{"str", "hello"},
		{"int", int64(42)},
		{"float", 3.14},
		{"bool", true},
		{"uint", uint64(100)},
		{"invalid", "invalid_int"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			prop, exists := schema.Properties[tt.fieldName]
			if !exists {
				t.Fatalf("Property %s not found", tt.fieldName)
			}
			if prop.Default != tt.expected {
				t.Errorf("Default value for %s = %v, want %v", tt.fieldName, prop.Default, tt.expected)
			}
		})
	}
}

func TestComplexUnmarshaling(t *testing.T) {
	type PointerStruct struct {
		Name     *string `mcp:"name,Name field"`
		Age      *int    `mcp:"age,Age field"`
		Optional *bool   `mcp:"optional,Optional field"`
	}

	tests := []struct {
		name     string
		params   map[string]interface{}
		target   interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name: "Pointer fields",
			params: map[string]interface{}{
				"name": "John",
				"age":  float64(30),
			},
			target: &PointerStruct{},
			expected: &PointerStruct{
				Name: stringPtr("John"),
				Age:  intPtr(30),
			},
			wantErr: false,
		},
		{
			name:    "Non-pointer target",
			params:  map[string]interface{}{},
			target:  SimpleStruct{},
			wantErr: true,
		},
		{
			name:    "Non-struct target",
			params:  map[string]interface{}{},
			target:  stringPtr("test"),
			wantErr: true,
		},
		{
			name: "Type conversion errors",
			params: map[string]interface{}{
				"name": 123,
				"age":  "not_a_number",
			},
			target:  &SimpleStruct{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UnmarshalParams(tt.params, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.expected != nil {
				targetValue := reflect.ValueOf(tt.target).Elem()
				expectedValue := reflect.ValueOf(tt.expected).Elem()

				if !reflect.DeepEqual(targetValue.Interface(), expectedValue.Interface()) {
					t.Errorf("UnmarshalParams() result = %+v, want %+v", targetValue.Interface(), expectedValue.Interface())
				}
			}
		})
	}
}

func TestUnsupportedTypes(t *testing.T) {
	type UnsupportedStruct struct {
		Channel chan int `mcp:"channel,Channel field"`
	}

	structType := reflect.TypeOf(UnsupportedStruct{})
	_, err := GenerateSchemaFromStruct(structType)
	if err == nil {
		t.Error("Expected error for unsupported type, got nil")
	}
}

func TestErrorCases(t *testing.T) {
	t.Run("Non-struct type", func(t *testing.T) {
		_, err := GenerateSchemaFromStruct(reflect.TypeOf("string"))
		if err == nil {
			t.Error("Expected error for non-struct type, got nil")
		}
	})

	t.Run("Nil pointer type", func(t *testing.T) {
		var ptr *SimpleStruct
		_, err := GenerateSchemaFromStruct(reflect.TypeOf(ptr))
		if err != nil {
			t.Errorf("Unexpected error for pointer type: %v", err)
		}
	})
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
