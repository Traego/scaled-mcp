package httphandlers

import (
	"fmt"
	"github.com/traego/scaled-mcp/pkg/auth"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"net/http"
	"strings"

	"github.com/traego/scaled-mcp/pkg/utils"

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

	if ai := auth.GetAuthInfo(ctx); ai != nil && h.serverInfo.GetAuthHandler() != nil {
		ser, err := h.serverInfo.GetAuthHandler().Serialize(ai)
		if err != nil {
			handleError(w, fmt.Errorf("unable to serialize auth"), mcpRequest.Message.ID)
		}
		wrapped.AuthInfo = ser
	}

	_, rid, err := h.actorSystem.ActorOf(ctx, "root")
	if err != nil {
		handleError(w, err, mcpRequest)
		return
	}

	err = rid.SendAsync(ctx, san, &wrapped)
	if err != nil {
		if strings.HasSuffix(err.Error(), " not found") {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(string("session not found")))
			return
		} else {
			handleError(w, err, mcpRequest)
			return
		}
	}

	// Return 202 Accepted with no content as per the 2024 spec
	w.WriteHeader(http.StatusAccepted)
}
