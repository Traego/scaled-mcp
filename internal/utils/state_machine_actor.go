package utils

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/tochemey/goakt/v3/actor"
)

// StateID represents the identifier for a specific state
type StateID string

// Data represents the internal data carried by the state machine actor
type Data interface{}

type MessageHandlingResult struct {
	NextStateId *StateID
	NextData    Data
}

// StateHandler defines the behavior for a specific state
type StateHandler func(ctx *actor.ReceiveContext, data Data) (MessageHandlingResult, error)

// StateHandlerMap maps state identifiers to their handlers
type StateHandlerMap map[StateID]StateHandler

// StateMachineActor implements an actor that behaves as a finite state machine
type StateMachineActor struct {
	// Current state identifier
	currentState StateID

	// Initial state identifier
	initialState StateID

	// State handlers
	stateHandlers StateHandlerMap

	// Actor data
	data Data

	// Unhandled message handler (optional)
	unhandledHandler func(ctx *actor.ReceiveContext, data Data, message interface{}) Data
}

// NewStateMachineActor creates a new state machine actor
func NewStateMachineActor(initialState StateID, data Data) *StateMachineActor {
	return &StateMachineActor{
		currentState:     initialState,
		initialState:     initialState,
		stateHandlers:    make(StateHandlerMap),
		data:             data,
		unhandledHandler: nil,
	}
}

// When registers a handler for a specific state
func (a *StateMachineActor) When(state StateID, handler StateHandler) *StateMachineActor {
	a.stateHandlers[state] = handler
	return a
}

// WhenUnhandled registers a handler for unhandled messages
func (a *StateMachineActor) WhenUnhandled(handler func(ctx *actor.ReceiveContext, data Data, message interface{}) Data) *StateMachineActor {
	a.unhandledHandler = handler
	return a
}

// PreStart is called when the actor is started
func (a *StateMachineActor) PreStart(ctx context.Context) error {
	slog.DebugContext(ctx, "Starting state machine actor", "initial_state", a.initialState)
	return nil
}

// Receive handles messages sent to the actor
func (a *StateMachineActor) Receive(ctx *actor.ReceiveContext) {
	message := ctx.Message()

	// Log state transition information for debugging
	msgType := reflect.TypeOf(message).String()
	ctx.Logger().Debug("StateMachineActor processing message",
		"current_state", a.currentState,
		"message_type", msgType)

	// If the current state has a handler, process the message
	if handler, exists := a.stateHandlers[a.currentState]; exists {
		result, err := handler(ctx, a.data)
		if err != nil {
			ctx.Err(fmt.Errorf("error processing message: %w", err))
		}

		// Check if state transition is needed
		if result.NextStateId != nil && *result.NextStateId != a.currentState {
			ctx.Logger().Debug("StateMachineActor state transition",
				"from", a.currentState,
				"to", *result.NextStateId)
			a.currentState = *result.NextStateId
		}

		// Update data
		a.data = result.NextData
		return
	}

	// If message was not handled by the current state handler
	if a.unhandledHandler != nil {
		// Use the unhandled message handler
		a.data = a.unhandledHandler(ctx, a.data, message)
	} else {
		// Mark message as unhandled
		ctx.Unhandled()
		ctx.Logger().Warn("Unhandled message in state machine actor",
			"current_state", a.currentState,
			"message_type", msgType)
	}
}

// PostStop is called when the actor is stopped
func (a *StateMachineActor) PostStop(ctx context.Context) error {
	slog.DebugContext(ctx, "Stopping state machine actor", "final_state", a.currentState)
	return nil
}

// Stay returns the same state and data, indicating no transition is needed
func Stay(nextData Data) (MessageHandlingResult, error) {
	return MessageHandlingResult{
		NextStateId: nil,
		NextData:    nextData,
	}, nil
}

// Goto initiates a state transition
func Goto(nextState StateID, nextData Data) (MessageHandlingResult, error) {
	return MessageHandlingResult{
		NextStateId: &nextState,
		NextData:    nextData,
	}, nil
}

// Shutdown initiates actor shutdown
func Shutdown(ctx *actor.ReceiveContext) {
	ctx.Shutdown()
}

// GetCurrentState returns the current state of the actor
func (a *StateMachineActor) GetCurrentState() StateID {
	return a.currentState
}

// GetData returns the current data of the actor
func (a *StateMachineActor) GetData() Data {
	return a.data
}
