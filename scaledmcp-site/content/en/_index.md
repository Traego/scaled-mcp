---
title: "Scaled MCP - Horizontally Scalable Message Context Protocol"
type: docs
bookToc: false
bookFlatSection: true
---

# Scaled MCP Server

[![CI Status](https://github.com/traego/scaled-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/traego/scaled-mcp/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/traego/scaled-mcp.svg)](https://pkg.go.dev/github.com/traego/scaled-mcp)
[![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=flat)](https://goreportcard.com/report/github.com/traego/scaled-mcp)
[![codecov](https://codecov.io/gh/traego/scaled-mcp/branch/main/graph/badge.svg)](https://codecov.io/gh/traego/scaled-mcp)
[![License](https://img.shields.io/github/license/traego/scaled-mcp)](https://github.com/traego/scaled-mcp/blob/main/LICENSE)

A horizontally scalable MCP (Message Context Protocol) server implementation that supports load-balanced deployments.

## Overview

The Scaled MCP Server is a Go library that implements the MCP 2025-03 specification with support for horizontal scaling. It's designed to be embedded in your application and provides flexible configuration options.

## Features

- **HTTP Transport**: Flexible HTTP transport with main `/mcp` endpoint, optional SSE endpoint, and capabilities negotiation
- **Session Management**: Distributed session management with Redis or in-memory options
- **Actor System**: Uses an actor-based architecture for handling sessions and message routing
- **Horizontal Scaling**: Support for load-balanced deployments across multiple nodes

## Installation

```bash
go get github.com/traego/scaled-mcp@latest
```

> **Note:** This library requires Go 1.24 or higher.

## Key Libraries

The Scaled MCP project consists of three main libraries:

1. **scaled-mcp-server**: A horizontally scalable MCP server implementation
2. **scaled-mcp-client**: A 2025 spec compatible client with options
3. **mcp-http-test-harness**: A tester designed exclusively for testing MCP servers (of either 2024 or 2025-03 specs)

## Next Steps

Continue reading the documentation to learn more about:

- [Getting Started](/docs/getting-started) - A quick start guide
- [Core Concepts](/docs/concepts) - Understanding the key components
- [Examples](/docs/examples) - Example code for different use cases
- [API Reference](/docs/reference) - Detailed API documentation
