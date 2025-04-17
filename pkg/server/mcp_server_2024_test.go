package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmaxmax/go-sse"

	"github.com/traego/scaled-mcp/pkg/client"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/resources"
	"github.com/traego/scaled-mcp/test/testutils"
)

// TestMCPServer2024 tests the MCP server with the 2024 spec.
func TestMCPServer2024(t *testing.T) {
	// Get a random available port
	port, err := testutils.GetAvailablePort()
	require.NoError(t, err, "Failed to get available port")

	// Create a server config with 2024 compatibility enabled
	cfg := config.DefaultConfig()
	cfg.BackwardCompatible20241105 = true
	cfg.HTTP.Port = port

	registry := resources.NewStaticToolRegistry()
	err = registry.RegisterTool(protocol.Tool{
		Name:        "test_tool",
		Description: "Does Stuff",
		InputSchema: protocol.InputSchema{},
	}, func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		return nil, nil
	})
	require.NoError(t, err, "Failed to register tool")

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
	defer mcpServer.Stop(ctx)

	// Get the server's HTTP address
	serverAddr := "http://localhost:" + strconv.Itoa(cfg.HTTP.Port)

	// Create client options with 2024 protocol version
	options := client.DefaultClientOptions()
	options.ProtocolVersion = protocol.ProtocolVersion20241105
	options.ClientInfo = client.ClientInfo{
		Name:    "test-client",
		Version: "1.0.0",
	}

	t.Run("Basic Initialization", func(t *testing.T) {
		// Create a new MCP client
		mcpClient, err := client.NewMcpClient(serverAddr, options)
		require.NoError(t, err, "Failed to create MCP client")

		// Use a separate context for this test that we can cancel
		testCtx, testCancel := context.WithCancel(context.Background())

		// Connect the client
		err = mcpClient.Connect(testCtx)
		require.NoError(t, err, "Failed to connect MCP client")

		// Verify that the client is initialized
		assert.True(t, mcpClient.IsInitialized(), "McpClient should be initialized")

		// Verify the protocol version
		assert.Equal(t, protocol.ProtocolVersion20241105, mcpClient.GetProtocolVersion(),
			"Protocol version should be 2024-11-05")

		// Verify the connection method
		assert.Equal(t, client.ConnectionMethodSSE, mcpClient.GetConnectionMethod(),
			"Connection method should be SSE for 2024 spec")

		// Test sending a request
		resp, err := mcpClient.SendRequest(testCtx, "tools/list", nil)
		require.NoError(t, err, "Failed to send roots/list request")
		assert.NotNil(t, resp, "Response should not be nil")
		assert.Nil(t, resp.Error, "Response should not contain an error")

		// Clean up
		testCancel()
		_ = mcpClient.Close(context.Background())
	})

	t.Run("Method not found", func(t *testing.T) {
		// Create a new MCP client
		mcpClient, err := client.NewMcpClient(serverAddr, options)
		require.NoError(t, err, "Failed to create MCP client")

		// Use a separate context for this test that we can cancel
		testCtx, testCancel := context.WithCancel(context.Background())

		// Connect the client
		err = mcpClient.Connect(testCtx)
		require.NoError(t, err, "Failed to connect MCP client")

		// Verify that the client is initialized
		assert.True(t, mcpClient.IsInitialized(), "McpClient should be initialized")

		// Verify the protocol version
		assert.Equal(t, protocol.ProtocolVersion20241105, mcpClient.GetProtocolVersion(),
			"Protocol version should be 2024-11-05")

		// Verify the connection method
		assert.Equal(t, client.ConnectionMethodSSE, mcpClient.GetConnectionMethod(),
			"Connection method should be SSE for 2024 spec")

		// Test sending a request
		_, err = mcpClient.SendRequest(testCtx, "roots/list", nil)
		require.Error(t, err, "Expected error for roots/list request")

		// Clean up
		testCancel()
		_ = mcpClient.Close(context.Background())
	})

	t.Run("SSE Connection", func(t *testing.T) {
		// Create a new MCP client
		mcpClient, err := client.NewMcpClient(serverAddr, options)
		require.NoError(t, err, "Failed to create MCP client")

		// Use a separate context for this test that we can cancel
		testCtx, testCancel := context.WithCancel(context.Background())

		// Connect the client
		err = mcpClient.Connect(testCtx)
		require.NoError(t, err, "Failed to connect MCP client")

		// Verify that the client is initialized
		assert.True(t, mcpClient.IsInitialized(), "McpClient should be initialized")

		// Verify the protocol version
		assert.Equal(t, protocol.ProtocolVersion20241105, mcpClient.GetProtocolVersion(),
			"Protocol version should be 2024-11-05")

		// Verify the connection method
		assert.Equal(t, client.ConnectionMethodSSE, mcpClient.GetConnectionMethod(),
			"Connection method should be SSE for 2024 spec")

		// Test sending a request
		resp, err := mcpClient.SendRequest(testCtx, "tools/list", nil)
		require.NoError(t, err, "Failed to send tools/list request")
		assert.NotNil(t, resp, "Response should not be nil")
		assert.Nil(t, resp.Error, "Response should not contain an error")

		// Clean up
		testCancel()
		_ = mcpClient.Close(context.Background())
	})

	t.Run("Multiple Clients", func(t *testing.T) {
		// Create a separate context for this test that we can cancel
		testCtx, testCancel := context.WithCancel(context.Background())
		defer testCancel()

		// Create multiple clients
		numClients := 5
		clients := make([]client.McpClient, numClients)

		for i := 0; i < numClients; i++ {
			c, err := client.NewMcpClient(serverAddr, options)
			require.NoError(t, err, "Failed to create MCP client")
			clients[i] = c

			// Connect each client
			err = c.Connect(testCtx)
			require.NoError(t, err, "Failed to connect MCP client")

			// Verify the protocol version
			assert.Equal(t, protocol.ProtocolVersion20241105, c.GetProtocolVersion(),
				"Protocol version should be 2024-11-05")

			// Verify the connection method
			assert.Equal(t, client.ConnectionMethodSSE, c.GetConnectionMethod(),
				"Connection method should be SSE for 2024 spec")
		}

		// Verify that all clients are initialized
		for i, c := range clients {
			assert.True(t, c.IsInitialized(), "McpClient %d should be initialized", i)

			// Test sending a request with each client
			resp, err := c.SendRequest(testCtx, "tools/list", nil)
			require.NoError(t, err, "Failed to send tools/list request with client %d", i)
			assert.NotNil(t, resp, "Response should not be nil")
			assert.Nil(t, resp.Error, "Response should not contain an error")
		}

		// Clean up all clients
		for _, c := range clients {
			_ = c.Close(context.Background())
		}
	})

	t.Run("Invalid Protocol Version", func(t *testing.T) {
		// Create a separate context for this test that we can cancel
		testCtx, testCancel := context.WithCancel(context.Background())
		defer testCancel()

		// First establish an SSE connection to get a session ID
		req, err := http.NewRequestWithContext(testCtx, http.MethodGet, serverAddr+"/sse", nil)
		require.NoError(t, err, "Failed to create SSE request")

		// Create a new SSE connection
		sseConn := sse.NewConnection(req)

		// Set up channels to receive events and session ID
		connectionEstablished := make(chan struct{})
		connectionError := make(chan error, 1)
		endpoint := ""

		// Subscribe to all events
		sseConn.SubscribeToAll(func(event sse.Event) {
			// Check if this is a session ID event
			if event.Type == "endpoint" {
				endpoint = event.Data
			}

			// Signal that we've received an event
			select {
			case connectionEstablished <- struct{}{}:
			default:
				// Already signaled
			}
		})

		// Start the connection in a goroutine
		go func() {
			err := sseConn.Connect()
			if err != nil {
				select {
				case connectionError <- err:
				case <-testCtx.Done():
					// Context canceled
				}
			}
		}()

		// Wait for the connection to be established with a timeout
		select {
		case <-testCtx.Done():
			t.Fatal("Context canceled while waiting for SSE connection")
		case err := <-connectionError:
			t.Fatalf("Error establishing SSE connection: %v", err)
		case <-connectionEstablished:
			// Connection established, continue with the test
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for SSE connection")
		}

		// Ensure we have a session ID
		require.NotEmpty(t, endpoint, "Failed to receive endpoint from SSE connection")

		// Now send initialize request with invalid protocol version
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

		// Create a request with the session ID as a query parameter
		invalidVersionURL := fmt.Sprintf("%s%s", serverAddr, endpoint)
		invalidVersionResp, err := http.Post(invalidVersionURL, "application/json",
			bytes.NewReader(reqBody))
		require.NoError(t, err, "Failed to make invalid version request")
		defer func() {
			_ = invalidVersionResp.Body.Close()
		}()

		// For 2024 spec, the request should be accepted
		// and the error will be sent via the SSE channel
		assert.Equal(t, http.StatusAccepted, invalidVersionResp.StatusCode,
			"Invalid protocol version should return 202 Accepted")

		// Clean up
		testCancel()
	})

	t.Run("Properly fail on message before initialization", func(t *testing.T) {
		// Create a separate context for this test that we can cancel
		testCtx, testCancel := context.WithCancel(context.Background())
		defer testCancel()

		// Send a request without initializing first
		uninitializedRequest := protocol.JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      "test-uninitialized",
			Method:  "roots/list", // Any method other than initialize
			Params:  nil,
		}

		reqBody, err := json.Marshal(uninitializedRequest)
		require.NoError(t, err, "Failed to marshal uninitialized request")

		req, err := http.NewRequestWithContext(testCtx, http.MethodPost, serverAddr+"/messages", bytes.NewReader(reqBody))
		require.NoError(t, err, "Failed to create uninitialized request")

		uninitializedResp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Failed to make uninitialized request")

		// Should fail with 404 Not Found since there's no session
		require.Equal(t, http.StatusNotFound, uninitializedResp.StatusCode)
		defer func() {
			_ = uninitializedResp.Body.Close()
		}()
	})

	t.Run("Tools List Should Work", func(t *testing.T) {
		// Create a new MCP client
		mcpClient, err := client.NewMcpClient(serverAddr, options)
		require.NoError(t, err, "Failed to create MCP client")

		// Use a separate context for this test that we can cancel
		testCtx, testCancel := context.WithCancel(context.Background())

		// Connect the client
		err = mcpClient.Connect(testCtx)
		require.NoError(t, err, "Failed to connect MCP client")

		// Verify that the client is initialized
		assert.True(t, mcpClient.IsInitialized(), "McpClient should be initialized")

		// Verify the protocol version
		assert.Equal(t, protocol.ProtocolVersion20241105, mcpClient.GetProtocolVersion(),
			"Protocol version should be 2024-11-05")

		// Verify the connection method
		assert.Equal(t, client.ConnectionMethodSSE, mcpClient.GetConnectionMethod(),
			"Connection method should be SSE for 2024 spec")

		// Test sending a request
		resp, err := mcpClient.SendRequest(testCtx, "tools/list", nil)
		require.NoError(t, err, "Failed to send tools/list request")
		assert.NotNil(t, resp, "Response should not be nil")

		// Type assert the response result to access nested fields
		resultMap, ok := resp.Result.(map[string]interface{})
		require.True(t, ok, "Result should be a map")

		tools, ok := resultMap["tools"].([]interface{})
		require.True(t, ok, "tools should be a slice")
		require.NotEmpty(t, tools, "tools slice should not be empty")

		// Check that the first tool is initialize or another expected value
		// This is a simple check to ensure we got valid data back
		toolName, ok := tools[0].(map[string]interface{})["name"].(string)
		require.True(t, ok, "tool should be a string")
		assert.Equal(t, toolName, "test_tool")

		assert.Nil(t, resp.Error, "Response should not contain an error")

		// Clean up
		testCancel()
		_ = mcpClient.Close(context.Background())
	})
}
