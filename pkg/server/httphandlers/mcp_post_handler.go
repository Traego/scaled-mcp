package httphandlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/traego/scaled-mcp/pkg/actors"

	"github.com/tochemey/goakt/v3/actor"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/utils"
)

// HandleMCPPost handles an MCP request
func (h *MCPHandler) HandleMCPPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionId := r.Header.Get("Mcp-Session-Id")
	demandInitialize := sessionId == ""

	mcpRequest, err := parseMessageRequest(r)
	if err != nil {
		handleError(w, err, "")
		return
	}

	if demandInitialize {
		h.handleMcpInitDemand(ctx, w, r, mcpRequest)
		return
	} else {
		h.handleMcpMessages(ctx, sessionId, w, r, mcpRequest)
		return
	}
}

func (h *MCPHandler) handleMcpMessages(ctx context.Context, sessionId string, w http.ResponseWriter, r *http.Request, mr McpRequest) {
	if !mr.IsBatch {
		protoMsg, err := protocol.ConvertJSONToProtoRequest(mr.Message)
		if err != nil {
			handleError(w, err, mr.Message.ID)
			return
		}

		_, sa, err := h.actorSystem.ActorOf(ctx, utils.GetSessionActorName(sessionId))
		if err != nil {
			handleError(w, err, mr.Message.ID)
			return
		}

		wrapped := mcppb.WrappedRequest{
			IsAsk:                 true,
			RespondToConnectionId: "",
			Request:               protoMsg,
		}

		respMsg, err := actor.Ask(ctx, sa, &wrapped, h.config.RequestTimeout)
		if err != nil {
			handleError(w, err, mr.Message.ID)
			return
		}

		rjm, ok := respMsg.(*mcppb.JsonRpcResponse)
		if !ok {
			err := actor.NewInternalError(fmt.Errorf("failed to parse json-rpc response type"))
			handleError(w, err, mr.Message.ID)
			return
		}
		rm, err := protocol.ConvertProtoToJSONResponse(rjm)
		if err != nil {
			handleError(w, err, mr.Message.ID)
			return
		}

		err = writeMessage(w, rm, nil)
		if err != nil {
			handleError(w, err, mr.Message.ID)
			return
		}
		return
	} else {
		err := actor.NewInternalError(fmt.Errorf("batch messaging not implemented"))
		handleError(w, err, mr.Message.ID)
		return
		//msgs := mr.Messages
		//// Technically speaking, we should actually escalate to an SSE here
		//err := rh.HandleBatch(ctx, msgs)
		//if err != nil {
		//	// For batch requests, this is really just to handle a broad error, the expectation is that we're
		//	// writing errors out from the handle batch function
		//	handleError(w, err, "")
		//}
	}
}

func (h *MCPHandler) handleMcpInitDemand(ctx context.Context, w http.ResponseWriter, r *http.Request, mr McpRequest) {
	// If no session and it's a post, check that it's an initialize message. If it's not, it's a bad request
	if mr.IsBatch {
		slog.Debug("Received batch request without sessionId (expecting single initialize message")
		respErr := protocol.NewInvalidRequestError("batch requests are disallowed before initialization", "")
		handleError(w, respErr, "")
		return
	} else {
		msg := mr.Message
		if msg.Method == "initialize" {
			sessionId, err := utils.GenerateSecureID(20)
			if err != nil {
				handleError(w, err, msg.ID)
				return
			}

			sa := actors.NewMcpSessionStateMachine(h.serverInfo, sessionId)
			san := utils.GetSessionActorName(sessionId)
			act, err := h.actorSystem.Spawn(ctx, san, sa)
			if err != nil {
				handleError(w, err, msg.ID)
				return
			}

			protoInit, err := protocol.ConvertJSONToProtoRequest(msg)
			if err != nil {
				handleError(w, err, msg.ID)
				return
			}

			wrapped := mcppb.WrappedRequest{
				IsAsk:                 true,
				RespondToConnectionId: "",
				Request:               protoInit,
			}

			initResp, err := actor.Ask(ctx, act, &wrapped, h.config.RequestTimeout)
			if err != nil {
				handleError(w, err, msg.ID)
				return
			}

			jrr, ok := initResp.(*mcppb.JsonRpcResponse)
			if !ok {
				handleError(w, fmt.Errorf("unable to parse init response"), msg.ID)
				return
			}

			ir, err := protocol.ConvertProtoToJSONResponse(jrr)
			if err != nil {
				handleError(w, err, msg.ID)
				return
			}

			err = writeMessage(w, ir, &sessionId)
			if err != nil {
				handleError(w, err, msg.ID)
			}
			return
		} else {
			respErr := protocol.NewInvalidRequestError("missing Mcp-Session-Id, expecting initialize message", msg.ID)
			handleError(w, respErr, "")
			return
		}
	}
}
