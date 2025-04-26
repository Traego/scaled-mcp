package client

import (
	"context"
	"encoding/json"
	"github.com/traego/scaled-mcp/internal/actors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/resources"
	"github.com/traego/scaled-mcp/pkg/server"
	"github.com/traego/scaled-mcp/pkg/utils"
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

	// Verify that the session actor is initialized
	// Get the session ID from the client
	sessionID := mcpClient.GetSessionID()
	require.NotEmpty(t, sessionID, "Session ID should not be empty")

	// Get the actor system from the server
	actorSystem := mcpServer.GetActorSystem()
	require.NotNil(t, actorSystem, "Actor system should not be nil")

	// Get the session actor name using the utility function
	sessionActorName := utils.GetSessionActorName(sessionID)
	slog.Info("Looking for session actor", "name", sessionActorName)

	// Find the session actor
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Wait a bit to ensure the session actor has processed the initialization
	time.Sleep(100 * time.Millisecond)

	// Check if the session actor exists
	_, sessionActor, err := actorSystem.ActorOf(ctx, sessionActorName)
	require.NoError(t, err, "Failed to find session actor")
	require.NotNil(t, sessionActor, "Session actor should not be nil")

	// We can't directly check the actor's state, but we can verify it's running
	// which means it didn't shut down due to initialization failure
	assert.True(t, sessionActor.IsRunning(), "Session actor should be running")
	sma, ok := sessionActor.Actor().(*utils.StateMachineActor)
	assert.True(t, ok, "Session actor should be *utils.StateMachineActor")
	sd, ok := sma.GetData().(*actors.SessionData)
	assert.True(t, ok, "Session actor should be actors.SessionData")

	assert.Equal(t, sd.ClientNotificationsInitialized, true, "Session actor should be initialized")

	// We can also try to send a request through the client to verify the session is working
	// If the session actor wasn't properly initialized, this would fail
	toolsList, err := mcpClient.ListTools(ctx)
	require.NoError(t, err, "Failed to list tools - session actor might not be properly initialized")
	require.NotNil(t, toolsList, "Tools list should not be nil")

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

		content, ok := resultMap["content"].([]interface{})
		if !ok {
			t.Logf("content is not array")
			t.FailNow()
		}
		resContent, ok := content[0].(map[string]interface{})
		require.True(t, ok, "resContent should be map")

		res := resContent["text"]
		// Parse the JSON string in res
		var echoMap map[string]interface{}
		err = json.Unmarshal([]byte(res.(string)), &echoMap)
		require.NoError(t, err, "Failed to parse echo JSON")
		echoVal, ok := echoMap["echo"]
		require.True(t, ok, "echo key should be present")
		assert.Equal(t, "Hello, world!", echoVal, "Echo response should match input")

	})
}
