package httphandlers

import (
	"encoding/json"
	"github.com/traego/scaled-mcp/pkg/auth"
	"log/slog"
	"net/http"
)

func (h *MCPHandler) SessionPreflightHandler(tryIncludePrincipalId bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set appropriate headers for the response
		w.Header().Set("Content-Type", "application/json")

		// If we don't have a session manager initialized, return an error
		if h.serverInfo.GetSessionManager() == nil {
			slog.Error("session manager not initialized")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		var principalId *string
		if tryIncludePrincipalId {
			ai := auth.GetAuthInfo(r.Context())
			if ai != nil {
				p := ai.GetPrincipalId()
				principalId = &p
			}
		}

		// Generate a secure signed session ID that includes a random component and a signature
		sessionID, err := h.serverInfo.GetSessionManager().GenerateSessionId(principalId)
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
	})
}
