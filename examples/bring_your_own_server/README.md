# Bring Your Own Server Example

This example demonstrates how to use the MCP server as an HTTP handler with your own HTTP server. This is the simplest and most flexible way to integrate MCP into your existing HTTP infrastructure.

## Key Pattern

The key pattern demonstrated here is:

```go
// Create MCP server
mcpServer, err := server.NewMcpServer(cfg)

// Create HTTP server with the MCP server as handler
httpServer := &http.Server{
    Addr:    ":8080",
    Handler: mcpServer,
}

// Start the MCP server (this doesn't start HTTP)
mcpServer.Start(ctx)

// Start the HTTP server
httpServer.ListenAndServe()
```

## Middleware Integration

The example also shows how to wrap the MCP server with middleware:

```go
// Create middleware
headerMiddleware := func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Example", "true")
        next.ServeHTTP(w, r)
    })
}

// Apply middleware to MCP server
httpServer := &http.Server{
    Addr:    ":8080",
    Handler: headerMiddleware(mcpServer),
}
```

## Running the Example

```bash
go run main.go
```

Then connect to the MCP server at http://localhost:8080/mcp using any MCP client.

## Benefits

This pattern gives you complete control over:

1. HTTP server configuration
2. Middleware application
3. TLS settings
4. Graceful shutdown

It's the recommended approach for integrating MCP into existing HTTP servers or when you need fine-grained control over the HTTP layer.
