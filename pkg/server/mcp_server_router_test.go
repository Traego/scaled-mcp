package server

import (
	"context"
	"fmt"
	"log/slog"
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

// TestMcpServerWithRouter tests the pattern where the user provides their own chi router
// The MCP server will register Handlers on the router and create/start its own HTTP server
func TestMcpServerWithRouter(t *testing.T) {
	ctx := context.Background()

	// Get available port for testing
	port, err := testutils.GetAvailablePort()
	require.NoError(t, err, "Failed to get available port")

	// Create default config
	cfg := config.DefaultConfig()
	cfg.HTTP.Port = port

	// Create a custom router with middleware
	router := chi.NewRouter()

	// Add middleware to inject X-Test-Header
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test-Header", "true")
			next.ServeHTTP(w, r)
		})
	})

	// Create MCP server with custom router
	mcpServer, err := NewMcpServer(cfg, WithRouter(router))
	require.NoError(t, err, "Failed to create MCP server")

	// Start the MCP server - this will register routes and start the HTTP server
	err = mcpServer.Start(ctx)
	require.NoError(t, err, "Failed to start MCP server")

	// Create an HTTP server with the Chi router
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	// Start the HTTP server in a goroutine
	httpErrCh := make(chan error, 1)
	go func() {
		slog.InfoContext(ctx, "Starting HTTP server", "port", port)
		httpErrCh <- httpServer.ListenAndServe()
	}()

	// Give the servers a moment to start up
	time.Sleep(100 * time.Millisecond)

	// Check if there was an immediate error starting the HTTP server
	select {
	case err := <-httpErrCh:
		require.NoError(t, err, "Failed to start HTTP server")
	default:
		// No error, continue with the test
	}

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
	assert.Equal(t, []string{"true"}, resp.Headers["X-Test-Header"], "Response should contain X-Test-Header")

	// Add cleanup to shut down the server when the test completes
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Shutdown the MCP server (which will also shut down the HTTP server)
		mcpServer.Stop(ctx)
	})
}
