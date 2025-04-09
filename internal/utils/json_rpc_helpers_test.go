package utils

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
)

func TestCreateErrorResponseFromJsonRpcError(t *testing.T) {
	testCases := []struct {
		name        string
		request     *mcppb.JsonRpcRequest
		jsonRpcErr  *protocol.JsonRpcError
		expectedRes *mcppb.JsonRpcResponse
	}{
		{
			name: "with string ID and no data",
			request: &mcppb.JsonRpcRequest{
				Jsonrpc: "2.0",
				Id: &mcppb.JsonRpcRequest_StringId{
					StringId: "test-id",
				},
				Method: "test/method",
			},
			jsonRpcErr: &protocol.JsonRpcError{
				Code:    -32600,
				Message: "Invalid request",
			},
			expectedRes: &mcppb.JsonRpcResponse{
				Jsonrpc: "2.0",
				Id: &mcppb.JsonRpcResponse_StringId{
					StringId: "test-id",
				},
				Response: &mcppb.JsonRpcResponse_Error{
					Error: &mcppb.JsonRpcError{
						Code:    -32600,
						Message: "Invalid request",
					},
				},
			},
		},
		{
			name: "with int ID and no data",
			request: &mcppb.JsonRpcRequest{
				Jsonrpc: "2.0",
				Id: &mcppb.JsonRpcRequest_IntId{
					IntId: 42,
				},
				Method: "test/method",
			},
			jsonRpcErr: &protocol.JsonRpcError{
				Code:    -32601,
				Message: "Method not found",
			},
			expectedRes: &mcppb.JsonRpcResponse{
				Jsonrpc: "2.0",
				Id: &mcppb.JsonRpcResponse_IntId{
					IntId: 42,
				},
				Response: &mcppb.JsonRpcResponse_Error{
					Error: &mcppb.JsonRpcError{
						Code:    -32601,
						Message: "Method not found",
					},
				},
			},
		},
		{
			name: "with string ID and data",
			request: &mcppb.JsonRpcRequest{
				Jsonrpc: "2.0",
				Id: &mcppb.JsonRpcRequest_StringId{
					StringId: "test-id",
				},
				Method: "test/method",
			},
			jsonRpcErr: &protocol.JsonRpcError{
				Code:    -32602,
				Message: "Invalid params",
				Data: map[string]interface{}{
					"missing": "param1",
					"details": "Required parameter is missing",
				},
			},
			expectedRes: &mcppb.JsonRpcResponse{
				Jsonrpc: "2.0",
				Id: &mcppb.JsonRpcResponse_StringId{
					StringId: "test-id",
				},
				Response: &mcppb.JsonRpcResponse_Error{
					Error: &mcppb.JsonRpcError{
						Code:     -32602,
						Message:  "Invalid params",
						DataJson: `{"details":"Required parameter is missing","missing":"param1"}`,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function
			response := CreateErrorResponseFromJsonRpcError(tc.request, tc.jsonRpcErr)

			// Verify the response
			assert.Equal(t, tc.expectedRes.Jsonrpc, response.Jsonrpc)

			// Check ID based on type
			switch id := tc.expectedRes.Id.(type) {
			case *mcppb.JsonRpcResponse_StringId:
				assert.Equal(t, id.StringId, response.GetStringId())
			case *mcppb.JsonRpcResponse_IntId:
				assert.Equal(t, id.IntId, response.GetIntId())
			}

			// Check error
			assert.Equal(t, tc.expectedRes.GetError().Code, response.GetError().Code)
			assert.Equal(t, tc.expectedRes.GetError().Message, response.GetError().Message)

			// If data is provided, check it matches
			if tc.jsonRpcErr.Data != nil {
				// Parse both JSON strings to compare the actual data
				var expectedData, actualData map[string]interface{}
				err := json.Unmarshal([]byte(tc.expectedRes.GetError().DataJson), &expectedData)
				require.NoError(t, err)

				err = json.Unmarshal([]byte(response.GetError().DataJson), &actualData)
				require.NoError(t, err)

				assert.Equal(t, expectedData, actualData)
			} else {
				assert.Empty(t, response.GetError().DataJson)
			}
		})
	}
}

func TestCreateErrorResponse(t *testing.T) {
	testCases := []struct {
		name        string
		request     *mcppb.JsonRpcRequest
		code        int32
		message     string
		data        interface{}
		expectedRes *mcppb.JsonRpcResponse
	}{
		{
			name: "with string ID and no data",
			request: &mcppb.JsonRpcRequest{
				Jsonrpc: "2.0",
				Id: &mcppb.JsonRpcRequest_StringId{
					StringId: "test-id",
				},
				Method: "test/method",
			},
			code:    -32600,
			message: "Invalid request",
			data:    nil,
			expectedRes: &mcppb.JsonRpcResponse{
				Jsonrpc: "2.0",
				Id: &mcppb.JsonRpcResponse_StringId{
					StringId: "test-id",
				},
				Response: &mcppb.JsonRpcResponse_Error{
					Error: &mcppb.JsonRpcError{
						Code:    -32600,
						Message: "Invalid request",
					},
				},
			},
		},
		{
			name: "with int ID and no data",
			request: &mcppb.JsonRpcRequest{
				Jsonrpc: "2.0",
				Id: &mcppb.JsonRpcRequest_IntId{
					IntId: 42,
				},
				Method: "test/method",
			},
			code:    -32601,
			message: "Method not found",
			data:    nil,
			expectedRes: &mcppb.JsonRpcResponse{
				Jsonrpc: "2.0",
				Id: &mcppb.JsonRpcResponse_IntId{
					IntId: 42,
				},
				Response: &mcppb.JsonRpcResponse_Error{
					Error: &mcppb.JsonRpcError{
						Code:    -32601,
						Message: "Method not found",
					},
				},
			},
		},
		{
			name: "with string ID and data",
			request: &mcppb.JsonRpcRequest{
				Jsonrpc: "2.0",
				Id: &mcppb.JsonRpcRequest_StringId{
					StringId: "test-id",
				},
				Method: "test/method",
			},
			code:    -32602,
			message: "Invalid params",
			data: map[string]interface{}{
				"missing": "param1",
				"details": "Required parameter is missing",
			},
			expectedRes: &mcppb.JsonRpcResponse{
				Jsonrpc: "2.0",
				Id: &mcppb.JsonRpcResponse_StringId{
					StringId: "test-id",
				},
				Response: &mcppb.JsonRpcResponse_Error{
					Error: &mcppb.JsonRpcError{
						Code:     -32602,
						Message:  "Invalid params",
						DataJson: `{"details":"Required parameter is missing","missing":"param1"}`,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function
			response := CreateErrorResponse(tc.request, tc.code, tc.message, tc.data)

			// Verify the response
			assert.Equal(t, tc.expectedRes.Jsonrpc, response.Jsonrpc)

			// Check ID based on type
			switch id := tc.expectedRes.Id.(type) {
			case *mcppb.JsonRpcResponse_StringId:
				assert.Equal(t, id.StringId, response.GetStringId())
			case *mcppb.JsonRpcResponse_IntId:
				assert.Equal(t, id.IntId, response.GetIntId())
			}

			// Check error
			assert.Equal(t, tc.expectedRes.GetError().Code, response.GetError().Code)
			assert.Equal(t, tc.expectedRes.GetError().Message, response.GetError().Message)

			// If data is provided, check it matches
			if tc.data != nil {
				// Parse both JSON strings to compare the actual data
				var expectedData, actualData map[string]interface{}
				err := json.Unmarshal([]byte(tc.expectedRes.GetError().DataJson), &expectedData)
				require.NoError(t, err)

				err = json.Unmarshal([]byte(response.GetError().DataJson), &actualData)
				require.NoError(t, err)

				assert.Equal(t, expectedData, actualData)
			} else {
				assert.Empty(t, response.GetError().DataJson)
			}
		})
	}
}
