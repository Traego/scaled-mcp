package httphandlers

import (
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"net/http"

	"github.com/traego/scaled-mcp/internal/utils"

	"github.com/tochemey/goakt/v3/actor"
	"github.com/traego/scaled-mcp/pkg/protocol"
)

/*
POST /messages
Route Message to Session Actor
*/

// This is backwards compatibility for 2024 for client to server messages
func (h *MCPHandler) HandleMessagePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionId := r.URL.Query().Get("sessionId")

	mcpRequest, err := parseMessageRequest(r)
	if err != nil {
		handleError(w, err, "")
		return
	}

	if mcpRequest.IsBatch {
		// in the 2024 spec batch is not allowed
		respErr := protocol.NewInvalidRequestError("batched json rpc calls are not allowed in the 2024-11-05 spec", "")
		handleError(w, respErr, "")
		return
	}

	san := utils.GetSessionActorName(sessionId)
	_, act, err := h.actorSystem.ActorOf(ctx, san)
	if err != nil {
		handleError(w, err, mcpRequest)
		return
	}

	protoMsg, err := protocol.ConvertJSONToProtoRequest(mcpRequest.Message)
	if err != nil {
		handleError(w, err, mcpRequest)
		return
	}

	wrapped := mcppb.WrappedRequest{
		IsAsk:                 false,
		RespondToConnectionId: utils.GetDefaultSSEConnectionName(sessionId),
		Request:               protoMsg,
	}

	err = actor.Tell(ctx, act, &wrapped)
	if err != nil {
		handleError(w, err, mcpRequest.Message.ID)
		return
	}

	// Return 202 Accepted with no content as per the 2024 spec
	w.WriteHeader(http.StatusAccepted)
}
