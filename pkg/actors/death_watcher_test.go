package actors

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tochemey/goakt/v3/actor"
	"github.com/tochemey/goakt/v3/goaktpb"

	"github.com/traego/scaled-mcp/internal/logger"
)

func TestDeathWatcher(t *testing.T) {
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

	t.Run("should receive termination notification", func(t *testing.T) {
		// Create a test actor that will be terminated
		testActor := &testTerminatingActor{}
		testActorPID, err := actorSystem.Spawn(ctx, "test-actor", testActor)
		require.NoError(t, err)

		// Create the death watcher actor
		_, notifications, err := SpawnDeathWatcher(t.Context(), actorSystem, testActorPID)
		require.NoError(t, err)

		poison := goaktpb.PoisonPill{}
		err = actor.Tell(ctx, testActorPID, &poison)
		require.NoError(t, err)

		// Wait for the termination notification
		select {
		case terminated := <-notifications:
			assert.NotNil(t, terminated)
			assert.Equal(t, testActorPID.ID(), terminated.GetActorId())
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for termination notification")
		}
	})

	t.Run("should handle channel full condition", func(t *testing.T) {
		// Create a death watcher with a channel that will be full
		// We use a channel with buffer size 0 to simulate a full channel
		dw := &DeathWatcher{
			notifications: make(chan *goaktpb.Terminated),
		}
		deathWatcherPID, err := actorSystem.Spawn(ctx, "death-watcher-full", dw)
		require.NoError(t, err)

		// Create a test actor that will be terminated
		testActor := &testTerminatingActor{}
		testActorPID, err := actorSystem.Spawn(ctx, "test-actor-full", testActor)
		require.NoError(t, err)

		// Watch the test actor
		deathWatcherPID.Watch(testActorPID)
		require.NoError(t, err)

		// Block the channel by not reading from it
		// This will force the default case in the select statement

		// Terminate the test actor
		poison := goaktpb.PoisonPill{}
		err = actor.Tell(ctx, testActorPID, &poison)
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(500 * time.Millisecond)

		// No assertion needed here, we're just testing that the actor doesn't crash
		// when the channel is full
	})

	t.Run("should handle nil notifications channel", func(t *testing.T) {
		// Create a test actor that will be terminated
		testActor := &testTerminatingActor{}
		testActorPID, err := actorSystem.Spawn(ctx, "test-actor-nil", testActor)
		require.NoError(t, err)

		// Create and throw away channel
		dwa, _, err := SpawnDeathWatcher(t.Context(), actorSystem, testActorPID)
		require.NoError(t, err)

		// Terminate the test actor
		poison := goaktpb.PoisonPill{}
		err = actor.Tell(ctx, testActorPID, &poison)
		require.NoError(t, err)

		// Give some time for the message to be processed
		time.Sleep(500 * time.Millisecond)

		// If termination was successful, the actor should shut down. If the shutdown crashed, it will be alive
		require.False(t, dwa.IsRunning(), "death watcher should not be running")
	})
}

// testTerminatingActor is a simple actor that can be terminated
type testTerminatingActor struct{}

func (t *testTerminatingActor) PreStart(ctx context.Context) error {
	return nil
}

func (t *testTerminatingActor) Receive(ctx *actor.ReceiveContext) {
	// Do nothing
}

func (t *testTerminatingActor) PostStop(ctx context.Context) error {
	return nil
}

var _ actor.Actor = (*testTerminatingActor)(nil)
