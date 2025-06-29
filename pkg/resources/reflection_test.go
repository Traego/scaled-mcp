package resources

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestParseDefaultValueComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		typ      reflect.Type
		expected interface{}
	}{
		{"string", "hello", reflect.TypeOf(""), "hello"},
		{"string_empty", "", reflect.TypeOf(""), ""},

		{"int", "42", reflect.TypeOf(int(0)), int64(42)},
		{"int8", "127", reflect.TypeOf(int8(0)), int64(127)},
		{"int16", "32767", reflect.TypeOf(int16(0)), int64(32767)},
		{"int32", "2147483647", reflect.TypeOf(int32(0)), int64(2147483647)},
		{"int64", "9223372036854775807", reflect.TypeOf(int64(0)), int64(9223372036854775807)},
		{"int_invalid", "not_a_number", reflect.TypeOf(int(0)), "not_a_number"},

		{"uint", "42", reflect.TypeOf(uint(0)), uint64(42)},
		{"uint8", "255", reflect.TypeOf(uint8(0)), uint64(255)},
		{"uint16", "65535", reflect.TypeOf(uint16(0)), uint64(65535)},
		{"uint32", "4294967295", reflect.TypeOf(uint32(0)), uint64(4294967295)},
		{"uint64", "18446744073709551615", reflect.TypeOf(uint64(0)), uint64(18446744073709551615)},
		{"uint_invalid", "not_a_number", reflect.TypeOf(uint(0)), "not_a_number"},

		{"float32", "3.14", reflect.TypeOf(float32(0)), 3.14},
		{"float64", "2.718281828", reflect.TypeOf(float64(0)), 2.718281828},
		{"float_invalid", "not_a_float", reflect.TypeOf(float64(0)), "not_a_float"},

		{"bool_true", "true", reflect.TypeOf(bool(false)), true},
		{"bool_false", "false", reflect.TypeOf(bool(false)), false},
		{"bool_1", "1", reflect.TypeOf(bool(false)), true},
		{"bool_0", "0", reflect.TypeOf(bool(false)), false},
		{"bool_invalid", "maybe", reflect.TypeOf(bool(false)), "maybe"},

		{"ptr_string", "hello", reflect.TypeOf((*string)(nil)), "hello"},
		{"ptr_int", "42", reflect.TypeOf((*int)(nil)), int64(42)},
		{"ptr_bool", "true", reflect.TypeOf((*bool)(nil)), true},

		{"unsupported_chan", "test", reflect.TypeOf(make(chan int)), "test"},
		{"unsupported_func", "test", reflect.TypeOf(func() {}), "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDefaultValue(tt.value, tt.typ)
			if result != tt.expected {
				t.Errorf("parseDefaultValue(%q, %v) = %v, want %v", tt.value, tt.typ, result, tt.expected)
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
		{
			name: "Nil parameter value",
			params: map[string]interface{}{
				"name": nil,
				"age":  float64(25),
			},
			target: &SimpleStruct{},
			expected: &SimpleStruct{
				Name: "",
				Age:  25,
			},
			wantErr: false,
		},
		{
			name: "Direct assignment",
			params: map[string]interface{}{
				"name": "Direct",
				"age":  int(30),
			},
			target: &SimpleStruct{},
			expected: &SimpleStruct{
				Name: "Direct",
				Age:  30,
			},
			wantErr: false,
		},
		{
			name: "Type conversion - int64 to int",
			params: map[string]interface{}{
				"name": "Test",
				"age":  int64(42),
			},
			target: &SimpleStruct{},
			expected: &SimpleStruct{
				Name: "Test",
				Age:  42,
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

func TestSetFieldValueComprehensive(t *testing.T) {
	type TestStruct struct {
		StringField  string
		IntField     int
		Int8Field    int8
		Int16Field   int16
		Int32Field   int32
		Int64Field   int64
		UintField    uint
		Uint8Field   uint8
		Uint16Field  uint16
		Uint32Field  uint32
		Uint64Field  uint64
		Float32Field float32
		Float64Field float64
		BoolField    bool
		SliceField   []string
		PtrString    *string
		PtrInt       *int
	}

	tests := []struct {
		name        string
		fieldName   string
		paramValue  interface{}
		wantErr     bool
		expectedVal interface{}
	}{
		{"string_direct", "StringField", "hello", false, "hello"},
		{"int_assignable", "IntField", 123, false, 123},

		{"int_from_float64", "IntField", float64(42), false, 42},
		{"int_from_int64", "IntField", int64(42), false, 42},
		{"int_direct", "IntField", 42, false, 42},
		{"int_error_string", "IntField", "invalid", true, nil},
		{"int8_from_float64", "Int8Field", float64(127), false, int8(127)},
		{"int16_from_float64", "Int16Field", float64(32767), false, int16(32767)},
		{"int32_from_float64", "Int32Field", float64(2147483647), false, int32(2147483647)},
		{"int64_from_float64", "Int64Field", float64(42), false, int64(42)},

		{"uint_from_float64", "UintField", float64(42), false, uint(42)},
		{"uint_from_uint64", "UintField", uint64(42), false, uint(42)},
		{"uint_direct", "UintField", uint(42), false, uint(42)},
		{"uint_error_string", "UintField", "invalid", true, nil},
		{"uint8_from_float64", "Uint8Field", float64(255), false, uint8(255)},
		{"uint16_from_float64", "Uint16Field", float64(65535), false, uint16(65535)},
		{"uint32_from_float64", "Uint32Field", float64(4294967295), false, uint32(4294967295)},
		{"uint64_from_float64", "Uint64Field", float64(42), false, uint64(42)},

		{"float32_from_float64", "Float32Field", float64(3.14), false, float32(3.14)},
		{"float64_direct", "Float64Field", 3.14159, false, 3.14159},
		{"float_error_string", "Float64Field", "invalid", true, nil},

		{"bool_direct", "BoolField", true, false, true},
		{"bool_error_string", "BoolField", "invalid", true, nil},

		{"ptr_string_nil_field", "PtrString", "hello", false, "hello"},
		{"ptr_int_nil_field", "PtrInt", float64(42), false, 42},

		{"nil_value", "StringField", nil, false, ""},

		{"assignable_type", "StringField", "direct_assign", false, "direct_assign"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := &TestStruct{}
			targetValue := reflect.ValueOf(target).Elem()
			fieldValue := targetValue.FieldByName(tt.fieldName)

			err := setFieldValue(fieldValue, tt.paramValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("setFieldValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.expectedVal != nil {
				actualVal := fieldValue.Interface()
				if fieldValue.Kind() == reflect.Ptr && !fieldValue.IsNil() {
					actualVal = fieldValue.Elem().Interface()
				}
				if actualVal != tt.expectedVal {
					t.Errorf("setFieldValue() result = %v, want %v", actualVal, tt.expectedVal)
				}
			}
		})
	}
}

func TestMustRegisterStructToolPanic(t *testing.T) {
	registry := NewStaticToolRegistry()

	validHandler := func(ctx context.Context, input *SimpleStruct) (interface{}, error) {
		return nil, nil
	}

	tool := protocol.Tool{
		Name:        "test-duplicate",
		Description: "Duplicate tool",
		InputSchema: protocol.InputSchema{Type: "object"},
	}

	err := registry.RegisterTool(tool, func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatalf("Failed to register initial tool: %v", err)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustRegisterStructTool should have panicked with duplicate tool registration")
		}
	}()

	MustRegisterStructTool(registry, "test-duplicate", "Duplicate tool", validHandler)
}

type CalculatorOutput struct {
	Result    float64 `mcp:"result,The calculation result,required"`
	Operation string  `mcp:"operation,The operation performed,required"`
	A         float64 `mcp:"a,First operand,required"`
	B         float64 `mcp:"b,Second operand,required"`
}

func TestRegisterStructToolWithTypes(t *testing.T) {
	registry := NewStaticToolRegistry()

	handler := func(ctx context.Context, input *SimpleStruct) (*CalculatorOutput, error) {
		return &CalculatorOutput{
			Result:    float64(input.Age * 2),
			Operation: "double_age",
			A:         float64(input.Age),
			B:         2.0,
		}, nil
	}

	err := RegisterTool(registry, "calculator", "Doubles the age", handler)
	assert.NoError(t, err)

	tool, err := registry.GetTool(context.Background(), "calculator")
	assert.NoError(t, err)
	assert.Equal(t, "calculator", tool.Name)
	assert.NotNil(t, tool.OutputSchema)
	assert.Equal(t, "object", tool.OutputSchema.Type)
	assert.Contains(t, tool.OutputSchema.Properties, "result")
	assert.Contains(t, tool.OutputSchema.Properties, "operation")
	assert.Contains(t, tool.OutputSchema.Properties, "a")
	assert.Contains(t, tool.OutputSchema.Properties, "b")

	result, err := registry.CallTool(context.Background(), "calculator", map[string]interface{}{
		"name": "John",
		"age":  25,
	})
	assert.NoError(t, err)

	output, ok := result.(*CalculatorOutput)
	assert.True(t, ok)
	assert.Equal(t, 50.0, output.Result)
	assert.Equal(t, "double_age", output.Operation)
	assert.Equal(t, 25.0, output.A)
	assert.Equal(t, 2.0, output.B)
}

func TestStructHelpersWithTypes(t *testing.T) {
	registry := NewStaticToolRegistry()

	handler := func(ctx context.Context, input *SimpleStruct) (*CalculatorOutput, error) {
		return &CalculatorOutput{
			Result:    float64(input.Age),
			Operation: "identity",
			A:         float64(input.Age),
			B:         1.0,
		}, nil
	}

	err := RegisterTool(registry, "identity", "Returns the age", handler)
	assert.NoError(t, err)

	tool, err := registry.GetTool(context.Background(), "identity")
	assert.NoError(t, err)
	assert.NotNil(t, tool.OutputSchema)
	assert.Equal(t, "object", tool.OutputSchema.Type)
	assert.Contains(t, tool.OutputSchema.Properties, "result")

	result, err := registry.CallTool(context.Background(), "identity", map[string]interface{}{
		"name": "Alice",
		"age":  30,
	})
	assert.NoError(t, err)

	output, ok := result.(*CalculatorOutput)
	assert.True(t, ok)
	assert.Equal(t, 30.0, output.Result)
	assert.Equal(t, "identity", output.Operation)
}

func TestMustRegisterStructToolWithTypesPanic(t *testing.T) {
	registry := NewStaticToolRegistry()

	validHandler := func(ctx context.Context, input *SimpleStruct) (*CalculatorOutput, error) {
		return &CalculatorOutput{
			Result:    float64(input.Age),
			Operation: "test",
			A:         float64(input.Age),
			B:         1.0,
		}, nil
	}

	assert.NotPanics(t, func() {
		MustRegisterTool(registry, "valid_tool", "A valid tool", validHandler)
	})

	assert.Panics(t, func() {
		MustRegisterTool(registry, "", "Empty name tool", validHandler)
	})
}

func TestGoTypeToSchemaTypeComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		input    reflect.Type
		expected string
		wantErr  bool
	}{
		{"string", reflect.TypeOf(""), "string", false},
		{"int", reflect.TypeOf(int(0)), "integer", false},
		{"int8", reflect.TypeOf(int8(0)), "integer", false},
		{"int16", reflect.TypeOf(int16(0)), "integer", false},
		{"int32", reflect.TypeOf(int32(0)), "integer", false},
		{"int64", reflect.TypeOf(int64(0)), "integer", false},
		{"uint", reflect.TypeOf(uint(0)), "integer", false},
		{"uint8", reflect.TypeOf(uint8(0)), "integer", false},
		{"uint16", reflect.TypeOf(uint16(0)), "integer", false},
		{"uint32", reflect.TypeOf(uint32(0)), "integer", false},
		{"uint64", reflect.TypeOf(uint64(0)), "integer", false},
		{"float32", reflect.TypeOf(float32(0)), "number", false},
		{"float64", reflect.TypeOf(float64(0)), "number", false},
		{"bool", reflect.TypeOf(bool(false)), "boolean", false},

		{"slice", reflect.TypeOf([]string{}), "array", false},
		{"array", reflect.TypeOf([5]string{}), "array", false},
		{"map", reflect.TypeOf(map[string]interface{}{}), "object", false},
		{"struct", reflect.TypeOf(struct{}{}), "object", false},
		{"interface", reflect.TypeOf((*interface{})(nil)).Elem(), "object", false},

		{"ptr_string", reflect.TypeOf((*string)(nil)), "string", false},
		{"ptr_int", reflect.TypeOf((*int)(nil)), "integer", false},
		{"ptr_float", reflect.TypeOf((*float64)(nil)), "number", false},
		{"ptr_bool", reflect.TypeOf((*bool)(nil)), "boolean", false},
		{"ptr_slice", reflect.TypeOf((*[]string)(nil)), "array", false},
		{"ptr_map", reflect.TypeOf((*map[string]interface{})(nil)), "object", false},
		{"ptr_struct", reflect.TypeOf((*struct{})(nil)), "object", false},

		{"function", reflect.TypeOf(func() {}), "", true},
		{"channel", reflect.TypeOf(make(chan int)), "", true},
		{"complex64", reflect.TypeOf(complex64(0)), "", true},
		{"complex128", reflect.TypeOf(complex128(0)), "", true},
		{"uintptr", reflect.TypeOf(uintptr(0)), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := goTypeToSchemaType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("goTypeToSchemaType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("goTypeToSchemaType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseStructFieldErrors(t *testing.T) {
	type InvalidStruct struct {
		BadField chan int `mcp:"bad,Bad field"`
	}

	structType := reflect.TypeOf(InvalidStruct{})
	field := structType.Field(0)

	_, _, _, err := parseStructField(field)
	if err == nil {
		t.Error("parseStructField should return error for unsupported type")
	}
}
