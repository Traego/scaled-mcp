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

// TestPromptServerInfo is an in-memory implementation of config.McpServerInfo for testing prompts
type TestPromptServerInfo struct {
	FeatureRegistry resources.FeatureRegistry
	ServerCaps      protocol.ServerCapabilities
	ServerConfig    *config.ServerConfig
}

func NewTestPromptServerInfo() *TestPromptServerInfo {
	// Create a mock prompt registry
	promptRegistry := &MockPromptRegistry{
		prompts: map[string]resources.Prompt{
			"test-prompt": {
				Name:        "test-prompt",
				Description: "A test prompt",
				Messages: []resources.PromptMessage{
					{
						Role:    "system",
						Content: "You are a test assistant",
					},
				},
			},
			"template-prompt": {
				Name:        "template-prompt",
				Description: "A template prompt",
				Messages: []resources.PromptMessage{
					{
						Role:    "system",
						Content: "You are {{role}}",
					},
				},
			},
		},
	}

	return &TestPromptServerInfo{
		FeatureRegistry: resources.FeatureRegistry{
			PromptRegistry: promptRegistry,
		},
		ServerCaps: protocol.ServerCapabilities{
			Prompts: &protocol.PromptsServerCapability{},
			Experimental: map[string]interface{}{
				"version": "1.0.0",
			},
		},
		ServerConfig: &config.ServerConfig{
			ProtocolVersion: protocol.ProtocolVersion20250326,
		},
	}
}

func (s *TestPromptServerInfo) GetFeatureRegistry() resources.FeatureRegistry {
	return s.FeatureRegistry
}

func (s *TestPromptServerInfo) GetServerCapabilities() protocol.ServerCapabilities {
	return s.ServerCaps
}

func (s *TestPromptServerInfo) GetServerConfig() *config.ServerConfig {
	return s.ServerConfig
}

func (s *TestPromptServerInfo) GetExecutors() config.MethodHandler {
	return nil // Not needed for these tests
}

func (s *TestPromptServerInfo) GetAuthHandler() config.AuthHandler {
	return nil
}

// MockPromptRegistry is a mock implementation of the PromptRegistry interface
type MockPromptRegistry struct {
	prompts map[string]resources.Prompt
}

func (m *MockPromptRegistry) ListPrompts(ctx context.Context, opts resources.PromptListOptions) resources.PromptListResult {
	result := resources.PromptListResult{
		Prompts: make([]resources.Prompt, 0, len(m.prompts)),
	}

	for _, prompt := range m.prompts {
		result.Prompts = append(result.Prompts, prompt)
	}

	return result
}

func (m *MockPromptRegistry) GetPrompt(ctx context.Context, name string) (resources.Prompt, bool) {
	prompt, found := m.prompts[name]
	return prompt, found
}

func (m *MockPromptRegistry) ProcessPrompt(ctx context.Context, name string, arguments map[string]string) ([]resources.PromptMessage, error) {
	prompt, found := m.prompts[name]
	if !found {
		return nil, resources.ErrPromptNotFound
	}

	// Process template variables in messages
	processedMessages := make([]resources.PromptMessage, len(prompt.Messages))
	for i, msg := range prompt.Messages {
		processedMsg := resources.PromptMessage{
			Role: msg.Role,
		}

		// Simple template replacement for testing
		content := msg.Content
		if msg.Role == "system" && content == "You are {{role}}" && arguments["role"] != "" {
			content = "You are " + arguments["role"]
		}
		processedMsg.Content = content

		processedMessages[i] = processedMsg
	}

	return processedMessages, nil
}

func TestPromptExecutor_CanHandleMethod(t *testing.T) {
	// Create a test server info
	serverInfo := NewTestPromptServerInfo()

	// Create a prompt executor
	executor := NewPromptExecutor(serverInfo)

	// Test that the executor can handle known methods
	assert.True(t, executor.CanHandleMethod("prompts/list"))
	assert.True(t, executor.CanHandleMethod("prompts/get"))

	// Test that the executor cannot handle unknown methods
	assert.False(t, executor.CanHandleMethod("unknown/method"))
	assert.False(t, executor.CanHandleMethod("prompts/unknown"))
}

func TestPromptExecutor_HandleMethod_List(t *testing.T) {
	// Create a test server info
	serverInfo := NewTestPromptServerInfo()

	// Create a prompt executor
	executor := NewPromptExecutor(serverInfo)

	// Test the prompts/list method
	ctx := context.Background()
	req := &mcppb.JsonRpcRequest{
		Jsonrpc: "2.0",
		Id: &mcppb.JsonRpcRequest_StringId{
			StringId: "1",
		},
		Method: "prompts/list",
	}

	resp, err := executor.HandleMethod(ctx, "prompts/list", req)
	require.NoError(t, err)
	assert.Equal(t, "1", resp.GetStringId())
	assert.Equal(t, "2.0", resp.Jsonrpc)

	// Parse the result
	var result map[string]interface{}
	err = json.Unmarshal([]byte(resp.GetResultJson()), &result)
	require.NoError(t, err)

	// Verify the prompts list
	promptsArray, ok := result["prompts"].([]interface{})
	require.True(t, ok, "Result should contain a 'prompts' array")
	assert.Len(t, promptsArray, 2, "Should have 2 prompts")

	// Check that the prompts have the expected structure
	for _, promptInterface := range promptsArray {
		prompt, ok := promptInterface.(map[string]interface{})
		require.True(t, ok, "Prompt should be a map")

		assert.Contains(t, []string{"test-prompt", "template-prompt"}, prompt["name"].(string))
		assert.NotEmpty(t, prompt["description"])
		assert.NotNil(t, prompt["messages"])
	}
}

func TestPromptExecutor_HandleMethod_Get(t *testing.T) {
	// Create a test server info
	serverInfo := NewTestPromptServerInfo()

	// Create a prompt executor
	executor := NewPromptExecutor(serverInfo)

	// Test the prompts/get method
	ctx := context.Background()

	t.Run("Get Existing Prompt", func(t *testing.T) {
		paramsJSON, _ := json.Marshal(map[string]interface{}{
			"name": "test-prompt",
		})

		req := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "1",
			},
			Method:     "prompts/get",
			ParamsJson: string(paramsJSON),
		}

		resp, err := executor.HandleMethod(ctx, "prompts/get", req)
		require.NoError(t, err)
		assert.Equal(t, "1", resp.GetStringId())
		assert.Equal(t, "2.0", resp.Jsonrpc)

		// Parse the result
		var result map[string]interface{}
		err = json.Unmarshal([]byte(resp.GetResultJson()), &result)
		require.NoError(t, err)

		// Verify the prompt details
		assert.Equal(t, "A test prompt", result["description"])

		messages, ok := result["messages"].([]interface{})
		require.True(t, ok, "Result should contain a 'messages' array")
		assert.Len(t, messages, 1, "Should have 1 message")

		message, ok := messages[0].(map[string]interface{})
		require.True(t, ok, "Message should be a map")
		assert.Equal(t, "system", message["role"])
		assert.Equal(t, "You are a test assistant", message["content"])
	})

	t.Run("Get Prompt With Template Processing", func(t *testing.T) {
		paramsJSON, _ := json.Marshal(map[string]interface{}{
			"name": "template-prompt",
			"arguments": map[string]interface{}{
				"role": "a helpful assistant",
			},
		})

		req := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "2",
			},
			Method:     "prompts/get",
			ParamsJson: string(paramsJSON),
		}

		resp, err := executor.HandleMethod(ctx, "prompts/get", req)
		require.NoError(t, err)
		assert.Equal(t, "2", resp.GetStringId())

		// Parse the result
		var result map[string]interface{}
		err = json.Unmarshal([]byte(resp.GetResultJson()), &result)
		require.NoError(t, err)

		// Verify the processed template
		messages, ok := result["messages"].([]interface{})
		require.True(t, ok, "Result should contain a 'messages' array")

		message, ok := messages[0].(map[string]interface{})
		require.True(t, ok, "Message should be a map")
		assert.Equal(t, "You are a helpful assistant", message["content"], "Template should be processed")
	})

	t.Run("Get Non-Existent Prompt", func(t *testing.T) {
		paramsJSON, _ := json.Marshal(map[string]interface{}{
			"name": "non-existent-prompt",
		})

		req := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "3",
			},
			Method:     "prompts/get",
			ParamsJson: string(paramsJSON),
		}

		resp, err := executor.HandleMethod(ctx, "prompts/get", req)
		assert.Error(t, err, "Should return error for non-existent prompt")
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Missing Name Parameter", func(t *testing.T) {
		paramsJSON, _ := json.Marshal(map[string]interface{}{
			// Missing "name" parameter
		})

		req := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "4",
			},
			Method:     "prompts/get",
			ParamsJson: string(paramsJSON),
		}

		resp, err := executor.HandleMethod(ctx, "prompts/get", req)
		assert.Error(t, err, "Should return error for missing name parameter")
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "prompt name is required")
	})
}

func TestPromptExecutor_HandleMethod_InvalidMethod(t *testing.T) {
	// Create a test server info
	serverInfo := NewTestPromptServerInfo()

	// Create a prompt executor
	executor := NewPromptExecutor(serverInfo)

	// Test handling an invalid method
	ctx := context.Background()
	req := &mcppb.JsonRpcRequest{
		Jsonrpc: "2.0",
		Id: &mcppb.JsonRpcRequest_StringId{
			StringId: "1",
		},
		Method: "prompts/invalid",
	}

	resp, err := executor.HandleMethod(ctx, "prompts/invalid", req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Method not found")
	jre := err.(*protocol.JsonRpcError)
	jre.Code = -32601
}

func TestPromptExecutor_HandleMethod_NilPromptRegistry(t *testing.T) {
	// Create a test server info with nil prompt registry
	serverInfo := NewTestPromptServerInfo()
	serverInfo.FeatureRegistry.PromptRegistry = nil

	// Create a prompt executor
	executor := NewPromptExecutor(serverInfo)

	// Test that methods return an error when prompt registry is nil
	ctx := context.Background()
	req := &mcppb.JsonRpcRequest{
		Jsonrpc: "2.0",
		Id: &mcppb.JsonRpcRequest_StringId{
			StringId: "1",
		},
		Method: "prompts/list",
	}

	resp, err := executor.HandleMethod(ctx, "prompts/list", req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Method not found")
	jre := err.(*protocol.JsonRpcError)
	jre.Code = -32601
}

func TestPromptExecutor_HandleMethod_IntId(t *testing.T) {
	// Create a test server info
	serverInfo := NewTestPromptServerInfo()

	// Create a prompt executor
	executor := NewPromptExecutor(serverInfo)

	// Test with an integer ID
	ctx := context.Background()
	req := &mcppb.JsonRpcRequest{
		Jsonrpc: "2.0",
		Id: &mcppb.JsonRpcRequest_IntId{
			IntId: 42,
		},
		Method: "prompts/list",
	}

	resp, err := executor.HandleMethod(ctx, "prompts/list", req)
	require.NoError(t, err)
	assert.Equal(t, int64(42), resp.GetIntId())
	assert.Equal(t, "2.0", resp.Jsonrpc)
}
