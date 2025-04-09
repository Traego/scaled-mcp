package protocol

import (
	"encoding/json"
	"fmt"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
)

// convertJSONToProtoRequest converts a JSON-RPC message to a protobuf request
func ConvertJSONToProtoRequest(message JSONRPCMessage) (*mcppb.JsonRpcRequest, error) {
	// Create the base request
	req := &mcppb.JsonRpcRequest{
		Jsonrpc: message.JSONRPC,
		Method:  message.Method,
	}

	// Convert the ID
	if message.ID != nil {
		switch id := message.ID.(type) {
		case float64:
			req.Id = &mcppb.JsonRpcRequest_IntId{IntId: int64(id)}
		case string:
			req.Id = &mcppb.JsonRpcRequest_StringId{StringId: id}
		default:
			// For null or other types, treat as null
			req.Id = &mcppb.JsonRpcRequest_NullId{NullId: true}
		}
	} else {
		// No ID means it's a notification
		req.Id = &mcppb.JsonRpcRequest_NullId{NullId: true}
	}

	// Convert the params
	if message.Params != nil {
		paramsJSON, err := json.Marshal(message.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		req.ParamsJson = string(paramsJSON)
	}

	return req, nil
}

// convertProtoToJSONResponse converts a protobuf response to a JSON-RPC message
func ConvertProtoToJSONResponse(protoResp *mcppb.JsonRpcResponse) (JSONRPCMessage, error) {
	// Create the base response
	jsonResp := JSONRPCMessage{
		JSONRPC: protoResp.Jsonrpc,
	}

	// Convert the ID
	switch id := protoResp.Id.(type) {
	case *mcppb.JsonRpcResponse_IntId:
		jsonResp.ID = id.IntId
	case *mcppb.JsonRpcResponse_StringId:
		jsonResp.ID = id.StringId
	case *mcppb.JsonRpcResponse_NullId:
		jsonResp.ID = nil
	}

	// Convert the result or error
	switch resp := protoResp.Response.(type) {
	case *mcppb.JsonRpcResponse_ResultJson:
		if resp.ResultJson != "" {
			var result interface{}
			if err := json.Unmarshal([]byte(resp.ResultJson), &result); err != nil {
				return jsonResp, fmt.Errorf("failed to unmarshal result: %w", err)
			}
			jsonResp.Result = result
		} else {
			jsonResp.Result = map[string]interface{}{}
		}
	case *mcppb.JsonRpcResponse_Error:
		errorObj := map[string]interface{}{
			"code":    resp.Error.Code,
			"message": resp.Error.Message,
		}

		// Add error data if present
		if resp.Error.DataJson != "" {
			var data interface{}
			if err := json.Unmarshal([]byte(resp.Error.DataJson), &data); err == nil {
				errorObj["data"] = data
			}
		}

		jsonResp.Error = errorObj
	}

	return jsonResp, nil
}
