package client

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/resources"
	"github.com/traego/scaled-mcp/pkg/server"
	"github.com/traego/scaled-mcp/test/testutils"
	"log/slog"
)

// TestToolsClient tests the tool-related client functionality
func TestToolsClient(t *testing.T) {
	// Get a random available port
	port, err := testutils.GetAvailablePort()
	require.NoError(t, err, "Failed to get available port")

	// Create a server config with 2025 compatibility (default)
	cfg := config.DefaultConfig()
	cfg.BackwardCompatible20241105 = false
	cfg.HTTP.Port = port

	// Create a tool registry with test tools
	registry := resources.NewStaticToolRegistry()

	// Register a test tool that returns a simple response
	err = registry.RegisterTool(protocol.Tool{
		Name:        "test/echo",
		Description: "Echoes back the input message",
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.SchemaProperty{
				"message": {
					Type:        "string",
					Description: "Message to echo",
				},
			},
		},
	}, func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		// Simply echo back the message
		return map[string]interface{}{
			"echo": params["message"],
		}, nil
	})
	require.NoError(t, err)

	// Register another test tool for testing FindTool
	err = registry.RegisterTool(protocol.Tool{
		Name:        "test/add",
		Description: "Adds two numbers",
		InputSchema: protocol.InputSchema{
			Type: "object",
			Properties: map[string]protocol.SchemaProperty{
				"a": {
					Type:        "number",
					Description: "First number",
				},
				"b": {
					Type:        "number",
					Description: "Second number",
				},
			},
		},
	}, func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		// Add the two numbers
		a, _ := params["a"].(float64)
		b, _ := params["b"].(float64)
		return map[string]interface{}{
			"result": a + b,
		}, nil
	})
	require.NoError(t, err)

	// Create a new MCP server with the tool registry
	mcpServer, err := server.NewMcpServer(cfg, server.WithToolRegistry(registry))
	require.NoError(t, err, "Failed to create MCP server")

	// Start the server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = mcpServer.Start(ctx)
	require.NoError(t, err, "Failed to start MCP server")

	// Ensure server is stopped after the test
	defer mcpServer.Stop(ctx)

	// Get the server's HTTP address
	serverAddr := "http://localhost:" + strconv.Itoa(cfg.HTTP.Port)

	// Create client options with 2025 protocol version
	options := DefaultClientOptions()
	options.ProtocolVersion = protocol.ProtocolVersion20250326
	options.ClientInfo = ClientInfo{
		Name:    "test-client",
		Version: "1.0.0",
	}

	// Create a new MCP client
	mcpClient, err := NewMcpClient(serverAddr, options)
	require.NoError(t, err, "Failed to create MCP client")

	defer func() {
		_ = mcpClient.Close(context.Background())
	}()

	// Connect the client
	err = mcpClient.Connect(ctx)
	require.NoError(t, err, "Failed to connect MCP client")

	// Verify that the client is initialized
	assert.True(t, mcpClient.IsInitialized(), "McpClient should be initialized")

	t.Run("ListTools", func(t *testing.T) {
		// Test the ListTools method
		toolsList, err := mcpClient.ListTools(ctx)
		require.NoError(t, err, "Failed to list tools")
		require.NotNil(t, toolsList, "ToolsList should not be nil")

		// Verify that we have the expected number of tools
		require.Len(t, toolsList.Tools, 2, "Should have 2 tools")

		// Verify tool names
		toolNames := make([]string, 0, len(toolsList.Tools))
		for _, tool := range toolsList.Tools {
			toolNames = append(toolNames, tool.Name)
		}

		assert.Contains(t, toolNames, "test/echo", "Should have test/echo tool")
		assert.Contains(t, toolNames, "test/add", "Should have test/add tool")

		// Log the tools for debugging
		slog.Info("Retrieved tools", "tools", toolNames)
	})

	t.Run("FindTool", func(t *testing.T) {
		// Test finding a tool that exists
		echoTool, err := mcpClient.FindTool(ctx, "test/echo")
		require.NoError(t, err, "Failed to find test/echo tool")
		require.NotNil(t, echoTool, "Tool should not be nil")

		assert.Equal(t, "test/echo", echoTool.Name, "Tool name should match")
		assert.Equal(t, "Echoes back the input message", echoTool.Description, "Tool description should match")

		// Test finding a tool that doesn't exist
		nonExistentTool, err := mcpClient.FindTool(ctx, "non/existent")
		assert.Error(t, err, "Should return error for non-existent tool")
		assert.Nil(t, nonExistentTool, "Tool should be nil for non-existent tool")
		assert.Contains(t, err.Error(), "tool not found", "Error should indicate tool not found")
	})

	t.Run("CallTool", func(t *testing.T) {
		// First, get the list of tools to verify they exist
		toolsList, err := mcpClient.ListTools(ctx)
		require.NoError(t, err, "Failed to list tools")

		// Verify that our test tools exist
		toolNames := make([]string, 0, len(toolsList.Tools))
		for _, tool := range toolsList.Tools {
			toolNames = append(toolNames, tool.Name)
		}
		slog.Info("Available tools for testing", "tools", toolNames)

		// Test calling the echo tool
		echoParams := map[string]interface{}{
			"message": "Hello, world!",
		}

		slog.Info("Calling test/echo tool", "params", echoParams)
		echoResp, err := mcpClient.CallTool(ctx, "test/echo", echoParams)
		if err != nil {
			t.Logf("Error calling test/echo tool: %v", err)
			t.FailNow()
		}
		require.NoError(t, err, "Failed to call test/echo tool")
		require.NotNil(t, echoResp, "Response should not be nil")

		// Verify the response
		slog.Info("Echo response received", "response", echoResp)

		resultMap, ok := echoResp.Result.(map[string]interface{})
		if !ok {
			t.Logf("Result is not a map: %T %+v", echoResp.Result, echoResp.Result)
			t.FailNow()
		}
		require.True(t, ok, "Result should be a map")

		echo, ok := resultMap["echo"].(string)
		if !ok {
			t.Logf("Echo is not a string: %T %+v", resultMap["echo"], resultMap["echo"])
			t.FailNow()
		}
		require.True(t, ok, "echo should be a string")
		assert.Equal(t, "Hello, world!", echo, "Echo response should match input")
	})
}
