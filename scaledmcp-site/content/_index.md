---
title: "Scaled MCP - Horizontally Scalable Message Context Protocol"
type: docs
---

# Scaled MCP Server

[![CI Status](https://github.com/traego/scaled-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/traego/scaled-mcp/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/traego/scaled-mcp.svg)](https://pkg.go.dev/github.com/traego/scaled-mcp)
[![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=flat)](https://goreportcard.com/report/github.com/traego/scaled-mcp)
[![codecov](https://codecov.io/gh/traego/scaled-mcp/branch/main/graph/badge.svg)](https://codecov.io/gh/traego/scaled-mcp)
[![License](https://img.shields.io/github/license/traego/scaled-mcp)](https://github.com/traego/scaled-mcp/blob/main/LICENSE)

A horizontally scalable implementation of the Message Context Protocol (MCP) specification, designed according to Go best practices for libraries.

## Overview

The Scaled MCP Server is a Go library that implements the MCP 2025-03 specification with support for horizontal scaling. It's designed to be embedded in your application and provides flexible configuration options.

The project consists of three main libraries:

1. **scaled-mcp-server**: A horizontally scalable MCP server implementation
2. **scaled-mcp-client**: A 2025 spec compatible client with options
3. **mcp-http-test-harness**: A tester designed exclusively for testing MCP servers (of either 2024 or 2025-03 specs)

## Key Features

- **HTTP Transport**: Flexible HTTP transport with main `/mcp` endpoint, optional SSE endpoint, and capabilities negotiation
- **Session Management**: Distributed session management with Redis or in-memory options
- **Actor System**: Uses goakt v3.2.0 for an actor-based architecture (each client session has its own actor)
- **Horizontal Scaling**: Support for load-balanced deployments across multiple nodes
- **Tool Management**: Static and dynamic tool registry options
- **Resource Management**: Support for text, binary, and templated resources

## Installation

```bash
go get github.com/traego/scaled-mcp@latest
```

> **Note:** This library requires Go 1.24 or higher.

## Design Philosophy

The Scaled MCP libraries follow these design principles:

- **Go Best Practices**: Following idiomatic Go patterns for library modules
- **Real Testing**: Using real types and test implementations instead of mocks
- **Interface-Based Design**: Testing interfaces rather than implementation details
- **Logging**: Consistently using `slog` for structured logging
- **Horizontal Scaling**: Built from the ground up for distributed deployments

## Getting Started

To quickly get started with Scaled MCP, check out the [Getting Started](/docs/getting-started/) guide or explore the [examples](/docs/examples/).
