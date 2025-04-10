package mcp2025

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/server"
)

// TestMCPServer2025 tests the MCP server with the 2025 spec
func TestMCPServer2025(t *testing.T) {
	// Create a server config with 2024 compatibility disabled (2025 spec only)
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

	// Test cases
	t.Run("Health Check", func(t *testing.T) {
		resp, err := http.Get(serverAddr + "/health")
		require.NoError(t, err, "Failed to make health check request")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Health check should return 200 OK")

		var result map[string]string
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err, "Failed to decode health check response")

		assert.Equal(t, "ok", result["status"], "Health check should return status ok")
	})

	t.Run("Initialize Session", func(t *testing.T) {
		// Create a JSON-RPC initialize request
		initRequest := protocol.JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      "test-1",
			Method:  "initialize",
			Params: map[string]interface{}{
				"client_info": map[string]interface{}{
					"name":    "test-client",
					"version": "1.0.0",
				},
			},
		}

		reqBody, err := json.Marshal(initRequest)
		require.NoError(t, err, "Failed to marshal initialize request")

		// Send the request to the /mcp endpoint (2025 spec)
		resp, err := http.Post(serverAddr+"/mcp", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err, "Failed to make initialize request")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Initialize request should return 200 OK")

		var result protocol.JSONRPCMessage
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err, "Failed to decode initialize response")

		assert.Equal(t, "2.0", result.JSONRPC, "Response should have JSONRPC version 2.0")
		assert.Equal(t, initRequest.ID, result.ID, "Response should have the same ID as the request")

		// The result should contain server_info
		resultMap, ok := result.Result.(map[string]interface{})
		require.True(t, ok, "Result should be a map")

		serverInfo, ok := resultMap["server_info"].(map[string]interface{})
		require.True(t, ok, "Result should contain server_info")

		assert.NotEmpty(t, serverInfo["name"], "Server name should not be empty")
		assert.NotEmpty(t, serverInfo["version"], "Server version should not be empty")

		// Check for capabilities in the response
		capabilities, ok := resultMap["capabilities"].(map[string]interface{})
		require.True(t, ok, "Result should contain capabilities")

		// Verify that the capabilities include at least the basic ones
		assert.NotNil(t, capabilities["prompts"], "Capabilities should include prompts")
		assert.NotNil(t, capabilities["resources"], "Capabilities should include resources")
		assert.NotNil(t, capabilities["tools"], "Capabilities should include tools")
	})

	t.Run("SSE Connection", func(t *testing.T) {
		// First initialize a session
		initRequest := protocol.JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      "test-sse",
			Method:  "initialize",
			Params: map[string]interface{}{
				"client_info": map[string]interface{}{
					"name":    "test-client",
					"version": "1.0.0",
				},
			},
		}

		reqBody, err := json.Marshal(initRequest)
		require.NoError(t, err, "Failed to marshal initialize request")

		resp, err := http.Post(serverAddr+"/mcp", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err, "Failed to make initialize request")

		var result protocol.JSONRPCMessage
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err, "Failed to decode initialize response")
		resp.Body.Close()

		resultMap, ok := result.Result.(map[string]interface{})
		require.True(t, ok, "Result should be a map")

		sessionID, ok := resultMap["session_id"].(string)
		require.True(t, ok, "Result should contain session_id")

		// Now connect to the SSE endpoint using GET on /mcp
		req, err := http.NewRequest("GET", serverAddr+"/mcp?session_id="+sessionID, nil)
		require.NoError(t, err, "Failed to create SSE request")

		req.Header.Set("Accept", "text/event-stream")

		client := &http.Client{}
		sseResp, err := client.Do(req)
		require.NoError(t, err, "Failed to make SSE request")
		defer sseResp.Body.Close()

		assert.Equal(t, http.StatusOK, sseResp.StatusCode, "SSE request should return 200 OK")
		assert.Equal(t, "text/event-stream", sseResp.Header.Get("Content-Type"), "SSE response should have text/event-stream content type")
	})

	t.Run("Send Message", func(t *testing.T) {
		// First initialize a session
		initRequest := protocol.JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      "test-msg",
			Method:  "initialize",
			Params: map[string]interface{}{
				"client_info": map[string]interface{}{
					"name":    "test-client",
					"version": "1.0.0",
				},
			},
		}

		reqBody, err := json.Marshal(initRequest)
		require.NoError(t, err, "Failed to marshal initialize request")

		resp, err := http.Post(serverAddr+"/mcp", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err, "Failed to make initialize request")

		var result protocol.JSONRPCMessage
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err, "Failed to decode initialize response")
		resp.Body.Close()

		resultMap, ok := result.Result.(map[string]interface{})
		require.True(t, ok, "Result should be a map")

		sessionID, ok := resultMap["session_id"].(string)
		require.True(t, ok, "Result should contain session_id")

		// Now send a message
		msgRequest := protocol.JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      "test-msg-1",
			Method:  "message",
			Params: map[string]interface{}{
				"session_id": sessionID,
				"message":    "Hello, world!",
			},
		}

		reqBody, err = json.Marshal(msgRequest)
		require.NoError(t, err, "Failed to marshal message request")

		resp, err = http.Post(serverAddr+"/mcp", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err, "Failed to make message request")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Message request should return 200 OK")

		var msgResult protocol.JSONRPCMessage
		err = json.NewDecoder(resp.Body).Decode(&msgResult)
		require.NoError(t, err, "Failed to decode message response")

		assert.Equal(t, "2.0", msgResult.JSONRPC, "Response should have JSONRPC version 2.0")
		assert.Equal(t, msgRequest.ID, msgResult.ID, "Response should have the same ID as the request")
	})

	t.Run("Batch Request", func(t *testing.T) {
		// First initialize a session
		initRequest := protocol.JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      "test-batch",
			Method:  "initialize",
			Params: map[string]interface{}{
				"client_info": map[string]interface{}{
					"name":    "test-client",
					"version": "1.0.0",
				},
			},
		}

		reqBody, err := json.Marshal(initRequest)
		require.NoError(t, err, "Failed to marshal initialize request")

		resp, err := http.Post(serverAddr+"/mcp", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err, "Failed to make initialize request")

		var result protocol.JSONRPCMessage
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err, "Failed to decode initialize response")
		resp.Body.Close()

		resultMap, ok := result.Result.(map[string]interface{})
		require.True(t, ok, "Result should be a map")

		sessionID, ok := resultMap["session_id"].(string)
		require.True(t, ok, "Result should contain session_id")

		// Now send a batch request
		batch := []protocol.JSONRPCMessage{
			{
				JSONRPC: "2.0",
				ID:      "batch-1",
				Method:  "message",
				Params: map[string]interface{}{
					"session_id": sessionID,
					"message":    "First message",
				},
			},
			{
				JSONRPC: "2.0",
				ID:      "batch-2",
				Method:  "message",
				Params: map[string]interface{}{
					"session_id": sessionID,
					"message":    "Second message",
				},
			},
		}

		batchBody, err := json.Marshal(batch)
		require.NoError(t, err, "Failed to marshal batch request")

		resp, err = http.Post(serverAddr+"/mcp", "application/json", bytes.NewReader(batchBody))
		require.NoError(t, err, "Failed to make batch request")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Batch request should return 200 OK")

		var batchResult []protocol.JSONRPCMessage
		err = json.NewDecoder(resp.Body).Decode(&batchResult)
		require.NoError(t, err, "Failed to decode batch response")

		assert.Len(t, batchResult, 2, "Batch response should contain 2 results")
		assert.Equal(t, batch[0].ID, batchResult[0].ID, "First response should have the same ID as the first request")
		assert.Equal(t, batch[1].ID, batchResult[1].ID, "Second response should have the same ID as the second request")
	})
}

// TestMCPServerErrors2025 tests error handling in the MCP server with the 2025 spec
func TestMCPServerErrors2025(t *testing.T) {
	// Create a server config with 2024 compatibility disabled (2025 spec only)
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

	t.Run("Invalid JSON", func(t *testing.T) {
		// Send invalid JSON
		resp, err := http.Post(serverAddr+"/mcp", "application/json", bytes.NewReader([]byte("invalid json")))
		require.NoError(t, err, "Failed to make invalid JSON request")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Invalid JSON should return 200 OK with JSON-RPC error")

		var result protocol.JSONRPCMessage
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err, "Failed to decode error response")

		assert.Equal(t, "2.0", result.JSONRPC, "Response should have JSONRPC version 2.0")
		assert.NotNil(t, result.Error, "Response should contain an error")

		errorObj, ok := result.Error.(map[string]interface{})
		require.True(t, ok, "Error should be a map")

		assert.Equal(t, float64(protocol.ErrParse), errorObj["code"], "Error code should be parse error")
	})

	t.Run("Invalid Method", func(t *testing.T) {
		// Send request with invalid method
		invalidMethodRequest := protocol.JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      "test-invalid-method",
			Method:  "invalid_method",
			Params:  map[string]interface{}{},
		}

		reqBody, err := json.Marshal(invalidMethodRequest)
		require.NoError(t, err, "Failed to marshal invalid method request")

		resp, err := http.Post(serverAddr+"/mcp", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err, "Failed to make invalid method request")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Invalid method should return 200 OK with JSON-RPC error")

		var result protocol.JSONRPCMessage
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err, "Failed to decode error response")

		assert.Equal(t, "2.0", result.JSONRPC, "Response should have JSONRPC version 2.0")
		assert.Equal(t, invalidMethodRequest.ID, result.ID, "Response should have the same ID as the request")
		assert.NotNil(t, result.Error, "Response should contain an error")

		errorObj, ok := result.Error.(map[string]interface{})
		require.True(t, ok, "Error should be a map")

		assert.Equal(t, float64(protocol.ErrMethodNotFound), errorObj["code"], "Error code should be method not found")
	})

	t.Run("Invalid Session ID", func(t *testing.T) {
		// Send message with invalid session ID
		invalidSessionRequest := protocol.JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      "test-invalid-session",
			Method:  "message",
			Params: map[string]interface{}{
				"session_id": "invalid-session-id",
				"message":    "Hello, world!",
			},
		}

		reqBody, err := json.Marshal(invalidSessionRequest)
		require.NoError(t, err, "Failed to marshal invalid session request")

		resp, err := http.Post(serverAddr+"/mcp", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err, "Failed to make invalid session request")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Invalid session should return 200 OK with JSON-RPC error")

		var result protocol.JSONRPCMessage
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err, "Failed to decode error response")

		assert.Equal(t, "2.0", result.JSONRPC, "Response should have JSONRPC version 2.0")
		assert.Equal(t, invalidSessionRequest.ID, result.ID, "Response should have the same ID as the request")
		assert.NotNil(t, result.Error, "Response should contain an error")

		errorObj, ok := result.Error.(map[string]interface{})
		require.True(t, ok, "Error should be a map")

		// The error code should be invalid params or similar
		assert.NotEqual(t, float64(0), errorObj["code"], "Error code should not be 0")
	})

	t.Run("Missing Required Parameters", func(t *testing.T) {
		// Send message without required parameters
		missingParamsRequest := protocol.JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      "test-missing-params",
			Method:  "message",
			// Missing session_id and message
			Params: map[string]interface{}{},
		}

		reqBody, err := json.Marshal(missingParamsRequest)
		require.NoError(t, err, "Failed to marshal missing params request")

		resp, err := http.Post(serverAddr+"/mcp", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err, "Failed to make missing params request")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Missing params should return 200 OK with JSON-RPC error")

		var result protocol.JSONRPCMessage
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err, "Failed to decode error response")

		assert.Equal(t, "2.0", result.JSONRPC, "Response should have JSONRPC version 2.0")
		assert.Equal(t, missingParamsRequest.ID, result.ID, "Response should have the same ID as the request")
		assert.NotNil(t, result.Error, "Response should contain an error")

		errorObj, ok := result.Error.(map[string]interface{})
		require.True(t, ok, "Error should be a map")

		assert.Equal(t, float64(protocol.ErrInvalidParams), errorObj["code"], "Error code should be invalid params")
	})
}
