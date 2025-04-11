//go:build race
// +build race

package server

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traego/scaled-mcp/pkg/client"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/resources"
	"github.com/traego/scaled-mcp/test/testutils"
	"strconv"
	"testing"
)

// TestMCPServer2025 tests the MCP server with the 2025 spec.
func TestMCPServer2025(t *testing.T) {
	// Get a random available port
	port, err := testutils.GetAvailablePort()
	require.NoError(t, err, "Failed to get available port")

	// Create a server config with 2025 compatibility (default)
	cfg := config.DefaultConfig()
	cfg.BackwardCompatible20241105 = false
	cfg.HTTP.Port = port

	registry := resources.NewStaticToolRegistry()
	err = registry.RegisterTool(resources.Tool{
		Name:        "Amazing Tool",
		Description: "Does amazing things",
		InputSchema: resources.InputSchema{},
	}, func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		return nil, nil
	})
	require.NoError(t, err)

	// Create a new MCP server
	mcpServer, err := NewMcpServer(cfg, WithToolRegistry(registry))
	require.NoError(t, err, "Failed to create MCP server")

	// Start the server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = mcpServer.Start(ctx)
	require.NoError(t, err, "Failed to start MCP server")

	defer cancel()
	// Ensure server is stopped after the test
	//defer mcpServer.Stop(ctx)

	// Get the server's HTTP address
	serverAddr := "http://localhost:" + strconv.Itoa(cfg.HTTP.Port)

	// Create client options with 2025 protocol version
	options := client.DefaultClientOptions()
	options.ProtocolVersion = protocol.ProtocolVersion20250326
	options.ClientInfo = client.ClientInfo{
		Name:    "test-client",
		Version: "1.0.0",
	}

	t.Run("Basic Initialization", func(t *testing.T) {
		// Create a new MCP client
		mcpClient, err := client.NewMcpClient(serverAddr, options)
		defer func() {
			_ = mcpClient.Close(context.Background())
		}()
		require.NoError(t, err, "Failed to create MCP client")

		// Connect the client
		err = mcpClient.Connect(ctx)
		require.NoError(t, err, "Failed to connect MCP client")

		// Verify that the client is initialized
		assert.True(t, mcpClient.IsInitialized(), "McpClient should be initialized")

		// Verify the protocol version
		assert.Equal(t, protocol.ProtocolVersion20250326, mcpClient.GetProtocolVersion(),
			"Protocol version should be 2025-03-26")

		// Verify the connection method
		assert.Equal(t, client.ConnectionMethodHTTP, mcpClient.GetConnectionMethod(),
			"Connection method should be HTTP for 2025 spec")

		// Test sending a request
		resp, err := mcpClient.SendRequest(ctx, "tools/list", nil)
		require.NoError(t, err, "Failed to send tools/list request")
		assert.NotNil(t, resp, "Response should not be nil")

		// Type assert the response result to access nested fields
		resultMap, ok := resp.Result.(map[string]interface{})
		require.True(t, ok, "Result should be a map")

		tools, ok := resultMap["tools"].([]interface{})
		require.True(t, ok, "tools should be a slice")
		require.NotEmpty(t, tools, "tools slice should not be empty")

		require.Len(t, tools, 1, "tools len should be 1")

		// Check that the first tool is initialize or another expected value
		toolName, ok := tools[0].(map[string]interface{})["name"].(string)
		require.True(t, ok, "tool should be a string")
		assert.Equal(t, toolName, "Amazing Tool")

		assert.Nil(t, resp.Error, "Response should not contain an error")
	})
}
