---
title: "Static Tool Server"
linkTitle: "Static Tool Server"
weight: 2
description: >
  Implementing and registering static tools with the MCP server.
---

## Static Tool Server Example

This example demonstrates how to create and register static tools with the MCP server. It shows how to create a static tool registry, define tools with different input parameter styles, and register handlers for the tools.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/resources"
	"github.com/traego/scaled-mcp/pkg/server"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Configure logging
	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(logHandler))

	// Create registries
	toolRegistry := resources.NewStaticToolRegistry()
	promptRegistry := resources.NewStaticPromptRegistry()
	resourceRegistry := resources.NewStaticResourceRegistry()

	// Register tools with the server
	registerEchoTool(toolRegistry)
	registerGreetingTool(toolRegistry)

	// Create server with the registries
	cfg := config.DefaultConfig()
	cfg.BackwardCompatible20241105 = true
	cfg.HTTP.Port = 9985

	mcpServer, err := server.NewMcpServer(cfg,
		server.WithToolRegistry(toolRegistry),
		server.WithPromptRegistry(promptRegistry),
		server.WithResourceRegistry(resourceRegistry),
	)
	if err != nil {
		slog.Error("Failed to create MCP server", "error", err)
		os.Exit(1)
	}

	// Start the server
	go func() {
		if err := mcpServer.Start(ctx); err != nil && err != http.ErrServerClosed {
			slog.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("Example static registries server is available")
	slog.Info("Tools available: echo, greeting")

	// Wait for termination signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	sctx, c2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer c2()

	// Shutdown the server
	mcpServer.Stop(sctx)
}

// registerEchoTool registers an echo tool with the server
func registerEchoTool(registry *resources.StaticToolRegistry) {
	// Create the echo tool
	echoTool := resources.NewTool("echo").
		WithDescription("Echo back the input message").
		WithInputs([]resources.ToolInput{
			{
				Name:        "message",
				Type:        "string",
				Description: "The message to echo back",
				Required:    true,
			},
		}).
		Build()

	// Register the tool with its handler
	err := registry.RegisterTool(echoTool, func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		message, ok := params["message"].(string)
		if !ok {
			return nil, fmt.Errorf("message parameter must be a string")
		}

		return map[string]interface{}{
			"echo": message,
		}, nil
	})

	if err != nil {
		slog.Error("Failed to register echo tool", "error", err)
	}
}

// registerGreetingTool registers a greeting tool with the server
func registerGreetingTool(registry *resources.StaticToolRegistry) {
	// Create the greeting tool
	greetingTool := resources.NewTool("greeting").
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
		Build()

	// Register the tool with its handler
	err := registry.RegisterTool(greetingTool, func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
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
	})

	if err != nil {
		slog.Error("Failed to register greeting tool", "error", err)
	}
}
```

## Key Points

- **Tool Registry Creation**: Using `resources.NewStaticToolRegistry()` to create a registry for static tools
- **Tool Definition Options**: The example shows defining tools using the `WithInputs` method for cleaner code
- **Parameter Validation**: Handlers validate that parameters have the expected types
- **Error Handling**: Proper error handling for parameter validation and registration
- **Logging**: Using structured logging with `slog` for better observability

## Alternative Tool Definition

Besides the `WithInputs` method shown above, you can also define tools using individual parameter methods:

```go
calculatorTool := resources.NewTool("calculator").
    WithDescription("Performs basic arithmetic operations").
    WithString("operation").
    Required().
    Description("Operation to perform (add, subtract, multiply, divide)").
    Add().
    WithNumber("a").
    Required().
    Description("First operand").
    Add().
    WithNumber("b").
    Required().
    Description("Second operand").
    Add().
    Build()
```

## Testing the Static Tool Server

Once running, you can test the server using any MCP client by connecting to `http://localhost:9985/mcp`. The server exposes the `echo` and `greeting` tools, which you can use in your MCP client interactions.
