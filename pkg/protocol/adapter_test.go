package protocol

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
)

func TestConvertJSONToProtoRequest(t *testing.T) {
	t.Run("request with string ID", func(t *testing.T) {
		// Create a JSON-RPC request with a string ID
		jsonReq := JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      "request-1",
			Method:  "initialize",
			Params: map[string]interface{}{
				"protocolVersion": ProtocolVersion20250326,
				"clientInfo": map[string]interface{}{
					"name":    "test-client",
					"version": "1.0.0",
				},
			},
		}

		// Convert to protobuf
		protoReq, err := ConvertJSONToProtoRequest(jsonReq)
		require.NoError(t, err)
		require.NotNil(t, protoReq)

		// Verify the conversion
		assert.Equal(t, "2.0", protoReq.Jsonrpc)
		assert.Equal(t, "initialize", protoReq.Method)

		// Verify ID is properly converted
		stringID, ok := protoReq.Id.(*mcppb.JsonRpcRequest_StringId)
		require.True(t, ok, "ID should be a string ID")
		assert.Equal(t, "request-1", stringID.StringId)

		// Verify params are properly converted
		var params map[string]interface{}
		err = json.Unmarshal([]byte(protoReq.ParamsJson), &params)
		require.NoError(t, err)
		assert.Equal(t, string(ProtocolVersion20250326), params["protocolVersion"])

		clientInfo, ok := params["clientInfo"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "test-client", clientInfo["name"])
		assert.Equal(t, "1.0.0", clientInfo["version"])
	})

	t.Run("request with numeric ID", func(t *testing.T) {
		// Create a JSON-RPC request with a numeric ID
		jsonReq := JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      float64(42),
			Method:  "getResource",
			Params: map[string]interface{}{
				"uri": "resource:test",
			},
		}

		// Convert to protobuf
		protoReq, err := ConvertJSONToProtoRequest(jsonReq)
		require.NoError(t, err)
		require.NotNil(t, protoReq)

		// Verify the conversion
		assert.Equal(t, "2.0", protoReq.Jsonrpc)
		assert.Equal(t, "getResource", protoReq.Method)

		// Verify ID is properly converted
		intID, ok := protoReq.Id.(*mcppb.JsonRpcRequest_IntId)
		require.True(t, ok, "ID should be an int ID")
		assert.Equal(t, int64(42), intID.IntId)

		// Verify params are properly converted
		var params map[string]interface{}
		err = json.Unmarshal([]byte(protoReq.ParamsJson), &params)
		require.NoError(t, err)
		assert.Equal(t, "resource:test", params["uri"])
	})

	t.Run("notification (null ID)", func(t *testing.T) {
		// Create a JSON-RPC notification (no ID)
		jsonReq := JSONRPCMessage{
			JSONRPC: "2.0",
			Method:  "resourceChanged",
			Params: map[string]interface{}{
				"uri": "resource:test",
			},
		}

		// Convert to protobuf
		protoReq, err := ConvertJSONToProtoRequest(jsonReq)
		require.NoError(t, err)
		require.NotNil(t, protoReq)

		// Verify the conversion
		assert.Equal(t, "2.0", protoReq.Jsonrpc)
		assert.Equal(t, "resourceChanged", protoReq.Method)

		// Verify ID is properly converted to null
		_, ok := protoReq.Id.(*mcppb.JsonRpcRequest_NullId)
		require.True(t, ok, "ID should be a null ID for notifications")

		// Verify params are properly converted
		var params map[string]interface{}
		err = json.Unmarshal([]byte(protoReq.ParamsJson), &params)
		require.NoError(t, err)
		assert.Equal(t, "resource:test", params["uri"])
	})

	t.Run("request without params", func(t *testing.T) {
		// Create a JSON-RPC request without params
		jsonReq := JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      "no-params",
			Method:  "ping",
		}

		// Convert to protobuf
		protoReq, err := ConvertJSONToProtoRequest(jsonReq)
		require.NoError(t, err)
		require.NotNil(t, protoReq)

		// Verify the conversion
		assert.Equal(t, "2.0", protoReq.Jsonrpc)
		assert.Equal(t, "ping", protoReq.Method)

		// Verify ID is properly converted
		stringID, ok := protoReq.Id.(*mcppb.JsonRpcRequest_StringId)
		require.True(t, ok, "ID should be a string ID")
		assert.Equal(t, "no-params", stringID.StringId)

		// Verify params are empty
		assert.Empty(t, protoReq.ParamsJson)
	})
}

func TestConvertProtoToJSONResponse(t *testing.T) {
	t.Run("success response with string ID", func(t *testing.T) {
		// Create a protobuf success response with string ID
		resultJSON := `{"status":"success","message":"Operation completed"}`
		protoResp := &mcppb.JsonRpcResponse{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcResponse_StringId{
				StringId: "request-1",
			},
			Response: &mcppb.JsonRpcResponse_ResultJson{
				ResultJson: resultJSON,
			},
		}

		// Convert to JSON-RPC
		jsonResp, err := ConvertProtoToJSONResponse(protoResp)
		require.NoError(t, err)

		// Verify the conversion
		assert.Equal(t, "2.0", jsonResp.JSONRPC)
		assert.Equal(t, "request-1", jsonResp.ID)

		// Verify result is properly converted
		resultMap, ok := jsonResp.Result.(map[string]interface{})
		require.True(t, ok, "Result should be a map")
		assert.Equal(t, "success", resultMap["status"])
		assert.Equal(t, "Operation completed", resultMap["message"])

		// Verify error is nil
		assert.Nil(t, jsonResp.Error)
	})

	t.Run("success response with numeric ID", func(t *testing.T) {
		// Create a protobuf success response with numeric ID
		resultJSON := `{"resources":[{"uri":"resource:test"}]}`
		protoResp := &mcppb.JsonRpcResponse{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcResponse_IntId{
				IntId: 42,
			},
			Response: &mcppb.JsonRpcResponse_ResultJson{
				ResultJson: resultJSON,
			},
		}

		// Convert to JSON-RPC
		jsonResp, err := ConvertProtoToJSONResponse(protoResp)
		require.NoError(t, err)

		// Verify the conversion
		assert.Equal(t, "2.0", jsonResp.JSONRPC)
		assert.Equal(t, int64(42), jsonResp.ID)

		// Verify result is properly converted
		resultMap, ok := jsonResp.Result.(map[string]interface{})
		require.True(t, ok, "Result should be a map")
		resources, ok := resultMap["resources"].([]interface{})
		require.True(t, ok, "Resources should be an array")
		require.Len(t, resources, 1)
		resource, ok := resources[0].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "resource:test", resource["uri"])

		// Verify error is nil
		assert.Nil(t, jsonResp.Error)
	})

	t.Run("empty result", func(t *testing.T) {
		// Create a protobuf response with empty result
		protoResp := &mcppb.JsonRpcResponse{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcResponse_StringId{
				StringId: "empty-result",
			},
			Response: &mcppb.JsonRpcResponse_ResultJson{
				ResultJson: "",
			},
		}

		// Convert to JSON-RPC
		jsonResp, err := ConvertProtoToJSONResponse(protoResp)
		require.NoError(t, err)

		// Verify the conversion
		assert.Equal(t, "2.0", jsonResp.JSONRPC)
		assert.Equal(t, "empty-result", jsonResp.ID)

		// Verify result is an empty object
		resultMap, ok := jsonResp.Result.(map[string]interface{})
		require.True(t, ok, "Result should be a map")
		assert.Empty(t, resultMap)

		// Verify error is nil
		assert.Nil(t, jsonResp.Error)
	})

	t.Run("error response", func(t *testing.T) {
		// Create a protobuf error response
		protoResp := &mcppb.JsonRpcResponse{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcResponse_StringId{
				StringId: "error-request",
			},
			Response: &mcppb.JsonRpcResponse_Error{
				Error: &mcppb.JsonRpcError{
					Code:     -32601,
					Message:  "Method not found",
					DataJson: `{"details":"The requested method does not exist"}`,
				},
			},
		}

		// Convert to JSON-RPC
		jsonResp, err := ConvertProtoToJSONResponse(protoResp)
		require.NoError(t, err)

		// Verify the conversion
		assert.Equal(t, "2.0", jsonResp.JSONRPC)
		assert.Equal(t, "error-request", jsonResp.ID)

		// Verify result is nil
		assert.Nil(t, jsonResp.Result)

		// Verify error is properly converted
		errorMap, ok := jsonResp.Error.(map[string]interface{})
		require.True(t, ok, "Error should be a map")

		// Use a type-agnostic comparison for numeric values
		code := errorMap["code"].(int32)
		assert.Equal(t, -32601, int(code), "Error code should be -32601")
		assert.Equal(t, "Method not found", errorMap["message"])

		// Verify error data
		data, ok := errorMap["data"].(map[string]interface{})
		require.True(t, ok, "Error data should be a map")
		assert.Equal(t, "The requested method does not exist", data["details"])
	})

	t.Run("error response without data", func(t *testing.T) {
		// Create a protobuf error response without data
		protoResp := &mcppb.JsonRpcResponse{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcResponse_NullId{
				NullId: true,
			},
			Response: &mcppb.JsonRpcResponse_Error{
				Error: &mcppb.JsonRpcError{
					Code:    -32700,
					Message: "Parse error",
				},
			},
		}

		// Convert to JSON-RPC
		jsonResp, err := ConvertProtoToJSONResponse(protoResp)
		require.NoError(t, err)

		// Verify the conversion
		assert.Equal(t, "2.0", jsonResp.JSONRPC)
		assert.Nil(t, jsonResp.ID)

		// Verify result is nil
		assert.Nil(t, jsonResp.Result)

		// Verify error is properly converted
		errorMap, ok := jsonResp.Error.(map[string]interface{})
		require.True(t, ok, "Error should be a map")

		// Use a type-agnostic comparison for numeric values
		code := errorMap["code"].(int32)
		assert.Equal(t, -32700, int(code), "Error code should be -32700")
		assert.Equal(t, "Parse error", errorMap["message"])

		// Verify error data is not present
		_, exists := errorMap["data"]
		assert.False(t, exists, "Error data should not be present")
	})
}
