package utils

import (
	"encoding/json"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
)

// createErrorResponseFromJsonRpcError creates a response from a JsonRpcError
func CreateErrorResponseFromJsonRpcError(req *mcppb.JsonRpcRequest, err *protocol.JsonRpcError) *mcppb.JsonRpcResponse {
	response := &mcppb.JsonRpcResponse{
		Jsonrpc: "2.0",
	}

	// Copy the ID from the request
	switch id := req.Id.(type) {
	case *mcppb.JsonRpcRequest_IntId:
		response.Id = &mcppb.JsonRpcResponse_IntId{IntId: id.IntId}
	case *mcppb.JsonRpcRequest_StringId:
		response.Id = &mcppb.JsonRpcResponse_StringId{StringId: id.StringId}
	}

	// Create error object
	errorObj := &mcppb.JsonRpcError{
		Code:    int32(err.Code),
		Message: err.Message,
	}

	// Add error data if provided
	if err.Data != nil {
		dataJSON, jsonErr := json.Marshal(err.Data)
		if jsonErr == nil {
			errorObj.DataJson = string(dataJSON)
		}
	}

	response.Response = &mcppb.JsonRpcResponse_Error{
		Error: errorObj,
	}

	return response
}

// createErrorResponse creates a JSON-RPC error response
func CreateErrorResponse(req *mcppb.JsonRpcRequest, code int32, message string, data interface{}) *mcppb.JsonRpcResponse {
	response := &mcppb.JsonRpcResponse{
		Jsonrpc: "2.0",
	}

	// Copy the ID from the request
	switch id := req.Id.(type) {
	case *mcppb.JsonRpcRequest_IntId:
		response.Id = &mcppb.JsonRpcResponse_IntId{IntId: id.IntId}
	case *mcppb.JsonRpcRequest_StringId:
		response.Id = &mcppb.JsonRpcResponse_StringId{StringId: id.StringId}
	}

	// Create the error
	errorObj := &mcppb.JsonRpcError{
		Code:    code,
		Message: message,
	}

	// Add error data if provided
	if data != nil {
		dataJSON, err := json.Marshal(data)
		if err == nil {
			errorObj.DataJson = string(dataJSON)
		}
	}

	response.Response = &mcppb.JsonRpcResponse_Error{
		Error: errorObj,
	}

	return response
}
