package actors

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tochemey/goakt/v3/actor"
	"github.com/tochemey/goakt/v3/goaktpb"

	"github.com/traego/scaled-mcp/internal/logger"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/utils"
)

// InMemoryChannel is an in-memory implementation of OneWayChannel for testing
type InMemoryChannel struct {
	mu           sync.Mutex
	messages     []Message
	endpoints    []string
	closed       bool
	sendFunc     func(string, interface{}) error
	endpointFunc func(string) error
	closeFunc    func()
}

// Message represents a message sent through the channel
type Message struct {
	EventType string
	Data      interface{}
}

// NewInMemoryChannel creates a new in-memory channel
func NewInMemoryChannel() *InMemoryChannel {
	return &InMemoryChannel{
		messages:  make([]Message, 0),
		endpoints: make([]string, 0),
		closed:    false,
	}
}

// Send records a message sent through the channel
func (c *InMemoryChannel) Send(eventType string, data interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return assert.AnError
	}

	c.messages = append(c.messages, Message{
		EventType: eventType,
		Data:      data,
	})

	if c.sendFunc != nil {
		return c.sendFunc(eventType, data)
	}

	return nil
}

// SendEndpoint records an endpoint sent through the channel
func (c *InMemoryChannel) SendEndpoint(endpoint string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return assert.AnError
	}

	c.endpoints = append(c.endpoints, endpoint)

	if c.endpointFunc != nil {
		return c.endpointFunc(endpoint)
	}

	return nil
}

// Close marks the channel as closed
func (c *InMemoryChannel) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true

	if c.closeFunc != nil {
		c.closeFunc()
	}
}

// GetMessages returns all messages sent through the channel
func (c *InMemoryChannel) GetMessages() []Message {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.messages
}

// GetEndpoints returns all endpoints sent through the channel
func (c *InMemoryChannel) GetEndpoints() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.endpoints
}

// IsClosed returns whether the channel is closed
func (c *InMemoryChannel) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.closed
}

// SetSendFunc sets a custom function to be called when Send is called
func (c *InMemoryChannel) SetSendFunc(f func(string, interface{}) error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sendFunc = f
}

// SetEndpointFunc sets a custom function to be called when SendEndpoint is called
func (c *InMemoryChannel) SetEndpointFunc(f func(string) error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.endpointFunc = f
}

// SetCloseFunc sets a custom function to be called when Close is called
func (c *InMemoryChannel) SetCloseFunc(f func()) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closeFunc = f
}

// MockSessionActor is a mock implementation of the session actor
type MockSessionActor struct {
	registerFunc func(*mcppb.RegisterConnection) *mcppb.RegisterConnectionResponse
}

// NewMockSessionActor creates a new mock session actor
func NewMockSessionActor(registerFunc func(*mcppb.RegisterConnection) *mcppb.RegisterConnectionResponse) *MockSessionActor {
	if registerFunc == nil {
		// Default to successful registration
		registerFunc = func(*mcppb.RegisterConnection) *mcppb.RegisterConnectionResponse {
			return &mcppb.RegisterConnectionResponse{Success: true}
		}
	}

	return &MockSessionActor{
		registerFunc: registerFunc,
	}
}

func (m *MockSessionActor) PreStart(ctx context.Context) error {
	return nil
}

func (m *MockSessionActor) Receive(ctx *actor.ReceiveContext) {
	msg := ctx.Message()
	switch msg := msg.(type) {
	case *mcppb.RegisterConnection:
		ctx.Response(m.registerFunc(msg))
	default:
		// Do nothing
	}
}

func (m *MockSessionActor) PostStop(ctx context.Context) error {
	return nil
}

func TestClientConnectionActor(t *testing.T) {
	// Create a new actor system
	ctx := context.Background()
	actorSystem, err := actor.NewActorSystem("test-system",
		actor.WithPassivationDisabled(),
		actor.WithLogger(logger.DiscardSlogLogger),
	)
	require.NoError(t, err)

	// Start the actor system
	err = actorSystem.Start(ctx)
	require.NoError(t, err)

	// Ensure we clean up after the test
	t.Cleanup(func() {
		err := actorSystem.Stop(ctx)
		require.NoError(t, err)
	})

	t.Run("should initialize with default SSE connection", func(t *testing.T) {
		// Create a channel
		channel := NewInMemoryChannel()

		// Create the client connection actor
		sessionId := "test-session-id"
		cca := NewClientConnectionActor(
			config.DefaultConfig(),
			sessionId,
			nil,
			channel,
			true,
			true, // defaultSseConnection = true
			"",
		)

		// Spawn the actor
		ccaPID, err := actorSystem.Spawn(ctx, "test-client-conn", cca)
		require.NoError(t, err)

		// Give some time for the actor to initialize
		time.Sleep(100 * time.Millisecond)

		// Verify the actor was created correctly
		assert.NotNil(t, ccaPID)

		// We don't send PostStart in this test - just checking initialization
		// Clean up without asserting - the actor may already be stopping
		poison := &goaktpb.PoisonPill{}
		_ = actor.Tell(ctx, ccaPID, poison)

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("should initialize with custom connection ID", func(t *testing.T) {
		// Create a channel
		channel := NewInMemoryChannel()

		// Create the client connection actor
		sessionId := "test-session-id"
		cca := NewClientConnectionActor(
			config.DefaultConfig(),
			sessionId,
			nil,
			channel,
			true,
			false, // defaultSseConnection = false
			"",
		)

		// Spawn the actor
		ccaPID, err := actorSystem.Spawn(ctx, "test-client-conn-custom", cca)
		require.NoError(t, err)

		// Give some time for the actor to initialize
		time.Sleep(100 * time.Millisecond)

		// Verify the actor was created correctly
		assert.NotNil(t, ccaPID)

		// Clean up without asserting - the actor may already be stopping
		poison := &goaktpb.PoisonPill{}
		_ = actor.Tell(ctx, ccaPID, poison)

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("should handle session actor not found", func(t *testing.T) {
		// Create a channel
		channel := NewInMemoryChannel()

		// Create the client connection actor
		sessionId := "nonexistent-session"
		cca := NewClientConnectionActor(
			config.DefaultConfig(),
			sessionId,
			nil,
			channel,
			true,
			true,
			"",
		)

		// Spawn the actor
		ccaPID, err := actorSystem.Spawn(ctx, "test-client-conn-no-session", cca)
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(500 * time.Millisecond)

		// Verify the connection is closed instead of sending an error
		assert.True(t, channel.IsClosed(), "Expected connection to be closed when session actor not found")

		// The actor should have shut itself down, but we'll try to clean up anyway
		err = ccaPID.Shutdown(ctx)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("should handle successful session registration", func(t *testing.T) {
		// Create a channel
		channel := NewInMemoryChannel()

		// Create a mock session actor
		mockSession := NewMockSessionActor(nil) // Use default success response

		// Spawn the mock session actor
		sessionId := "test-session-success"
		sessionActorName := utils.GetSessionActorName(sessionId)
		sessionPID, err := actorSystem.Spawn(ctx, sessionActorName, mockSession)
		require.NoError(t, err)

		// Create the client connection actor
		cca := NewClientConnectionActor(
			config.DefaultConfig(),
			sessionId,
			nil,
			channel,
			true,
			true,
			"",
		)

		// Spawn the actor
		ccaPID, err := actorSystem.Spawn(ctx, "test-client-conn-success", cca)
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(500 * time.Millisecond)

		// Verify an endpoint was sent
		assert.GreaterOrEqual(t, len(channel.GetEndpoints()), 1)

		// Clean up
		err = sessionPID.Shutdown(ctx)
		require.NoError(t, err)
		err = ccaPID.Shutdown(ctx)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("should handle failed session registration", func(t *testing.T) {
		// Create a channel
		channel := NewInMemoryChannel()

		// Create a mock session actor with failed registration
		mockSession := NewMockSessionActor(func(*mcppb.RegisterConnection) *mcppb.RegisterConnectionResponse {
			return &mcppb.RegisterConnectionResponse{
				Success: false,
				Error:   "registration failed",
			}
		})

		// Spawn the mock session actor
		sessionId := "test-session-fail"
		sessionActorName := utils.GetSessionActorName(sessionId)
		sessionPID, err := actorSystem.Spawn(ctx, sessionActorName, mockSession)
		require.NoError(t, err)

		// Create the client connection actor
		cca := NewClientConnectionActor(
			config.DefaultConfig(),
			sessionId,
			nil,
			channel,
			true,
			true,
			"",
		)

		// Spawn the actor
		_, err = actorSystem.Spawn(ctx, "test-client-conn-fail", cca)
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(500 * time.Millisecond)

		// Clean up
		poison := &goaktpb.PoisonPill{}
		err = actor.Tell(ctx, sessionPID, poison)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("should handle JSON-RPC response messages", func(t *testing.T) {
		// Create a channel
		channel := NewInMemoryChannel()

		// Create a mock session actor
		mockSession := NewMockSessionActor(nil) // Use default success response

		// Spawn the mock session actor
		sessionId := "test-session-json"
		sessionActorName := utils.GetSessionActorName(sessionId)
		sessionPID, err := actorSystem.Spawn(ctx, sessionActorName, mockSession)
		require.NoError(t, err)

		// Create the client connection actor
		cca := NewClientConnectionActor(
			config.DefaultConfig(),
			sessionId,
			nil,
			channel,
			false, // Don't send endpoint
			true,
			"",
		)

		// Spawn the actor
		ccaPID, err := actorSystem.Spawn(ctx, "test-client-conn-json", cca)
		require.NoError(t, err)

		// Send PostStart message to trigger session registration
		err = actor.Tell(ctx, ccaPID, &goaktpb.PostStart{})
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(500 * time.Millisecond)

		// Create a JSON-RPC response message
		jsonRpcResponse := &mcppb.JsonRpcResponse{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcResponse_StringId{
				StringId: "test-id",
			},
		}

		// Send the JSON-RPC response to the client connection actor
		err = actor.Tell(ctx, ccaPID, jsonRpcResponse)
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(500 * time.Millisecond)

		// Verify a message was sent
		messages := channel.GetMessages()
		assert.GreaterOrEqual(t, len(messages), 1)
		assert.Equal(t, "message", messages[0].EventType)

		// Clean up
		poison := &goaktpb.PoisonPill{}
		err = actor.Tell(ctx, sessionPID, poison)
		require.NoError(t, err)
		_ = actor.Tell(ctx, ccaPID, poison)

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("should handle terminated messages", func(t *testing.T) {
		// Create a channel
		channel := NewInMemoryChannel()

		// Create a mock session actor
		mockSession := NewMockSessionActor(nil) // Use default success response

		// Spawn the mock session actor
		sessionId := "test-session-json"
		sessionActorName := utils.GetSessionActorName(sessionId)
		_, err := actorSystem.Spawn(ctx, sessionActorName, mockSession)
		require.NoError(t, err)

		cca := NewClientConnectionActor(
			config.DefaultConfig(),
			sessionId,
			nil,
			channel,
			false,
			true,
			"",
		)

		// Spawn the actor
		ccaPID, err := actorSystem.Spawn(ctx, "test-client-conn-terminated", cca)
		require.NoError(t, err)

		// wait for the actor to be initialized
		time.Sleep(time.Second)

		// Create a terminated message
		terminatedMsg := &goaktpb.Terminated{
			ActorId: utils.GetSessionActorName(sessionId),
		}

		// Send the message to the actor
		err = actor.Tell(ctx, ccaPID, terminatedMsg)
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(500 * time.Millisecond)
	})

	t.Run("should handle unknown messages", func(t *testing.T) {
		// Create a channel
		channel := NewInMemoryChannel()

		// Create a mock session actor
		mockSession := NewMockSessionActor(nil) // Use default success response

		// Spawn the mock session actor
		sessionId := "test-session-unknown"
		sessionActorName := utils.GetSessionActorName(sessionId)
		sessionPID, err := actorSystem.Spawn(ctx, sessionActorName, mockSession)
		require.NoError(t, err)

		// Create the client connection actor
		cca := NewClientConnectionActor(
			config.DefaultConfig(),
			sessionId,
			nil,
			channel,
			true,
			true,
			"",
		)

		// Spawn the actor
		ccaPID, err := actorSystem.Spawn(ctx, "test-client-conn-unknown", cca)
		require.NoError(t, err)

		// Send PostStart message to trigger session registration
		err = actor.Tell(ctx, ccaPID, &goaktpb.PostStart{})
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(500 * time.Millisecond)

		// Send an unknown message type
		// Use PoisonPill here as an unexpected message - in ClientConnectionActor's Receive method
		// PoisonPill isn't handled explicitly, so it will hit the default case
		err = actor.Tell(ctx, ccaPID, &goaktpb.PoisonPill{})
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(500 * time.Millisecond)

		// Clean up
		poison := &goaktpb.PoisonPill{}
		err = actor.Tell(ctx, sessionPID, poison)
		require.NoError(t, err)
		_ = actor.Tell(ctx, ccaPID, poison)

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("should handle SendEndpoint error", func(t *testing.T) {
		// Create a channel with a custom error function for SendEndpoint
		channel := NewInMemoryChannel()
		channel.SetEndpointFunc(func(string) error {
			return assert.AnError
		})

		// Create a mock session actor
		mockSession := NewMockSessionActor(nil) // Use default success response

		// Spawn the mock session actor
		sessionId := "test-session-endpoint-error"
		sessionActorName := utils.GetSessionActorName(sessionId)
		sessionPID, err := actorSystem.Spawn(ctx, sessionActorName, mockSession)
		require.NoError(t, err)

		// Create the client connection actor
		cca := NewClientConnectionActor(
			config.DefaultConfig(),
			sessionId,
			nil,
			channel,
			true,
			true,
			"",
		)

		// Spawn the actor
		ccaPID, err := actorSystem.Spawn(ctx, "test-client-conn-endpoint-error", cca)
		require.NoError(t, err)

		// Send PostStart message to trigger session registration
		err = actor.Tell(ctx, ccaPID, &goaktpb.PostStart{})
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(500 * time.Millisecond)

		// Verify endpoint was attempted
		assert.GreaterOrEqual(t, len(channel.GetEndpoints()), 1)

		// Clean up
		poison := &goaktpb.PoisonPill{}
		err = actor.Tell(ctx, sessionPID, poison)
		require.NoError(t, err)
		_ = actor.Tell(ctx, ccaPID, poison)

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("should handle Send error", func(t *testing.T) {
		// Create a channel with custom Send function that returns an error
		channel := NewInMemoryChannel()
		channel.SetSendFunc(func(eventType string, data interface{}) error {
			return assert.AnError
		})

		// Create a mock session actor
		mockSession := NewMockSessionActor(nil) // Use default success response

		// Spawn the mock session actor
		sessionId := "test-session-send-error"
		sessionActorName := utils.GetSessionActorName(sessionId)
		sessionPID, err := actorSystem.Spawn(ctx, sessionActorName, mockSession)
		require.NoError(t, err)

		// Create the client connection actor
		cca := NewClientConnectionActor(
			config.DefaultConfig(),
			sessionId,
			nil,
			channel,
			true,
			true,
			"",
		)

		// Spawn the actor
		ccaPID, err := actorSystem.Spawn(ctx, "test-client-conn-send-error", cca)
		require.NoError(t, err)

		// Send PostStart message to trigger session registration
		err = actor.Tell(ctx, ccaPID, &goaktpb.PostStart{})
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(500 * time.Millisecond)

		// Create a JSON-RPC response message
		jsonRpcResponse := &mcppb.JsonRpcResponse{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcResponse_StringId{
				StringId: "test-id",
			},
		}

		// Send the JSON-RPC response to the client connection actor
		err = actor.Tell(ctx, ccaPID, jsonRpcResponse)
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(500 * time.Millisecond)

		// Clean up
		poison := &goaktpb.PoisonPill{}
		err = actor.Tell(ctx, sessionPID, poison)
		require.NoError(t, err)
		_ = actor.Tell(ctx, ccaPID, poison)

		time.Sleep(100 * time.Millisecond)
	})
}
