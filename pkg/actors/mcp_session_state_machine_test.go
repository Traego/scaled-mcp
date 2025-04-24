package actors

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tochemey/goakt/v3/actor"

	"github.com/traego/scaled-mcp/internal/logger"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/resources"
	"github.com/traego/scaled-mcp/pkg/utils"
)

// TestExecutor is a real implementation of config.MethodHandler for testing
type TestExecutor struct {
	methodHandlers map[string]func(context.Context, *mcppb.JsonRpcRequest) (*mcppb.JsonRpcResponse, error)
}

func NewTestExecutor() *TestExecutor {
	return &TestExecutor{
		methodHandlers: map[string]func(context.Context, *mcppb.JsonRpcRequest) (*mcppb.JsonRpcResponse, error){
			"test/method": func(ctx context.Context, req *mcppb.JsonRpcRequest) (*mcppb.JsonRpcResponse, error) {
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

				response.Response = &mcppb.JsonRpcResponse_ResultJson{
					ResultJson: `{"success": true}`,
				}

				return response, nil
			},
		},
	}
}

func (e *TestExecutor) CanHandleMethod(method string) bool {
	_, exists := e.methodHandlers[method]
	return exists
}

func (e *TestExecutor) HandleMethod(ctx context.Context, method string, req *mcppb.JsonRpcRequest) (*mcppb.JsonRpcResponse, error) {
	handler, exists := e.methodHandlers[method]
	if !exists {
		return nil, protocol.NewMethodNotFoundError(method, req.Id)
	}
	return handler(ctx, req)
}

// TestServerInfo is a real implementation of config.McpServerInfo for testing
type TestServerInfo struct {
	serverCaps   protocol.ServerCapabilities
	serverConfig *config.ServerConfig
	executors    config.MethodHandler
	registry     resources.FeatureRegistry
}

func NewTestServerInfo(executors config.MethodHandler) config.McpServerInfo {
	return &TestServerInfo{
		serverCaps: protocol.ServerCapabilities{
			Tools: &protocol.ToolsServerCapability{
				ListChanged: true,
			},
			Experimental: map[string]interface{}{
				"version": "1.0.0",
			},
		},
		serverConfig: &config.ServerConfig{
			ProtocolVersion: protocol.ProtocolVersion20250326,
			Session: config.SessionConfig{
				TTL: 1 * time.Minute,
			},
		},
		executors: executors,
		registry:  resources.FeatureRegistry{},
	}
}

func (s *TestServerInfo) GetServerCapabilities() protocol.ServerCapabilities {
	return s.serverCaps
}

func (s *TestServerInfo) GetServerConfig() *config.ServerConfig {
	return s.serverConfig
}

func (s *TestServerInfo) GetExecutors() config.MethodHandler {
	return s.executors
}

func (s *TestServerInfo) GetFeatureRegistry() resources.FeatureRegistry {
	return s.registry
}

// TestConnectionActor is a real implementation of a client connection actor for testing
type TestConnectionActor struct {
	receivedMessages []interface{}
	t                *testing.T
}

func NewTestConnectionActor(t *testing.T) *TestConnectionActor {
	return &TestConnectionActor{
		receivedMessages: make([]interface{}, 0),
		t:                t,
	}
}

func (a *TestConnectionActor) PreStart(ctx context.Context) error {
	return nil
}

func (a *TestConnectionActor) Receive(ctx *actor.ReceiveContext) {
	msg := ctx.Message()
	a.receivedMessages = append(a.receivedMessages, msg)

	// Log the received message for debugging
	if a.t != nil {
		a.t.Logf("TestConnectionActor received message: %T", msg)
	}

	// If it's a RegisterConnection message, respond with success
	if _, ok := msg.(*mcppb.RegisterConnection); ok {
		ctx.Response(&mcppb.RegisterConnectionResponse{Success: true})
	}
}

func (a *TestConnectionActor) PostStop(ctx context.Context) error {
	return nil
}

func (a *TestConnectionActor) GetReceivedMessages() []interface{} {
	return a.receivedMessages
}

// initializeSession is a helper function that performs the complete initialization dance for a session actor
// It sends both the initialize request and the notifications/initialized notification
func initializeSession(
	ctx context.Context,
	t *testing.T,
	pid *actor.PID,
	protocolVersion protocol.ProtocolVersion,
	connectionId string,
) (*protocol.InitializeResult, error) {
	// Create initialize request
	initializeParams := protocol.InitializeParams{
		ProtocolVersion: protocolVersion,
		ClientInfo: protocol.ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}

	paramsJSON, err := json.Marshal(initializeParams)
	require.NoError(t, err)

	initRequest := &mcppb.JsonRpcRequest{
		Jsonrpc: "2.0",
		Id: &mcppb.JsonRpcRequest_StringId{
			StringId: "init-req",
		},
		Method:     "initialize",
		ParamsJson: string(paramsJSON),
	}

	wrappedRequest := &mcppb.WrappedRequest{
		Request:               initRequest,
		IsAsk:                 true,
		RespondToConnectionId: connectionId,
	}

	var result protocol.InitializeResult

	// Send initialize request
	resp, err := actor.Ask(ctx, pid, wrappedRequest, 60*time.Second)
	if err != nil {
		return nil, err
	}

	// Verify response
	jsonRpcResponse, ok := resp.(*mcppb.JsonRpcResponse)
	require.True(t, ok)

	// Parse the result
	err = json.Unmarshal([]byte(jsonRpcResponse.GetResultJson()), &result)
	if err != nil {
		return nil, err
	}

	// Now send the notifications/initialized notification to complete the initialization dance
	initializedNotification := &mcppb.JsonRpcRequest{
		Jsonrpc: "2.0",
		Method:  "notifications/initialized",
	}

	wrappedNotification := &mcppb.WrappedRequest{
		Request:               initializedNotification,
		IsAsk:                 false, // Notifications are always Tell, not Ask
		RespondToConnectionId: connectionId,
	}

	err = actor.Tell(ctx, pid, wrappedNotification)
	if err != nil {
		return nil, err
	}

	// Wait a bit for the notification to be processed
	time.Sleep(50 * time.Millisecond)

	return &result, nil
}

func TestMcpSessionStateMachine(t *testing.T) {
	// Create a new actor system for testing
	ctx := context.Background()
	actorSystem, err := actor.NewActorSystem("test-system",
		actor.WithPassivationDisabled(),
		actor.WithLogger(logger.DiscardSlogLogger))
	require.NoError(t, err)

	err = actorSystem.Start(ctx)
	require.NoError(t, err)

	_, err = actorSystem.Spawn(ctx, "root", &RootActor{})
	require.NoError(t, err)

	// Ensure we clean up after the test
	t.Cleanup(func() {
		err := actorSystem.Stop(ctx)
		require.NoError(t, err)
	})

	t.Run("should handle PostStart message", func(t *testing.T) {
		// Create server info with test executor
		executor := NewTestExecutor()
		serverInfo := NewTestServerInfo(executor)

		// Create session actor
		sessionID := "test-session-1"
		sessionActor := NewMcpSessionStateMachine(serverInfo, sessionID)

		// Spawn the actor
		pid, err := actorSystem.Spawn(ctx, "test-session-1", sessionActor)
		require.NoError(t, err)

		// PostStart is automatically called when the actor is spawned
		// Wait for the actor to process the PostStart message
		time.Sleep(50 * time.Millisecond)

		// Verify the actor is still alive
		_, err = actor.Ask(ctx, pid, &mcppb.RegisterConnection{
			ConnectionId: "test-conn-1",
		}, 100*time.Millisecond)
		require.NoError(t, err, "Actor should be alive after PostStart")

		// Clean up
		err = pid.Shutdown(ctx)
		require.NoError(t, err)
	})

	t.Run("should handle RegisterConnection message", func(t *testing.T) {
		// Create server info with test executor
		executor := NewTestExecutor()
		serverInfo := NewTestServerInfo(executor)

		// Create session actor
		sessionID := "test-session-2"
		sessionActor := NewMcpSessionStateMachine(serverInfo, sessionID)

		// Spawn the actor
		pid, err := actorSystem.Spawn(ctx, "test-session-2", sessionActor)
		require.NoError(t, err)

		// Send RegisterConnection message
		resp, err := actor.Ask(ctx, pid, &mcppb.RegisterConnection{
			ConnectionId: "test-conn-2",
		}, 100*time.Millisecond)
		require.NoError(t, err)

		// Verify response
		registerResp, ok := resp.(*mcppb.RegisterConnectionResponse)
		require.True(t, ok)
		assert.True(t, registerResp.Success)

		// Clean up
		err = pid.Shutdown(ctx)
		require.NoError(t, err)
	})

	t.Run("should transition to initialized state on initialize request", func(t *testing.T) {
		// Create server info with test executor
		executor := NewTestExecutor()
		serverInfo := NewTestServerInfo(executor)

		// Create session actor
		sessionID := "test-session-3"
		sessionActor := NewMcpSessionStateMachine(serverInfo, sessionID)

		// Get the state machine to verify state transitions
		stateMachine, ok := sessionActor.(*utils.StateMachineActor)
		require.True(t, ok)

		// Spawn the actor
		pid, err := actorSystem.Spawn(ctx, "test-session-3", sessionActor)
		require.NoError(t, err)

		// Perform the complete initialization dance
		result, err := initializeSession(ctx, t, pid, protocol.ProtocolVersion20250326, "test-conn-3")
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify the result
		assert.Equal(t, protocol.ProtocolVersion20250326, result.ProtocolVersion)
		assert.Equal(t, sessionID, result.SessionID)

		// Verify state transition
		assert.Equal(t, StateInitialized, stateMachine.GetCurrentState())

		// Clean up
		err = pid.Shutdown(ctx)
		require.NoError(t, err)
	})

	t.Run("should reject non-initialize requests in uninitialized state", func(t *testing.T) {
		// Create server info with test executor
		executor := NewTestExecutor()
		serverInfo := NewTestServerInfo(executor)

		// Create session actor
		sessionID := "test-session-4"
		sessionActor := NewMcpSessionStateMachine(serverInfo, sessionID)

		// Spawn the actor
		pid, err := actorSystem.Spawn(ctx, "test-session-4", sessionActor)
		require.NoError(t, err)

		// Create a non-initialize request
		nonInitRequest := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "non-init-1",
			},
			Method:     "test/method",
			ParamsJson: "{}",
		}

		wrappedRequest := &mcppb.WrappedRequest{
			Request:               nonInitRequest,
			IsAsk:                 true, // Use Ask instead of Tell
			RespondToConnectionId: "",   // Not needed for Ask
		}

		// Send non-initialize request in uninitialized state and get response directly
		response, err := actor.Ask(ctx, pid, wrappedRequest, 500*time.Millisecond)
		require.NoError(t, err)
		require.NotNil(t, response)

		// Verify the response is a JSON-RPC response
		jsonRpcResponse, ok := response.(*mcppb.JsonRpcResponse)
		require.True(t, ok, "Response should be a JsonRpcResponse")

		// Verify error response
		assert.Equal(t, "2.0", jsonRpcResponse.Jsonrpc)
		assert.Equal(t, "non-init-1", jsonRpcResponse.GetStringId())

		// Check that it's an error response
		errorResp := jsonRpcResponse.GetError()
		require.NotNil(t, errorResp)
		assert.Equal(t, int32(-32002), errorResp.Code)
		assert.Contains(t, errorResp.Message, "Server not initialized")

		// Clean up
		err = pid.Shutdown(ctx)
		require.NoError(t, err)
	})

	t.Run("should handle shutdown request in initialized state", func(t *testing.T) {
		// Create server info with test executor
		executor := NewTestExecutor()
		serverInfo := NewTestServerInfo(executor)

		// Create session actor
		sessionID := "test-session-5"
		sessionActor := NewMcpSessionStateMachine(serverInfo, sessionID)

		// Get the state machine to verify state transitions
		stateMachine, ok := sessionActor.(*utils.StateMachineActor)
		require.True(t, ok)

		// Spawn the actor
		pid, err := actorSystem.Spawn(ctx, "test-session-5", sessionActor)
		require.NoError(t, err)

		// Perform the complete initialization dance
		_, err = initializeSession(ctx, t, pid, protocol.ProtocolVersion20250326, "test-conn-5")
		require.NoError(t, err)

		// Verify state transition to initialized
		assert.Equal(t, StateInitialized, stateMachine.GetCurrentState())

		// Now send shutdown request
		shutdownRequest := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "shutdown-5",
			},
			Method: "shutdown",
		}

		wrappedShutdownRequest := &mcppb.WrappedRequest{
			Request:               shutdownRequest,
			IsAsk:                 true,
			RespondToConnectionId: "test-conn-5",
		}

		// Send shutdown request
		resp, err := actor.Ask(ctx, pid, wrappedShutdownRequest, 100*time.Millisecond)
		require.NoError(t, err)

		// Verify response
		jsonRpcResponse, ok := resp.(*mcppb.JsonRpcResponse)
		require.True(t, ok)
		assert.Equal(t, "2.0", jsonRpcResponse.Jsonrpc)
		assert.Equal(t, "shutdown-5", jsonRpcResponse.GetStringId())

		// Verify state transition to shutdown
		assert.Equal(t, StateShutdown, stateMachine.GetCurrentState())

		// Clean up
		err = pid.Shutdown(ctx)
		require.NoError(t, err)
	})

	t.Run("should handle non-lifecycle requests in initialized state", func(t *testing.T) {
		// Create server info with test executor
		executor := NewTestExecutor()
		serverInfo := NewTestServerInfo(executor)

		// Create session actor
		sessionID := "test-session-4"
		sessionActor := NewMcpSessionStateMachine(serverInfo, sessionID)

		// Get the state machine to verify state transitions
		stateMachine, ok := sessionActor.(*utils.StateMachineActor)
		require.True(t, ok)

		// Spawn the actor
		pid, err := actorSystem.Spawn(ctx, "test-session-4", sessionActor)
		require.NoError(t, err)

		// Perform the complete initialization dance
		_, err = initializeSession(ctx, t, pid, protocol.ProtocolVersion20250326, "test-conn-4")
		require.NoError(t, err)

		// Verify state transition
		assert.Equal(t, StateInitialized, stateMachine.GetCurrentState())

		// Now send a non-lifecycle request
		testRequest := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "test-method-1",
			},
			Method:     "test/method",
			ParamsJson: "{}",
		}

		wrappedRequest := &mcppb.WrappedRequest{
			Request:               testRequest,
			IsAsk:                 true,
			RespondToConnectionId: "test-conn-4",
		}

		// Send test method request
		resp, err := actor.Ask(ctx, pid, wrappedRequest, 100*time.Millisecond)
		require.NoError(t, err)

		// Verify response
		jsonRpcResponse, ok := resp.(*mcppb.JsonRpcResponse)
		require.True(t, ok)
		assert.Equal(t, "2.0", jsonRpcResponse.Jsonrpc)
		assert.Equal(t, "test-method-1", jsonRpcResponse.GetStringId())

		// Parse the result
		var resultMap map[string]interface{}
		err = json.Unmarshal([]byte(jsonRpcResponse.GetResultJson()), &resultMap)
		require.NoError(t, err)

		// Verify the result
		success, ok := resultMap["success"]
		require.True(t, ok)
		assert.Equal(t, true, success)

		// Clean up
		err = pid.Shutdown(ctx)
		require.NoError(t, err)
	})

	t.Run("should handle TryCleanupPreInitialized message for uninitialized session", func(t *testing.T) {
		// Create server info with test executor
		executor := NewTestExecutor()
		serverInfo := NewTestServerInfo(executor)

		// Create session actor
		sessionID := "test-session-7"
		sessionActor := NewMcpSessionStateMachine(serverInfo, sessionID)

		// Spawn the actor
		pid, err := actorSystem.Spawn(ctx, "test-session-7", sessionActor)
		require.NoError(t, err)

		// Verify the actor is initially alive
		_, err = actor.Ask(ctx, pid, &mcppb.RegisterConnection{
			ConnectionId: "test-conn-7",
		}, 100*time.Millisecond)
		require.NoError(t, err, "Actor should be alive initially")

		// Send TryCleanupPreInitialized message to trigger cleanup of uninitialized session
		err = actor.Tell(ctx, pid, &mcppb.TryCleanupIfUninitialized{})
		require.NoError(t, err)

		// Give some time for the message to be processed and actor to stop
		time.Sleep(200 * time.Millisecond)

		// Try to send a message to the actor - this should fail if the actor has stopped
		_, err = actor.Ask(ctx, pid, &mcppb.RegisterConnection{
			ConnectionId: "test-conn-7-after",
		}, 100*time.Millisecond)

		// The actor should be stopped since it wasn't initialized
		assert.Error(t, err, "Actor should be stopped after TryCleanupPreInitialized for uninitialized session")
	})

	t.Run("should handle CheckSessionTTL message for expired session", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping test in short mode")
		}

		// Create server info with test executor but with a very short TTL
		executor := NewTestExecutor()
		serverInfo := &TestServerInfo{
			serverCaps: protocol.ServerCapabilities{},
			serverConfig: &config.ServerConfig{
				ProtocolVersion: protocol.ProtocolVersion20250326,
				Session: config.SessionConfig{
					TTL: 100 * time.Millisecond, // Short TTL for testing
				},
			},
			executors: executor,
			registry:  resources.FeatureRegistry{},
		}

		// Create session actor
		sessionID := "test-session-8"
		sessionActor := NewMcpSessionStateMachine(serverInfo, sessionID)

		// Spawn the actor
		pid, err := actorSystem.Spawn(ctx, "test-session-8", sessionActor)
		require.NoError(t, err)

		// Initialize the session
		initRequest := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "init-1",
			},
			Method: "initialize",
			ParamsJson: `{
				"protocolVersion": "2025-03-26",
				"capabilities": {},
				"clientInfo": {
					"name": "test-client",
					"version": "1.0.0"
				}
			}`,
		}

		wrappedRequest := &mcppb.WrappedRequest{
			Request: initRequest,
			IsAsk:   true,
		}

		//Send initialize request
		initResponse, err := actor.Ask(ctx, pid, wrappedRequest, 10*time.Second)
		require.NoError(t, err, "Initialize request should succeed")
		require.NotNil(t, initResponse, "Initialize response should not be nil")

		// Wait for the session to expire
		time.Sleep(1 * time.Second)

		isAlive := pid.IsRunning()
		require.False(t, isAlive, "Actor must have died")
	})

	t.Run("should handle unhandled message", func(t *testing.T) {
		// Create server info with test executor
		executor := NewTestExecutor()
		serverInfo := NewTestServerInfo(executor)

		// Create session actor
		sessionID := "test-session-9"
		sessionActor := NewMcpSessionStateMachine(serverInfo, sessionID)

		// Spawn the actor
		pid, err := actorSystem.Spawn(ctx, "test-session-9", sessionActor)
		require.NoError(t, err)

		// Send an unknown message type
		err = actor.Tell(ctx, pid, &mcppb.StringMsg{Message: "unknown"})
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(100 * time.Millisecond)

		// The actor should still be alive
		_, err = actor.Ask(ctx, pid, &mcppb.RegisterConnection{
			ConnectionId: "test-conn-9",
		}, 100*time.Millisecond)
		require.NoError(t, err, "Actor should still be alive after handling unhandled message")

		// Clean up
		err = pid.Shutdown(ctx)
		require.NoError(t, err)
	})

	t.Run("should ignore messages in shutdown state", func(t *testing.T) {
		// Create server info with test executor
		executor := NewTestExecutor()
		serverInfo := NewTestServerInfo(executor)

		// Create a test connection actor to receive the response
		connActor := NewTestConnectionActor(t)
		connPid, err := actorSystem.Spawn(ctx, "test-conn-10", connActor)
		require.NoError(t, err)

		// Create session actor
		sessionID := "test-session-10"
		sessionActor := NewMcpSessionStateMachine(serverInfo, sessionID)

		// Get the state machine to verify state transitions
		stateMachine, ok := sessionActor.(*utils.StateMachineActor)
		require.True(t, ok)

		// Spawn the actor
		pid, err := actorSystem.Spawn(ctx, "test-session-10", sessionActor)
		require.NoError(t, err)

		// Register the connection
		registerResp, err := actor.Ask(ctx, pid, &mcppb.RegisterConnection{
			ConnectionId: "test-conn-10",
		}, 500*time.Millisecond)
		require.NoError(t, err)
		require.NotNil(t, registerResp)

		// Perform the complete initialization dance
		_, err = initializeSession(ctx, t, pid, protocol.ProtocolVersion20250326, "test-conn-10")
		require.NoError(t, err)

		// Verify state is initialized
		assert.Equal(t, StateInitialized, stateMachine.GetCurrentState())

		// Now send shutdown request
		shutdownRequest := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "shutdown-10",
			},
			Method: "shutdown",
		}

		wrappedShutdownRequest := &mcppb.WrappedRequest{
			Request:               shutdownRequest,
			IsAsk:                 false, // Use Tell instead of Ask
			RespondToConnectionId: "test-conn-10",
		}

		// Send shutdown request
		err = actor.Tell(ctx, pid, wrappedShutdownRequest)
		require.NoError(t, err)

		// Wait for the response to be sent to the connection actor
		time.Sleep(500 * time.Millisecond)

		// Verify state transition
		assert.Equal(t, StateShutdown, stateMachine.GetCurrentState())

		// Now try to send a message in shutdown state
		testRequest := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "test-method-10",
			},
			Method:     "test/method",
			ParamsJson: "{}",
		}

		wrappedTestRequest := &mcppb.WrappedRequest{
			Request:               testRequest,
			IsAsk:                 false, // Use Tell instead of Ask
			RespondToConnectionId: "test-conn-10",
		}

		// Send test method request in shutdown state
		err = actor.Tell(ctx, pid, wrappedTestRequest)
		require.NoError(t, err)

		// Wait for the message to be processed
		time.Sleep(500 * time.Millisecond)

		// Send CheckSessionTTL message to trigger final shutdown
		err = actor.Tell(ctx, pid, &mcppb.CheckSessionTTL{})
		require.NoError(t, err)

		// Give more time for the message to be processed and actor to stop
		time.Sleep(500 * time.Millisecond)

		// Try to send a message to the actor - this should fail if the actor has stopped
		_, err = actor.Ask(ctx, pid, &mcppb.RegisterConnection{
			ConnectionId: "test-conn-10-after",
		}, 500*time.Millisecond)

		// The actor should be stopped after CheckSessionTTL in shutdown state
		assert.Error(t, err, "Actor should be stopped after CheckSessionTTL in shutdown state")

		// Clean up
		err = connPid.Shutdown(ctx)
		require.NoError(t, err)
	})

	t.Run("should not shut down after pre-initialize timeout if already initialized", func(t *testing.T) {
		// Create server info with test executor
		executor := NewTestExecutor()
		// Create a server config with a very short TTL for faster testing
		serverInfo := &TestServerInfo{
			serverCaps: protocol.ServerCapabilities{
				Tools: &protocol.ToolsServerCapability{
					ListChanged: true,
				},
			},
			serverConfig: &config.ServerConfig{
				ProtocolVersion: protocol.ProtocolVersion20250326,
				//Session: config.SessionConfig{
				//	// Use a very short TTL to speed up the test
				//	TTL: 60 * time.Second,
				//},
			},
			executors: executor,
		}

		// Create session actor
		sessionID := "test-session-ttl"
		sessionActor := NewMcpSessionStateMachine(serverInfo, sessionID)

		// Get the state machine to verify state transitions
		stateMachine, ok := sessionActor.(*utils.StateMachineActor)
		require.True(t, ok)

		// Spawn the actor
		pid, err := actorSystem.Spawn(ctx, "test-session-ttl", sessionActor)
		require.NoError(t, err)

		// Perform the complete initialization dance
		_, err = initializeSession(ctx, t, pid, protocol.ProtocolVersion20250326, "test-conn-ttl")
		require.NoError(t, err)

		// Verify state transition to initialized
		assert.Equal(t, StateInitialized, stateMachine.GetCurrentState())

		// Wait for the pre-initialize timeout to occur (TTL/10 + a bit more)
		// This should trigger the TryCleanupPreInitialized message
		waitTime := 100 * time.Millisecond
		t.Logf("Waiting %v for pre-initialize timeout", waitTime)
		time.Sleep(waitTime)

		// The actor should still be alive and in the initialized state
		// If the bug exists, the actor will have shut down
		isRunning := pid.IsRunning()
		t.Logf("Actor running status: %v", isRunning)

		// Try to send a message to the actor to verify it's still alive
		registerResp, err := actor.Ask(ctx, pid, &mcppb.RegisterConnection{
			ConnectionId: "test-conn-ttl-after",
		}, 100*time.Millisecond)

		// If the bug exists, this will fail with an error
		if err != nil {
			t.Logf("Failed to send message to actor: %v", err)
			t.Fail()
		} else {
			t.Logf("Successfully sent message to actor, response: %v", registerResp)
		}

		// Verify the actor is still in the initialized state
		assert.Equal(t, StateInitialized, stateMachine.GetCurrentState())

		// Clean up
		err = pid.Shutdown(ctx)
		require.NoError(t, err)
	})
}
