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
	ProtocolVersion string
	ClientInfo      protocol.ClientInfo
	Initialized     bool

	// Last activity time
	LastActivity time.Time

	// Session timeout duration
	SessionTimeout time.Duration

	// Connection actors
	ClientConnectionActors map[string]*actor.PID
}

// NewMcpSessionStateMachine creates a new MCP session state machine actor
func NewMcpSessionStateMachine(serverInfo config.McpServerInfo, sessionID string) actor.Actor {
	// Initialize session data
	data := &SessionData{
		SessionID:              sessionID,
		ServerInfo:             serverInfo,
		LastActivity:           time.Now(),
		SessionTimeout:         serverInfo.GetServerConfig().Session.TTL,
		ClientConnectionActors: make(map[string]*actor.PID),
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
	sessionData.LastActivity = time.Now()

	message := ctx.Message()
	switch msg := message.(type) {
	case *goaktpb.PostStart:
		return handlePostStart(ctx, sessionData)
	case *mcppb.RegisterConnection:
		return handleRegisterConnection(ctx, sessionData, msg)
	case *mcppb.WrappedRequest:
		return handleWrappedRequestUninitialized(ctx, sessionData, msg)
	case *mcppb.TryCleanupPreInitialized:
		return handleTryCleanupPreInitialized(ctx, sessionData)
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
	sessionData.LastActivity = time.Now()

	message := ctx.Message()
	switch msg := message.(type) {
	case *mcppb.RegisterConnection:
		return handleRegisterConnection(ctx, sessionData, msg)
	case *mcppb.WrappedRequest:
		return handleWrappedRequestInitialized(ctx, sessionData, msg)
	case *mcppb.CheckSessionTTL:
		return handleCheckSessionTTL(ctx, sessionData)
	case *mcppb.TryCleanupPreInitialized:
		return handleTryCleanupPreInitialized(ctx, sessionData)
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

// handlePostStart handles the PostStart message
func handlePostStart(ctx *actor.ReceiveContext, sessionData *SessionData) (utils.MessageHandlingResult, error) {
	ctx.Logger().Debug("mcp session actor finished starting, sending cleanup message", "session_id", sessionData.SessionID)
	err := ctx.ActorSystem().ScheduleOnce(ctx.Context(), &mcppb.TryCleanupPreInitialized{}, ctx.Self(), sessionData.ServerInfo.GetServerConfig().Session.TTL/10)
	if err != nil {
		return utils.MessageHandlingResult{}, fmt.Errorf("failed to send cleanup pre-initialized message: %w", err)
	}

	//actorutils.ScheduleOnce(ctx.Context(), ctx.Self().ActorSystem(), ctx.Self().Name(), &mcppb.TryCleanupPreInitialized{}, sessionData.ServerInfo.GetServerConfig().Session.TTL/10)
	return utils.Stay(sessionData)
}

// handleRegisterConnection handles the RegisterConnection message
func handleRegisterConnection(ctx *actor.ReceiveContext, sessionData *SessionData, msg *mcppb.RegisterConnection) (utils.MessageHandlingResult, error) {
	sender := ctx.Sender()
	sessionData.ClientConnectionActors[msg.GetConnectionId()] = sender
	ctx.Response(&mcppb.RegisterConnectionResponse{Success: true})
	return utils.Stay(sessionData)
}

// handleWrappedRequestUninitialized handles wrapped requests in the uninitialized state
func handleWrappedRequestUninitialized(ctx *actor.ReceiveContext, sessionData *SessionData, msg *mcppb.WrappedRequest) (utils.MessageHandlingResult, error) {
	// In uninitialized state, we only accept initialize requests
	switch msg.Request.Method {
	case "initialize":
		response := handleInitialize(ctx.Context(), sessionData, msg.Request)
		sendResponse(ctx, sessionData, msg, response)

		// Transition to initialized state
		nextState := StateInitialized
		return utils.MessageHandlingResult{
			NextStateId: &nextState,
			NextData:    sessionData,
		}, nil

	case "notifications/initialized":
		// This is a notification that initialization is complete
		nextState := StateInitialized
		return utils.MessageHandlingResult{
			NextStateId: &nextState,
			NextData:    sessionData,
		}, nil

	default:
		// Return error for non-initialize requests in uninitialized state
		ctx.Logger().Debug("mcp session actor got non-lifecycle message before being initialized", "session_id", sessionData.SessionID)
		errorResp := utils.CreateErrorResponse(msg.Request, -32002, "Server not initialized", nil)
		sendResponse(ctx, sessionData, msg, errorResp)

		return utils.Stay(sessionData)
	}
}

// handleWrappedRequestInitialized handles wrapped requests in the initialized state
func handleWrappedRequestInitialized(ctx *actor.ReceiveContext, sessionData *SessionData, msg *mcppb.WrappedRequest) (utils.MessageHandlingResult, error) {
	// Handle the request based on the method
	switch msg.Request.Method {
	case "shutdown":
		response := handleShutdown(msg.Request)
		sendResponse(ctx, sessionData, msg, response)

		// Transition to shutdown state
		nextState := StateShutdown
		return utils.MessageHandlingResult{
			NextStateId: &nextState,
			NextData:    sessionData,
		}, nil

	default:
		// Handle non-lifecycle messages
		response, err := handleNonLifecycleRequest(ctx.Context(), sessionData, msg.Request.Id, msg.Request)
		if err != nil {
			var retErr *mcppb.JsonRpcResponse

			var jsonRpcError *protocol.JsonRpcError
			if errors.As(err, &jsonRpcError) {
				retErr = utils.CreateErrorResponseFromJsonRpcError(msg.Request, jsonRpcError)
			} else {
				hndlErr := protocol.NewInternalError("problem handling message", msg.Request.Id)
				retErr = utils.CreateErrorResponseFromJsonRpcError(msg.Request, hndlErr)
			}

			sendResponse(ctx, sessionData, msg, retErr)
			slog.ErrorContext(ctx.Context(), "problem handling non-lifecycle message", "session_id", sessionData.SessionID, "err", err)
			return utils.Stay(sessionData)
		}

		sendResponse(ctx, sessionData, msg, response)
		return utils.Stay(sessionData)
	}
}

// handleTryCleanupPreInitialized handles the TryCleanupPreInitialized message
func handleTryCleanupPreInitialized(ctx *actor.ReceiveContext, sessionData *SessionData) (utils.MessageHandlingResult, error) {
	// Check if we're still in uninitialized state after the timeout
	// If so, shut down the actor
	if ctx.Self() != nil {
		err := ctx.Self().Shutdown(ctx.Context())
		if err != nil {
			slog.ErrorContext(ctx.Context(), "error in shutdown", "session_id", sessionData.SessionID)
		}
	}

	err := ctx.ActorSystem().Schedule(ctx.Context(), &mcppb.CheckSessionTTL{}, ctx.Self(), sessionData.ServerInfo.GetServerConfig().Session.TTL/6)
	if err != nil {
		return utils.MessageHandlingResult{}, fmt.Errorf("problem scheduling check_session_ttl: %w", err)
	}

	// Schedule the session TTL check
	//actorutils.Schedule(ctx.Context(), ctx.Self().ActorSystem(), ctx.Self().Name(), &mcppb.CheckSessionTTL{}, sessionData.ServerInfo.GetServerConfig().Session.TTL/6)

	return utils.Stay(sessionData)
}

// handleCheckSessionTTL handles the CheckSessionTTL message
func handleCheckSessionTTL(ctx *actor.ReceiveContext, sessionData *SessionData) (utils.MessageHandlingResult, error) {
	if sessionData.LastActivity.Add(sessionData.SessionTimeout).Before(time.Now()) {
		ctx.Logger().Debug("mcp session actor timeout", "session_id", sessionData.SessionID)
		utils.Shutdown(ctx)
	}
	return utils.Stay(sessionData)
}

// sendResponse sends a response to the client
func sendResponse(ctx *actor.ReceiveContext, sessionData *SessionData, wrappedMsg *mcppb.WrappedRequest, response *mcppb.JsonRpcResponse) {
	if wrappedMsg.IsAsk {
		ctx.Response(response)
	} else {
		rc, ok := sessionData.ClientConnectionActors[wrappedMsg.RespondToConnectionId]
		if !ok {
			slog.ErrorContext(ctx.Context(), "could not find actor to respond for connection to", "connectionId", wrappedMsg.RespondToConnectionId)
			return
		}
		ctx.Tell(rc, response)
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
	supportedVersions := []string{"2024-11-05", "2025-03-26"}
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
	sessionData.Initialized = true

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
