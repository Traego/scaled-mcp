---
title: "About Scaled MCP"
linkTitle: "About"
weight: 10
menu:
  main:
    weight: 10
---

# About Scaled MCP

A horizontally scalable implementation of the Message Context Protocol (MCP) specification

## What is Scaled MCP?

Scaled MCP is a Go library that implements the MCP 2025-03 specification with support for horizontal scaling. It's designed to be embedded in your application and provides a flexible, production-ready implementation of the protocol.

The project consists of three main libraries:

1. **scaled-mcp-server**: A horizontally scalable MCP server implementation that can be deployed across multiple nodes
2. **scaled-mcp-client**: A 2025 spec compatible client with various configuration options
3. **mcp-http-test-harness**: A testing framework designed exclusively for testing MCP servers (supporting both 2024 and 2025-03 specs)

## Design Philosophy

The Scaled MCP libraries are designed with the following principles in mind:

- **Go Best Practices**: Following idiomatic Go patterns and best practices for library modules
- **Production-Ready**: Built with real-world deployment scenarios in mind
- **Horizontal Scaling**: Support for distributed deployments from the ground up
- **Flexibility**: Modular design that allows for customization and extension
- **Testing**: Comprehensive test coverage and tools for testing your implementations

## Key Features

- **HTTP Transport**: Flexible HTTP transport with main `/mcp` endpoint, optional SSE endpoint, and capabilities negotiation
- **Session Management**: Distributed session management with Redis or in-memory options
- **Actor System**: Uses goakt v3.2.0 for an actor-based architecture, providing reliable message passing
- **Horizontal Scaling**: Support for load-balanced deployments across multiple nodes
- **Tool Management**: Both static and dynamic tool registry options
- **Resource Management**: Support for text, binary, and templated resources

## License

Scaled MCP is licensed under the [MIT License](https://github.com/traego/scaled-mcp/blob/main/LICENSE).

## Contributing

We welcome contributions to the Scaled MCP project! To contribute:

1. Fork the repository on [GitHub](https://github.com/traego/scaled-mcp)
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests to ensure they pass (`go test ./...`)
5. Commit your changes (`git commit -m 'Add some amazing feature'`)
6. Push to your branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Development Guidelines

- Follow Go best practices for library code
- Use `slog` for logging
- Write tests for new features
- Use real types and test implementations rather than mocks
- Test the interface behavior rather than internal implementation details
