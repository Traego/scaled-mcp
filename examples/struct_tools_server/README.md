# Struct-Based Tools Example

This example demonstrates the new reflection-based attribute model for defining MCP tools using Go structs with tags.

## Features

- **Automatic Schema Generation**: Tool input schemas are automatically generated from Go struct types using reflection
- **Struct Tags**: Use `mcp` tags to specify field metadata (name, description, required, defaults)
- **Type-Safe Handlers**: Handler functions receive typed structs instead of generic `map[string]interface{}`
- **Backward Compatibility**: Works alongside existing manual schema definition methods

## Running the Example

```bash
go run main.go
```

The server will start on port 9986 and provide three tools:

1. **calculator** - Performs basic arithmetic operations
2. **greeting** - Generates personalized greetings in multiple languages
3. **weather** - Returns mock weather information with optional forecasts

## Struct Tag Format

The `mcp` struct tag follows this format:
```
mcp:"name,description,required,default=value"
```

- **name**: Field name in the schema (defaults to lowercase field name)
- **description**: Human-readable description of the field
- **required**: Mark field as required
- **default=value**: Set a default value for optional fields
- **-**: Skip field (not included in schema)

## Example Struct Definition

```go
type CalculatorInput struct {
    Operation string  `mcp:"operation,The operation to perform (add subtract multiply divide),required"`
    A         float64 `mcp:"a,First operand,required"`
    B         float64 `mcp:"b,Second operand,required"`
}
```

This automatically generates a JSON schema equivalent to:
```json
{
  "type": "object",
  "properties": {
    "operation": {
      "type": "string",
      "description": "The operation to perform (add subtract multiply divide)"
    },
    "a": {
      "type": "number",
      "description": "First operand"
    },
    "b": {
      "type": "number", 
      "description": "Second operand"
    }
  },
  "required": ["operation", "a", "b"]
}
```

## Type-Safe Handler

```go
func calculatorHandler(ctx context.Context, input *CalculatorInput) (interface{}, error) {
    // Direct access to typed fields: input.Operation, input.A, input.B
    switch input.Operation {
    case "add":
        return map[string]interface{}{"result": input.A + input.B}, nil
    // ...
    }
}
```

## Supported Go Types

- `string` → `"string"`
- `int`, `int64`, etc. → `"integer"`
- `float64`, `float32` → `"number"`
- `bool` → `"boolean"`
- `[]T`, `[N]T` → `"array"`
- `struct`, `map` → `"object"`
- `*T` → Same as `T` (pointer types for optional fields)
