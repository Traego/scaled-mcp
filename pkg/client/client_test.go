package client

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/server"
)

// TestMCPClient2024 tests the MCP client with a 2024-compatible server
func TestMCPClient2024(t *testing.T) {
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
	options := DefaultClientOptions()
	options.ProtocolVersion = ProtocolVersion20241105
	options.ClientInfo = ClientInfo{
		Name:    "test-client",
		Version: "1.0.0",
	}

	// Create a new MCP client
	client, err := NewMcpClient(serverAddr, options)
	require.NoError(t, err, "Failed to create MCP client")

	// Connect to the server
	err = client.Connect(ctx)
	require.NoError(t, err, "Failed to connect MCP client")

	// Ensure client is closed after the test
	defer client.Close(ctx)

	// Verify that the client is initialized
	assert.True(t, client.IsInitialized(), "McpClient should be initialized")

	// Test sending a request
	resp, err := client.SendRequest(ctx, "roots/list", nil)
	require.NoError(t, err, "Failed to send roots/list request")
	assert.NotNil(t, resp, "Response should not be nil")
	assert.Nil(t, resp.Error, "Response should not contain an error")

	// Test sending a notification
	err = client.SendNotification(ctx, "notifications/test", map[string]interface{}{
		"message": "test notification",
	})
	require.NoError(t, err, "Failed to send test notification")
}

// TestMCPClient2025 tests the MCP client with a 2025-compatible server
func TestMCPClient2025(t *testing.T) {
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
	options := DefaultClientOptions()
	options.ProtocolVersion = ProtocolVersion20250326
	options.ClientInfo = ClientInfo{
		Name:    "test-client",
		Version: "1.0.0",
	}

	// Create a new MCP client
	client, err := NewMcpClient(serverAddr, options)
	require.NoError(t, err, "Failed to create MCP client")

	// Connect to the server
	err = client.Connect(ctx)
	require.NoError(t, err, "Failed to connect MCP client")

	// Ensure client is closed after the test
	defer client.Close(ctx)

	// Verify that the client is initialized
	assert.True(t, client.IsInitialized(), "McpClient should be initialized")

	// Test sending a request
	resp, err := client.SendRequest(ctx, "roots/list", nil)
	require.NoError(t, err, "Failed to send roots/list request")
	assert.NotNil(t, resp, "Response should not be nil")
	assert.Nil(t, resp.Error, "Response should not contain an error")

	// Test sending a notification
	err = client.SendNotification(ctx, "notifications/test", map[string]interface{}{
		"message": "test notification",
	})
	require.NoError(t, err, "Failed to send test notification")
}

// TestMCPClientAuto tests the MCP client with auto protocol version detection
func TestMCPClientAuto(t *testing.T) {
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

	// Create client options with auto protocol version detection
	options := DefaultClientOptions()
	options.ProtocolVersion = ProtocolVersionAuto
	options.ClientInfo = ClientInfo{
		Name:    "test-client",
		Version: "1.0.0",
	}

	// Create a new MCP client
	client, err := NewMcpClient(serverAddr, options)
	require.NoError(t, err, "Failed to create MCP client")

	// Connect to the server
	err = client.Connect(ctx)
	require.NoError(t, err, "Failed to connect MCP client")

	// Ensure client is closed after the test
	defer client.Close(ctx)

	// Verify that the client is initialized
	assert.True(t, client.IsInitialized(), "McpClient should be initialized")

	// Test sending a request
	resp, err := client.SendRequest(ctx, "roots/list", nil)
	require.NoError(t, err, "Failed to send roots/list request")
	assert.NotNil(t, resp, "Response should not be nil")
	assert.Nil(t, resp.Error, "Response should not contain an error")
}

// TestMCPClientEventHandler tests the event handler functionality
func TestMCPClientEventHandler(t *testing.T) {
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

	// Create client options
	options := DefaultClientOptions()
	options.ProtocolVersion = ProtocolVersion20241105
	options.ClientInfo = ClientInfo{
		Name:    "test-client",
		Version: "1.0.0",
	}

	// Create a new MCP client
	client, err := NewMcpClient(serverAddr, options)
	require.NoError(t, err, "Failed to create MCP client")

	// Create a channel to receive events
	eventCh := make(chan *protocol.JSONRPCMessage, 1)

	// Add an event handler
	handler := EventHandlerFunc(func(event *protocol.JSONRPCMessage) {
		select {
		case eventCh <- event:
		default:
			// Channel is full, which shouldn't happen in this test
			t.Error("Event channel is full")
		}
	})
	client.AddEventHandler(handler)

	// Connect to the server
	err = client.Connect(ctx)
	require.NoError(t, err, "Failed to connect MCP client")

	// Ensure client is closed after the test
	defer client.Close(ctx)

	// Send a request that should trigger an event
	_, err = client.SendRequest(ctx, "roots/list", nil)
	require.NoError(t, err, "Failed to send roots/list request")

	// Wait for the event
	select {
	case event := <-eventCh:
		assert.NotNil(t, event, "Event should not be nil")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for event")
	}

	// Remove the event handler
	client.RemoveEventHandler(handler)
}

// TestMCPClientErrors tests error handling in the MCP client
func TestMCPClientErrors(t *testing.T) {
	// Create a server config
	cfg := config.DefaultConfig()

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

	// Test with invalid server URL
	t.Run("Invalid Server URL", func(t *testing.T) {
		options := DefaultClientOptions()
		client, err := NewMcpClient("http://invalid-server:12345", options)
		require.NoError(t, err, "Creating client with invalid URL should not fail")

		err = client.Connect(ctx)
		assert.Error(t, err, "Connect with invalid URL should fail")
	})

	// Test sending request without connecting
	t.Run("Request Without Connect", func(t *testing.T) {
		options := DefaultClientOptions()
		client, err := NewMcpClient(serverAddr, options)
		require.NoError(t, err, "Failed to create MCP client")

		_, err = client.SendRequest(ctx, "roots/list", nil)
		assert.Error(t, err, "Sending request without connecting should fail")
	})

	// Test sending notification without connecting
	t.Run("Notification Without Connect", func(t *testing.T) {
		options := DefaultClientOptions()
		client, err := NewMcpClient(serverAddr, options)
		require.NoError(t, err, "Failed to create MCP client")

		err = client.SendNotification(ctx, "notifications/test", nil)
		assert.Error(t, err, "Sending notification without connecting should fail")
	})

	// Test with invalid method
	t.Run("Invalid Method", func(t *testing.T) {
		options := DefaultClientOptions()
		client, err := NewMcpClient(serverAddr, options)
		require.NoError(t, err, "Failed to create MCP client")

		err = client.Connect(ctx)
		require.NoError(t, err, "Failed to connect MCP client")
		defer client.Close(ctx)

		resp, err := client.SendRequest(ctx, "invalid_method", nil)
		require.NoError(t, err, "Request with invalid method should not fail at transport level")
		assert.NotNil(t, resp.Error, "Response should contain an error")
	})
}
