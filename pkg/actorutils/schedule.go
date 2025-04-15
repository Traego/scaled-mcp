package actorutils

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tochemey/goakt/v3/actor"
	"google.golang.org/protobuf/proto"
)

// Schedule schedules a recurring message to be sent to the target actor at the specified interval.
// The scheduling will continue until the provided context is cancelled.
// The first message is sent after the interval has elapsed.
func Schedule(ctx context.Context, actorSystem actor.ActorSystem, targetActor string, message proto.Message, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// Context was cancelled, stop the loop
				slog.DebugContext(ctx, "scheduled message cancelled")
				return
			case <-ticker.C:
				// Time to send the message
				_, rid, err := actorSystem.ActorOf(ctx, "root")
				if err != nil {
					slog.ErrorContext(ctx, fmt.Sprintf("failed to get the root actor: %v", err))
				}

				err = rid.SendAsync(ctx, targetActor, message)
				if err != nil {
					if err.Error() == "actor is not alive" {
						slog.DebugContext(ctx, "actor is not alive, shutting down")
						return
					} else {
						slog.ErrorContext(ctx, "failed to send scheduled message",
							"error", err)
					}
				}
			}
		}
	}()
}

// ScheduleOnce schedules a message to be sent once to the target actor after the specified interval.
// The message will not be sent if the provided context is cancelled before the interval elapses.
func ScheduleOnce(ctx context.Context, actorSystem actor.ActorSystem, targetActor string, message proto.Message, interval time.Duration) {
	go func() {
		select {
		case <-ctx.Done():
			// Context was cancelled, don't send the message
			slog.DebugContext(ctx, "scheduled one-time message cancelled")
			return
		case <-time.After(interval):
			_, rid, err := actorSystem.ActorOf(ctx, "root")
			if err != nil {
				slog.ErrorContext(ctx, fmt.Sprintf("failed to get the root actor: %v", err))
				return
			}

			// Time to send the message
			err = rid.SendAsync(ctx, targetActor, message)
			if err != nil {
				slog.ErrorContext(ctx, "failed to send scheduled message",
					"error", err)
			}
		}
	}()
}
