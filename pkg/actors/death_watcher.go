package actors

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/tochemey/goakt/v3/actor"
	"github.com/tochemey/goakt/v3/goaktpb"
)

type DeathWatcher struct {
	notifications chan *goaktpb.Terminated
	pid           *actor.PID
}

func SpawnDeathWatcher(ctx context.Context, actorSystem actor.ActorSystem, pid *actor.PID) (*actor.PID, <-chan *goaktpb.Terminated, error) {
	notifications := make(chan *goaktpb.Terminated)
	dw := &DeathWatcher{
		notifications: notifications,
		pid:           pid,
	}

	dwa, err := actorSystem.Spawn(ctx, "death-watcher"+pid.Name(), dw)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to spawn death watcher: %w", err)
	}

	pid.Watch(dwa)

	return dwa, notifications, nil
}

func (d *DeathWatcher) PreStart(ctx context.Context) error {
	return nil
}

func (d *DeathWatcher) Receive(ctx *actor.ReceiveContext) {
	message := ctx.Message()

	// Handle different message types
	switch msg := message.(type) {
	case *goaktpb.PostStart:
		// d.pid.Watch(ctx.Self())
		return
	case *goaktpb.Terminated:
		if msg.GetActorId() == d.pid.ID() {
			slog.Debug("DeathWatcher received termination notification",
				"actor", ctx.Sender().ID())

			// Only try to send if the channel exists
			if d.notifications != nil {
				select {
				case d.notifications <- msg:
					// Message sent successfully
				default:
					// Channel is full or closed, log and continue
					slog.Warn("Failed to send termination notification, channel might be full or closed",
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
