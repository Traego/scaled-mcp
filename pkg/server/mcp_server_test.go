package server

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traego/scaled-mcp/pkg/client"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/test/testutils"
)

func TestMcpServerWithHttpServer(t *testing.T) {
	ctx := context.Background()

	// Get available port for testing
	port, err := testutils.GetAvailablePort()
	require.NoError(t, err, "Failed to get available port")

	// Create default config
	cfg := config.DefaultConfig()

	// Create a custom router with middleware
	router := chi.NewRouter()

	// Add middleware to inject X-Test-Header
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test-Header", "true")
			next.ServeHTTP(w, r)
		})
	})

	// Create HTTP server with the router
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
	}

	// Create MCP server with both custom HTTP server and router
	mcpServer, err := NewMcpServer(cfg, WithHttpServer(server), WithRouter(router))
	require.NoError(t, err, "Failed to create MCP server")

	// Start the MCP server - this will register routes but not start the HTTP server
	err = mcpServer.Start(ctx)
	require.NoError(t, err, "Failed to start MCP server")

	// Start the HTTP server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("HTTP server error: %v", err)
		}
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Create a client to test the server
	c, err := client.NewMcpClient(fmt.Sprintf("http://localhost:%d", port), client.DefaultClientOptions())
	require.NoError(t, err, "Failed to create client")

	err = c.Connect(ctx)
	require.NoError(t, err, "Failed to connect to server")

	// Test sending a request
	resp, err := c.SendRequest(ctx, "tools/list", nil)
	require.NoError(t, err, "Failed to send tools/list request")
	assert.NotNil(t, resp, "Response should not be nil")
	assert.Nil(t, resp.Error, "Response should not contain an error")

	// TODO: Add assertion to verify X-Test-Header was set
	// This would require a direct HTTP request rather than using the client

	assert.Equal(t, resp.Headers["X-Test-Header"], []string{"true"}, "Response should contain true")
	// Add cleanup to shut down both servers when the test completes
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Shutdown the MCP server
		mcpServer.Stop(ctx)

		// Shutdown the HTTP server
		_ = server.Shutdown(ctx)
	})
}
