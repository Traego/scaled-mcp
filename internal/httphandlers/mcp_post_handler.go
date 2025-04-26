package httphandlers

import (
	"context"
	"fmt"
	"github.com/traego/scaled-mcp/internal/actors"
	"github.com/traego/scaled-mcp/pkg/auth"
	"log/slog"
	"net/http"

	"github.com/tochemey/goakt/v3/actor"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/utils"
)

// HandleMCPPost handles an MCP request
func (h *MCPHandler) HandleMCPPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	mcpRequest, err := parseMessageRequest(r)
	if err != nil {
		handleError(w, err, "")
		return
	}

	sessionId := r.Header.Get("Mcp-Session-Id")
	if sessionId == "" && mcpRequest.Message.Method == "initialize" {
		// We only allow endpoint style sessions urls for the very first initialize. This is ambiguous in the spec,
		// so we're assuming the endpoint is still the preferred way to do this
		sessionId = r.URL.Query().Get("sessionId")
	}

	demandInitialize := sessionId == ""

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

		san := utils.GetSessionActorName(sessionId)

		wrapped := mcppb.WrappedRequest{
			IsAsk:                 true,
			RespondToConnectionId: "",
			Request:               protoMsg,
		}

		if ai := auth.GetAuthInfo(ctx); ai != nil && h.serverInfo.GetAuthHandler() != nil {
			ser, err := h.serverInfo.GetAuthHandler().Serialize(ai)
			if err != nil {
				handleError(w, fmt.Errorf("unable to serialize auth"), mr.Message.ID)
			}
			wrapped.AuthInfo = ser
		}

		_, rid, err := h.actorSystem.ActorOf(ctx, "root")
		if err != nil {
			handleError(w, err, mr.Message.ID)
		}

		// So there's a one off one way only request which is notifications/initialized that we need to handle specially
		if protocol.IsOnewayMethod(mr.Message.Method) {
			err = rid.SendAsync(ctx, san, &wrapped)
			if err != nil {
				handleError(w, err, mr.Message.ID)
				return
			}
		} else {
			respMsg, err := rid.SendSync(ctx, san, &wrapped, h.config.RequestTimeout)
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
		}
	} else {
		err := actor.NewInternalError(fmt.Errorf("batch messaging not implemented"))
		handleError(w, err, mr.Message.ID)
		return
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
			_, err = h.actorSystem.Spawn(ctx, san, sa)
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

			if ai := auth.GetAuthInfo(ctx); ai != nil && h.serverInfo.GetAuthHandler() != nil {
				ser, err := h.serverInfo.GetAuthHandler().Serialize(ai)
				if err != nil {
					handleError(w, fmt.Errorf("unable to serialize auth"), mr.Message.ID)
				}
				wrapped.AuthInfo = ser
			}

			_, rid, err := h.actorSystem.ActorOf(ctx, san)
			if err != nil {
				handleError(w, err, msg.ID)
				return
			}

			initResp, err := rid.SendSync(ctx, san, &wrapped, h.config.RequestTimeout)
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
