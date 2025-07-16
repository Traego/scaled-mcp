package httphandlers

import (
	"encoding/json"
	"fmt"
	actors2 "github.com/traego/scaled-mcp/internal/actors"
	"github.com/traego/scaled-mcp/internal/channels"
	"github.com/traego/scaled-mcp/pkg/auth"
	"github.com/traego/scaled-mcp/pkg/utils"
	"log/slog"
	"net/http"
)

func (h *MCPHandler) SSEGetWithBasePath(basePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.SSEGetFunc(w, r, basePath)
	}
}

// This is backwards compatibility for 2024 SSE sessions, for server to client messages
func (h *MCPHandler) HandleSSEGet(w http.ResponseWriter, r *http.Request) {
	h.SSEGetFunc(w, r, "")
}

func (h *MCPHandler) HandleSessionPreFlight(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Set appropriate headers for the response
	w.Header().Set("Content-Type", "application/json")

	// If we don't have a session manager initialized, return an error
	if h.serverInfo.GetSessionManager() == nil {
		slog.Error("session manager not initialized")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var uniqueId *string
	if ai := auth.GetAuthInfo(ctx); ai != nil && h.serverInfo.GetAuthHandler() != nil {
		uid := ai.GetPrincipalId()
		uniqueId = &uid
	}

	// Generate a secure signed session ID that includes a random component and a signature
	sessionID, err := h.serverInfo.GetSessionManager().GenerateSessionId(uniqueId)
	if err != nil {
		slog.Error("failed to generate session ID", slog.Any("error", err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return the session ID in a JSON response
	response := map[string]string{
		"sessionId": sessionID,
	}

	// Convert the response to JSON
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		slog.Error("failed to marshal response", slog.Any("error", err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Write the response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(jsonResponse)
	if err != nil {
		slog.Error("failed to write response", slog.Any("error", err))
	}
}

func (h *MCPHandler) SSEGetFunc(w http.ResponseWriter, r *http.Request, basePath string) {
	// I think this is easy...spin up the death watcher, spin up the connection watcher, wait for death to come
	ctx := r.Context() // TODO Add logging details around these

	var uniqueId *string
	if ai := auth.GetAuthInfo(ctx); ai != nil && h.serverInfo.GetAuthHandler() != nil {
		uid := ai.GetPrincipalId()
		uniqueId = &uid
	}

	// To read the session_id cookie
	cookie, err := r.Cookie("session_id")
	if err != nil && err != http.ErrNoCookie {
		handleError(w, err, "")
		return
	}

	setCookie := true

	var sessionId string
	if cookie == nil {
		if r.URL.Query().Get("sessionId") != "" {
			// If session ID is provided in the query string, verify it's a valid signed ID
			sessionId = r.URL.Query().Get("sessionId")

			// Verify the signed session ID with a 15-minute max age
			err = h.serverInfo.GetSessionManager().VerifySessionId(sessionId, uniqueId)
			if err != nil {
				http.Error(w, "Access denied", http.StatusForbidden)
			}
			// Use the verified random ID component
			setCookie = false
		} else {
			sessionId, err = h.serverInfo.GetSessionManager().GenerateSessionId(uniqueId)
			if err != nil {
				handleError(w, err, "")
				return
			}
		}
	} else {
		sessionId = cookie.Value
	}

	san := utils.GetSessionActorName(sessionId)

	// Ensure the session actor exists; spawn only if we don't find it running
	_, existingPid, _ := h.actorSystem.ActorOf(ctx, san)
	if existingPid == nil {
		sa := actors2.NewMcpSessionStateMachine(h.serverInfo, sessionId)
		_, err = h.actorSystem.Spawn(ctx, san, sa)
		if err != nil {
			handleError(w, err, "")
			return
		}
	}

	// Create an SSE channel for communication
	channel := channels.NewSSEChannel(w, r, sessionId, h.config.HTTP.SSLEnabled, setCookie)

	clientId, err := utils.GenerateSecureID(20)
	if err != nil {
		handleError(w, err, "")
	}

	cca := actors2.NewClientConnectionActor(h.config, sessionId, nil, channel, true, true, basePath)
	clientActorName := fmt.Sprintf("%s-client", clientId)
	clientActor, err := h.actorSystem.Spawn(ctx, clientActorName, cca)
	if err != nil {
		respErr := fmt.Errorf("error spawning sse session: %w", err)
		handleError(w, respErr, "")
	}

	_, dc, err := actors2.SpawnDeathWatcher(ctx, h.actorSystem, clientActor)
	if err != nil {
		handleError(w, err, "")
	}

	select {
	case <-dc:
	case <-channel.Done:
	}
}
