---
title: "External HTTP Server"
linkTitle: "External HTTP Server"
weight: 3
description: >
  Using your own HTTP server with the MCP transport.
---

## External HTTP Server Example

This example demonstrates how to use your own HTTP server with the MCP transport. This approach gives you full control over the HTTP server configuration, middleware, and additional routes.

```go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/server"
	"github.com/traego/scaled-mcp/pkg/transport"
)

func main() {
	// Create a server with default configuration
	cfg := config.DefaultConfig()
	cfg.Session.UseInMemory = true
	
	// Create the MCP server but don't start the HTTP server
	srv, err := server.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	
	// Create a custom router
	r := chi.NewRouter()
	
	// Add middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	
	// Add CORS middleware - important when using an external server
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	
	// Add your custom routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to the MCP server!"))
	})
	
	// Create the HTTP transport with the custom router
	httpTransport := transport.NewHTTPTransport(
		cfg,
		srv.GetActorSystem(),
		srv.GetSessionManager(),
		transport.WithExternalRouter(r),
	)
	
	// Start the MCP server without HTTP
	if err := srv.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	
	// Start the HTTP transport
	if err := httpTransport.Start(); err != nil {
		log.Fatalf("Failed to start HTTP transport: %v", err)
	}
	
	// Start your HTTP server
	log.Printf("Starting HTTP server on %s:%d", cfg.HTTP.Host, cfg.HTTP.Port)
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}
```

## Key Points

- **Server Creation**: Using `server.NewServer(cfg)` instead of `server.NewMcpServer(cfg)` to create the server without starting the HTTP transport
- **Custom Router**: Using the Chi router for custom middleware and route handling
- **CORS Configuration**: When using an external HTTP server, you need to configure CORS settings on your router
- **External Router**: Using `transport.WithExternalRouter(r)` to provide your custom router to the HTTP transport
- **Manual Transport Start**: Calling `httpTransport.Start()` to start the transport separately from the server

## Additional Configuration

When using an external HTTP server, you can customize various aspects of the HTTP transport:

```go
// Create the HTTP transport with additional options
httpTransport := transport.NewHTTPTransport(
    cfg,
    srv.GetActorSystem(),
    srv.GetSessionManager(),
    transport.WithExternalRouter(r),
    transport.WithPathPrefix("/api/mcp"),    // Custom path prefix
    transport.WithSSEEnabled(true),          // Enable SSE endpoint
    transport.WithSSEPathPrefix("/api/sse"), // Custom SSE path prefix
)
```

## Integration with Existing APIs

This approach is particularly useful when integrating MCP with an existing API server. You can maintain your current API routes and add MCP functionality alongside them.
