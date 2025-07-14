package context

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tochemey/goakt/v3/actor"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
)

type McpContext struct {
	sessionID    string
	actorContext *actor.ReceiveContext
	ctx          context.Context
}

func NewMcpContext(sessionID string, actorContext *actor.ReceiveContext, ctx context.Context) *McpContext {
	return &McpContext{
		sessionID:    sessionID,
		actorContext: actorContext,
		ctx:          ctx,
	}
}

func (mc *McpContext) GetSessionID() string {
	return mc.sessionID
}

func (mc *McpContext) GetContext() context.Context {
	return mc.ctx
}

func (mc *McpContext) Elicit(message string, schema map[string]interface{}) (*protocol.ElicitationResponse, error) {
	request := &protocol.ElicitationRequest{
		Message:         message,
		RequestedSchema: schema,
	}

	paramsJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal elicitation request: %w", err)
	}

	jsonRpcReq := &mcppb.JsonRpcRequest{
		Jsonrpc:    "2.0",
		Method:     "elicitation/create",
		ParamsJson: string(paramsJSON),
		Id:         &mcppb.JsonRpcRequest_StringId{StringId: fmt.Sprintf("elicit-%s", mc.sessionID)},
	}

	_ = jsonRpcReq
	return nil, fmt.Errorf("elicitation not yet implemented")
}
