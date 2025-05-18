package httphandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tochemey/goakt/v3/actor"
	"github.com/traego/scaled-mcp/pkg/auth"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/resources"
)

func TestNewMCPHandler(t *testing.T) {
	cfg := &config.ServerConfig{}
	actorSystem, _ := actor.NewActorSystem("test-system")
	serverInfo := &mockServerInfo{}

	handler := NewMCPHandler(cfg, actorSystem, serverInfo)

	assert.NotNil(t, handler)
	assert.Equal(t, cfg, handler.config)
	assert.Equal(t, actorSystem, handler.actorSystem)
	assert.Equal(t, serverInfo, handler.serverInfo)
}

func TestParseMessageRequest(t *testing.T) {
	t.Run("Single message", func(t *testing.T) {
		message := protocol.JSONRPCMessage{
			JSONRPC: "2.0",
			Method:  "test.method",
			ID:      1,
		}

		messageJSON, err := json.Marshal(message)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(messageJSON))

		result, err := parseMessageRequest(req)
		require.NoError(t, err)

		assert.False(t, result.IsBatch)
		assert.Equal(t, message.JSONRPC, result.Message.JSONRPC)
		assert.Equal(t, message.Method, result.Message.Method)
		assert.Equal(t, float64(1), result.Message.ID)
	})

	t.Run("Batch messages", func(t *testing.T) {
		messages := []protocol.JSONRPCMessage{
			{
				JSONRPC: "2.0",
				Method:  "test.method1",
				ID:      1,
			},
			{
				JSONRPC: "2.0",
				Method:  "test.method2",
				ID:      2,
			},
		}

		messagesJSON, err := json.Marshal(messages)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(messagesJSON))

		result, err := parseMessageRequest(req)
		require.NoError(t, err)

		assert.True(t, result.IsBatch)
		assert.Len(t, result.Messages, 2)
		assert.Equal(t, messages[0].JSONRPC, result.Messages[0].JSONRPC)
		assert.Equal(t, messages[0].Method, result.Messages[0].Method)
		assert.Equal(t, float64(1), result.Messages[0].ID)
		assert.Equal(t, messages[1].JSONRPC, result.Messages[1].JSONRPC)
		assert.Equal(t, messages[1].Method, result.Messages[1].Method)
		assert.Equal(t, float64(2), result.Messages[1].ID)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader([]byte("invalid json")))

		_, err := parseMessageRequest(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse body")
	})

	t.Run("Read body error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/mcp", &errorReader{})

		_, err := parseMessageRequest(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read body")
	})
}

func TestWriteMessage(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		w := httptest.NewRecorder()

		message := protocol.JSONRPCMessage{
			JSONRPC: "2.0",
			Method:  "test.method",
			ID:      1,
			Result:  "test result",
		}

		sessionID := "test-session-id"
		err := writeMessage(w, message, &sessionID)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		assert.Equal(t, sessionID, w.Header().Get("Mcp-Session-Id"))

		var responseMessage protocol.JSONRPCMessage
		err = json.Unmarshal(w.Body.Bytes(), &responseMessage)
		require.NoError(t, err)
		assert.Equal(t, message.JSONRPC, responseMessage.JSONRPC)
		assert.Equal(t, message.Method, responseMessage.Method)
		assert.Equal(t, float64(1), responseMessage.ID)
		assert.Equal(t, message.Result, responseMessage.Result)
	})

	t.Run("No session ID", func(t *testing.T) {
		w := httptest.NewRecorder()

		message := protocol.JSONRPCMessage{
			JSONRPC: "2.0",
			Method:  "test.method",
			ID:      1,
			Result:  "test result",
		}

		err := writeMessage(w, message, nil)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		assert.Empty(t, w.Header().Get("Mcp-Session-Id"))

		var responseMessage protocol.JSONRPCMessage
		err = json.Unmarshal(w.Body.Bytes(), &responseMessage)
		require.NoError(t, err)
		assert.Equal(t, message.JSONRPC, responseMessage.JSONRPC)
		assert.Equal(t, message.Method, responseMessage.Method)
		assert.Equal(t, float64(1), responseMessage.ID)
		assert.Equal(t, message.Result, responseMessage.Result)
	})
}

func TestHandleError(t *testing.T) {
	t.Run("JSON-RPC error", func(t *testing.T) {
		w := httptest.NewRecorder()

		jsonRpcError := &protocol.JsonRpcError{
			Code:    protocol.ErrInvalidParams,
			Message: "Invalid params",
			ID:      1,
		}

		handleError(w, jsonRpcError, 1)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var responseMessage protocol.JSONRPCMessage
		err := json.Unmarshal(w.Body.Bytes(), &responseMessage)
		require.NoError(t, err)
		assert.Equal(t, "2.0", responseMessage.JSONRPC)
		assert.Equal(t, float64(1), responseMessage.ID)
		assert.NotNil(t, responseMessage.Error)

		errorMap, ok := responseMessage.Error.(map[string]interface{})
		require.True(t, ok, "Error should be a map")
		assert.Equal(t, float64(protocol.ErrInvalidParams), errorMap["code"])
		assert.Equal(t, "Invalid params", errorMap["message"])
	})

	t.Run("JSON-RPC error with nil ID", func(t *testing.T) {
		w := httptest.NewRecorder()

		jsonRpcError := &protocol.JsonRpcError{
			Code:    protocol.ErrInvalidParams,
			Message: "Invalid params",
			ID:      nil,
		}

		handleError(w, jsonRpcError, 1)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var responseMessage protocol.JSONRPCMessage
		err := json.Unmarshal(w.Body.Bytes(), &responseMessage)
		require.NoError(t, err)
		assert.Equal(t, "2.0", responseMessage.JSONRPC)
		assert.Equal(t, float64(1), responseMessage.ID)
		assert.NotNil(t, responseMessage.Error)

		errorMap, ok := responseMessage.Error.(map[string]interface{})
		require.True(t, ok, "Error should be a map")
		assert.Equal(t, float64(protocol.ErrInvalidParams), errorMap["code"])
		assert.Equal(t, "Invalid params", errorMap["message"])
	})

	t.Run("Non-JSON-RPC error", func(t *testing.T) {
		w := httptest.NewRecorder()

		regularError := errors.New("regular error")

		handleError(w, regularError, 1)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var responseMessage protocol.JSONRPCMessage
		err := json.Unmarshal(w.Body.Bytes(), &responseMessage)
		require.NoError(t, err)
		assert.Equal(t, "2.0", responseMessage.JSONRPC)
		assert.Equal(t, float64(1), responseMessage.ID)
		assert.NotNil(t, responseMessage.Error)

		errorMap, ok := responseMessage.Error.(map[string]interface{})
		require.True(t, ok, "Error should be a map")
		assert.Equal(t, float64(protocol.ErrInternal), errorMap["code"])
		assert.Equal(t, "Internal error: regular error", errorMap["message"])
	})
}

type mockServerInfo struct{}

type mockToolRegistry struct{}

func (m *mockToolRegistry) GetTool(ctx context.Context, name string) (protocol.Tool, error) {
	return protocol.Tool{}, errors.New("not implemented")
}

func (m *mockToolRegistry) ListTools(ctx context.Context, opts protocol.ToolListOptions) (protocol.ToolListResult, error) {
	return protocol.ToolListResult{}, errors.New("not implemented")
}

func (m *mockToolRegistry) CallTool(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
	return nil, errors.New("not implemented")
}

type mockPromptRegistry struct{}

func (m *mockPromptRegistry) GetPrompt(ctx context.Context, name string) (resources.Prompt, bool) {
	return resources.Prompt{}, false
}

func (m *mockPromptRegistry) ListPrompts(ctx context.Context, opts resources.PromptListOptions) resources.PromptListResult {
	return resources.PromptListResult{}
}

func (m *mockPromptRegistry) ProcessPrompt(ctx context.Context, name string, arguments map[string]string) ([]resources.PromptMessage, error) {
	return nil, errors.New("not implemented")
}

type mockResourceRegistry struct{}

func (m *mockResourceRegistry) ListResources(ctx context.Context, opts resources.ResourceListOptions) resources.ResourceListResult {
	return resources.ResourceListResult{}
}

func (m *mockResourceRegistry) ReadResource(ctx context.Context, uri string) ([]resources.ResourceContents, error) {
	return nil, errors.New("not implemented")
}

func (m *mockResourceRegistry) SubscribeResource(ctx context.Context, uri string) error {
	return errors.New("not implemented")
}

func (m *mockResourceRegistry) UnsubscribeResource(ctx context.Context, uri string) error {
	return errors.New("not implemented")
}

func (m *mockResourceRegistry) ListResourceTemplates(ctx context.Context, opts resources.ResourceTemplateListOptions) resources.ResourceTemplateListResult {
	return resources.ResourceTemplateListResult{}
}

func (m *mockServerInfo) GetFeatureRegistry() resources.FeatureRegistry {
	return resources.FeatureRegistry{
		ToolRegistry:     &mockToolRegistry{},
		PromptRegistry:   &mockPromptRegistry{},
		ResourceRegistry: &mockResourceRegistry{},
	}
}

func (m *mockServerInfo) GetServerCapabilities() protocol.ServerCapabilities {
	return protocol.ServerCapabilities{}
}

func (m *mockServerInfo) GetServerConfig() *config.ServerConfig {
	return nil
}

func (m *mockServerInfo) GetExecutors() config.MethodHandler {
	return nil
}

func (m *mockServerInfo) GetAuthHandler() config.AuthHandler {
	return &mockAuthHandler{}
}

func (m *mockServerInfo) GetTraceHandler() config.TraceHandler {
	return nil
}

type mockAuthInfo struct{}

func (m *mockAuthInfo) GetPrincipalId() string {
	return "test-principal"
}

type mockAuthHandler struct{}

func (m *mockAuthHandler) ExtractAuth(r *http.Request) auth.AuthInfo {
	return &mockAuthInfo{}
}

func (m *mockAuthHandler) Serialize(auth auth.AuthInfo) ([]byte, error) {
	return nil, nil
}

func (m *mockAuthHandler) Deserialize(b []byte) (auth.AuthInfo, error) {
	return &mockAuthInfo{}, nil
}

type errorReader struct{}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}
