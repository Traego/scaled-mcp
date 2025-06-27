# Generics-Based Tools Example

This example demonstrates the new generics-based tool registration with two type parameters for input and output schemas.

## Features

- **Compile-time Type Safety**: Both input and output types are checked at compile time
- **Automatic Schema Generation**: Tool schemas are automatically generated from Go struct types for both input and output
- **Zero Reflection at Registration**: Uses generics instead of reflection for better performance
- **Better IDE Support**: Full IntelliSense and type checking in IDEs

## Usage

### Define Input and Output Structs

```go
type CalculatorInput struct {
    Operation string  `mcp:"operation,The operation to perform,required"`
    A         float64 `mcp:"a,First operand,required"`
    B         float64 `mcp:"b,Second operand,required"`
}

type CalculatorOutput struct {
    Result    float64 `mcp:"result,The calculation result,required"`
    Operation string  `mcp:"operation,The operation performed,required"`
}
```

### Create Type-Safe Handler

```go
func calculatorHandler(ctx context.Context, input *CalculatorInput) (*CalculatorOutput, error) {
    result := input.A + input.B
    return &CalculatorOutput{
        Result:    result,
        Operation: input.Operation,
    }, nil
}
```

### Register with Generics

```go
// Method 1: Direct registration
err := registry.RegisterStructToolWithTypes("calculator", "Performs arithmetic", calculatorHandler)

// Method 2: Convenience function
err := resources.RegisterStructToolWithTypes(registry, "calculator", "Performs arithmetic", calculatorHandler)
```

## Running the Example

```bash
cd examples/generics_tools_server
go run main.go
```

The server will be available on port 9988 with the following tools:
- `calculator_typed`: Performs arithmetic operations with typed input/output
- `greeting_typed`: Generates greetings with typed input/output

## Benefits over Reflection-Based Approach

1. **Compile-time Safety**: Errors caught at compile time instead of runtime
2. **Better Performance**: No reflection overhead at registration time
3. **IDE Support**: Full type checking and IntelliSense
4. **Self-Documenting**: Handler signatures clearly show input and output types
5. **Future-Proof**: Aligns with MCP June 2025 specification features

## Backward Compatibility

This generics-based approach works alongside the existing reflection-based methods. You can mix both approaches in the same application without any issues.
