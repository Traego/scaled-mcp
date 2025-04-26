package executors

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/resources"
)

// TestUtilitiesServerInfo is an in-memory implementation of config.McpServerInfo for testing utilities
type TestUtilitiesServerInfo struct {
	FeatureRegistry resources.FeatureRegistry
	ServerCaps      protocol.ServerCapabilities
	ServerConfig    *config.ServerConfig
}

func NewTestUtilitiesServerInfo() *TestUtilitiesServerInfo {
	return &TestUtilitiesServerInfo{
		FeatureRegistry: resources.FeatureRegistry{},
		ServerCaps: protocol.ServerCapabilities{
			Logging: &protocol.LoggingServerCapability{},
			Experimental: map[string]interface{}{
				"version": "1.0.0",
			},
		},
		ServerConfig: &config.ServerConfig{
			ProtocolVersion: protocol.ProtocolVersion20250326,
		},
	}
}

func (s *TestUtilitiesServerInfo) GetFeatureRegistry() resources.FeatureRegistry {
	return s.FeatureRegistry
}

func (s *TestUtilitiesServerInfo) GetServerCapabilities() protocol.ServerCapabilities {
	return s.ServerCaps
}

func (s *TestUtilitiesServerInfo) GetServerConfig() *config.ServerConfig {
	return s.ServerConfig
}

func (s *TestUtilitiesServerInfo) GetExecutors() config.MethodHandler {
	return nil // Not needed for these tests
}

func (s *TestUtilitiesServerInfo) GetAuthHandler() config.AuthHandler {
	return nil // Not needed for these tests
}

func TestUtilitiesExecutor_CanHandleMethod(t *testing.T) {
	// Create a test server info
	serverInfo := NewTestUtilitiesServerInfo()

	// Create a utilities executor
	executor := NewUtilitiesExecutor(serverInfo)

	// Test that the executor can handle known methods
	assert.True(t, executor.CanHandleMethod("ping"))

	// Test that the executor cannot handle unknown methods
	assert.False(t, executor.CanHandleMethod("unknown/method"))
	assert.False(t, executor.CanHandleMethod("utilities/unknown"))
}

func TestUtilitiesExecutor_HandleMethod_Ping(t *testing.T) {
	// Create a test server info
	serverInfo := NewTestUtilitiesServerInfo()

	// Create a utilities executor
	executor := NewUtilitiesExecutor(serverInfo)

	// Test the ping method
	ctx := context.Background()
	req := &mcppb.JsonRpcRequest{
		Jsonrpc: "2.0",
		Id: &mcppb.JsonRpcRequest_StringId{
			StringId: "1",
		},
		Method: "ping",
	}

	resp, err := executor.HandleMethod(ctx, "ping", req)
	require.NoError(t, err)
	assert.Equal(t, "1", resp.GetStringId())
	assert.Equal(t, "2.0", resp.Jsonrpc)

	// Parse the result
	var result map[string]interface{}
	err = json.Unmarshal([]byte(resp.GetResultJson()), &result)
	require.NoError(t, err)

	// The ping response should be an empty object according to the implementation
	assert.Empty(t, result)
}

func TestUtilitiesExecutor_HandleMethod_InvalidMethod(t *testing.T) {
	// Create a test server info
	serverInfo := NewTestUtilitiesServerInfo()

	// Create a utilities executor
	executor := NewUtilitiesExecutor(serverInfo)

	// Test handling an invalid method
	ctx := context.Background()
	req := &mcppb.JsonRpcRequest{
		Jsonrpc: "2.0",
		Id: &mcppb.JsonRpcRequest_StringId{
			StringId: "1",
		},
		Method: "utilities/invalid",
	}

	resp, err := executor.HandleMethod(ctx, "utilities/invalid", req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestUtilitiesExecutor_HandleMethod_IntId(t *testing.T) {
	// Create a test server info
	serverInfo := NewTestUtilitiesServerInfo()

	// Create a utilities executor
	executor := NewUtilitiesExecutor(serverInfo)

	// Test the ping method with an integer ID
	ctx := context.Background()
	req := &mcppb.JsonRpcRequest{
		Jsonrpc: "2.0",
		Id: &mcppb.JsonRpcRequest_IntId{
			IntId: 42,
		},
		Method: "ping",
	}

	resp, err := executor.HandleMethod(ctx, "ping", req)
	require.NoError(t, err)
	assert.Equal(t, int64(42), resp.GetIntId())
	assert.Equal(t, "2.0", resp.Jsonrpc)

	// Parse the result
	var result map[string]interface{}
	err = json.Unmarshal([]byte(resp.GetResultJson()), &result)
	require.NoError(t, err)

	// The ping response should be an empty object according to the implementation
	assert.Empty(t, result)
}
