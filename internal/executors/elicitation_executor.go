package executors

import (
	"context"
	"encoding/json"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
)

type ElicitationExecutor struct {
	serverInfo config.McpServerInfo
}

func NewElicitationExecutor(serverInfo config.McpServerInfo) *ElicitationExecutor {
	return &ElicitationExecutor{
		serverInfo: serverInfo,
	}
}

func (e *ElicitationExecutor) CanHandleMethod(method string) bool {
	return method == "elicitation/create"
}

func (e *ElicitationExecutor) HandleMethod(ctx context.Context, method string, req *mcppb.JsonRpcRequest) (*mcppb.JsonRpcResponse, error) {
	switch method {
	case "elicitation/create":
		return e.handleElicitationCreate(ctx, req)
	default:
		return nil, protocol.NewMethodNotFoundError(method, req.Id)
	}
}

func (e *ElicitationExecutor) handleElicitationCreate(ctx context.Context, req *mcppb.JsonRpcRequest) (*mcppb.JsonRpcResponse, error) {
	var elicitReq protocol.ElicitationRequest
	if err := json.Unmarshal([]byte(req.ParamsJson), &elicitReq); err != nil {
		return nil, protocol.NewInvalidParamsError("Invalid elicitation request parameters", req.Id)
	}

	response := &protocol.ElicitationResponse{
		Action: protocol.ElicitationActionAccept,
		Content: map[string]interface{}{
			"mock": "This is a mock elicitation response",
		},
	}

	resultJSON, err := json.Marshal(response)
	if err != nil {
		return nil, protocol.NewInternalError("Failed to marshal elicitation response", req.Id)
	}

	jsonRpcResp := &mcppb.JsonRpcResponse{
		Jsonrpc: "2.0",
		Response: &mcppb.JsonRpcResponse_ResultJson{
			ResultJson: string(resultJSON),
		},
	}

	// Copy the ID from the request
	switch id := req.Id.(type) {
	case *mcppb.JsonRpcRequest_IntId:
		jsonRpcResp.Id = &mcppb.JsonRpcResponse_IntId{IntId: id.IntId}
	case *mcppb.JsonRpcRequest_StringId:
		jsonRpcResp.Id = &mcppb.JsonRpcResponse_StringId{StringId: id.StringId}
	}

	return jsonRpcResp, nil
}
