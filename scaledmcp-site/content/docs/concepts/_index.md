---
title: "Core Concepts"
linkTitle: "Concepts"
weight: 2
description: >
  Understanding the key components and concepts of the Scaled MCP implementation.
---

Scaled MCP is designed with a flexible, modular architecture that follows Go best practices for library modules. This section explains the key concepts and components that make up the system.

## Architecture Overview

The Scaled MCP architecture consists of several key components:

1. **HTTP Transport**: Handles the HTTP communication layer with flexible configuration options
2. **Session Management**: Manages client sessions across distributed deployments
3. **Actor System**: Provides a reliable message-passing infrastructure for handling client requests
4. **Tool Registry**: Manages the available tools that can be used by clients
5. **Prompt Registry**: Stores and manages prompt templates
6. **Resource Registry**: Manages resources that can be accessed by clients

These components work together to create a scalable, reliable MCP implementation that can be used in production environments.
