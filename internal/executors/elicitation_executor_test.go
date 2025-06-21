package executors

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/resources"
)

type TestElicitationServerInfo struct {
	serverConfig *config.ServerConfig
}

func (s *TestElicitationServerInfo) GetServerCapabilities() protocol.ServerCapabilities {
	return protocol.ServerCapabilities{
		Elicitation: &protocol.ElicitationCapability{},
	}
}

func (s *TestElicitationServerInfo) GetServerConfig() *config.ServerConfig {
	return s.serverConfig
}

func (s *TestElicitationServerInfo) GetExecutors() config.MethodHandler {
	return nil
}

func (s *TestElicitationServerInfo) GetFeatureRegistry() resources.FeatureRegistry {
	return resources.FeatureRegistry{}
}

func (s *TestElicitationServerInfo) GetAuthHandler() config.AuthHandler {
	return nil
}

func (s *TestElicitationServerInfo) GetTraceHandler() config.TraceHandler {
	return nil
}

func (s *TestElicitationServerInfo) GetSessionStartupCallback() config.SessionStartupCallback {
	return nil
}

func TestNewElicitationExecutor(t *testing.T) {
	serverInfo := &TestElicitationServerInfo{
		serverConfig: &config.ServerConfig{},
	}
	
	executor := NewElicitationExecutor(serverInfo)
	
	assert.NotNil(t, executor)
	assert.Equal(t, serverInfo, executor.serverInfo)
}

func TestElicitationExecutor_CanHandleMethod(t *testing.T) {
	serverInfo := &TestElicitationServerInfo{
		serverConfig: &config.ServerConfig{},
	}
	executor := NewElicitationExecutor(serverInfo)
	
	tests := []struct {
		method   string
		expected bool
	}{
		{"elicitation/create", true},
		{"tools/list", false},
		{"prompts/list", false},
		{"resources/list", false},
		{"ping", false},
		{"", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			result := executor.CanHandleMethod(tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestElicitationExecutor_HandleMethod_ElicitationCreate(t *testing.T) {
	serverInfo := &TestElicitationServerInfo{
		serverConfig: &config.ServerConfig{},
	}
	executor := NewElicitationExecutor(serverInfo)
	ctx := context.Background()
	
	elicitRequest := protocol.ElicitationRequest{
		Message: "Please provide your name",
		RequestedSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []string{"name"},
		},
	}
	
	paramsJSON, err := json.Marshal(elicitRequest)
	require.NoError(t, err)
	
	req := &mcppb.JsonRpcRequest{
		Jsonrpc:    "2.0",
		Method:     "elicitation/create",
		ParamsJson: string(paramsJSON),
		Id:         &mcppb.JsonRpcRequest_StringId{StringId: "test-1"},
	}
	
	response, err := executor.HandleMethod(ctx, "elicitation/create", req)
	require.NoError(t, err)
	require.NotNil(t, response)
	
	assert.Equal(t, "2.0", response.Jsonrpc)
	assert.Equal(t, "test-1", response.GetStringId())
	
	var elicitResponse protocol.ElicitationResponse
	err = json.Unmarshal([]byte(response.GetResultJson()), &elicitResponse)
	require.NoError(t, err)
	
	assert.Equal(t, protocol.ElicitationActionAccept, elicitResponse.Action)
	assert.NotNil(t, elicitResponse.Content)
	assert.Contains(t, elicitResponse.Content, "mock")
}

func TestElicitationExecutor_HandleMethod_InvalidParams(t *testing.T) {
	serverInfo := &TestElicitationServerInfo{
		serverConfig: &config.ServerConfig{},
	}
	executor := NewElicitationExecutor(serverInfo)
	ctx := context.Background()
	
	req := &mcppb.JsonRpcRequest{
		Jsonrpc:    "2.0",
		Method:     "elicitation/create",
		ParamsJson: "invalid json",
		Id:         &mcppb.JsonRpcRequest_StringId{StringId: "test-2"},
	}
	
	response, err := executor.HandleMethod(ctx, "elicitation/create", req)
	require.Error(t, err)
	assert.Nil(t, response)
	
	assert.Contains(t, err.Error(), "Invalid elicitation request parameters")
}

func TestElicitationExecutor_HandleMethod_UnsupportedMethod(t *testing.T) {
	serverInfo := &TestElicitationServerInfo{
		serverConfig: &config.ServerConfig{},
	}
	executor := NewElicitationExecutor(serverInfo)
	ctx := context.Background()
	
	req := &mcppb.JsonRpcRequest{
		Jsonrpc: "2.0",
		Method:  "unsupported/method",
		Id:      &mcppb.JsonRpcRequest_StringId{StringId: "test-3"},
	}
	
	response, err := executor.HandleMethod(ctx, "unsupported/method", req)
	require.Error(t, err)
	assert.Nil(t, response)
	
	assert.Contains(t, err.Error(), "Method not found")
}

func TestElicitationExecutor_HandleMethod_IntId(t *testing.T) {
	serverInfo := &TestElicitationServerInfo{
		serverConfig: &config.ServerConfig{},
	}
	executor := NewElicitationExecutor(serverInfo)
	ctx := context.Background()
	
	elicitRequest := protocol.ElicitationRequest{
		Message: "Test with int ID",
		RequestedSchema: map[string]interface{}{
			"type": "string",
		},
	}
	
	paramsJSON, err := json.Marshal(elicitRequest)
	require.NoError(t, err)
	
	req := &mcppb.JsonRpcRequest{
		Jsonrpc:    "2.0",
		Method:     "elicitation/create",
		ParamsJson: string(paramsJSON),
		Id:         &mcppb.JsonRpcRequest_IntId{IntId: 42},
	}
	
	response, err := executor.HandleMethod(ctx, "elicitation/create", req)
	require.NoError(t, err)
	require.NotNil(t, response)
	
	assert.Equal(t, "2.0", response.Jsonrpc)
	assert.Equal(t, int64(42), response.GetIntId())
}
