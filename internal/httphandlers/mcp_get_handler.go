package httphandlers

import (
	"fmt"
	actors2 "github.com/traego/scaled-mcp/internal/actors"
	"github.com/traego/scaled-mcp/internal/channels"
	"log/slog"
	"net/http"
)

/*
GET /mcp
Create ConnectionActor
Register Client Stream with Session Actor
*/

// This is a user establishing an SSE session with the server for server to client comms
func (h *MCPHandler) HandleMCPGet(w http.ResponseWriter, r *http.Request) {
	// I think this is easy...spin up the death watcher, spin up the connection watcher, wait for death to come
	ctx := r.Context() // TODO Add logging details around these

	sessionId := r.Header.Get("Mcp-Session-Id")

	if sessionId == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Create an SSE channel for communication
	channel := channels.NewSSEChannel(w, r)

	cca := actors2.NewClientConnectionActor(h.config, sessionId, nil, channel, true, false, "")
	clientActorName := fmt.Sprintf("%s-client", sessionId)
	clientActor, err := h.actorSystem.Spawn(ctx, clientActorName, cca)
	if err != nil {
		respErr := fmt.Errorf("error spawning mcp session: %w", err)
		handleError(w, respErr, "")
	}

	_, dc, err := actors2.SpawnDeathWatcher(ctx, h.actorSystem, clientActor)
	if err != nil {
		respErr := fmt.Errorf("error spawning connection watcher: %w", err)
		handleError(w, respErr, "")
	}

	select {
	case <-dc:
	case <-channel.Done:
	}

	slog.DebugContext(ctx, "Shutting down MCP Long Lived Session")
}
