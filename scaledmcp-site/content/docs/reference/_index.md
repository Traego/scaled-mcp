---
title: "API Reference"
linkTitle: "Reference"
weight: 4
description: >
  Detailed API reference for the Scaled MCP libraries.
---

This section provides detailed API documentation for the key components of the Scaled MCP libraries. For complete API documentation, please refer to the [GoDoc](https://pkg.go.dev/github.com/traego/scaled-mcp).

## Core Libraries

The Scaled MCP project consists of three main libraries:

1. **scaled-mcp-server**: A horizontally scalable MCP server implementation
2. **scaled-mcp-client**: A 2025 spec compatible client with various options
3. **mcp-http-test-harness**: A testing framework for MCP servers

## Key Packages

### config

The `config` package provides configuration structures and defaults for the Scaled MCP server:

```go
// DefaultConfig returns a default configuration for the MCP server
func DefaultConfig() Config

// Config is the top-level configuration for the MCP server
type Config struct {
    HTTP                  HTTPConfig
    Session               SessionConfig
    Actor                 ActorConfig
    BackwardCompatible20241105 bool
}
```

### server

The `server` package provides the core server implementation:

```go
// NewMcpServer creates a new MCP server with the given configuration and options
func NewMcpServer(cfg config.Config, opts ...Option) (*McpServer, error)

// Start starts the server
func (s *McpServer) Start(ctx context.Context) error

// Stop stops the server
func (s *McpServer) Stop(ctx context.Context) error
```

### resources

The `resources` package provides tools, prompts, and resource management:

```go
// Tool interfaces and implementations
func NewTool(id string) *ToolBuilder
func NewStaticToolRegistry() *StaticToolRegistry
func NewDynamicToolRegistry(provider DynamicToolProvider) *DynamicToolRegistry

// Prompt interfaces and implementations
func NewPrompt(id string) *PromptBuilder
func NewStaticPromptRegistry() *StaticPromptRegistry

// Resource interfaces and implementations
func NewResource(id, name string) *ResourceBuilder
func NewResourceTemplate(id, name string) *ResourceTemplateBuilder
func NewStaticResourceRegistry() *StaticResourceRegistry
```

### transport

The `transport` package provides HTTP communication for the MCP protocol:

```go
// NewHTTPTransport creates a new HTTP transport
func NewHTTPTransport(cfg config.Config, actorSystem *goakt.ActorSystem, sessionManager session.Manager, opts ...Option) *HTTPTransport

// WithExternalRouter provides an external router to the HTTP transport
func WithExternalRouter(router chi.Router) Option

// WithPathPrefix sets a custom path prefix for the MCP endpoint
func WithPathPrefix(prefix string) Option

// WithSSEEnabled enables or disables the SSE endpoint
func WithSSEEnabled(enabled bool) Option
```

## Session Management

The Scaled MCP server provides two session store implementations:

1. **In-Memory Store**: For development and testing
2. **Redis Store**: For distributed deployments

## Actor System

The Scaled MCP server uses the goakt actor system for message passing and session management:

```go
// Session actor messages
type McpSessionInitialize struct {}
type McpSessionMessage struct {}
type McpSessionClose struct {}
```

## MCP Protocol

The Scaled MCP server implements the MCP 2025-03 specification with:

- Main `/mcp` endpoint for core protocol
- Optional SSE endpoint for events
- Capabilities negotiation to determine supported features
