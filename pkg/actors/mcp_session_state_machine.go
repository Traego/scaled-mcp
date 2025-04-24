package actors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/tochemey/goakt/v3/actor"
	"github.com/tochemey/goakt/v3/goaktpb"

	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/utils"
)

// Session states
const (
	StateUninitialized utils.StateID = "uninitialized"
	StateInitialized   utils.StateID = "initialized"
	StateShutdown      utils.StateID = "shutdown"
)

// SessionData contains the session actor's state
type SessionData struct {
	// Session ID
	SessionID string

	// Server configuration
	ServerInfo config.McpServerInfo

	// MCP protocol state
	ProtocolVersion protocol.ProtocolVersion
	ClientInfo      protocol.ClientInfo

	// Last activity time
	LastActivity time.Time

	// Session timeout duration
	InitializeTimeout time.Duration

	// Session timeout duration
	SessionTimeout time.Duration

	// Connection actors
	ClientConnectionActors map[string]*actor.PID

	// Flag to track if the session is initialized
	ClientNotificationsInitialized bool
}

// NewMcpSessionStateMachine creates a new MCP session state machine actor
func NewMcpSessionStateMachine(serverInfo config.McpServerInfo, sessionID string) actor.Actor {
	// Initialize session data
	sessionTimeout := 5 * time.Minute
	if serverInfo.GetServerConfig().Session.TTL > 0 {
		sessionTimeout = serverInfo.GetServerConfig().Session.TTL
	}

	initializeTimeout := sessionTimeout / 10
	if serverInfo.GetServerConfig().Session.InitializeTimeout > 0 {
		initializeTimeout = serverInfo.GetServerConfig().Session.InitializeTimeout
	}

	data := &SessionData{
		SessionID:                      sessionID,
		ServerInfo:                     serverInfo,
		LastActivity:                   time.Now(),
		InitializeTimeout:              initializeTimeout,
		SessionTimeout:                 sessionTimeout,
		ClientConnectionActors:         make(map[string]*actor.PID),
		ClientNotificationsInitialized: false,
	}

	// Create state machine starting in uninitialized state
	fsm := utils.NewStateMachineActor(sessionID, StateUninitialized, data)

	// Configure state handlers
	fsm.When(StateUninitialized, handleUninitializedState).
		When(StateInitialized, handleInitializedState).
		When(StateShutdown, handleShutdownState).
		WhenUnhandled(handleUnhandledMessage)

	return fsm
}

// handleUninitializedState handles messages in the uninitialized state
func handleUninitializedState(ctx *actor.ReceiveContext, data utils.Data) (utils.MessageHandlingResult, error) {
	sessionData := data.(*SessionData)

	message := ctx.Message()
	switch msg := message.(type) {
	case *goaktpb.PostStart:
		return handlePostStartUninitialized(ctx, sessionData)
	case *mcppb.RegisterConnection:
		return handleRegisterConnection(ctx, sessionData, msg)
	case *mcppb.WrappedRequest:
		return handleWrappedRequestUninitialized(ctx, sessionData, msg)
	case *mcppb.TryCleanupIfUninitialized:
		return handleTryCleanupIfUninitialized(ctx, sessionData)
	case *mcppb.CheckSessionTTL:
		return handleCheckSessionTTL(ctx, sessionData)
	default:
		// Log unhandled message
		slog.WarnContext(ctx.Context(), "Uninitialized state: Received unknown message type",
			"session_id", sessionData.SessionID,
			"message_type", fmt.Sprintf("%T", msg))
		ctx.Unhandled()
		return utils.Stay(sessionData)
	}
}

// handleInitializedState handles messages in the initialized state
func handleInitializedState(ctx *actor.ReceiveContext, data utils.Data) (utils.MessageHandlingResult, error) {
	sessionData := data.(*SessionData)

	message := ctx.Message()
	switch msg := message.(type) {
	case *mcppb.RegisterConnection:
		return handleRegisterConnection(ctx, sessionData, msg)
	case *mcppb.WrappedRequest:
		return handleWrappedRequestInitialized(ctx, sessionData, msg)
	case *mcppb.CheckSessionTTL:
		return handleCheckSessionTTL(ctx, sessionData)
	case *mcppb.TryCleanupIfUninitialized:
		return handleTryCleanupInitialized(ctx, sessionData)
	default:
		// Log unhandled message
		slog.WarnContext(ctx.Context(), "Initialized state: Received unknown message type",
			"session_id", sessionData.SessionID,
			"message_type", fmt.Sprintf("%T", msg))
		ctx.Unhandled()
		return utils.Stay(sessionData)
	}
}

// handleShutdownState handles messages in the shutdown state
func handleShutdownState(ctx *actor.ReceiveContext, data utils.Data) (utils.MessageHandlingResult, error) {
	sessionData := data.(*SessionData)

	// In shutdown state, we don't process any messages except for final cleanup
	message := ctx.Message()
	switch msg := message.(type) {
	case *mcppb.CheckSessionTTL:
		// Always shutdown when in shutdown state
		utils.Shutdown(ctx)
		return utils.Stay(sessionData)
	default:
		// Log unhandled message
		slog.WarnContext(ctx.Context(), "Shutdown state: Received message, ignoring",
			"session_id", sessionData.SessionID,
			"message_type", fmt.Sprintf("%T", msg))
		return utils.Stay(sessionData)
	}
}

// handleUnhandledMessage handles messages that are not handled by any state
func handleUnhandledMessage(ctx *actor.ReceiveContext, data utils.Data, message interface{}) utils.Data {
	sessionData := data.(*SessionData)
	slog.WarnContext(ctx.Context(), "Received unhandled message",
		"session_id", sessionData.SessionID,
		"message_type", fmt.Sprintf("%T", message))
	return sessionData
}

// handlePostStartUninitialized handles the PostStart message
func handlePostStartUninitialized(ctx *actor.ReceiveContext, sessionData *SessionData) (utils.MessageHandlingResult, error) {
	ctx.Logger().Info("mcp session actor finished starting, sending cleanup message", "session_id", sessionData.SessionID)
	err := ctx.ActorSystem().ScheduleOnce(ctx.Context(), &mcppb.TryCleanupIfUninitialized{}, ctx.Self(), sessionData.InitializeTimeout)
	if err != nil {
		return utils.MessageHandlingResult{}, fmt.Errorf("failed to send cleanup message: %w", err)
	}

	return utils.Stay(sessionData)
}

// handleRegisterConnection handles the RegisterConnection message
func handleRegisterConnection(ctx *actor.ReceiveContext, sessionData *SessionData, msg *mcppb.RegisterConnection) (utils.MessageHandlingResult, error) {
	sender := ctx.Sender()
	sessionData.LastActivity = time.Now()
	sessionData.ClientConnectionActors[msg.GetConnectionId()] = sender
	ctx.Response(&mcppb.RegisterConnectionResponse{Success: true})
	return utils.Stay(sessionData)
}

// handleWrappedRequestUninitialized handles wrapped requests in the uninitialized state
func handleWrappedRequestUninitialized(rctx *actor.ReceiveContext, sessionData *SessionData, msg *mcppb.WrappedRequest) (utils.MessageHandlingResult, error) {
	ctx := context.WithValue(rctx.Context(), utils.SessionIdCtx, sessionData.SessionID)
	// In uninitialized state, we only accept initialize requests
	switch msg.Request.Method {
	case "initialize":
		response := handleInitialize(ctx, sessionData, msg.Request)
		sendResponse(rctx, ctx, sessionData, msg, response)
		sessionData.LastActivity = time.Now()

		// Transition to initialized state
		nextState := StateInitialized
		return utils.MessageHandlingResult{
			NextStateId: &nextState,
			NextData:    sessionData,
		}, nil

	default:
		// Return error for non-initialize requests in uninitialized state
		rctx.Logger().Info("mcp session actor got non-lifecycle message before being initialized", "session_id", sessionData.SessionID)
		errorResp := utils.CreateErrorResponse(msg.Request, -32002, "Server not initialized", nil)
		sendResponse(rctx, ctx, sessionData, msg, errorResp)

		return utils.Stay(sessionData)
	}
}

// handleWrappedRequestInitialized handles wrapped requests in the initialized state
func handleWrappedRequestInitialized(rctx *actor.ReceiveContext, sessionData *SessionData, msg *mcppb.WrappedRequest) (utils.MessageHandlingResult, error) {
	ctx := context.WithValue(rctx.Context(), utils.SessionIdCtx, sessionData.SessionID)
	// Handle the request based on the method
	switch msg.Request.Method {
	case "shutdown":
		response := handleShutdown(msg.Request)
		sendResponse(rctx, ctx, sessionData, msg, response)

		// Transition to shutdown state
		nextState := StateShutdown
		return utils.MessageHandlingResult{
			NextStateId: &nextState,
			NextData:    sessionData,
		}, nil

	case "notifications/initialized":
		slog.InfoContext(ctx, "Handling notifications/initialized request", "session_id", sessionData.SessionID)
		// This is a notification that initialization is complete
		sessionData.LastActivity = time.Now()
		sessionData.ClientNotificationsInitialized = true
		sessionData.LastActivity = time.Now()
		return utils.Stay(sessionData)
	default:
		// Handle non-lifecycle messages
		response, err := handleNonLifecycleRequest(ctx, sessionData, msg.Request.Id, msg.Request)
		if err != nil {
			var retErr *mcppb.JsonRpcResponse

			var jsonRpcError *protocol.JsonRpcError
			if errors.As(err, &jsonRpcError) {
				retErr = utils.CreateErrorResponseFromJsonRpcError(msg.Request, jsonRpcError)
			} else {
				hndlErr := protocol.NewInternalError("problem handling message", msg.Request.Id)
				retErr = utils.CreateErrorResponseFromJsonRpcError(msg.Request, hndlErr)
			}

			sendResponse(rctx, ctx, sessionData, msg, retErr)
			slog.ErrorContext(ctx, "problem handling non-lifecycle message", "session_id", sessionData.SessionID, "err", err)
			return utils.Stay(sessionData)
		}

		sendResponse(rctx, ctx, sessionData, msg, response)
		sessionData.LastActivity = time.Now()
		return utils.Stay(sessionData)
	}
}

// handleTryCleanupIfUninitialized handles the TryCleanupIfUninitialized message
func handleTryCleanupIfUninitialized(ctx *actor.ReceiveContext, sessionData *SessionData) (utils.MessageHandlingResult, error) {
	slog.InfoContext(ctx.Context(), "handling cleanup request - session is uninitialized, shutting down", "session_id", sessionData.SessionID)
	err := ctx.Self().Shutdown(ctx.Context())
	if err != nil {
		ctx.Logger().Error("failed to shut down session actor", "session_id", sessionData.SessionID, "error", err)
		return utils.MessageHandlingResult{}, err
	}
	return utils.Stay(sessionData)
}

// handleTryCleanupIfUninitialized handles the TryCleanupIfUninitialized message
func handleTryCleanupInitialized(ctx *actor.ReceiveContext, sessionData *SessionData) (utils.MessageHandlingResult, error) {
	slog.InfoContext(ctx.Context(), "handling cleanup request - session is initialized, scheduling periodic session check", "session_id", sessionData.SessionID)

	sessionData.LastActivity = time.Now()

	err := ctx.ActorSystem().Schedule(ctx.Context(), &mcppb.CheckSessionTTL{}, ctx.Self(), sessionData.SessionTimeout/2)
	if err != nil {
		return utils.MessageHandlingResult{}, fmt.Errorf("problem scheduling check_session_ttl: %w", err)
	}

	return utils.Stay(sessionData)
}

// handleCheckSessionTTL handles the CheckSessionTTL message
func handleCheckSessionTTL(ctx *actor.ReceiveContext, sessionData *SessionData) (utils.MessageHandlingResult, error) {
	timeoutAt := sessionData.LastActivity.Add(sessionData.SessionTimeout)
	slog.InfoContext(ctx.Context(), "checking if session is alive", "session_id", sessionData.SessionID, "timeout_at", timeoutAt)
	if timeoutAt.Before(time.Now()) {
		slog.InfoContext(ctx.Context(), fmt.Sprintf("session has had no activity since %s, shutting down", sessionData.LastActivity.String()), "session_id", sessionData.SessionID)
		ctx.Logger().Info("mcp session actor timeout", "session_id", sessionData.SessionID)
		utils.Shutdown(ctx)
	}
	return utils.Stay(sessionData)
}

// sendResponse sends a response to the client
func sendResponse(rctx *actor.ReceiveContext, ctx context.Context, sessionData *SessionData, wrappedMsg *mcppb.WrappedRequest, response *mcppb.JsonRpcResponse) {
	if wrappedMsg.IsAsk {
		rctx.Response(response)
	} else {
		rc, ok := sessionData.ClientConnectionActors[wrappedMsg.RespondToConnectionId]
		if !ok {
			slog.ErrorContext(ctx, "could not find actor to respond for connection to", "connectionId", wrappedMsg.RespondToConnectionId)
			return
		}
		rctx.Tell(rc, response)
	}
}

// handleInitialize processes an initialize request
func handleInitialize(ctx context.Context, sessionData *SessionData, req *mcppb.JsonRpcRequest) *mcppb.JsonRpcResponse {
	slog.InfoContext(ctx, "Handling initialize request", "session_id", sessionData.SessionID)

	// Create base response
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

	// Parse the parameters
	if req.ParamsJson == "" {
		return utils.CreateErrorResponse(req, -32602, "Invalid params: missing parameters", nil)
	}

	// Parse into our type
	var params protocol.InitializeParams
	if err := json.Unmarshal([]byte(req.ParamsJson), &params); err != nil {
		return utils.CreateErrorResponse(req, -32602, "Invalid params: "+err.Error(), nil)
	}

	// Check protocol version
	supportedVersions := []protocol.ProtocolVersion{protocol.ProtocolVersion20241105, protocol.ProtocolVersion20250326}
	versionSupported := false
	for _, v := range supportedVersions {
		if params.ProtocolVersion == v {
			versionSupported = true
			break
		}
	}

	if !versionSupported {
		errorData := map[string]interface{}{
			"supportedVersions": supportedVersions,
		}
		return utils.CreateErrorResponse(req, -32602, "Unsupported protocol version", errorData)
	}

	// Store client info and capabilities
	sessionData.ProtocolVersion = params.ProtocolVersion
	sessionData.ClientInfo = params.ClientInfo
	sessionData.LastActivity = time.Now()

	// Create the result
	result := protocol.InitializeResult{
		ProtocolVersion: params.ProtocolVersion,
		ServerInfo: protocol.ServerInfo{
			Name:    "scaled-mcp-server",
			Version: "1.0.0", // TODO: Get from config
		},
		Capabilities: sessionData.ServerInfo.GetServerCapabilities(),
		SessionID:    sessionData.SessionID,
	}

	// Convert result to JSON
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return utils.CreateErrorResponse(req, -32603, "Internal error: "+err.Error(), nil)
	}

	response.Response = &mcppb.JsonRpcResponse_ResultJson{
		ResultJson: string(resultJSON),
	}
	return response
}

// handleShutdown processes a shutdown request
func handleShutdown(req *mcppb.JsonRpcRequest) *mcppb.JsonRpcResponse {
	// Create base response
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

	// Create empty result
	response.Response = &mcppb.JsonRpcResponse_ResultJson{
		ResultJson: "{}",
	}

	return response
}

// handleNonLifecycleRequest processes other MCP requests
func handleNonLifecycleRequest(ctx context.Context, sessionData *SessionData, messageId interface{}, req *mcppb.JsonRpcRequest) (*mcppb.JsonRpcResponse, error) {
	// Create base response
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

	// Check if this is a tool-related method
	exc := sessionData.ServerInfo.GetExecutors()
	if sessionData.ServerInfo.GetExecutors().CanHandleMethod(req.Method) {
		resp, err := exc.HandleMethod(ctx, req.Method, req)
		if err != nil {
			return nil, fmt.Errorf("problem handling non-lifecycle request: %w", err)
		}
		return resp, nil
	} else {
		return nil, protocol.NewMethodNotFoundError(req.Method, messageId)
	}
}
