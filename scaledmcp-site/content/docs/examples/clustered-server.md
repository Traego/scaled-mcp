---
title: "Clustered Server"
linkTitle: "Clustered Server"
weight: 4
description: >
  Setting up a horizontally scaled deployment with multiple server instances.
---

## Clustered Server Example

This example demonstrates how to configure and run a clustered deployment of the Scaled MCP server. Clustered deployments are essential for horizontal scaling, providing high availability and increased capacity.

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
	"github.com/traego/scaled-mcp/pkg/resources"
	"github.com/traego/scaled-mcp/pkg/server"
)

func main() {
	// Get node information from environment variables
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		nodeName = "node1"
	}
	
	nodeAddress := os.Getenv("NODE_ADDRESS")
	if nodeAddress == "" {
		nodeAddress = "127.0.0.1"
	}
	
	nodePort := os.Getenv("NODE_PORT")
	if nodePort == "" {
		nodePort = "9090"
	}
	
	// Configure logging
	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(logHandler))
	
	// Create configuration for clustered deployment
	cfg := config.DefaultConfig()
	
	// Configure actor system for clustering
	cfg.Actor.Name = nodeName
	cfg.Actor.Address = nodeAddress
	cfg.Actor.Port = nodePort
	cfg.Actor.Cluster.Enabled = true
	
	// Add seed nodes for cluster formation
	cfg.Actor.Cluster.Seeds = []string{
		"node1:9090",
		"node2:9090",
		"node3:9090",
	}
	
	// Use Redis for distributed session management
	cfg.Session.UseInMemory = false
	cfg.Session.Redis.Address = os.Getenv("REDIS_ADDRESS")
	if cfg.Session.Redis.Address == "" {
		cfg.Session.Redis.Address = "localhost:6379"
	}
	cfg.Session.Redis.Password = os.Getenv("REDIS_PASSWORD")
	cfg.Session.Redis.DB = 0
	cfg.Session.TTL = 30 * time.Minute
	
	// Create static tool registry
	registry := resources.NewStaticToolRegistry()
	
	// Define and register a simple echo tool
	echoTool := resources.NewTool("echo").
		WithDescription("Echo back the input message").
		WithString("message").
		Required().
		Description("The message to echo back").
		Add().
		Build()
	
	registry.RegisterTool(echoTool)
	
	// Create the server with the tool registry
	mcpServer, err := server.NewMcpServer(cfg,
		server.WithToolRegistry(registry),
		server.WithServerInfo("Clustered MCP Server", "1.0.0"),
	)
	if err != nil {
		slog.Error("Failed to create server", "error", err)
		os.Exit(1)
	}
	
	// Set up the tool handler
	registry.SetToolHandler("echo", func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		message := params["message"].(string)
		return map[string]interface{}{
			"echo": message,
			"node": nodeName,
		}, nil
	})
	
	// Start the server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	if err := mcpServer.Start(ctx); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
	
	slog.Info("Clustered MCP server started", 
		"node", nodeName, 
		"address", nodeAddress, 
		"port", nodePort)
	
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

- **Actor Configuration**: Each node needs unique identification (`Name`, `Address`, `Port`)
- **Cluster Configuration**: Enable clustering and provide seed nodes for cluster formation
- **Redis for Sessions**: Using Redis for distributed session management is required for clustered deployments
- **Environment Variables**: Using environment variables for configuration allows for easier containerization

## Deployment Considerations

When deploying a clustered MCP server, consider the following:

### 1. Load Balancing

Requests can be distributed across multiple nodes:

```
        ┌─────────────┐
        │ Load        │
        │ Balancer    │
        └───────┬─────┘
                │
    ┌───────────┴───────────┐
    │                       │
┌───▼───┐             ┌─────▼───┐
│ MCP   │             │ MCP     │
│ Node1 │             │ Node2   │
└───┬───┘             └─────┬───┘
    │                       │
    └─────────┬─────────────┘
              │
        ┌─────▼─────┐
        │           │
        │  Redis    │
        │           │
        └───────────┘
```

### 2. Session Affinity

While not required due to the distributed session management, session affinity (sticky sessions) can improve performance by reducing session data retrieval from Redis.

### 3. Actor Clustering

The goakt actor system automatically handles routing messages to the correct actor, regardless of which node it's on. This is transparent to the application code.

### 4. Containerization

This example is well-suited for containerization with Docker and orchestration with Kubernetes:

```yaml
# Docker Compose example
version: '3'
services:
  redis:
    image: redis:alpine
    ports:
      - "6379:6379"
    
  node1:
    build: .
    environment:
      - NODE_NAME=node1
      - NODE_ADDRESS=node1
      - NODE_PORT=9090
      - REDIS_ADDRESS=redis:6379
    ports:
      - "8081:8080"
    depends_on:
      - redis
    
  node2:
    build: .
    environment:
      - NODE_NAME=node2
      - NODE_ADDRESS=node2
      - NODE_PORT=9090
      - REDIS_ADDRESS=redis:6379
    ports:
      - "8082:8080"
    depends_on:
      - redis
```
