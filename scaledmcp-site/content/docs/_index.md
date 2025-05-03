---
title: "Documentation"
linkTitle: "Documentation"
weight: 20
menu:
  main:
    weight: 20
---

> This documentation covers the Scaled MCP server, a horizontally scalable implementation of the Message Context Protocol (MCP) specification.

## What is Scaled MCP?

The Scaled MCP Server is a Go library that implements the MCP 2025-03 specification with support for horizontal scaling. It's designed to be embedded in your application and provides flexible configuration options.

There are three main libraries included in this project:

1. **scaled-mcp-server**: A horizontally scalable MCP server implementation
2. **scaled-mcp-client**: A 2025 spec compatible client with various options
3. **mcp-http-test-harness**: A testing framework designed exclusively for testing MCP servers (compatible with both 2024 and 2025-03 specs)

## Key Features

- **HTTP Transport**: Flexible HTTP transport with main `/mcp` endpoint, optional SSE endpoint, and capabilities negotiation
- **Session Management**: Distributed session management with Redis or in-memory options
- **Actor System**: Uses goakt v3.2.0 for an actor-based architecture (each client session has its own actor)
- **Horizontal Scaling**: Support for load-balanced deployments across multiple nodes
- **Tool Management**: Static and dynamic tool registry options
- **Resource Management**: Support for text, binary, and templated resources
