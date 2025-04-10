package httphandlers

import (
	"fmt"
	"github.com/traego/scaled-mcp/internal/utils"
	"github.com/traego/scaled-mcp/pkg/actors"
	"github.com/traego/scaled-mcp/pkg/channels"
	"net/http"
)

// This is backwards compatibility for 2024 SSE sessions, for server to client messages
func (h *MCPHandler) HandleSSEGet(w http.ResponseWriter, r *http.Request) {
	// I think this is easy...spin up the death watcher, spin up the connection watcher, wait for death to come
	ctx := r.Context() // TODO Add logging details around these

	sessionId, err := utils.GenerateSecureID(20)
	if err != nil {
		handleError(w, err, "")
		return
	}

	sa := actors.NewMcpSessionStateMachine(h.serverInfo, sessionId)
	san := utils.GetSessionActorName(sessionId)
	_, err = h.actorSystem.Spawn(ctx, san, sa)
	if err != nil {
		handleError(w, err, "")
		return
	}

	// Create an SSE channel for communication
	channel := channels.NewSSEChannel(w, r)

	cca := actors.NewClientConnectionActor(h.config, sessionId, nil, channel, true, true)
	clientActorName := fmt.Sprintf("%s-client", sessionId)
	clientActor, err := h.actorSystem.Spawn(ctx, clientActorName, cca)
	if err != nil {
		respErr := fmt.Errorf("error spawning session: %w", err)
		handleError(w, respErr, "")
	}

	_, dc, err := actors.SpawnDeathWatcher(ctx, h.actorSystem, clientActor)
	if err != nil {
		handleError(w, err, "")
	}

	select {
	case <-dc:
	case <-channel.Done:
	}
}
