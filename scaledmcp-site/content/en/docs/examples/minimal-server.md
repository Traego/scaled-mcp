---
title: "Minimal Server"
linkTitle: "Minimal Server"
weight: 1
description: >
  A bare-bones MCP server implementation.
---

## Minimal Server Example

This example demonstrates the simplest possible implementation of an MCP server. It uses the default configuration with minimal customization and an in-memory session store.

```go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/server"
)

func main() {
	ctx, cancelAll := context.WithCancel(context.Background())

	// Configure logging
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// Create a server configuration
	cfg := config.DefaultConfig()
	cfg.BackwardCompatible20241105 = true

	// Customize configuration if needed
	cfg.HTTP.Port = 9985

	// Create the MCP server with default options
	// This will create a new HTTP server internally
	mcpServer, err := server.NewMcpServer(cfg)
	if err != nil {
		slog.Error("Failed to create MCP server", "error", err)
		os.Exit(1)
	}

	// Start the server
	if err := mcpServer.Start(ctx); err != nil {
		slog.Error("Failed to start MCP server", "error", err)
		os.Exit(1)
	}

	slog.Info("MCP server started", "port", cfg.HTTP.Port)

	// Wait for termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down MCP server...")

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cancelAll()

	mcpServer.Stop(ctx)

	slog.Info("MCP server stopped")
}
```

## Key Points

- **Default Configuration**: Using `config.DefaultConfig()` provides sensible defaults to get started quickly
- **BackwardCompatible**: The `BackwardCompatible20241105` flag enables compatibility with the 2024-11-05 MCP specification
- **Graceful Shutdown**: The server handles termination signals and performs a clean shutdown
- **In-Memory Sessions**: By default, sessions are stored in memory, which is suitable for development

## Testing the Server

Once running, you can test the server using any MCP client by connecting to `http://localhost:9985/mcp`. The server doesn't expose any custom tools in this example, but it will respond to basic MCP protocol messages.
