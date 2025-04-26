package actors

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/tochemey/goakt/v3/actor"
	"github.com/tochemey/goakt/v3/goaktpb"
)

type ActorNotStarted struct {
	WatchId string
}

func (e *ActorNotStarted) GetWatchId() string { return e.WatchId }

type ActorTerminatedMessage struct {
	WatchId string
	ActorId string
}

func (e *ActorTerminatedMessage) GetWatchId() string { return e.WatchId }

type DeathWatchMessage interface {
	GetWatchId() string
}

type DeathWatcher struct {
	notifications chan DeathWatchMessage // *goaktpb.Terminated
	pid           *actor.PID
	watchId       string
}

func SpawnDeathWatcher(ctx context.Context, actorSystem actor.ActorSystem, pid *actor.PID) (*actor.PID, <-chan DeathWatchMessage, error) {
	notifications := make(chan DeathWatchMessage)
	watchId := uuid.New().String()

	dw := &DeathWatcher{
		notifications: notifications,
		pid:           pid,
		watchId:       watchId,
	}

	deathWatchName := "death-watcher" + watchId
	slog.DebugContext(ctx, "spawning death watcher "+deathWatchName)

	dwa, err := actorSystem.Spawn(ctx, deathWatchName, dw)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to spawn death watcher: %w", err)
	}

	time.Sleep(100 * time.Millisecond)

	_, lookup, _ := actorSystem.ActorOf(ctx, pid.Name())
	if lookup != nil && lookup.IsRunning() {
		slog.InfoContext(ctx, "found actor, starting death watch", "watchId", watchId, "actorId", pid.ID())
		dwa.Watch(pid)
	} else {
		slog.InfoContext(ctx, "did not find actor, sending term message and shutting down", "watchId", watchId)
		go func() {
			notifications <- &ActorNotStarted{watchId}
		}()
		_ = dwa.Shutdown(ctx)
	}

	return dwa, notifications, nil
}

func (d *DeathWatcher) PreStart(ctx context.Context) error {
	return nil
}

func (d *DeathWatcher) Receive(ctx *actor.ReceiveContext) {
	message := ctx.Message()

	ctx.Logger().Info("death watcher received message", message)

	// Handle different message types
	switch msg := message.(type) {
	case *goaktpb.Terminated:
		if msg.GetActorId() == d.pid.ID() {
			ctx.Logger().Debug("DeathWatcher received termination notification",
				"actor", ctx.Sender().ID())

			// Only try to send if the channel exists
			if d.notifications != nil {
				select {
				case d.notifications <- &ActorTerminatedMessage{WatchId: d.watchId, ActorId: msg.GetActorId()}:
					// Message sent successfully
				default:
					// Channel is full or closed, log and continue
					ctx.Logger().Warn("Failed to send termination notification, channel might be full or closed",
						"actor", ctx.Sender().ID())
				}
			}

			// Shutdown, our purpose in life is complete
			ctx.Shutdown()
		}
	}
}

func (d *DeathWatcher) PostStop(ctx context.Context) error {
	return nil
}

var _ actor.Actor = (*DeathWatcher)(nil)
