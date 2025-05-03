---
title: "Getting Started"
linkTitle: "Getting Started"
weight: 1
description: >
  Quick start guide to using the Scaled MCP server.
---

## Installation

To use the Scaled MCP server in your Go project, simply add it to your dependencies:

```bash
go get github.com/traego/scaled-mcp@latest
```

> **Note:** This library requires Go 1.24 or higher.

## Basic Usage

Here's a minimal example to get a basic MCP server up and running:

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
	
	// Use in-memory session store for simplicity
	cfg.Session.UseInMemory = true
	
	// Create the MCP server with default options
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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cancelAll()

	mcpServer.Stop(shutdownCtx)

	slog.Info("MCP server stopped")
}
```

## Next Steps

After getting a basic server running, you'll likely want to:

1. [Add tools](/docs/concepts/tools/) to make your server useful
2. [Configure session management](/docs/concepts/sessions/) for distributed deployments
3. [Set up custom prompts](/docs/concepts/prompts/) for your application
4. [Explore the examples](/docs/examples/) to see more complex use cases

## Configuration Options

The Scaled MCP server provides a rich set of configuration options. Here are the key ones:

```go
// Create a server with customized configuration
cfg := config.DefaultConfig()

// HTTP settings
cfg.HTTP.Host = "0.0.0.0"  // Bind to all interfaces
cfg.HTTP.Port = 8080        // Custom port
cfg.HTTP.ReadTimeout = 30 * time.Second
cfg.HTTP.WriteTimeout = 30 * time.Second
cfg.HTTP.ShutdownTimeout = 10 * time.Second

// Session settings
cfg.Session.UseInMemory = false // Use Redis instead
cfg.Session.Redis.Address = "localhost:6379"
cfg.Session.Redis.Password = "password"
cfg.Session.Redis.DB = 0
cfg.Session.TTL = 30 * time.Minute

// Actor system settings
cfg.Actor.Address = "127.0.0.1"
cfg.Actor.Port = 9090
```
