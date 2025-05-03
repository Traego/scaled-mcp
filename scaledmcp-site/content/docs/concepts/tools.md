---
title: "Tool Management"
linkTitle: "Tools"
weight: 1
description: >
  Managing and implementing tools in the Scaled MCP server.
---

## Tool Basics

Tools are functions that can be exposed to clients connecting to your MCP server. They allow clients to perform operations that the server implements, such as calculations, data retrieval, or other function calls.

The Scaled MCP server provides two types of tool registries:

1. **Static Tool Registry**: Tools are defined and registered at startup
2. **Dynamic Tool Registry**: Tools are provided by a custom provider implementation that can determine available tools at runtime

## Creating and Registering Tools

### Static Tool Registry

The static tool registry is the simplest way to define and register tools:

```go
// Create a static tool registry
registry := resources.NewStaticToolRegistry()

// Define a tool
calculatorTool := resources.NewTool("calculator").
    WithDescription("Performs basic arithmetic operations").
    WithInputs([]resources.ToolInput{
        {
            Name:        "operation",
            Type:        "string",
            Description: "Operation to perform (add, subtract, multiply, divide)",
            Required:    true,
        },
        {
            Name:        "a",
            Type:        "number",
            Description: "First operand",
            Required:    true,
        },
        {
            Name:        "b",
            Type:        "number",
            Description: "Second operand",
            Required:    true,
        },
    }).
    Build()

// Register the tool with the registry
err := registry.RegisterTool(calculatorTool, func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    // Extract parameters
    operation, ok := params["operation"].(string)
    if !ok {
        return nil, fmt.Errorf("operation parameter must be a string")
    }
    
    a, ok := params["a"].(float64)
    if !ok {
        return nil, fmt.Errorf("a parameter must be a number")
    }
    
    b, ok := params["b"].(float64)
    if !ok {
        return nil, fmt.Errorf("b parameter must be a number")
    }
    
    // Perform the calculation
    var result float64
    switch operation {
    case "add":
        result = a + b
    case "subtract":
        result = a - b
    case "multiply":
        result = a * b
    case "divide":
        if b == 0 {
            return nil, fmt.Errorf("division by zero")
        }
        result = a / b
    default:
        return nil, fmt.Errorf("unknown operation %s", operation)
    }
    
    return map[string]interface{}{
        "result": result,
    }, nil
})
```

### Dynamic Tool Registry

For more complex scenarios, you can implement a custom tool provider:

```go
// Define a custom tool provider
type MyToolProvider struct {}

func (p *MyToolProvider) GetAvailableTools(ctx context.Context, sessionID string) ([]resources.Tool, error) {
    // Implement logic to determine available tools based on the session or other factors
    // This could involve database lookups, permission checks, etc.
    return []resources.Tool{
        // Return tools based on logic
    }, nil
}

// Create a dynamic tool registry with the provider
provider := &MyToolProvider{}
registry := resources.NewDynamicToolRegistry(provider)
```

## Tool Handler Implementation

When implementing tool handlers, follow these best practices:

1. **Parameter Validation**: Always validate parameters to ensure they have the correct types and values
2. **Error Handling**: Return clear, actionable error messages when parameters are invalid or operations fail
3. **Context Awareness**: Respect the context passed to the handler
4. **Concurrency**: Implement handlers that are safe for concurrent use

## Using Tools in the Server

Once you've defined your tools and created a registry, you can provide it to the server:

```go
// Create server with the tool registry
mcpServer, err := server.NewMcpServer(cfg,
    server.WithToolRegistry(registry),
)
```
