---
title: "Session Management"
linkTitle: "Sessions"
weight: 2
description: >
  Managing distributed sessions across multiple server instances.
---

## Session Management Overview

In the Scaled MCP server, session management is a critical component for horizontal scaling. Each client session must be tracked consistently, even as requests are distributed across multiple server instances.

The Scaled MCP implementation provides two session store options:

1. **In-Memory Store**: Simple, non-distributed storage for development and testing
2. **Redis Store**: Distributed session storage suitable for production deployments

## In-Memory Session Store

The in-memory session store is suitable for development and testing environments where you're running a single instance of the server:

```go
// Configure the server to use in-memory session store
cfg := config.DefaultConfig()
cfg.Session.UseInMemory = true

// Create and start the server
mcpServer, err := server.NewMcpServer(cfg)
```

## Redis Session Store

For production environments with multiple server instances, use the Redis session store:

```go
// Configure the server to use Redis session store
cfg := config.DefaultConfig()
cfg.Session.UseInMemory = false
cfg.Session.Redis.Address = "redis:6379"
cfg.Session.Redis.Password = "your-password" // Optional
cfg.Session.Redis.DB = 0                    // Optional
cfg.Session.TTL = 30 * time.Minute         // Session timeout

// Create and start the server
mcpServer, err := server.NewMcpServer(cfg)
```

## Session Lifecycle

When a client connects to the MCP server, the following session lifecycle events occur:

1. **Session Creation**: When a client sends an initialize request, a new session is created
2. **Actor Creation**: A new session actor is created and assigned to the session
3. **Session Management**: The session is tracked in the session store with a TTL
4. **Session Renewal**: Each client interaction renews the session TTL
5. **Session Termination**: Sessions are terminated when the client disconnects or the TTL expires

## Custom Session Management

If you need custom session management beyond what's provided, you can implement the `SessionManager` interface:

```go
type SessionManager interface {
    InitializeSession(ctx context.Context, sessionID string, systemPid *actor.PID) error
    GetSessionActor(ctx context.Context, sessionID string) (*actor.PID, error)
    CloseSession(ctx context.Context, sessionID string) error
}
```

Then, provide your custom implementation when creating the server:

```go
// Create a custom session manager
mySessionManager := NewMyCustomSessionManager()

// Create the server with the custom session manager
mcpServer, err := server.NewMcpServer(cfg,
    server.WithSessionManager(mySessionManager),
)
```
