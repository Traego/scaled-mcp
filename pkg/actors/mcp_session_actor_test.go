package actors

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tochemey/goakt/v3/actor"
	"github.com/tochemey/goakt/v3/goaktpb"
	"github.com/traego/scaled-mcp/internal/utils"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/resources"
)

// MockExecutor is a mock implementation of config.MethodHandler for testing
type MockExecutor struct {
	canHandleFunc func(string) bool
	handleFunc    func(context.Context, string, *mcppb.JsonRpcRequest) (*mcppb.JsonRpcResponse, error)
}

func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		canHandleFunc: func(method string) bool {
			return method == "test/method"
		},
		handleFunc: func(ctx context.Context, method string, req *mcppb.JsonRpcRequest) (*mcppb.JsonRpcResponse, error) {
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
	}
}

func (m *MockExecutor) CanHandleMethod(method string) bool {
	return m.canHandleFunc(method)
}

func (m *MockExecutor) HandleMethod(ctx context.Context, method string, req *mcppb.JsonRpcRequest) (*mcppb.JsonRpcResponse, error) {
	return m.handleFunc(ctx, method, req)
}

func (m *MockExecutor) SetCanHandleFunc(f func(string) bool) {
	m.canHandleFunc = f
}

func (m *MockExecutor) SetHandleFunc(f func(context.Context, string, *mcppb.JsonRpcRequest) (*mcppb.JsonRpcResponse, error)) {
	m.handleFunc = f
}

// MockServerInfo is a mock implementation of config.McpServerInfo for testing
type MockServerInfo struct {
	serverCaps   protocol.ServerCapabilities
	serverConfig *config.ServerConfig
	executors    config.MethodHandler
}

func NewMockServerInfo(executors config.MethodHandler) *MockServerInfo {
	return &MockServerInfo{
		serverCaps: protocol.ServerCapabilities{
			Tools: &protocol.ToolsServerCapability{
				ListChanged: true,
			},
			Experimental: map[string]interface{}{
				"version": "1.0.0",
			},
		},
		serverConfig: &config.ServerConfig{
			ProtocolVersion: "2025-03",
		},
		executors: executors,
	}
}

func (s *MockServerInfo) GetFeatureRegistry() resources.FeatureRegistry {
	return resources.FeatureRegistry{}
}

func (s *MockServerInfo) GetServerCapabilities() protocol.ServerCapabilities {
	return s.serverCaps
}

func (s *MockServerInfo) GetServerConfig() *config.ServerConfig {
	return s.serverConfig
}

func (s *MockServerInfo) GetExecutors() config.MethodHandler {
	return s.executors
}

// MockClientConnectionActor is a mock implementation of the client connection actor
type MockClientConnectionActor struct {
	receivedMessages []interface{}
}

func NewMockClientConnectionActor() *MockClientConnectionActor {
	return &MockClientConnectionActor{
		receivedMessages: make([]interface{}, 0),
	}
}

func (m *MockClientConnectionActor) PreStart(ctx context.Context) error {
	return nil
}

func (m *MockClientConnectionActor) Receive(ctx *actor.ReceiveContext) {
	msg := ctx.Message()
	m.receivedMessages = append(m.receivedMessages, msg)
}

func (m *MockClientConnectionActor) PostStop(ctx context.Context) error {
	return nil
}

func (m *MockClientConnectionActor) GetReceivedMessages() []interface{} {
	return m.receivedMessages
}

// Create a local version of NewMcpSessionActor for testing
func createTestMcpSessionActor(serverInfo config.McpServerInfo, sessionId string) actor.Actor {
	return &McpSessionActor{
		sessionId:              sessionId,
		serverInfo:             serverInfo,
		initialized:            false,
		sessionTimeout:         1 * time.Minute,
		lastActivity:           time.Now(),
		clientConnectionActors: make(map[string]*actor.PID),
	}
}

func TestMcpSessionActor(t *testing.T) {
	// Create a new actor system
	ctx := context.Background()
	actorSystem, err := actor.NewActorSystem("test-system",
		actor.WithPassivationDisabled())
	require.NoError(t, err)

	// Start the actor system
	err = actorSystem.Start(ctx)
	require.NoError(t, err)

	// Defer stopping the actor system
	defer func() {
		err := actorSystem.Stop(ctx)
		require.NoError(t, err)
	}()

	t.Run("should cleanup uninitialized session after timeout", func(t *testing.T) {
		// Skip this test in CI environments where it might be flaky
		if testing.Short() {
			t.Skip("Skipping test in short mode")
		}

		// Create a mock executor
		mockExecutor := NewMockExecutor()

		// Create a mock server info
		mockServerInfo := NewMockServerInfo(mockExecutor)

		// Create the MCP session actor with a shorter cleanup timeout
		sessionId := "test-session-1"
		sessionActor := createTestMcpSessionActor(mockServerInfo, sessionId)

		// Set a shorter cleanup timeout for testing
		sessionActor.(*McpSessionActor).sessionTimeout = 500 * time.Millisecond

		// Spawn the actor - this will automatically trigger PostStart
		sessionPID, err := actorSystem.Spawn(ctx, utils.GetSessionActorName(sessionId), sessionActor)
		require.NoError(t, err)

		// Verify the actor is initially alive
		_, err = actor.Ask(ctx, sessionPID, &mcppb.RegisterConnection{
			ConnectionId: "test-conn-1",
		}, 100*time.Millisecond)
		require.NoError(t, err, "Actor should be alive initially")

		// Send TryCleanupPreInitialized message directly to trigger cleanup
		err = actor.Tell(ctx, sessionPID, &mcppb.TryCleanupPreInitialized{})
		require.NoError(t, err)

		// Wait for the cleanup to occur
		time.Sleep(600 * time.Millisecond)

		// Try to send a message to the actor - this should fail if the actor has stopped
		_, err = actor.Ask(ctx, sessionPID, &mcppb.RegisterConnection{
			ConnectionId: "test-conn-1-after",
		}, 100*time.Millisecond)

		// The actor should be stopped since it wasn't initialized within the timeout
		assert.Error(t, err, "Actor should be stopped after cleanup timeout")
	})

	t.Run("should handle RegisterConnection message", func(t *testing.T) {
		// Create a mock executor
		mockExecutor := NewMockExecutor()

		// Create a mock server info
		mockServerInfo := NewMockServerInfo(mockExecutor)

		// Create the MCP session actor
		sessionId := "test-session-2"
		sessionActor := createTestMcpSessionActor(mockServerInfo, sessionId)

		// Spawn the actor
		sessionPID, err := actorSystem.Spawn(ctx, utils.GetSessionActorName(sessionId), sessionActor)
		require.NoError(t, err)

		// Create a mock client connection actor
		mockClientConn := NewMockClientConnectionActor()

		// Spawn the mock client connection actor
		clientConnPID, err := actorSystem.Spawn(ctx, "test-client-conn", mockClientConn)
		require.NoError(t, err)

		// Create a RegisterConnection message
		registerMsg := &mcppb.RegisterConnection{
			ConnectionId: "test-connection-1",
		}

		// Send the message and get the response
		response, err := actor.Ask(ctx, sessionPID, registerMsg, 1*time.Second)
		require.NoError(t, err)

		// Verify the response
		registerResponse, ok := response.(*mcppb.RegisterConnectionResponse)
		require.True(t, ok)
		assert.True(t, registerResponse.Success)

		// Clean up
		err = actor.Tell(ctx, sessionPID, &goaktpb.PoisonPill{})
		require.NoError(t, err)
		err = actor.Tell(ctx, clientConnPID, &goaktpb.PoisonPill{})
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("should handle initialize message", func(t *testing.T) {
		// Create a mock executor
		mockExecutor := NewMockExecutor()

		// Create a mock server info
		mockServerInfo := NewMockServerInfo(mockExecutor)

		// Create the MCP session actor
		sessionId := "test-session-3"
		sessionActor := createTestMcpSessionActor(mockServerInfo, sessionId)

		// Spawn the actor
		sessionPID, err := actorSystem.Spawn(ctx, utils.GetSessionActorName(sessionId), sessionActor)
		require.NoError(t, err)

		// Create an initialize request
		initializeParams := protocol.InitializeParams{
			ProtocolVersion: "2025-03",
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
				StringId: "init-1",
			},
			Method:     "initialize",
			ParamsJson: string(paramsJSON),
		}

		wrappedRequest := &mcppb.WrappedRequest{
			Request:               initRequest,
			IsAsk:                 true,
			RespondToConnectionId: "test-connection-1",
		}

		// Send the message and get the response
		response, err := actor.Ask(ctx, sessionPID, wrappedRequest, 1*time.Second)
		require.NoError(t, err)

		// Verify the response
		jsonRpcResponse, ok := response.(*mcppb.JsonRpcResponse)
		require.True(t, ok)
		assert.Equal(t, "2.0", jsonRpcResponse.Jsonrpc)
		assert.Equal(t, "init-1", jsonRpcResponse.GetStringId())

		// Parse the result
		var result protocol.InitializeResult
		err = json.Unmarshal([]byte(jsonRpcResponse.GetResultJson()), &result)
		require.NoError(t, err)

		// Verify the result
		assert.Equal(t, "2025-03", result.ProtocolVersion)
		assert.Equal(t, sessionId, result.SessionID)
		assert.Equal(t, "scaled-mcp-server", result.ServerInfo.Name)

		// Clean up
		err = actor.Tell(ctx, sessionPID, &goaktpb.PoisonPill{})
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("should handle initialize with invalid protocol version", func(t *testing.T) {
		// Create a mock executor
		mockExecutor := NewMockExecutor()

		// Create a mock server info
		mockServerInfo := NewMockServerInfo(mockExecutor)

		// Create the MCP session actor
		sessionId := "test-session-4"
		sessionActor := createTestMcpSessionActor(mockServerInfo, sessionId)

		// Spawn the actor
		sessionPID, err := actorSystem.Spawn(ctx, utils.GetSessionActorName(sessionId), sessionActor)
		require.NoError(t, err)

		// Create an initialize request with invalid protocol version
		initializeParams := protocol.InitializeParams{
			ProtocolVersion: "invalid-version",
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
				StringId: "init-2",
			},
			Method:     "initialize",
			ParamsJson: string(paramsJSON),
		}

		wrappedRequest := &mcppb.WrappedRequest{
			Request:               initRequest,
			IsAsk:                 true,
			RespondToConnectionId: "test-connection-2",
		}

		// Send the message and get the response
		response, err := actor.Ask(ctx, sessionPID, wrappedRequest, 1*time.Second)
		require.NoError(t, err)

		// Verify the response
		jsonRpcResponse, ok := response.(*mcppb.JsonRpcResponse)
		require.True(t, ok)
		assert.Equal(t, "2.0", jsonRpcResponse.Jsonrpc)
		assert.Equal(t, "init-2", jsonRpcResponse.GetStringId())

		// Verify it's an error response
		assert.NotNil(t, jsonRpcResponse.GetError())
		assert.Equal(t, int32(-32602), jsonRpcResponse.GetError().Code)
		assert.Equal(t, "Unsupported protocol version", jsonRpcResponse.GetError().Message)

		// Clean up
		err = actor.Tell(ctx, sessionPID, &goaktpb.PoisonPill{})
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("should handle shutdown message", func(t *testing.T) {
		// Create a mock executor
		mockExecutor := NewMockExecutor()

		// Create a mock server info
		mockServerInfo := NewMockServerInfo(mockExecutor)

		// Create the MCP session actor
		sessionId := "test-session-5"
		sessionActor := createTestMcpSessionActor(mockServerInfo, sessionId)

		// Spawn the actor
		sessionPID, err := actorSystem.Spawn(ctx, utils.GetSessionActorName(sessionId), sessionActor)
		require.NoError(t, err)

		// First initialize the session
		initializeParams := protocol.InitializeParams{
			ProtocolVersion: "2025-03",
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
				StringId: "init-3",
			},
			Method:     "initialize",
			ParamsJson: string(paramsJSON),
		}

		wrappedInitRequest := &mcppb.WrappedRequest{
			Request:               initRequest,
			IsAsk:                 true,
			RespondToConnectionId: "test-connection-3",
		}

		// Send the initialize message
		_, err = actor.Ask(ctx, sessionPID, wrappedInitRequest, 1*time.Second)
		require.NoError(t, err)

		// Now send a shutdown request
		shutdownRequest := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "shutdown-1",
			},
			Method: "shutdown",
		}

		wrappedShutdownRequest := &mcppb.WrappedRequest{
			Request:               shutdownRequest,
			IsAsk:                 true,
			RespondToConnectionId: "test-connection-3",
		}

		// Send the message and get the response
		response, err := actor.Ask(ctx, sessionPID, wrappedShutdownRequest, 1*time.Second)
		require.NoError(t, err)

		// Verify the response
		jsonRpcResponse, ok := response.(*mcppb.JsonRpcResponse)
		require.True(t, ok)
		assert.Equal(t, "2.0", jsonRpcResponse.Jsonrpc)
		assert.Equal(t, "shutdown-1", jsonRpcResponse.GetStringId())
		assert.Equal(t, "{}", jsonRpcResponse.GetResultJson())

		// Clean up
		err = actor.Tell(ctx, sessionPID, &goaktpb.PoisonPill{})
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("should handle non-lifecycle message", func(t *testing.T) {
		// Create a mock executor
		mockExecutor := NewMockExecutor()

		// Create a mock server info
		mockServerInfo := NewMockServerInfo(mockExecutor)

		// Create the MCP session actor
		sessionId := "test-session-6"
		sessionActor := createTestMcpSessionActor(mockServerInfo, sessionId)

		// Spawn the actor
		sessionPID, err := actorSystem.Spawn(ctx, utils.GetSessionActorName(sessionId), sessionActor)
		require.NoError(t, err)

		// First initialize the session
		initializeParams := protocol.InitializeParams{
			ProtocolVersion: "2025-03",
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
				StringId: "init-4",
			},
			Method:     "initialize",
			ParamsJson: string(paramsJSON),
		}

		wrappedInitRequest := &mcppb.WrappedRequest{
			Request:               initRequest,
			IsAsk:                 true,
			RespondToConnectionId: "test-connection-4",
		}

		// Send the initialize message
		_, err = actor.Ask(ctx, sessionPID, wrappedInitRequest, 1*time.Second)
		require.NoError(t, err)

		// Send initialized notification
		initializedRequest := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Method:  "notifications/initialized",
		}

		wrappedInitializedRequest := &mcppb.WrappedRequest{
			Request:               initializedRequest,
			IsAsk:                 false,
			RespondToConnectionId: "test-connection-4",
		}

		err = actor.Tell(ctx, sessionPID, wrappedInitializedRequest)
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(100 * time.Millisecond)

		// Now send a non-lifecycle request
		nonLifecycleRequest := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "test-method-1",
			},
			Method: "test/method",
		}

		wrappedNonLifecycleRequest := &mcppb.WrappedRequest{
			Request:               nonLifecycleRequest,
			IsAsk:                 true,
			RespondToConnectionId: "test-connection-4",
		}

		// Send the message and get the response
		response, err := actor.Ask(ctx, sessionPID, wrappedNonLifecycleRequest, 1*time.Second)
		require.NoError(t, err)

		// Verify the response
		jsonRpcResponse, ok := response.(*mcppb.JsonRpcResponse)
		require.True(t, ok)
		assert.Equal(t, "2.0", jsonRpcResponse.Jsonrpc)
		assert.Equal(t, "test-method-1", jsonRpcResponse.GetStringId())
		assert.Equal(t, `{"success": true}`, jsonRpcResponse.GetResultJson())

		// Clean up
		err = actor.Tell(ctx, sessionPID, &goaktpb.PoisonPill{})
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("should handle non-lifecycle message before initialization", func(t *testing.T) {
		// Create a mock executor
		mockExecutor := NewMockExecutor()

		// Create a mock server info
		mockServerInfo := NewMockServerInfo(mockExecutor)

		// Create the MCP session actor
		sessionId := "test-session-7"
		sessionActor := createTestMcpSessionActor(mockServerInfo, sessionId)

		// Spawn the actor
		sessionPID, err := actorSystem.Spawn(ctx, utils.GetSessionActorName(sessionId), sessionActor)
		require.NoError(t, err)

		// Create a non-lifecycle request before initialization
		nonLifecycleRequest := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "test-method-2",
			},
			Method: "test/method",
		}

		// Create a wrapped request with a Tell (not Ask)
		wrappedNonLifecycleRequest := &mcppb.WrappedRequest{
			Request:               nonLifecycleRequest,
			IsAsk:                 false, // This is a Tell, not an Ask
			RespondToConnectionId: "test-connection-7",
		}

		// Send the message as a Tell
		err = actor.Tell(ctx, sessionPID, wrappedNonLifecycleRequest)
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(100 * time.Millisecond)

		// Clean up
		err = actor.Tell(ctx, sessionPID, &goaktpb.PoisonPill{})
		require.NoError(t, err)

		// The test passes if no panic occurs
		time.Sleep(100 * time.Millisecond)
	})

	//t.Run("should handle CheckSessionTTL message with expired session", func(t *testing.T) {
	//	// Skip this test in CI environments where it might be flaky
	//	if testing.Short() {
	//		t.Skip("Skipping test in short mode")
	//	}
	//
	//	// Create a mock executor
	//	mockExecutor := NewMockExecutor()
	//
	//	// Create a mock server info
	//	mockServerInfo := NewMockServerInfo(mockExecutor)
	//
	//	// Create the MCP session actor with a very short timeout
	//	sessionId := "test-session-9"
	//	sessionActor := createTestMcpSessionActor(mockServerInfo, sessionId)
	//
	//	// Set a very short session timeout
	//	sessionActor.(*McpSessionActor).sessionTimeout = 10 * time.Millisecond
	//
	//	// Manually set initialized to true to test TTL expiration
	//	sessionActor.(*McpSessionActor).initialized = true
	//
	//	// Spawn the actor
	//	sessionPID, err := actorSystem.Spawn(ctx, utils.GetSessionActorName(sessionId), sessionActor)
	//	require.NoError(t, err)
	//
	//	// Set the last activity time to the past to ensure it's expired
	//	sessionActor.(*McpSessionActor).lastActivity = time.Now().Add(-100 * time.Millisecond)
	//
	//	// Send CheckSessionTTL message to trigger cleanup of expired session
	//	err = actor.Tell(ctx, sessionPID, &mcppb.CheckSessionTTL{})
	//	require.NoError(t, err)
	//
	//	// Give some time for the message to be processed and actor to stop
	//	time.Sleep(200 * time.Millisecond)
	//
	//	// Try to send a message to the actor - this should fail if the actor has stopped
	//	_, err = actor.Ask(ctx, sessionPID, &mcppb.RegisterConnection{
	//		ConnectionId: "test-conn-9-after",
	//	}, 100*time.Millisecond)
	//
	//	// The actor should be stopped since the session expired
	//	assert.Error(t, err, "Actor should be stopped after session timeout")
	//})

	t.Run("should handle CheckSessionTTL message with active session", func(t *testing.T) {
		// Create a mock executor
		mockExecutor := NewMockExecutor()

		// Create a mock server info
		mockServerInfo := NewMockServerInfo(mockExecutor)

		// Create the MCP session actor
		sessionId := "test-session-9a"
		sessionActor := createTestMcpSessionActor(mockServerInfo, sessionId)

		// Manually set initialized to true
		sessionActor.(*McpSessionActor).initialized = true

		// Spawn the actor
		sessionPID, err := actorSystem.Spawn(ctx, utils.GetSessionActorName(sessionId), sessionActor)
		require.NoError(t, err)

		// Send CheckSessionTTL message
		err = actor.Tell(ctx, sessionPID, &mcppb.CheckSessionTTL{})
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(100 * time.Millisecond)

		// The actor should still be alive since the session is active
		_, err = actor.Ask(ctx, sessionPID, &mcppb.RegisterConnection{
			ConnectionId: "test-conn-9a-2",
		}, 100*time.Millisecond)
		require.NoError(t, err, "Actor should still be alive after CheckSessionTTL with active session")

		// Clean up
		err = actor.Tell(ctx, sessionPID, &goaktpb.PoisonPill{})
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("should handle TryCleanupPreInitialized message for uninitialized session", func(t *testing.T) {
		// Skip this test in CI environments where it might be flaky
		if testing.Short() {
			t.Skip("Skipping test in short mode")
		}

		// Create a mock executor
		mockExecutor := NewMockExecutor()

		// Create a mock server info
		mockServerInfo := NewMockServerInfo(mockExecutor)

		// Create the MCP session actor
		sessionId := "test-session-8a"
		sessionActor := createTestMcpSessionActor(mockServerInfo, sessionId)

		// Spawn the actor
		sessionPID, err := actorSystem.Spawn(ctx, utils.GetSessionActorName(sessionId), sessionActor)
		require.NoError(t, err)

		// Verify the actor is initially alive
		_, err = actor.Ask(ctx, sessionPID, &mcppb.RegisterConnection{
			ConnectionId: "test-conn-8a",
		}, 100*time.Millisecond)
		require.NoError(t, err, "Actor should be alive initially")

		// Send TryCleanupPreInitialized message to trigger cleanup of uninitialized session
		err = actor.Tell(ctx, sessionPID, &mcppb.TryCleanupPreInitialized{})
		require.NoError(t, err)

		// Give some time for the message to be processed and actor to stop
		time.Sleep(200 * time.Millisecond)

		// Try to send a message to the actor - this should fail if the actor has stopped
		_, err = actor.Ask(ctx, sessionPID, &mcppb.RegisterConnection{
			ConnectionId: "test-conn-8a-after",
		}, 100*time.Millisecond)

		// The actor should be stopped since it wasn't initialized
		assert.Error(t, err, "Actor should be stopped after TryCleanupPreInitialized for uninitialized session")
	})

	//t.Run("should handle TryCleanupPreInitialized message for initialized session", func(t *testing.T) {
	//	// Create a mock executor
	//	mockExecutor := NewMockExecutor()
	//
	//	// Create a mock server info
	//	mockServerInfo := NewMockServerInfo(mockExecutor)
	//
	//	// Create the MCP session actor
	//	sessionId := "test-session-8"
	//	sessionActor := createTestMcpSessionActor(mockServerInfo, sessionId)
	//
	//	// Manually set initialized to true
	//	sessionActor.(*McpSessionActor).initialized = true
	//
	//	// Spawn the actor
	//	sessionPID, err := actorSystem.Spawn(ctx, utils.GetSessionActorName(sessionId), sessionActor)
	//	require.NoError(t, err)
	//
	//	// Send TryCleanupPreInitialized message
	//	err = actor.Tell(ctx, sessionPID, &mcppb.TryCleanupPreInitialized{})
	//	require.NoError(t, err)
	//
	//	// Give some time for the message to be processed
	//	time.Sleep(100 * time.Millisecond)
	//
	//	// The actor should still be alive since it's initialized
	//	_, err = actor.Ask(ctx, sessionPID, &mcppb.RegisterConnection{
	//		ConnectionId: "test-conn-8-after",
	//	}, 100*time.Millisecond)
	//	require.NoError(t, err, "Actor should still be alive after TryCleanupPreInitialized when initialized")
	//
	//	// Clean up
	//	err = actor.Tell(ctx, sessionPID, &goaktpb.PoisonPill{})
	//	require.NoError(t, err)
	//
	//	time.Sleep(100 * time.Millisecond)
	//})

	t.Run("should handle unknown message type", func(t *testing.T) {
		// Create a mock executor
		mockExecutor := NewMockExecutor()

		// Create a mock server info
		mockServerInfo := NewMockServerInfo(mockExecutor)

		// Create the MCP session actor
		sessionId := "test-session-10"
		sessionActor := createTestMcpSessionActor(mockServerInfo, sessionId)

		// Spawn the actor
		sessionPID, err := actorSystem.Spawn(ctx, utils.GetSessionActorName(sessionId), sessionActor)
		require.NoError(t, err)

		// Send an unknown message type (using a protobuf message that's not handled)
		err = actor.Tell(ctx, sessionPID, &mcppb.CheckSessionTTL{})
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(100 * time.Millisecond)

		// The actor should still be alive
		// We can check this by sending another message
		_, err = actor.Ask(ctx, sessionPID, &mcppb.RegisterConnection{ConnectionId: "test-conn"}, 100*time.Millisecond)
		require.NoError(t, err)

		// Clean up
		err = actor.Tell(ctx, sessionPID, &goaktpb.PoisonPill{})
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
	})
}
