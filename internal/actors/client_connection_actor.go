package actors

import (
	"fmt"
	"github.com/traego/scaled-mcp/internal/channels"
	"log/slog"

	"github.com/google/uuid"
	"github.com/tochemey/goakt/v3/actor"
	"github.com/tochemey/goakt/v3/goaktpb"

	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/utils"
)

/*
NOTES
The high level concept is this actor represents either a one way or bidirectional client connection.
That is to say - for an SSE connection, you imagine this as the sink for messages produced by other parts of the applicatino
For a websocket connection, you would actually allow client requests to come up through here.


1. We need to do something to dedupe sessions
2. We're going to support websockets here too
*/

type ClientConnectionActor struct {
	cfg                  *config.ServerConfig
	params               *protocol.InitializeParams
	sessionId            string
	channel              channels.OneWayChannel
	sendEndpoint         bool
	connectionId         string
	defaultSseConnection bool
	basePath             string
}

// NewClientConnectionActor creates a new actor for handling client connections
// It supports both one-way (SSE) and two-way communication with clients
func NewClientConnectionActor(cfg *config.ServerConfig, sessionId string, params *protocol.InitializeParams, channel channels.OneWayChannel, sendEndpoint bool, defaultSseConnection bool, basePath string) actor.Actor {
	// I think here we actually need to do the negotiation, so that we can either start with one way or two way comms

	// TODO(arsene): this is a bit of a hack, we need to pass a logger in the constructor
	slog.Info("starting client connection actor")
	return &ClientConnectionActor{
		cfg:                  cfg,
		params:               params,
		sessionId:            sessionId,
		channel:              channel,
		sendEndpoint:         sendEndpoint,
		defaultSseConnection: defaultSseConnection,
		basePath:             basePath,
	}
}

func (c *ClientConnectionActor) PreStart(ctx *actor.Context) error {
	if c.defaultSseConnection {
		c.connectionId = utils.GetDefaultSSEConnectionName(c.sessionId)
	} else {
		cId := uuid.New().String()
		c.connectionId = fmt.Sprintf("%s-conn-", cId)
	}
	slog.Debug(fmt.Sprintf("Starting client connection %s actor for session %s", c.connectionId, c.sessionId))
	return nil
}

func (c *ClientConnectionActor) Receive(ctx *actor.ReceiveContext) {
	// For one way communication, this will always be messages coming from other parts of the system
	message := ctx.Message()
	slog.DebugContext(ctx.Context(), "ClientConnectionActor received message",
		"messageType", fmt.Sprintf("%T", message),
		"sessionId", c.sessionId,
		"connectionId", c.connectionId)

	// Handle different message types
	switch msg := message.(type) {
	case *goaktpb.PostStart:
		slog.DebugContext(ctx.Context(), "ClientConnectionActor processing PostStart message",
			"sessionId", c.sessionId,
			"connectionId", c.connectionId)

		san := utils.GetSessionActorName(c.sessionId)
		slog.DebugContext(ctx.Context(), "ClientConnectionActor getting session actor",
			"sessionActorName", san,
			"sessionId", c.sessionId)

		// Register with the session. If any issues, kill myself before doing anything else
		_, sa, err := ctx.ActorSystem().ActorOf(ctx.Context(), san)
		if err != nil {
			ctx.Logger().Error("error registering connection with session, shutting down", "sessionId", c.sessionId, "err", err)
			// Send an empty endpoint to signal failure
			slog.DebugContext(ctx.Context(), "ClientConnectionActor failed to get session actor, closing channel",
				"sessionId", c.sessionId,
				"sessionActorName", san,
				"error", err)
			c.channel.Close()
			ctx.Shutdown()
			return
		}

		// Let's watch the session, and if the session dies, we're killing ourselves
		slog.DebugContext(ctx.Context(), "ClientConnectionActor watching session actor",
			"sessionId", c.sessionId,
			"sessionActorName", san)
		sa.Watch(ctx.Self())

		reg := mcppb.RegisterConnection{ConnectionId: c.connectionId}
		slog.DebugContext(ctx.Context(), "ClientConnectionActor sending RegisterConnection message",
			"sessionId", c.sessionId,
			"connectionId", c.connectionId,
			"sessionActorName", san)

		registerResp := ctx.SendSync(san, &reg, c.cfg.RequestTimeout)
		slog.DebugContext(ctx.Context(), "ClientConnectionActor received RegisterConnection response",
			"sessionId", c.sessionId,
			"responseType", fmt.Sprintf("%T", registerResp))

		rr, ok := registerResp.(*mcppb.RegisterConnectionResponse)
		if !ok {
			ctx.Logger().Error("unexpected response to registering connection with session, shutting down", "sessionId", c.sessionId, "err", err)
			slog.DebugContext(ctx.Context(), "ClientConnectionActor received invalid response type, closing channel",
				"sessionId", c.sessionId,
				"expectedType", "*mcppb.RegisterConnectionResponse",
				"actualType", fmt.Sprintf("%T", registerResp))
			c.channel.Close()
			ctx.Shutdown()
			return
		}

		if !rr.GetSuccess() {
			ctx.Logger().Error("unexpected failure registering connection with session, shutting down", "sessionId", c.sessionId, "err", rr.GetError())
			slog.DebugContext(ctx.Context(), "ClientConnectionActor registration failed, closing channel",
				"sessionId", c.sessionId,
				"error", rr.GetError())
			c.channel.Close()
			ctx.Shutdown()
			return
		}

		slog.DebugContext(ctx.Context(), "ClientConnectionActor successfully registered with session",
			"sessionId", c.sessionId,
			"connectionId", c.connectionId)

		if c.sendEndpoint {
			slog.DebugContext(ctx.Context(), "ClientConnectionActor preparing to send endpoint",
				"sessionId", c.sessionId,
				"protocolVersion", c.cfg.ProtocolVersion,
				"basePath", c.basePath)

			var messageEndpoint string

			// Create the message endpoint URL with the sessionId
			// TODO [pw] I'm not wild about this - this is a cross dependency on config elsewhere that doesn't start
			// the mcp endpoint. Maybe we don't do this.....
			// Ok this is even more complex - this c.cfg is supplied from the client. So, if they hand in an invalid
			// protocol, I have to still start the sse session. Anywhere, this something not quite right here
			if c.cfg.ProtocolVersion == protocol.ProtocolVersion20250326 {
				messageEndpoint = fmt.Sprintf("%s%s?sessionId=%s", c.basePath, c.cfg.HTTP.MCPPath, c.sessionId)
				slog.DebugContext(ctx.Context(), "ClientConnectionActor using MCP path for endpoint",
					"sessionId", c.sessionId,
					"mcpPath", c.cfg.HTTP.MCPPath,
					"endpoint", messageEndpoint)
			} else {
				messageEndpoint = fmt.Sprintf("%s%s?sessionId=%s", c.basePath, c.cfg.HTTP.MessagePath, c.sessionId)
				slog.DebugContext(ctx.Context(), "ClientConnectionActor using Message path for endpoint",
					"sessionId", c.sessionId,
					"messagePath", c.cfg.HTTP.MessagePath,
					"endpoint", messageEndpoint)
			}

			// Send the endpoint event
			slog.DebugContext(ctx.Context(), "ClientConnectionActor sending endpoint to client",
				"sessionId", c.sessionId,
				"endpoint", messageEndpoint)

			err := c.channel.SendEndpoint(messageEndpoint)
			if err != nil {
				ctx.Logger().Error(fmt.Errorf("error sending message endpoint: %w", err))
				slog.DebugContext(ctx.Context(), "ClientConnectionActor failed to send endpoint",
					"sessionId", c.sessionId,
					"endpoint", messageEndpoint,
					"error", err)
			} else {
				slog.DebugContext(ctx.Context(), "ClientConnectionActor successfully sent endpoint",
					"sessionId", c.sessionId,
					"endpoint", messageEndpoint)
			}
		} else {
			slog.DebugContext(ctx.Context(), "ClientConnectionActor skipping endpoint send (not requested)",
				"sessionId", c.sessionId)
		}

	case *mcppb.JsonRpcResponse:
		// TODO(arsene): revisit this logging
		slog.DebugContext(ctx.Context(), fmt.Sprintf("Received message for client delivery sessionId = %s messageId = %s", c.sessionId, msg.Id))
		jm, err := protocol.ConvertProtoToJSONResponse(msg)
		if err != nil {
			ctx.Logger().Error("problem converting proto to json response", "err", err)
			slog.DebugContext(ctx.Context(), "ClientConnectionActor failed to convert proto to JSON response",
				"sessionId", c.sessionId,
				"messageId", msg.Id,
				"error", err)
			ctx.Err(err)
			return
		}

		slog.DebugContext(ctx.Context(), "ClientConnectionActor sending message to client",
			"sessionId", c.sessionId,
			"messageId", msg.Id)

		if err = c.channel.Send("message", jm); err != nil {
			ctx.Logger().Error("problem pushing json rpc response down channels channel", "err", err)
			slog.DebugContext(ctx.Context(), "ClientConnectionActor failed to send message to client",
				"sessionId", c.sessionId,
				"messageId", msg.Id,
				"error", err)
			ctx.Err(err)
			return
		}

		slog.DebugContext(ctx.Context(), "ClientConnectionActor successfully sent message to client",
			"sessionId", c.sessionId,
			"messageId", msg.Id)

	case *goaktpb.Terminated:
		slog.DebugContext(ctx.Context(), "ClientConnectionActor received Terminated message",
			"sessionId", c.sessionId,
			"terminatedActorId", msg.GetActorId())

		// If the session actor terminated, we should terminate as well
		if msg.GetActorId() == utils.GetSessionActorName(c.sessionId) {
			ctx.Logger().Info("session terminated, shutting down client connection", "sessionId", c.sessionId)
			slog.DebugContext(ctx.Context(), "ClientConnectionActor shutting down due to session termination",
				"sessionId", c.sessionId,
				"sessionActorName", utils.GetSessionActorName(c.sessionId))
			ctx.Shutdown()
		} else {
			slog.DebugContext(ctx.Context(), "ClientConnectionActor ignoring termination of unrelated actor",
				"sessionId", c.sessionId,
				"terminatedActorId", msg.GetActorId())
		}
	default:
		ctx.Logger().Error(fmt.Errorf("unable to handle message of type '%T'", msg))
		slog.DebugContext(ctx.Context(), "ClientConnectionActor received unhandled message type",
			"sessionId", c.sessionId,
			"messageType", fmt.Sprintf("%T", msg))
	}
}

func (c *ClientConnectionActor) PostStop(ctx *actor.Context) error {
	slog.Debug(fmt.Sprintf("Stopping client connection %s actor for session %s", c.connectionId, c.sessionId))
	return nil
}

var _ actor.Actor = (*ClientConnectionActor)(nil)
