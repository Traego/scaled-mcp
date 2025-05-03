---
title: "Dynamic Tool Server"
linkTitle: "Dynamic Tool Server"
weight: 5
description: >
  Implementing and using dynamic tool providers with the MCP server.
---

## Dynamic Tool Server Example

This example demonstrates how to create and use a dynamic tool registry with a custom tool provider. Dynamic tool providers allow you to determine available tools at runtime based on factors like session context, user permissions, or external systems.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/resources"
	"github.com/traego/scaled-mcp/pkg/server"
)

// Custom tool provider implementation
type ExampleToolProvider struct {
	// In a real implementation, you might have dependencies here
	// like database connections, service clients, etc.
}

// GetAvailableTools returns tools available for a given session
func (p *ExampleToolProvider) GetAvailableTools(ctx context.Context, sessionID string) ([]resources.Tool, error) {
	// In a real implementation, you might look up permissions for this session
	// or dynamically generate tools based on external data
	
	// For this example, we'll return different tools based on the session ID
	// to demonstrate the dynamic nature
	
	// Common tools available to all sessions
	tools := []resources.Tool{
		resources.NewTool("echo").
			WithDescription("Echo back the input message").
			WithInputs([]resources.ToolInput{
				{
					Name:        "message",
					Type:        "string",
					Description: "The message to echo back",
					Required:    true,
				},
			}).
			Build(),
	}
	
	// Special tools for specific sessions
	if len(sessionID) > 0 && sessionID[0] >= 'a' && sessionID[0] <= 'm' {
		// Sessions starting with a-m get the calculator tool
		tools = append(tools, resources.NewTool("calculator").
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
			Build())
	} else {
		// Other sessions get the greeting tool
		tools = append(tools, resources.NewTool("greeting").
			WithDescription("Generate a greeting for a person").
			WithInputs([]resources.ToolInput{
				{
					Name:        "name",
					Type:        "string",
					Description: "The name of the person to greet",
					Required:    true,
				},
				{
					Name:        "language",
					Type:        "string",
					Description: "The language for the greeting (en, es, fr)",
					Default:     "en",
				},
			}).
			Build())
	}
	
	return tools, nil
}

// HandleToolCall handles tool invocations
func (p *ExampleToolProvider) HandleToolCall(ctx context.Context, sessionID, toolID string, params map[string]interface{}) (interface{}, error) {
	// Handle the tool call based on the tool ID
	switch toolID {
	case "echo":
		message, ok := params["message"].(string)
		if !ok {
			return nil, fmt.Errorf("message parameter must be a string")
		}
		return map[string]interface{}{
			"echo": message,
		}, nil
		
	case "calculator":
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
		
	case "greeting":
		name, ok := params["name"].(string)
		if !ok {
			return nil, fmt.Errorf("name parameter must be a string")
		}
		
		language, _ := params["language"].(string)
		if language == "" {
			language = "en" // Default
		}
		
		var greeting string
		switch language {
		case "en":
			greeting = fmt.Sprintf("Hello, %s!", name)
		case "es":
			greeting = fmt.Sprintf("¡Hola, %s!", name)
		case "fr":
			greeting = fmt.Sprintf("Bonjour, %s!", name)
		default:
			greeting = fmt.Sprintf("Hello, %s!", name)
		}
		
		return map[string]interface{}{
			"greeting": greeting,
			"language": language,
		}, nil
		
	default:
		return nil, fmt.Errorf("unknown tool %s", toolID)
	}
}

func main() {
	// Configure logging
	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(logHandler))
	
	// Create server configuration
	cfg := config.DefaultConfig()
	cfg.HTTP.Port = 9985
	cfg.Session.UseInMemory = true
	
	// Create a custom tool provider
	toolProvider := &ExampleToolProvider{}
	
	// Create a dynamic tool registry with the provider
	registry := resources.NewDynamicToolRegistry(toolProvider)
	
	// Create server with the dynamic tool registry
	mcpServer, err := server.NewMcpServer(cfg,
		server.WithToolRegistry(registry),
		server.WithServerInfo("Dynamic Tool Server", "1.0.0"),
	)
	if err != nil {
		slog.Error("Failed to create server", "error", err)
		os.Exit(1)
	}
	
	// Start the server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	if err := mcpServer.Start(ctx); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
	
	slog.Info("Dynamic tool server started", "port", cfg.HTTP.Port)
	slog.Info("Tools available dynamically based on session ID")
	
	// Wait for termination signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	
	slog.Info("Shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	
	mcpServer.Stop(shutdownCtx)
	slog.Info("Server stopped")
}
```

## Key Points

- **Custom Tool Provider**: Creating a custom tool provider that implements the required interface
- **Dynamic Tool Selection**: Tools are dynamically selected based on session context
- **Centralized Tool Handling**: The provider handles both tool discovery and execution
- **Session-Based Logic**: Different sessions can see different tools based on any criteria you implement

## Tool Provider Interface

The `DynamicToolProvider` interface requires two methods:

```go
type DynamicToolProvider interface {
	// GetAvailableTools returns the tools available for a given session
	GetAvailableTools(ctx context.Context, sessionID string) ([]Tool, error)
	
	// HandleToolCall handles a tool call for a given session
	HandleToolCall(ctx context.Context, sessionID, toolID string, params map[string]interface{}) (interface{}, error)
}
```

## Real-World Applications

Dynamic tool providers are useful for several scenarios:

1. **User-Based Permissions**: Show different tools based on user roles and permissions
2. **Context-Aware Tools**: Provide tools that are relevant to the current conversation context
3. **Integration with External Systems**: Dynamically generate tools based on available external APIs
4. **A/B Testing**: Offer different tools to different user segments for testing
5. **Versioning**: Support multiple versions of tools for backward compatibility
