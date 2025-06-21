package httphandlers

import (
	"fmt"
	"github.com/traego/scaled-mcp/pkg/auth"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/telemetry"
	"net/http"
	"strings"

	"github.com/traego/scaled-mcp/pkg/utils"

	"github.com/traego/scaled-mcp/pkg/protocol"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

/*
POST /messages
Route Message to Session Actor
*/

// This is backwards compatibility for 2024 for client to server messages
func (h *MCPHandler) HandleMessagePost(w http.ResponseWriter, r *http.Request) {
	tracer := telemetry.GetTracer("scaled-mcp-server/http-handlers")
	ctx, span := tracer.Start(r.Context(), "HandleMessagePost",
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
		),
	)
	defer span.End()

	sessionId := r.URL.Query().Get("sessionId")
	span.SetAttributes(attribute.String("mcp.session_id", sessionId))

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

	if mcpRequest.IsBatch {
		respErr := protocol.NewInvalidRequestError("batched json rpc calls are not allowed in the 2024-11-05 spec", "")
		span.SetStatus(codes.Error, "Batch requests not allowed in 2024 spec")
		handleError(w, respErr, "")
		return
	}

	san := utils.GetSessionActorName(sessionId)
	span.SetAttributes(attribute.String("mcp.session_actor_name", san))

	protoMsg, err := protocol.ConvertJSONToProtoRequest(mcpRequest.Message)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to convert JSON to proto request")
		handleError(w, err, mcpRequest)
		return
	}

	traceId := utils.GetTraceId(ctx)
	if traceId == "" && trace.SpanContextFromContext(ctx).IsValid() {
		traceId = trace.SpanContextFromContext(ctx).TraceID().String()
	}

	wrapped := mcppb.WrappedRequest{
		IsAsk:                 false,
		RespondToConnectionId: utils.GetDefaultSSEConnectionName(sessionId),
		Request:               protoMsg,
		TraceId:               traceId,
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
		span.RecordError(err)
		if strings.HasSuffix(err.Error(), " not found") {
			span.SetStatus(codes.Error, "Session not found")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(string("session not found")))
			return
		} else {
			span.SetStatus(codes.Error, "Failed to send async message to actor")
			handleError(w, err, mcpRequest)
			return
		}
	}

	// Return 202 Accepted with no content as per the 2024 spec
	w.WriteHeader(http.StatusAccepted)
}
