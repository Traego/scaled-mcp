package httphandlers

import (
	"context"
	"fmt"
	"github.com/traego/scaled-mcp/internal/actors"
	"github.com/traego/scaled-mcp/pkg/auth"
	"github.com/traego/scaled-mcp/pkg/telemetry"
	"log/slog"
	"net/http"

	"github.com/tochemey/goakt/v3/actor"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// HandleMCPPost handles an MCP request
func (h *MCPHandler) HandleMCPPost(w http.ResponseWriter, r *http.Request) {
	tracer := telemetry.GetTracer("scaled-mcp-server/http-handlers")
	ctx, span := tracer.Start(r.Context(), "HandleMCPPost",
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
		),
	)
	defer span.End()

	mcpRequest, err := parseMessageRequest(r)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to parse message request")
		handleError(w, err, "")
		return
	}

	span.SetAttributes(
		attribute.String("mcp.method", mcpRequest.Message.Method),
		attribute.Bool("mcp.is_batch", mcpRequest.IsBatch),
	)

	sessionId := r.Header.Get("Mcp-Session-Id")
	if sessionId == "" && mcpRequest.Message.Method == "initialize" {
		sessionId = r.URL.Query().Get("sessionId")
	}

	span.SetAttributes(attribute.String("mcp.session_id", sessionId))

	demandInitialize := sessionId == ""

	if demandInitialize {
		span.SetAttributes(attribute.Bool("mcp.demand_initialize", true))
		h.handleMcpInitDemand(ctx, w, r, mcpRequest)
		return
	} else {
		span.SetAttributes(attribute.Bool("mcp.demand_initialize", false))
		h.handleMcpMessages(ctx, sessionId, w, r, mcpRequest)
		return
	}
}

func (h *MCPHandler) handleMcpMessages(ctx context.Context, sessionId string, w http.ResponseWriter, r *http.Request, mr McpRequest) {
	tracer := telemetry.GetTracer("scaled-mcp-server/http-handlers")
	ctx, span := tracer.Start(ctx, "handleMcpMessages",
		trace.WithAttributes(
			attribute.String("mcp.session_id", sessionId),
			attribute.Bool("mcp.is_batch", mr.IsBatch),
		),
	)
	defer span.End()

	if !mr.IsBatch {
		protoMsg, err := protocol.ConvertJSONToProtoRequest(mr.Message)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to convert JSON to proto request")
			handleError(w, err, mr.Message.ID)
			return
		}

		san := utils.GetSessionActorName(sessionId)
		span.SetAttributes(attribute.String("mcp.session_actor_name", san))

		traceId := utils.GetTraceId(ctx)
		if traceId == "" && trace.SpanContextFromContext(ctx).IsValid() {
			traceId = trace.SpanContextFromContext(ctx).TraceID().String()
		}

		wrapped := mcppb.WrappedRequest{
			IsAsk:                 true,
			RespondToConnectionId: "",
			Request:               protoMsg,
			TraceId:               traceId,
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
			span.SetAttributes(attribute.Bool("mcp.is_oneway", true))
			err = rid.SendAsync(ctx, san, &wrapped)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "Failed to send async message to actor")
				handleError(w, err, mr.Message.ID)
				return
			}
		} else {
			span.SetAttributes(attribute.Bool("mcp.is_oneway", false))
			respMsg, err := rid.SendSync(ctx, san, &wrapped, h.config.RequestTimeout)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "Failed to send sync message to actor")
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
	tracer := telemetry.GetTracer("scaled-mcp-server/http-handlers")
	ctx, span := tracer.Start(ctx, "handleMcpInitDemand",
		trace.WithAttributes(
			attribute.Bool("mcp.is_batch", mr.IsBatch),
		),
	)
	defer span.End()

	if mr.IsBatch {
		slog.Debug("Received batch request without sessionId (expecting single initialize message")
		respErr := protocol.NewInvalidRequestError("batch requests are disallowed before initialization", "")
		span.SetStatus(codes.Error, "Batch requests not allowed during initialization")
		handleError(w, respErr, "")
		return
	} else {
		msg := mr.Message
		span.SetAttributes(attribute.String("mcp.method", msg.Method))

		if msg.Method == "initialize" {
			sessionId, err := utils.GenerateSecureID(20)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "Failed to generate session ID")
				handleError(w, err, msg.ID)
				return
			}

			span.SetAttributes(attribute.String("mcp.session_id", sessionId))

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

			traceId := utils.GetTraceId(ctx)
			if traceId == "" && trace.SpanContextFromContext(ctx).IsValid() {
				traceId = trace.SpanContextFromContext(ctx).TraceID().String()
			}

			wrapped := mcppb.WrappedRequest{
				IsAsk:                 true,
				RespondToConnectionId: "",
				Request:               protoInit,
				TraceId:               traceId,
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
