package httphandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tochemey/goakt/v3/actor"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/protocol"
)

func TestHandleMCPPost_ParseError(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handler.HandleMCPPost(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response protocol.JSONRPCMessage
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotNil(t, response.Error)
}

func TestHandleMCPPost_MissingSessionId(t *testing.T) {
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

	message := protocol.JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  "unknown.method",
		ID:      1,
	}
	messageJSON, err := json.Marshal(message)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(messageJSON))
	w := httptest.NewRecorder()

	handler.HandleMCPPost(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response protocol.JSONRPCMessage
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Contains(t, response.Error.(map[string]interface{})["message"], "missing Mcp-Session-Id")
}

func TestHandleMCPPost_Initialize(t *testing.T) {
	t.Skip("TODO: Implement test for initialize scenario")
}
