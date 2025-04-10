package protocol

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONRPCMessageMarshaling(t *testing.T) {
	t.Run("request marshaling", func(t *testing.T) {
		// Create a request message
		req := JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      "request-1",
			Method:  "initialize",
			Params: map[string]interface{}{
				"protocolVersion": "2025-03",
				"clientInfo": map[string]interface{}{
					"name":    "test-client",
					"version": "1.0.0",
				},
			},
		}

		// Marshal to JSON
		data, err := json.Marshal(req)
		require.NoError(t, err)

		// Verify JSON structure
		var jsonMap map[string]interface{}
		err = json.Unmarshal(data, &jsonMap)
		require.NoError(t, err)

		// Check required fields according to JSON-RPC 2.0 spec
		assert.Equal(t, "2.0", jsonMap["jsonrpc"])
		assert.Equal(t, "request-1", jsonMap["id"])
		assert.Equal(t, "initialize", jsonMap["method"])

		// Check params
		params, ok := jsonMap["params"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "2025-03", params["protocolVersion"])

		// Result and error should not be present in a request
		_, hasResult := jsonMap["result"]
		_, hasError := jsonMap["error"]
		assert.False(t, hasResult, "result should not be present in a request")
		assert.False(t, hasError, "error should not be present in a request")
	})

	t.Run("success response marshaling", func(t *testing.T) {
		// Create a success response message
		resp := JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      42,
			Result: map[string]interface{}{
				"status": "success",
				"data": map[string]interface{}{
					"value": "test",
				},
			},
		}

		// Marshal to JSON
		data, err := json.Marshal(resp)
		require.NoError(t, err)

		// Verify JSON structure
		var jsonMap map[string]interface{}
		err = json.Unmarshal(data, &jsonMap)
		require.NoError(t, err)

		// Check required fields according to JSON-RPC 2.0 spec
		assert.Equal(t, "2.0", jsonMap["jsonrpc"])
		assert.Equal(t, float64(42), jsonMap["id"])

		// Check result
		result, ok := jsonMap["result"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "success", result["status"])

		// Method and error should not be present in a success response
		_, hasMethod := jsonMap["method"]
		_, hasError := jsonMap["error"]
		assert.False(t, hasMethod, "method should not be present in a response")
		assert.False(t, hasError, "error should not be present in a success response")
	})

	t.Run("error response marshaling", func(t *testing.T) {
		// Create an error response message
		resp := JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      "error-1",
			Error: map[string]interface{}{
				"code":    -32601,
				"message": "Method not found",
				"data":    "The requested method does not exist",
			},
		}

		// Marshal to JSON
		data, err := json.Marshal(resp)
		require.NoError(t, err)

		// Verify JSON structure
		var jsonMap map[string]interface{}
		err = json.Unmarshal(data, &jsonMap)
		require.NoError(t, err)

		// Check required fields according to JSON-RPC 2.0 spec
		assert.Equal(t, "2.0", jsonMap["jsonrpc"])
		assert.Equal(t, "error-1", jsonMap["id"])

		// Check error
		errorObj, ok := jsonMap["error"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(-32601), errorObj["code"])
		assert.Equal(t, "Method not found", errorObj["message"])
		assert.Equal(t, "The requested method does not exist", errorObj["data"])

		// Method and result should not be present in an error response
		_, hasMethod := jsonMap["method"]
		_, hasResult := jsonMap["result"]
		assert.False(t, hasMethod, "method should not be present in a response")
		assert.False(t, hasResult, "result should not be present in an error response")
	})

	t.Run("notification marshaling", func(t *testing.T) {
		// Create a notification message (no ID)
		notification := JSONRPCMessage{
			JSONRPC: "2.0",
			Method:  "resourceChanged",
			Params: map[string]interface{}{
				"uri": "resource:test",
			},
		}

		// Marshal to JSON
		data, err := json.Marshal(notification)
		require.NoError(t, err)

		// Verify JSON structure
		var jsonMap map[string]interface{}
		err = json.Unmarshal(data, &jsonMap)
		require.NoError(t, err)

		// Check required fields according to JSON-RPC 2.0 spec
		assert.Equal(t, "2.0", jsonMap["jsonrpc"])
		assert.Equal(t, "resourceChanged", jsonMap["method"])

		// ID should not be present in a notification
		_, hasID := jsonMap["id"]
		assert.False(t, hasID, "id should not be present in a notification")

		// Result and error should not be present in a notification
		_, hasResult := jsonMap["result"]
		_, hasError := jsonMap["error"]
		assert.False(t, hasResult, "result should not be present in a notification")
		assert.False(t, hasError, "error should not be present in a notification")
	})
}

func TestJSONRPCMessageUnmarshaling(t *testing.T) {
	t.Run("request unmarshaling", func(t *testing.T) {
		// JSON request
		jsonData := `{
			"jsonrpc": "2.0",
			"id": "request-1",
			"method": "initialize",
			"params": {
				"protocolVersion": "2025-03",
				"clientInfo": {
					"name": "test-client",
					"version": "1.0.0"
				}
			}
		}`

		// Unmarshal from JSON
		var req JSONRPCMessage
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		// Verify fields
		assert.Equal(t, "2.0", req.JSONRPC)
		assert.Equal(t, "request-1", req.ID)
		assert.Equal(t, "initialize", req.Method)

		// Check params
		params, ok := req.Params.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "2025-03", params["protocolVersion"])

		clientInfo, ok := params["clientInfo"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "test-client", clientInfo["name"])
		assert.Equal(t, "1.0.0", clientInfo["version"])

		// Result and error should be nil
		assert.Nil(t, req.Result)
		assert.Nil(t, req.Error)
	})

	t.Run("success response unmarshaling", func(t *testing.T) {
		// JSON success response
		jsonData := `{
			"jsonrpc": "2.0",
			"id": 42,
			"result": {
				"status": "success",
				"data": {
					"value": "test"
				}
			}
		}`

		// Unmarshal from JSON
		var resp JSONRPCMessage
		err := json.Unmarshal([]byte(jsonData), &resp)
		require.NoError(t, err)

		// Verify fields
		assert.Equal(t, "2.0", resp.JSONRPC)
		assert.Equal(t, float64(42), resp.ID)
		assert.Empty(t, resp.Method)

		// Check result
		result, ok := resp.Result.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "success", result["status"])

		data, ok := result["data"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "test", data["value"])

		// Error should be nil
		assert.Nil(t, resp.Error)
	})

	t.Run("error response unmarshaling", func(t *testing.T) {
		// JSON error response
		jsonData := `{
			"jsonrpc": "2.0",
			"id": "error-1",
			"error": {
				"code": -32601,
				"message": "Method not found",
				"data": "The requested method does not exist"
			}
		}`

		// Unmarshal from JSON
		var resp JSONRPCMessage
		err := json.Unmarshal([]byte(jsonData), &resp)
		require.NoError(t, err)

		// Verify fields
		assert.Equal(t, "2.0", resp.JSONRPC)
		assert.Equal(t, "error-1", resp.ID)
		assert.Empty(t, resp.Method)

		// Check error
		errorObj, ok := resp.Error.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(-32601), errorObj["code"])
		assert.Equal(t, "Method not found", errorObj["message"])
		assert.Equal(t, "The requested method does not exist", errorObj["data"])

		// Result should be nil
		assert.Nil(t, resp.Result)
	})

	t.Run("notification unmarshaling", func(t *testing.T) {
		// JSON notification
		jsonData := `{
			"jsonrpc": "2.0",
			"method": "resourceChanged",
			"params": {
				"uri": "resource:test"
			}
		}`

		// Unmarshal from JSON
		var notification JSONRPCMessage
		err := json.Unmarshal([]byte(jsonData), &notification)
		require.NoError(t, err)

		// Verify fields
		assert.Equal(t, "2.0", notification.JSONRPC)
		assert.Nil(t, notification.ID)
		assert.Equal(t, "resourceChanged", notification.Method)

		// Check params
		params, ok := notification.Params.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "resource:test", params["uri"])

		// Result and error should be nil
		assert.Nil(t, notification.Result)
		assert.Nil(t, notification.Error)
	})
}

func TestCapabilitiesMarshaling(t *testing.T) {
	t.Run("client capabilities marshaling", func(t *testing.T) {
		// Create client capabilities
		capabilities := ClientCapabilities{
			Roots: &RootsClientCapability{
				ListChanged: true,
			},
			Sampling: &SamplingClientCapability{},
			Experimental: map[string]interface{}{
				"customFeature": true,
			},
		}

		// Marshal to JSON
		data, err := json.Marshal(capabilities)
		require.NoError(t, err)

		// Verify JSON structure
		var jsonMap map[string]interface{}
		err = json.Unmarshal(data, &jsonMap)
		require.NoError(t, err)

		// Check fields
		roots, ok := jsonMap["roots"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, true, roots["listChanged"])

		_, hasSampling := jsonMap["sampling"]
		assert.True(t, hasSampling)

		experimental, ok := jsonMap["experimental"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, true, experimental["customFeature"])
	})

	t.Run("server capabilities marshaling", func(t *testing.T) {
		// Create server capabilities
		capabilities := ServerCapabilities{
			Prompts: &PromptsServerCapability{
				ListChanged: true,
			},
			Resources: &ResourcesServerCapability{
				Subscribe:   true,
				ListChanged: true,
			},
			Tools: &ToolsServerCapability{
				ListChanged: true,
			},
			Logging: &LoggingServerCapability{},
			Experimental: map[string]interface{}{
				"customFeature": "value",
			},
		}

		// Marshal to JSON
		data, err := json.Marshal(capabilities)
		require.NoError(t, err)

		// Verify JSON structure
		var jsonMap map[string]interface{}
		err = json.Unmarshal(data, &jsonMap)
		require.NoError(t, err)

		// Check fields
		prompts, ok := jsonMap["prompts"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, true, prompts["listChanged"])

		resources, ok := jsonMap["resources"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, true, resources["subscribe"])
		assert.Equal(t, true, resources["listChanged"])

		tools, ok := jsonMap["tools"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, true, tools["listChanged"])

		_, hasLogging := jsonMap["logging"]
		assert.True(t, hasLogging)

		experimental, ok := jsonMap["experimental"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "value", experimental["customFeature"])
	})
}

func TestInitializeStructsMarshaling(t *testing.T) {
	t.Run("initialize params marshaling", func(t *testing.T) {
		// Create initialize params
		params := InitializeParams{
			ProtocolVersion: "2025-03",
			ClientInfo: ClientInfo{
				Name:    "test-client",
				Version: "1.0.0",
			},
			Capabilities: ClientCapabilities{
				Roots: &RootsClientCapability{
					ListChanged: true,
				},
			},
		}

		// Marshal to JSON
		data, err := json.Marshal(params)
		require.NoError(t, err)

		// Verify JSON structure
		var jsonMap map[string]interface{}
		err = json.Unmarshal(data, &jsonMap)
		require.NoError(t, err)

		// Check fields
		assert.Equal(t, "2025-03", jsonMap["protocolVersion"])

		clientInfo, ok := jsonMap["clientInfo"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "test-client", clientInfo["name"])
		assert.Equal(t, "1.0.0", clientInfo["version"])

		capabilities, ok := jsonMap["capabilities"].(map[string]interface{})
		require.True(t, ok)
		roots, ok := capabilities["roots"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, true, roots["listChanged"])
	})

	t.Run("initialize result marshaling", func(t *testing.T) {
		// Create initialize result
		result := InitializeResult{
			ProtocolVersion: "2025-03",
			ServerInfo: ServerInfo{
				Name:    "test-server",
				Version: "1.0.0",
			},
			Capabilities: ServerCapabilities{
				Resources: &ResourcesServerCapability{
					Subscribe:   true,
					ListChanged: true,
				},
			},
			SessionID: "session-123",
		}

		// Marshal to JSON
		data, err := json.Marshal(result)
		require.NoError(t, err)

		// Verify JSON structure
		var jsonMap map[string]interface{}
		err = json.Unmarshal(data, &jsonMap)
		require.NoError(t, err)

		// Check fields
		assert.Equal(t, "2025-03", jsonMap["protocolVersion"])
		assert.Equal(t, "session-123", jsonMap["sessionId"])

		serverInfo, ok := jsonMap["serverInfo"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "test-server", serverInfo["name"])
		assert.Equal(t, "1.0.0", serverInfo["version"])

		capabilities, ok := jsonMap["capabilities"].(map[string]interface{})
		require.True(t, ok)
		resources, ok := capabilities["resources"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, true, resources["subscribe"])
		assert.Equal(t, true, resources["listChanged"])
	})
}
