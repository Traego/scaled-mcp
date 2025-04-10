package mcp2024

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traego/scaled-mcp/pkg/client"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/server"
)

// TestMCPServer2024 tests the MCP server with the 2024 spec.
func TestMCPServer2024(t *testing.T) {
	// Create a server config with 2024 compatibility enabled
	cfg := config.DefaultConfig()
	cfg.BackwardCompatible20241105 = true

	// Create a new MCP server
	mcpServer, err := server.NewMcpServer(cfg)
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

	// Create client options with 2024 protocol version
	options := client.DefaultClientOptions()
	options.ProtocolVersion = client.ProtocolVersion20241105
	options.ClientInfo = client.ClientInfo{
		Name:    "test-client",
		Version: "1.0.0",
	}

	t.Run("Basic Initialization", func(t *testing.T) {
		// Create a new MCP client
		mcpClient, err := client.NewMcpClient(serverAddr, options)
		defer mcpClient.Close(context.Background())
		require.NoError(t, err, "Failed to create MCP client")

		// Connect the client
		err = mcpClient.Connect(ctx)
		require.NoError(t, err, "Failed to connect MCP client")

		// Ensure client is closed after the test
		defer mcpClient.Close(ctx)

		// Verify that the client is initialized
		assert.True(t, mcpClient.IsInitialized(), "McpClient should be initialized")

		// Verify the protocol version
		assert.Equal(t, client.ProtocolVersion20241105, mcpClient.GetProtocolVersion(),
			"Protocol version should be 2024-11-05")

		// Verify the connection method
		assert.Equal(t, client.ConnectionMethodSSE, mcpClient.GetConnectionMethod(),
			"Connection method should be SSE for 2024 spec")

		// Test sending a request
		resp, err := mcpClient.SendRequest(ctx, "roots/list", nil)
		require.NoError(t, err, "Failed to send roots/list request")
		assert.NotNil(t, resp, "Response should not be nil")
		assert.Nil(t, resp.Error, "Response should not contain an error")
	})

	t.Run("SSE Connection", func(t *testing.T) {
		// Create client options with 2024 protocol version
		options := client.DefaultClientOptions()
		options.ProtocolVersion = client.ProtocolVersion20241105
		options.ClientInfo = client.ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		}

		// Create a new MCP client
		mcpClient, err := client.NewMcpClient(serverAddr, options)
		require.NoError(t, err, "Failed to create MCP client")

		// Connect the client
		err = mcpClient.Connect(ctx)
		require.NoError(t, err, "Failed to connect MCP client")

		// Ensure client is closed after the test
		defer mcpClient.Close(ctx)

		// Verify that the client is initialized
		assert.True(t, mcpClient.IsInitialized(), "McpClient should be initialized")

		// Verify the protocol version
		assert.Equal(t, client.ProtocolVersion20241105, mcpClient.GetProtocolVersion(),
			"Protocol version should be 2024-11-05")

		// Verify the connection method
		assert.Equal(t, client.ConnectionMethodSSE, mcpClient.GetConnectionMethod(),
			"Connection method should be SSE for 2024 spec")

		// Test sending a request
		resp, err := mcpClient.SendRequest(ctx, "roots/list", nil)
		require.NoError(t, err, "Failed to send roots/list request")
		assert.NotNil(t, resp, "Response should not be nil")
		assert.Nil(t, resp.Error, "Response should not contain an error")
	})

	t.Run("Multiple Clients", func(t *testing.T) {
		// Create multiple clients
		numClients := 5
		clients := make([]client.McpClient, numClients)

		for i := 0; i < numClients; i++ {
			c, err := client.NewMcpClient(serverAddr, options)
			require.NoError(t, err, "Failed to create MCP client")
			clients[i] = c

			// Connect each client
			err = c.Connect(ctx)
			require.NoError(t, err, "Failed to connect MCP client")

			// Ensure client is closed after the test
			defer c.Close(ctx)

			// Verify the protocol version
			assert.Equal(t, client.ProtocolVersion20241105, c.GetProtocolVersion(),
				"Protocol version should be 2024-11-05")

			// Verify the connection method
			assert.Equal(t, client.ConnectionMethodSSE, c.GetConnectionMethod(),
				"Connection method should be SSE for 2024 spec")
		}

		// Verify that all clients are initialized
		for i, client := range clients {
			assert.True(t, client.IsInitialized(), "McpClient %d should be initialized", i)

			// Test sending a request with each client
			resp, err := client.SendRequest(ctx, "roots/list", nil)
			require.NoError(t, err, "Failed to send roots/list request with client %d", i)
			assert.NotNil(t, resp, "Response should not be nil for client %d", i)
			assert.Nil(t, resp.Error, "Response should not contain an error for client %d", i)
		}
	})

	t.Run("Invalid Protocol Version", func(t *testing.T) {
		// Send initialize request with invalid protocol version directly
		invalidVersionRequest := protocol.JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      "test-invalid-version",
			Method:  "initialize",
			Params: map[string]interface{}{
				"protocolVersion": "1.0.0", // Invalid version
				"capabilities": map[string]interface{}{
					"roots": map[string]interface{}{
						"listChanged": true,
					},
				},
				"client_info": map[string]interface{}{
					"name":    "test-client",
					"version": "1.0.0",
				},
			},
		}

		reqBody, err := json.Marshal(invalidVersionRequest)
		require.NoError(t, err, "Failed to marshal invalid version request")

		invalidVersionResp, err := http.Post(serverAddr+"/messages", "application/json",
			bytes.NewReader(reqBody))
		require.NoError(t, err, "Failed to make invalid version request")
		defer invalidVersionResp.Body.Close()

		// For 2024 spec, the request should be accepted
		// and the error will be sent via the SSE channel
		assert.Equal(t, http.StatusAccepted, invalidVersionResp.StatusCode,
			"Invalid protocol version should return 202 Accepted")
	})
}

// TestMCPServer2025 tests the MCP server with the 2025 spec.
func TestMCPServer2025(t *testing.T) {
	// Create a server config with 2025 compatibility (default)
	cfg := config.DefaultConfig()
	cfg.BackwardCompatible20241105 = false

	// Create a new MCP server
	mcpServer, err := server.NewMcpServer(cfg)
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
	options := client.DefaultClientOptions()
	options.ProtocolVersion = client.ProtocolVersion20250326
	options.ClientInfo = client.ClientInfo{
		Name:    "test-client",
		Version: "1.0.0",
	}

	t.Run("Basic Initialization", func(t *testing.T) {
		// Create a new MCP client
		mcpClient, err := client.NewMcpClient(serverAddr, options)
		require.NoError(t, err, "Failed to create MCP client")

		// Connect the client
		err = mcpClient.Connect(ctx)
		require.NoError(t, err, "Failed to connect MCP client")

		// Ensure client is closed after the test
		defer mcpClient.Close(ctx)

		// Verify that the client is initialized
		assert.True(t, mcpClient.IsInitialized(), "McpClient should be initialized")

		// Verify the protocol version
		assert.Equal(t, client.ProtocolVersion20250326, mcpClient.GetProtocolVersion(),
			"Protocol version should be 2025-03-26")

		// Verify the connection method
		assert.Equal(t, client.ConnectionMethodHTTP, mcpClient.GetConnectionMethod(),
			"Connection method should be HTTP for 2025 spec")

		// Test sending a request
		resp, err := mcpClient.SendRequest(ctx, "roots/list", nil)
		require.NoError(t, err, "Failed to send roots/list request")
		assert.NotNil(t, resp, "Response should not be nil")
		assert.Nil(t, resp.Error, "Response should not contain an error")
	})
}
