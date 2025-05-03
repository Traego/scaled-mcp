---
title: "Actor System"
linkTitle: "Actor System"
weight: 3
description: >
  Understanding the actor-based architecture in Scaled MCP.
---

## Actor System Overview

The Scaled MCP server uses an actor-based architecture implemented with goakt v3.2.0. This approach provides an elegant solution for managing concurrent sessions and message processing in a distributed environment.

Key components of the actor system include:

1. **Session Actors**: Each client session has its own dedicated actor
2. **Message Passing**: The actor system provides a reliable message passing infrastructure
3. **Distributed Processing**: Supports distributed processing across multiple nodes

## Actor Communication

The actor system supports two types of communication:

1. **Asynchronous Communication (Tell)**: Used for one-way messages that don't require a response
2. **Synchronous Communication (Ask)**: Used for request-response patterns with timeouts

```go
// Asynchronous message (Tell)
actor.Tell(ctx, pid, message)

// Synchronous request-response with timeout (Ask)
response, err := actor.Ask(ctx, pid, message, timeout)
```

## Session Actor Lifecycle

When a client connects to the MCP server, a session actor is created and initialized:

1. **Actor Creation**: Using `actorSystem.Spawn(ctx, actorName, actorInstance)`
2. **Actor Lookup**: Using `actorSystem.ActorOf(ctx, actorName)` to find existing actors
3. **Initialization**: Session actors are initialized with a McpSessionInitialize message

For initialize requests, the actor is created but not pre-initialized, as initialization happens as part of processing the request.

## Horizontal Scaling with Actors

The actor system provides the foundation for horizontal scaling:

1. **Actor Addressability**: Actors can communicate across node boundaries
2. **Distributed State**: Session state is maintained consistently regardless of which node processes a request
3. **Transparent Location**: The actor system handles routing messages to the correct actor, regardless of its physical location

## Configuration

To configure the actor system for a clustered environment:

```go
// Configure the actor system for clustering
cfg := config.DefaultConfig()
cfg.Actor.Address = "node1.example.com" // The address of this node
cfg.Actor.Port = 9090                   // The port for actor communication
cfg.Actor.Cluster.Enabled = true        // Enable clustering
cfg.Actor.Cluster.Seeds = []string{     // Seed nodes for cluster formation
    "node1.example.com:9090",
    "node2.example.com:9090",
    "node3.example.com:9090",
}

// Create and start the server
mcpServer, err := server.NewMcpServer(cfg)
```

## Best Practices

When working with the actor system:

1. **Message Immutability**: Ensure messages passed between actors are immutable to avoid race conditions
2. **Context Propagation**: Always pass the context to actor method calls
3. **Error Handling**: Implement proper error handling within actors and when making Ask calls
4. **Actor Supervision**: Understand the supervision hierarchy to handle actor failures appropriately
