package utils

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tochemey/goakt/v3/actor"
	"github.com/tochemey/goakt/v3/log"

	"github.com/traego/scaled-mcp/internal/logger"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
)

// Define test states
const (
	StateUninitialized StateID = "uninitialized"
	StateInitialized   StateID = "initialized"
	StateActive        StateID = "active"
	StateShutdown      StateID = "shutdown"
)

// Test actor data
type TestActorData struct {
	Counter         int
	LastMessageType string
}

// Message content constants for tests
const (
	MsgContentInitialize = "initialize"
	MsgContentActivate   = "activate"
	MsgContentShutdown   = "shutdown"
	MsgContentUnknown    = "unknown"
)

func TestStateMachineActor(t *testing.T) {
	// Create a new actor system for testing
	ctx := context.Background()
	actorSystem, err := actor.NewActorSystem("test-system",
		actor.WithPassivationDisabled(),
		actor.WithLogger(logger.DiscardSlogLogger))
	require.NoError(t, err)

	err = actorSystem.Start(ctx)
	require.NoError(t, err)
	defer func() {
		err := actorSystem.Stop(ctx)
		require.NoError(t, err)
	}()

	t.Run("should transition through states correctly", func(t *testing.T) {
		// Create initial data
		initialData := &TestActorData{Counter: 0}

		// Create a state machine actor
		fsm := NewStateMachineActor("test"+uuid.New().String(), StateUninitialized, initialData)

		// Configure state handlers
		fsm.When(StateUninitialized, func(ctx *actor.ReceiveContext, data Data) (MessageHandlingResult, error) {
			actorData := data.(*TestActorData)

			// Check message content
			if msg, ok := ctx.Message().(*mcppb.StringMsg); ok {
				if msg.GetMessage() == MsgContentInitialize {
					actorData.Counter++
					actorData.LastMessageType = MsgContentInitialize
					nextState := StateInitialized
					return MessageHandlingResult{
						NextStateId: &nextState,
						NextData:    actorData,
					}, nil
				}
			}
			return Stay(actorData)
		}).When(StateInitialized, func(ctx *actor.ReceiveContext, data Data) (MessageHandlingResult, error) {
			actorData := data.(*TestActorData)

			if msg, ok := ctx.Message().(*mcppb.StringMsg); ok {
				if msg.GetMessage() == MsgContentActivate {
					actorData.Counter++
					actorData.LastMessageType = MsgContentActivate
					nextState := StateActive
					return MessageHandlingResult{
						NextStateId: &nextState,
						NextData:    actorData,
					}, nil
				} else if msg.GetMessage() == MsgContentShutdown {
					actorData.Counter++
					actorData.LastMessageType = MsgContentShutdown
					Shutdown(ctx)
					nextState := StateShutdown
					return MessageHandlingResult{
						NextStateId: &nextState,
						NextData:    actorData,
					}, nil
				}
			}

			return Stay(actorData)
		}).When(StateActive, func(ctx *actor.ReceiveContext, data Data) (MessageHandlingResult, error) {
			actorData := data.(*TestActorData)

			if msg, ok := ctx.Message().(*mcppb.StringMsg); ok {
				if msg.GetMessage() == MsgContentShutdown {
					actorData.Counter++
					actorData.LastMessageType = MsgContentShutdown
					Shutdown(ctx)
					nextState := StateShutdown
					return MessageHandlingResult{
						NextStateId: &nextState,
						NextData:    actorData,
					}, nil
				}
			}

			// For any other message, stay in the current state
			return Stay(actorData)
		}).WhenUnhandled(func(ctx *actor.ReceiveContext, data Data, message interface{}) Data {
			actorData := data.(*TestActorData)
			actorData.LastMessageType = MsgContentUnknown
			return actorData
		})

		// Spawn the actor
		pid, err := actorSystem.Spawn(ctx, "test-fsm", fsm)
		require.NoError(t, err)

		// Test state transitions
		err = actor.Tell(ctx, pid, &mcppb.StringMsg{Message: MsgContentInitialize})
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(50 * time.Millisecond)

		// Validate state change to Initialized
		assert.Equal(t, StateInitialized, fsm.GetCurrentState())

		// Test next state transition
		err = actor.Tell(ctx, pid, &mcppb.StringMsg{Message: MsgContentActivate})
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(50 * time.Millisecond)

		// Validate state change to Active and data updates
		assert.Equal(t, StateActive, fsm.GetCurrentState())
		actorData, ok := fsm.GetData().(*TestActorData)
		require.True(t, ok)
		assert.Equal(t, 2, actorData.Counter)
		assert.Equal(t, MsgContentActivate, actorData.LastMessageType)

		// Test shutdown
		err = actor.Tell(ctx, pid, &mcppb.StringMsg{Message: MsgContentShutdown})
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(50 * time.Millisecond)

		// Validate state change to Shutdown
		assert.Equal(t, StateShutdown, fsm.GetCurrentState())
	})

	t.Run("should handle messages in a state with no handler", func(t *testing.T) {
		// Create a new actor system for this test to avoid interference
		testCtx := context.Background()
		testActorSystem, err := actor.NewActorSystem("test-unhandled-system-1",
			actor.WithPassivationDisabled(),
			actor.WithLogger(log.DefaultLogger))
		require.NoError(t, err)

		err = testActorSystem.Start(testCtx)
		require.NoError(t, err)
		defer func() {
			err := testActorSystem.Stop(testCtx)
			require.NoError(t, err)
		}()

		// Create initial data
		initialData := &TestActorData{
			Counter:         0,
			LastMessageType: "",
		}

		// Create a state machine actor with a state that has no handler
		fsm := NewStateMachineActor("test-"+uuid.New().String(), "no-handler-state", initialData)

		// Only configure the unhandled handler
		fsm.WhenUnhandled(func(ctx *actor.ReceiveContext, data Data, message interface{}) Data {
			actorData := data.(*TestActorData)
			actorData.LastMessageType = MsgContentUnknown
			return actorData
		})

		// Spawn the actor
		pid, err := testActorSystem.Spawn(testCtx, "test-unhandled-fsm", fsm)
		require.NoError(t, err)

		// Send a message - since there's no handler for the current state,
		// it should be handled by the unhandled handler
		err = actor.Tell(testCtx, pid, &mcppb.StringMsg{Message: MsgContentUnknown})
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(50 * time.Millisecond)

		// Validate the message was handled by the unhandled handler
		actorData, ok := fsm.GetData().(*TestActorData)
		require.True(t, ok)
		assert.Equal(t, MsgContentUnknown, actorData.LastMessageType)
	})

	t.Run("should handle message in unhandled handler when no state handler exists", func(t *testing.T) {
		// Create a new actor system for this test to avoid interference
		testCtx := context.Background()
		testActorSystem, err := actor.NewActorSystem("test-unhandled-system-2",
			actor.WithPassivationDisabled(),
			actor.WithLogger(log.DefaultLogger))
		require.NoError(t, err)

		err = testActorSystem.Start(testCtx)
		require.NoError(t, err)
		defer func() {
			err := testActorSystem.Stop(testCtx)
			require.NoError(t, err)
		}()

		// Create a fresh actor system for this test to avoid any interference
		testActorSystem2, err := actor.NewActorSystem("test-unhandled-system-3",
			actor.WithPassivationDisabled(),
			actor.WithLogger(log.DefaultLogger))
		require.NoError(t, err)

		err = testActorSystem2.Start(testCtx)
		require.NoError(t, err)
		defer func() {
			err := testActorSystem2.Stop(testCtx)
			require.NoError(t, err)
		}()

		// Create initial data with a clean state
		initialData := &TestActorData{
			Counter:         0,
			LastMessageType: "",
		}

		// Create a state machine actor with only unhandled message handler
		// Do NOT register any state handlers
		fsm := NewStateMachineActor("test-"+uuid.New().String(), StateUninitialized, initialData)

		// Configure only unhandled handler
		fsm.WhenUnhandled(func(ctx *actor.ReceiveContext, data Data, message interface{}) Data {
			actorData := data.(*TestActorData)
			actorData.LastMessageType = "handled-by-unhandled"
			actorData.Counter = 1 // Fix the counter issue
			return actorData
		})

		// Spawn the actor
		pid, err := testActorSystem2.Spawn(testCtx, "test-only-unhandled", fsm)
		require.NoError(t, err)

		// Send message that has no direct handler
		err = actor.Tell(testCtx, pid, &mcppb.StringMsg{Message: MsgContentUnknown})
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(50 * time.Millisecond)

		// Validate unhandled message behavior
		actorData, ok := fsm.GetData().(*TestActorData)
		require.True(t, ok)
		assert.Equal(t, 1, actorData.Counter)
		assert.Equal(t, "handled-by-unhandled", actorData.LastMessageType)
	})
}

// Example of using the state machine actor
func ExampleStateMachineActor() {
	// Define states
	const (
		StateOff  StateID = "off"
		StateOn   StateID = "on"
		StateBusy StateID = "busy"
	)

	// Define message types
	const (
		MsgTurnOn  = "turn-on"
		MsgTurnOff = "turn-off"
		MsgProcess = "process"
	)

	// Define actor data
	type DeviceData struct {
		ProcessedItems int
	}

	// Create a state machine actor
	deviceActor := NewStateMachineActor(uuid.New().String(), StateOff, &DeviceData{ProcessedItems: 0})

	// Configure state handlers
	deviceActor.When(StateOff, func(ctx *actor.ReceiveContext, data Data) (MessageHandlingResult, error) {
		deviceData := data.(*DeviceData)

		// Check for turn on message
		if msg, ok := ctx.Message().(*mcppb.StringMsg); ok && msg.GetMessage() == MsgTurnOn {
			nextState := StateOn
			return Goto(nextState, deviceData)
		}

		return Stay(deviceData)
	}).When(StateOn, func(ctx *actor.ReceiveContext, data Data) (MessageHandlingResult, error) {
		deviceData := data.(*DeviceData)

		if msg, ok := ctx.Message().(*mcppb.StringMsg); ok {
			if msg.GetMessage() == MsgTurnOff {
				// Transition to "off" state
				nextState := StateOff
				return Goto(nextState, deviceData)
			} else if msg.GetMessage() == MsgProcess {
				// Transition to "busy" state
				nextState := StateBusy
				return Goto(nextState, deviceData)
			}
		}

		return Stay(deviceData)
	}).When(StateBusy, func(ctx *actor.ReceiveContext, data Data) (MessageHandlingResult, error) {
		deviceData := data.(*DeviceData)

		if msg, ok := ctx.Message().(*mcppb.StringMsg); ok {
			if msg.GetMessage() == MsgProcess {
				// Process the item
				deviceData.ProcessedItems++
				// Return to "on" state after processing
				nextState := StateOn
				return Goto(nextState, deviceData)
			} else if msg.GetMessage() == MsgTurnOff {
				// Force shutdown even when busy
				Shutdown(ctx)
				nextState := StateOff
				return Goto(nextState, deviceData)
			}
		}

		return Stay(deviceData)
	})

	// Note: In a real application, you would spawn this actor in an actor system
}
