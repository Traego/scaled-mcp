package httphandlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tochemey/goakt/v3/actor"
	"github.com/traego/scaled-mcp/pkg/config"
)

func TestHandleMCPGet(t *testing.T) {
	t.Run("missing session id", func(t *testing.T) {
		cfg := &config.ServerConfig{}
		actorSystem, err := actor.NewActorSystem("test-system")
		require.NoError(t, err)
		err = actorSystem.Start(context.Background())
		require.NoError(t, err)
		defer func() {
			_ = actorSystem.Stop(context.Background())
		}()

		serverInfo := &mockServerInfo{}
		handler := NewMCPHandler(cfg, actorSystem, serverInfo)

		req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
		w := httptest.NewRecorder()

		handler.HandleMCPGet(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("with valid session id", func(t *testing.T) {
		t.Skip("TODO: Implement test for valid session ID scenario")
	})
}
