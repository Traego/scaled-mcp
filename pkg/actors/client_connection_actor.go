package actors

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/tochemey/goakt/v3/actor"
	"github.com/tochemey/goakt/v3/goaktpb"
	"github.com/traego/scaled-mcp/internal/utils"
	"github.com/traego/scaled-mcp/pkg/channels"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"log/slog"
)

/*
NOTES
The high level concept is this actor represents either a one way or bidirectional client connection.
That is to say - for an SSE connection, you imagine this as the sink for messages produced by other parts of the applicatino
For a websocket connection, you would actually allow client requests to come up through here.


1. We need to do something to dedupe sessions
2. We're going to support websockets here too
*/

type clientConnectionActor struct {
	cfg                  *config.ServerConfig
	params               *protocol.InitializeParams
	sessionId            string
	channel              channels.OneWayChannel
	sendEndpoint         bool
	connectionId         string
	defaultSseConnection bool
}

// NewClientConnectionActor creates a new actor for handling client connections
// It supports both one-way (SSE) and two-way communication with clients
func NewClientConnectionActor(cfg *config.ServerConfig, sessionId string, params *protocol.InitializeParams, channel channels.OneWayChannel, sendEndpoint bool, defaultSseConnection bool) actor.Actor {
	// I think here we actually need to do the negotiation, so that we can either start with one way or two way comms

	slog.Info("starting client connection actor")
	return &clientConnectionActor{
		cfg:                  cfg,
		params:               params,
		sessionId:            sessionId,
		channel:              channel,
		sendEndpoint:         sendEndpoint,
		defaultSseConnection: defaultSseConnection,
	}
}

func (c *clientConnectionActor) PreStart(ctx context.Context) error {
	if c.defaultSseConnection {
		c.connectionId = utils.GetDefaultSSEConnectionName(c.sessionId)
	} else {
		cId := uuid.New().String()
		c.connectionId = fmt.Sprintf("%s-conn-", cId)
	}
	slog.Debug(fmt.Sprintf("Starting client connection %s actor for session %s", c.connectionId, c.sessionId))
	return nil
}

func (c *clientConnectionActor) Receive(ctx *actor.ReceiveContext) {
	// For one way communication, this will always be messages coming from other parts of the system
	message := ctx.Message()

	// Handle different message types
	switch msg := message.(type) {
	case *goaktpb.PostStart:
		{
			// Register with the session. If any issues, kill myself before doing anything else
			_, sa, err := ctx.ActorSystem().ActorOf(ctx.Context(), utils.GetSessionActorName(c.sessionId))
			if err != nil {
				ctx.Logger().Error("error registering connection with session, shutting down", "sessionId", c.sessionId, "err", err)
				// Send an empty endpoint to signal failure
				c.channel.Close()
				ctx.Shutdown()
				return
			}

			// Let's watch the session, and if the session dies, we're killing ourselves
			sa.Watch(ctx.Self())

			reg := mcppb.RegisterConnection{ConnectionId: c.connectionId}
			registerResp := ctx.Ask(sa, &reg, c.cfg.RequestTimeout)
			if err != nil {
				ctx.Logger().Error("error registering connection with session, shutting down", "sessionId", c.sessionId, "err", err)
				c.channel.Close()
				ctx.Shutdown()
				return
			}

			rr, ok := registerResp.(*mcppb.RegisterConnectionResponse)
			if !ok {
				ctx.Logger().Error("unexpected response to registering connection with session, shutting down", "sessionId", c.sessionId, "err", err)
				c.channel.Close()
				ctx.Shutdown()
				return
			}

			if !rr.GetSuccess() {
				ctx.Logger().Error("unexpected failure registering connection with session, shutting down", "sessionId", c.sessionId, "err", rr.GetError())
				c.channel.Close()
				ctx.Shutdown()
				return
			}

			if c.sendEndpoint {
				// Create the message endpoint URL with the sessionId
				messageEndpoint := fmt.Sprintf("%s?sessionId=%s", c.cfg.HTTP.MessagePath, c.sessionId)

				// Send the endpoint event
				err := c.channel.SendEndpoint(messageEndpoint)
				if err != nil {
					ctx.Logger().Error(fmt.Errorf("error sending message endpoint: %w", err))
				}
			}
		}
	case *mcppb.JsonRpcResponse:
		slog.Info(fmt.Sprintf("Received message for client delivery sessionId = %s messageId = %s", c.sessionId, msg.Id))
		jm, err := protocol.ConvertProtoToJSONResponse(msg)
		if err != nil {
			slog.Error("problem converting proto to json response", "err", err)
		}

		err = c.channel.Send("message", jm)
		if err != nil {
			slog.Error("problem pushing json rpc request down channels channel", "err", err)
		}
	case *goaktpb.Terminated:
		// If the session actor terminated, we should terminate as well
		if msg.GetActorId() == utils.GetSessionActorName(c.sessionId) {
			ctx.Logger().Info("session terminated, shutting down client connection", "sessionId", c.sessionId)
			ctx.Shutdown()
		}
	default:
		ctx.Logger().Error(fmt.Errorf("unable to handle message of type '%T'", msg))
	}
}

func (c *clientConnectionActor) PostStop(ctx context.Context) error {
	slog.Debug(fmt.Sprintf("Stopping client connection %s actor for session %s", c.connectionId, c.sessionId))
	return nil
}

var _ actor.Actor = (*clientConnectionActor)(nil)
