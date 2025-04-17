package executors

import (
	"encoding/json"

	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
)

// PrepareResponse creates a basic JSON-RPC response with the ID copied from the request.
// This is a common operation across all executors.
func PrepareResponse(req *mcppb.JsonRpcRequest) *mcppb.JsonRpcResponse {
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

	return response
}

// ParseParams extracts and parses the parameters from a JSON-RPC request.
// Returns the parsed parameters and any error that occurred during parsing.
func ParseParams(req *mcppb.JsonRpcRequest) (map[string]interface{}, error) {
	var params map[string]interface{}

	if req.ParamsJson != "" {
		if err := json.Unmarshal([]byte(req.ParamsJson), &params); err != nil {
			return nil, protocol.NewInvalidParamsError("Invalid parameters: "+err.Error(), req.Id)
		}
	} else {
		params = make(map[string]interface{})
	}

	return params, nil
}

// CheckFeature verifies if a required feature registry component is available.
// Returns a method not found error if the feature is not available.
func CheckFeature(available bool, method string, reqID interface{}) error {
	if !available {
		return protocol.NewMethodNotFoundError(method, reqID)
	}
	return nil
}

// ProcessRequest is a helper function that handles the common parts of processing a JSON-RPC request:
// 1. Checking if a required feature is available
// 2. Preparing a response with the correct ID
// 3. Parsing the parameters
//
// Returns the prepared response, parsed parameters, and any error that occurred during processing.
func ProcessRequest(method string, req *mcppb.JsonRpcRequest, featureAvailable bool) (*mcppb.JsonRpcResponse, map[string]interface{}, error) {
	// Check if the required feature is available
	if err := CheckFeature(featureAvailable, method, req.Id); err != nil {
		return nil, nil, err
	}

	// Create base response
	response := PrepareResponse(req)

	// Parse the params
	params, err := ParseParams(req)
	if err != nil {
		return nil, nil, err
	}

	return response, params, nil
}
