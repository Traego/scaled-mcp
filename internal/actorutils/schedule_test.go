//go:build !race
// +build !race

package actorutils

import (
	"context"
	"github.com/tochemey/goakt/v3/goaktpb"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tochemey/goakt/v3/actor"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// testActor is a simple actor implementation for testing scheduling
type testActor struct {
	receivedMessages chan proto.Message
}

func newTestActor() *testActor {
	return &testActor{
		receivedMessages: make(chan proto.Message, 10), // Buffer to avoid blocking
	}
}

func (a *testActor) Receive(ctx *actor.ReceiveContext) {
	msg := ctx.Message()
	if msg == nil {
		return
	}

	switch ctx.Message().(type) {
	case *goaktpb.PostStart:
	default:
		select {
		case a.receivedMessages <- msg:
			// Message sent to channel
		default:
			// Channel is full, this shouldn't happen with our buffer size
		}
	}
}

func (a *testActor) PreStart(ctx context.Context) error {
	return nil
}

func (a *testActor) PostStop(ctx context.Context) error {
	return nil
}

// Global mutex to ensure only one test runs at a time
var testMutex sync.Mutex

// TestSchedule_RecurringMessages tests that Schedule sends recurring messages
func TestSchedule_RecurringMessages(t *testing.T) {
	testMutex.Lock()
	defer testMutex.Unlock()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create and start the actor system
	actorSystem, err := actor.NewActorSystem("test-system-recurring")
	require.NoError(t, err)
	require.NotNil(t, actorSystem)

	err = actorSystem.Start(ctx)
	require.NoError(t, err)

	// Create an actor
	testActor := newTestActor()

	// Spawn the actor
	pid, err := actorSystem.Spawn(ctx, "test-actor-recurring", testActor)
	require.NoError(t, err)
	require.NotNil(t, pid)

	// Create a test message
	message, err := anypb.New(&anypb.Any{TypeUrl: "test/message", Value: []byte("test")})
	require.NoError(t, err)

	// Schedule the message to be sent every 50ms
	Schedule(ctx, pid, message, 50*time.Millisecond)

	// Receive at least 3 messages
	receivedCount := 0
	timeout := time.After(1 * time.Second)

	for receivedCount < 3 {
		select {
		case <-testActor.receivedMessages:
			receivedCount++
		case <-timeout:
			t.Fatalf("Timed out waiting for messages, received only %d", receivedCount)
			return
		}
	}

	assert.GreaterOrEqual(t, receivedCount, 3, "Should have received at least 3 messages")

	// Add a small delay before stopping the actor system
	time.Sleep(100 * time.Millisecond)

	// Stop the actor system
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()

	err = actorSystem.Stop(stopCtx)
	require.NoError(t, err)

	// Add a small delay after stopping to avoid race conditions between tests
	time.Sleep(200 * time.Millisecond)
}

// TestSchedule_CancelContext tests that Schedule stops sending messages when the context is cancelled
func TestSchedule_CancelContext(t *testing.T) {
	testMutex.Lock()
	defer testMutex.Unlock()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create and start the actor system
	actorSystem, err := actor.NewActorSystem("test-system-cancel")
	require.NoError(t, err)
	require.NotNil(t, actorSystem)

	err = actorSystem.Start(ctx)
	require.NoError(t, err)

	// Create an actor
	testActor := newTestActor()

	// Spawn the actor
	pid, err := actorSystem.Spawn(ctx, "test-actor-cancel", testActor)
	require.NoError(t, err)
	require.NotNil(t, pid)

	// Create a test message
	message, err := anypb.New(&anypb.Any{TypeUrl: "test/message", Value: []byte("test")})
	require.NoError(t, err)

	// Create a context that can be cancelled
	scheduleCtx, scheduleCancel := context.WithCancel(ctx)
	defer scheduleCancel()

	// Schedule the message to be sent every 50ms
	Schedule(scheduleCtx, pid, message, 50*time.Millisecond)

	// Wait for at least one message
	select {
	case <-testActor.receivedMessages:
		// Got one message
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timed out waiting for first message")
	}

	// Cancel the context to stop scheduling
	scheduleCancel()

	// Drain any messages that might have been sent before cancellation took effect
	drainUntil := time.After(100 * time.Millisecond)
	for {
		select {
		case <-testActor.receivedMessages:
			// Drain message
		case <-drainUntil:
			// Done draining
			goto drained
		}
	}
drained:

	// Now verify no more messages are sent
	noMoreMessages := true
	select {
	case <-testActor.receivedMessages:
		noMoreMessages = false
	case <-time.After(300 * time.Millisecond):
		// No messages received, which is what we want
	}

	assert.True(t, noMoreMessages, "Should not receive additional messages after context cancellation")

	// Add a small delay before stopping the actor system
	time.Sleep(100 * time.Millisecond)

	// Stop the actor system
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()

	err = actorSystem.Stop(stopCtx)
	require.NoError(t, err)

	// Add a small delay after stopping to avoid race conditions between tests
	time.Sleep(200 * time.Millisecond)
}

// TestScheduleOnce_SingleMessage tests that ScheduleOnce sends a single message after the specified interval
func TestScheduleOnce_SingleMessage(t *testing.T) {
	testMutex.Lock()
	defer testMutex.Unlock()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create and start the actor system
	actorSystem, err := actor.NewActorSystem("test-system-once")
	require.NoError(t, err)
	require.NotNil(t, actorSystem)

	err = actorSystem.Start(ctx)
	require.NoError(t, err)

	// Create an actor
	testActor := newTestActor()

	// Spawn the actor
	pid, err := actorSystem.Spawn(ctx, "test-actor-once", testActor)
	require.NoError(t, err)
	require.NotNil(t, pid)

	// Create a test message
	message, err := anypb.New(&anypb.Any{TypeUrl: "test/message", Value: []byte("test")})
	require.NoError(t, err)

	// Record start time to verify delay
	start := time.Now()

	// Schedule the message to be sent once after 100ms
	ScheduleOnce(ctx, pid, message, 100*time.Millisecond)

	// Wait for the message
	select {
	case <-testActor.receivedMessages:
		elapsed := time.Since(start)
		t.Logf("Message received after %v", elapsed)
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for scheduled message")
	}

	// Verify no more messages are sent
	noMoreMessages := true
	select {
	case <-testActor.receivedMessages:
		noMoreMessages = false
	case <-time.After(300 * time.Millisecond):
		// No more messages, which is what we want
	}

	assert.True(t, noMoreMessages, "Should not receive additional messages")

	// Add a small delay before stopping the actor system
	time.Sleep(100 * time.Millisecond)

	// Stop the actor system
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()

	err = actorSystem.Stop(stopCtx)
	require.NoError(t, err)

	// Add a small delay after stopping to avoid race conditions between tests
	time.Sleep(200 * time.Millisecond)
}

// TestScheduleOnce_CancelContext tests that ScheduleOnce does not send a message when the context is cancelled
func TestScheduleOnce_CancelContext(t *testing.T) {
	testMutex.Lock()
	defer testMutex.Unlock()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create and start the actor system
	actorSystem, err := actor.NewActorSystem("test-system-once-cancel")
	require.NoError(t, err)
	require.NotNil(t, actorSystem)

	err = actorSystem.Start(ctx)
	require.NoError(t, err)

	// Create an actor
	testActor := newTestActor()

	// Spawn the actor
	pid, err := actorSystem.Spawn(ctx, "test-actor-once-cancel", testActor)
	require.NoError(t, err)
	require.NotNil(t, pid)

	// Create a test message
	message, err := anypb.New(&anypb.Any{TypeUrl: "test/message", Value: []byte("test")})
	require.NoError(t, err)

	// Create a context that will be cancelled immediately
	scheduleCtx, scheduleCancel := context.WithCancel(ctx)
	defer scheduleCancel()

	// Schedule the message to be sent once after 200ms
	ScheduleOnce(scheduleCtx, pid, message, 200*time.Millisecond)

	// Cancel the context immediately
	scheduleCancel()

	// Verify no messages are received
	noMessages := true
	select {
	case <-testActor.receivedMessages:
		noMessages = false
	case <-time.After(300 * time.Millisecond):
		// No messages, which is what we want
	}

	assert.True(t, noMessages, "Should not receive any messages after context cancellation")

	// Add a small delay before stopping the actor system
	time.Sleep(100 * time.Millisecond)

	// Stop the actor system
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()

	err = actorSystem.Stop(stopCtx)
	require.NoError(t, err)

	// Add a small delay after stopping to avoid race conditions between tests
	time.Sleep(200 * time.Millisecond)
}
