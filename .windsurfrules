This is a library module, please follow go best practices for libraries.

There are 3 libraries here - scaled-mcp-server, which is a horizontally scalable mcp server, scaled-mcp-client which is a 2025 spec compatible client with options, and mcp-http-test-harness which is a tester designed exclusively for testing mcp servers (of either 2024 or 2025-03 specs)

Please only use slog for logging.

Please check in all the appropriate places BEFORE implementing an interface. Only implement interfaces if you're 100% sure they don't exist.

If you have an interface - don't ever create an adapter interface to simply wrap it, instead, pass that interface around.

Please don't use mocks - instead, use real types and test implementations, in memory implemenations, and when possible use concrete servers that call into in memory repositories.

When writing tests, please always examine the code under test to make sure you're using it correctly, and ask questions if it's unclear how to work with something.

When writing tests - be sure to test the interface, and try not to test the inners of the implementation too much. This is part of why I don't like mocks, they make assumptions about how things function internally. So, try your best to test method calls that are exercising the interface or the outward facing behavior of the implementation.