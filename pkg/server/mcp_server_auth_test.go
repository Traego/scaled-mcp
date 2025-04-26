package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/traego/scaled-mcp/pkg/auth"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/resources"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traego/scaled-mcp/pkg/client"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/test/testutils"
)

type TestAuthInfo struct {
	token string
}

func (t TestAuthInfo) GetPrincipalId() string {
	return t.token
}

var _ auth.AuthInfo = (*TestAuthInfo)(nil)

type AuthTestHandler struct{}

func (a AuthTestHandler) ExtractAuth(r *http.Request) auth.AuthInfo {
	authHeader := r.Header.Get("Authorization")
	return TestAuthInfo{token: authHeader}
}

func (t AuthTestHandler) Serialize(auth auth.AuthInfo) ([]byte, error) {
	switch a := auth.(type) {
	case TestAuthInfo:
		return []byte(a.token), nil
	default:
		return nil, errors.New("invalid auth type")
	}
}

func (t AuthTestHandler) Deserialize(bytes []byte) (auth.AuthInfo, error) {
	info := TestAuthInfo{token: string(bytes)}
	return &info, nil
}

var _ config.AuthHandler = (*AuthTestHandler)(nil)

// TestMcpServerWithAuth tests the auth flow through pattern
func TestMcpServerWithAuth(t *testing.T) {
	ctx := context.Background()

	// Get available port for testing
	port, err := testutils.GetAvailablePort()
	require.NoError(t, err, "Failed to get available port")

	// Create default config
	cfg := config.DefaultConfig()
	cfg.HTTP.Port = port

	authorized := false
	rw := sync.Mutex{}

	registry := resources.NewStaticToolRegistry()
	err = registry.RegisterTool(protocol.Tool{
		Name:        "test_tool",
		Description: "Does Stuff",
		InputSchema: protocol.InputSchema{},
	}, func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		ai := auth.GetAuthInfo(ctx)
		if ai != nil {
			rw.Lock()
			authorized = true
			rw.Unlock()
		}

		return nil, nil
	})
	require.NoError(t, err, "Failed to register tool")

	// Create MCP server
	mcpServer, err := NewMcpServer(cfg, WithToolRegistry(registry), WithAuthHandler(&AuthTestHandler{}))
	require.NoError(t, err, "Failed to create MCP server")

	err = mcpServer.Start(ctx)
	require.NoError(t, err, "Failed to start MCP server")

	defer mcpServer.Stop(ctx)
	require.NoError(t, err, "Failed to start MCP server")

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Create a client to test the server
	c, err := client.NewMcpClient(fmt.Sprintf("http://localhost:%d", port), client.DefaultClientOptions(), client.WithAuthHeader("AUTHD"))
	require.NoError(t, err, "Failed to create client")

	err = c.Connect(ctx)
	require.NoError(t, err, "Failed to connect to server")

	// Test sending a request
	resp, err := c.CallTool(ctx, "test_tool", struct{}{})
	require.NoError(t, err, "Failed to send test_tool call")
	assert.NotNil(t, resp, "Response should not be nil")
	assert.Nil(t, resp.Error, "Response should not contain an error")

	assert.True(t, authorized, "Request should have been authorized hitting the endpoint")

	// Add cleanup to shut down both servers when the test completes
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Shutdown the MCP server
		mcpServer.Stop(ctx)
	})
}
