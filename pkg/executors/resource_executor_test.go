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

// TestResourceServerInfo is an in-memory implementation of config.McpServerInfo for testing resources
type TestResourceServerInfo struct {
	FeatureRegistry resources.FeatureRegistry
	ServerCaps      protocol.ServerCapabilities
	ServerConfig    *config.ServerConfig
}

func NewTestResourceServerInfo() *TestResourceServerInfo {
	// Create a mock resource registry
	resourceRegistry := &MockResourceRegistry{
		resources: map[string][]resources.ResourceContents{
			"test-resource": {
				{
					URI:      "test-resource",
					MimeType: "text/plain",
					Content:  "This is a test resource",
				},
			},
		},
		resourceList: []resources.Resource{
			{
				URI:         "test-resource",
				Name:        "Test Resource",
				Description: "A test resource",
				MimeType:    "text/plain",
				Size:        22, // Length of "This is a test resource"
			},
		},
		resourceTemplates: []resources.ResourceTemplate{
			{
				URITemplate: "template/{id}",
				Name:        "Template Resource",
				Description: "A template for resources",
				MimeType:    "text/plain",
			},
		},
		subscriptions: make(map[string]bool),
	}

	return &TestResourceServerInfo{
		FeatureRegistry: resources.FeatureRegistry{
			ResourceRegistry: resourceRegistry,
		},
		ServerCaps: protocol.ServerCapabilities{
			Resources: &protocol.ResourcesServerCapability{},
			Experimental: map[string]interface{}{
				"version": "1.0.0",
			},
		},
		ServerConfig: &config.ServerConfig{
			ProtocolVersion: protocol.ProtocolVersion20250326,
		},
	}
}

func (s *TestResourceServerInfo) GetFeatureRegistry() resources.FeatureRegistry {
	return s.FeatureRegistry
}

func (s *TestResourceServerInfo) GetServerCapabilities() protocol.ServerCapabilities {
	return s.ServerCaps
}

func (s *TestResourceServerInfo) GetServerConfig() *config.ServerConfig {
	return s.ServerConfig
}

func (s *TestResourceServerInfo) GetExecutors() config.MethodHandler {
	return nil // Not needed for these tests
}

// MockResourceRegistry is a mock implementation of the ResourceRegistry interface
type MockResourceRegistry struct {
	resources         map[string][]resources.ResourceContents
	resourceList      []resources.Resource
	resourceTemplates []resources.ResourceTemplate
	subscriptions     map[string]bool
}

func (m *MockResourceRegistry) ListResources(ctx context.Context, opts resources.ResourceListOptions) resources.ResourceListResult {
	result := resources.ResourceListResult{
		Resources: make([]resources.Resource, 0, len(m.resourceList)),
	}

	// Simple implementation that ignores cursor for testing
	result.Resources = append(result.Resources, m.resourceList...)

	return result
}

func (m *MockResourceRegistry) ReadResource(ctx context.Context, uri string) ([]resources.ResourceContents, error) {
	contents, found := m.resources[uri]
	if !found {
		return nil, resources.ErrResourceNotFound
	}
	return contents, nil
}

func (m *MockResourceRegistry) SubscribeResource(ctx context.Context, uri string) error {
	if _, found := m.resources[uri]; !found {
		return resources.ErrResourceNotFound
	}
	m.subscriptions[uri] = true
	return nil
}

func (m *MockResourceRegistry) UnsubscribeResource(ctx context.Context, uri string) error {
	if _, found := m.resources[uri]; !found {
		return resources.ErrResourceNotFound
	}
	delete(m.subscriptions, uri)
	return nil
}

func (m *MockResourceRegistry) ListResourceTemplates(ctx context.Context, opts resources.ResourceTemplateListOptions) resources.ResourceTemplateListResult {
	result := resources.ResourceTemplateListResult{
		ResourceTemplates: make([]resources.ResourceTemplate, 0, len(m.resourceTemplates)),
	}

	// Simple implementation that ignores cursor for testing
	result.ResourceTemplates = append(result.ResourceTemplates, m.resourceTemplates...)

	return result
}

func TestResourceExecutor_CanHandleMethod(t *testing.T) {
	// Create a test server info
	serverInfo := NewTestResourceServerInfo()

	// Create a resource executor
	executor := NewResourceExecutor(serverInfo)

	// Test that the executor can handle known methods
	assert.True(t, executor.CanHandleMethod("resources/list"))
	assert.True(t, executor.CanHandleMethod("resources/read"))
	assert.True(t, executor.CanHandleMethod("resources/subscribe"))
	assert.True(t, executor.CanHandleMethod("resources/unsubscribe"))
	assert.True(t, executor.CanHandleMethod("resources/templates/list"))

	// Test that the executor cannot handle unknown methods
	assert.False(t, executor.CanHandleMethod("unknown/method"))
	assert.False(t, executor.CanHandleMethod("resources/unknown"))
}

func TestResourceExecutor_HandleMethod_List(t *testing.T) {
	// Create a test server info
	serverInfo := NewTestResourceServerInfo()

	// Create a resource executor
	executor := NewResourceExecutor(serverInfo)

	// Test context
	ctx := context.Background()

	// Test listing resources
	req := &mcppb.JsonRpcRequest{
		Jsonrpc: "2.0",
		Id: &mcppb.JsonRpcRequest_StringId{
			StringId: "1",
		},
		Method: "resources/list",
	}

	resp, err := executor.HandleMethod(ctx, "resources/list", req)
	require.NoError(t, err)
	assert.Equal(t, "1", resp.GetStringId())

	// Parse the result
	var result map[string]interface{}
	err = json.Unmarshal([]byte(resp.GetResultJson()), &result)
	require.NoError(t, err)

	// Verify the resources
	resources, ok := result["resources"].([]interface{})
	require.True(t, ok, "Result should contain a 'resources' array")
	assert.Equal(t, 1, len(resources), "Should have 1 resource")

	// Test with cursor
	paramsJSON, _ := json.Marshal(map[string]interface{}{
		"cursor": "some-cursor",
	})

	req = &mcppb.JsonRpcRequest{
		Jsonrpc: "2.0",
		Id: &mcppb.JsonRpcRequest_StringId{
			StringId: "2",
		},
		Method:     "resources/list",
		ParamsJson: string(paramsJSON),
	}

	resp, err = executor.HandleMethod(ctx, "resources/list", req)
	require.NoError(t, err)
	assert.Equal(t, "2", resp.GetStringId())
}

func TestResourceExecutor_HandleMethod_Read(t *testing.T) {
	// Create a test server info
	serverInfo := NewTestResourceServerInfo()

	// Create a resource executor
	executor := NewResourceExecutor(serverInfo)

	// Test context
	ctx := context.Background()

	t.Run("Read Existing Resource", func(t *testing.T) {
		paramsJSON, _ := json.Marshal(map[string]interface{}{
			"uri": "test-resource",
		})

		req := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "1",
			},
			Method:     "resources/read",
			ParamsJson: string(paramsJSON),
		}

		resp, err := executor.HandleMethod(ctx, "resources/read", req)
		require.NoError(t, err)
		assert.Equal(t, "1", resp.GetStringId())

		// Parse the result
		var result []interface{}
		err = json.Unmarshal([]byte(resp.GetResultJson()), &result)
		require.NoError(t, err)

		// Verify the resource contents
		require.Equal(t, 1, len(result), "Should have 1 resource content")
		content := result[0].(map[string]interface{})
		assert.Equal(t, "test-resource", content["uri"], "URI should match")
		assert.Equal(t, "text/plain", content["mimeType"], "MIME type should match")
		assert.Equal(t, "This is a test resource", content["content"], "Content should match")
	})

	t.Run("Read Non-Existent Resource", func(t *testing.T) {
		paramsJSON, _ := json.Marshal(map[string]interface{}{
			"uri": "non-existent-resource",
		})

		req := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "2",
			},
			Method:     "resources/read",
			ParamsJson: string(paramsJSON),
		}

		resp, err := executor.HandleMethod(ctx, "resources/read", req)
		assert.Error(t, err, "Should return error for non-existent resource")
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, resources.ErrResourceNotFound)
	})

	t.Run("Missing URI Parameter", func(t *testing.T) {
		paramsJSON, _ := json.Marshal(map[string]interface{}{
			// Missing "uri" parameter
		})

		req := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "3",
			},
			Method:     "resources/read",
			ParamsJson: string(paramsJSON),
		}

		resp, err := executor.HandleMethod(ctx, "resources/read", req)
		assert.Error(t, err, "Should return error for missing URI parameter")
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "resource URI is required")
	})

	t.Run("Empty URI Parameter", func(t *testing.T) {
		paramsJSON, _ := json.Marshal(map[string]interface{}{
			"uri": "",
		})

		req := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "4",
			},
			Method:     "resources/read",
			ParamsJson: string(paramsJSON),
		}

		resp, err := executor.HandleMethod(ctx, "resources/read", req)
		assert.Error(t, err, "Should return error for empty URI parameter")
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "resource URI must be a non-empty string")
	})
}

func TestResourceExecutor_HandleMethod_Subscribe(t *testing.T) {
	// Create a test server info
	serverInfo := NewTestResourceServerInfo()

	// Create a resource executor
	executor := NewResourceExecutor(serverInfo)

	// Test context
	ctx := context.Background()

	t.Run("Subscribe to Existing Resource", func(t *testing.T) {
		paramsJSON, _ := json.Marshal(map[string]interface{}{
			"uri": "test-resource",
		})

		req := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "1",
			},
			Method:     "resources/subscribe",
			ParamsJson: string(paramsJSON),
		}

		resp, err := executor.HandleMethod(ctx, "resources/subscribe", req)
		require.NoError(t, err)
		assert.Equal(t, "1", resp.GetStringId())

		// Parse the result
		var result map[string]interface{}
		err = json.Unmarshal([]byte(resp.GetResultJson()), &result)
		require.NoError(t, err)

		// Verify the success response
		success, ok := result["success"].(bool)
		require.True(t, ok, "Result should contain a 'success' boolean")
		assert.True(t, success, "Subscribe should be successful")
	})

	t.Run("Subscribe to Non-Existent Resource", func(t *testing.T) {
		paramsJSON, _ := json.Marshal(map[string]interface{}{
			"uri": "non-existent-resource",
		})

		req := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "2",
			},
			Method:     "resources/subscribe",
			ParamsJson: string(paramsJSON),
		}

		resp, err := executor.HandleMethod(ctx, "resources/subscribe", req)
		assert.Error(t, err, "Should return error for non-existent resource")
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "resource not found")
	})

	t.Run("Missing URI Parameter", func(t *testing.T) {
		paramsJSON, _ := json.Marshal(map[string]interface{}{
			// Missing "uri" parameter
		})

		req := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "3",
			},
			Method:     "resources/subscribe",
			ParamsJson: string(paramsJSON),
		}

		resp, err := executor.HandleMethod(ctx, "resources/subscribe", req)
		assert.Error(t, err, "Should return error for missing URI parameter")
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "resource URI is required")
	})
}

func TestResourceExecutor_HandleMethod_Unsubscribe(t *testing.T) {
	// Create a test server info
	serverInfo := NewTestResourceServerInfo()

	// Create a resource executor
	executor := NewResourceExecutor(serverInfo)

	// Test context
	ctx := context.Background()

	t.Run("Unsubscribe from Existing Resource", func(t *testing.T) {
		// First subscribe to the resource
		mockRegistry := serverInfo.GetFeatureRegistry().ResourceRegistry.(*MockResourceRegistry)
		mockRegistry.subscriptions["test-resource"] = true

		paramsJSON, _ := json.Marshal(map[string]interface{}{
			"uri": "test-resource",
		})

		req := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "1",
			},
			Method:     "resources/unsubscribe",
			ParamsJson: string(paramsJSON),
		}

		resp, err := executor.HandleMethod(ctx, "resources/unsubscribe", req)
		require.NoError(t, err)
		assert.Equal(t, "1", resp.GetStringId())

		// Parse the result
		var result map[string]interface{}
		err = json.Unmarshal([]byte(resp.GetResultJson()), &result)
		require.NoError(t, err)

		// Verify the success response
		success, ok := result["success"].(bool)
		require.True(t, ok, "Result should contain a 'success' boolean")
		assert.True(t, success, "Unsubscribe should be successful")

		// Verify the subscription was removed
		_, exists := mockRegistry.subscriptions["test-resource"]
		assert.False(t, exists, "Subscription should be removed")
	})

	t.Run("Unsubscribe from Non-Existent Resource", func(t *testing.T) {
		paramsJSON, _ := json.Marshal(map[string]interface{}{
			"uri": "non-existent-resource",
		})

		req := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "2",
			},
			Method:     "resources/unsubscribe",
			ParamsJson: string(paramsJSON),
		}

		resp, err := executor.HandleMethod(ctx, "resources/unsubscribe", req)
		assert.Error(t, err, "Should return error for non-existent resource")
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "resource not found")
	})

	t.Run("Missing URI Parameter", func(t *testing.T) {
		paramsJSON, _ := json.Marshal(map[string]interface{}{
			// Missing "uri" parameter
		})

		req := &mcppb.JsonRpcRequest{
			Jsonrpc: "2.0",
			Id: &mcppb.JsonRpcRequest_StringId{
				StringId: "3",
			},
			Method:     "resources/unsubscribe",
			ParamsJson: string(paramsJSON),
		}

		resp, err := executor.HandleMethod(ctx, "resources/unsubscribe", req)
		assert.Error(t, err, "Should return error for missing URI parameter")
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "resource URI is required")
	})
}

func TestResourceExecutor_HandleMethod_ListTemplates(t *testing.T) {
	// Create a test server info
	serverInfo := NewTestResourceServerInfo()

	// Create a resource executor
	executor := NewResourceExecutor(serverInfo)

	// Test context
	ctx := context.Background()

	// Test listing resource templates
	req := &mcppb.JsonRpcRequest{
		Jsonrpc: "2.0",
		Id: &mcppb.JsonRpcRequest_StringId{
			StringId: "1",
		},
		Method: "resources/templates/list",
	}

	resp, err := executor.HandleMethod(ctx, "resources/templates/list", req)
	require.NoError(t, err)
	assert.Equal(t, "1", resp.GetStringId())

	// Parse the result
	var result map[string]interface{}
	err = json.Unmarshal([]byte(resp.GetResultJson()), &result)
	require.NoError(t, err)

	// Verify the resource templates
	templates, ok := result["resourceTemplates"].([]interface{})
	require.True(t, ok, "Result should contain a 'resourceTemplates' array")
	assert.Equal(t, 1, len(templates), "Should have 1 resource template")

	// Test with cursor
	paramsJSON, _ := json.Marshal(map[string]interface{}{
		"cursor": "some-cursor",
	})

	req = &mcppb.JsonRpcRequest{
		Jsonrpc: "2.0",
		Id: &mcppb.JsonRpcRequest_StringId{
			StringId: "2",
		},
		Method:     "resources/templates/list",
		ParamsJson: string(paramsJSON),
	}

	resp, err = executor.HandleMethod(ctx, "resources/templates/list", req)
	require.NoError(t, err)
	assert.Equal(t, "2", resp.GetStringId())
}

func TestResourceExecutor_HandleMethod_InvalidMethod(t *testing.T) {
	// Create a test server info
	serverInfo := NewTestResourceServerInfo()

	// Create a resource executor
	executor := NewResourceExecutor(serverInfo)

	// Test context
	ctx := context.Background()

	// Test handling an invalid method
	req := &mcppb.JsonRpcRequest{
		Jsonrpc: "2.0",
		Id: &mcppb.JsonRpcRequest_StringId{
			StringId: "1",
		},
		Method: "resources/invalid",
	}

	resp, err := executor.HandleMethod(ctx, "resources/invalid", req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Method not found")
}

func TestResourceExecutor_HandleMethod_NilResourceRegistry(t *testing.T) {
	// Create a test server info with nil resource registry
	serverInfo := NewTestResourceServerInfo()
	serverInfo.FeatureRegistry.ResourceRegistry = nil

	// Create a resource executor
	executor := NewResourceExecutor(serverInfo)

	// Test context
	ctx := context.Background()

	// Test that methods return an error when resource registry is nil
	req := &mcppb.JsonRpcRequest{
		Jsonrpc: "2.0",
		Id: &mcppb.JsonRpcRequest_StringId{
			StringId: "1",
		},
		Method: "resources/list",
	}

	resp, err := executor.HandleMethod(ctx, "resources/list", req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Method not found")
}

func TestResourceExecutor_HandleMethod_IntId(t *testing.T) {
	// Create a test server info
	serverInfo := NewTestResourceServerInfo()

	// Create a resource executor
	executor := NewResourceExecutor(serverInfo)

	// Test context
	ctx := context.Background()

	// Test with an integer ID
	req := &mcppb.JsonRpcRequest{
		Jsonrpc: "2.0",
		Id: &mcppb.JsonRpcRequest_IntId{
			IntId: 42,
		},
		Method: "resources/list",
	}

	resp, err := executor.HandleMethod(ctx, "resources/list", req)
	require.NoError(t, err)
	assert.Equal(t, int64(42), resp.GetIntId())
	assert.Equal(t, "2.0", resp.Jsonrpc)
}
