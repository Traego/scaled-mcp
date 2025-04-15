package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/server"
)

func main() {
	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up logging
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// Create default config
	cfg := config.DefaultConfig()
	cfg.HTTP.Port = 8080 // Set the port for the example

	// Create MCP server
	mcpServer, err := server.NewMcpServer(cfg)
	if err != nil {
		slog.Error("Failed to create MCP server", "error", err)
		os.Exit(1)
	}

	// Create a simple middleware to add a header
	headerMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Example", "true")
			next.ServeHTTP(w, r)
		})
	}

	// Create HTTP server with the MCP server as handler, wrapped with middleware
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler: headerMiddleware(mcpServer),
	}

	// Start the MCP server - this will NOT start the HTTP server
	err = mcpServer.Start(ctx)
	if err != nil {
		slog.Error("Failed to start MCP server", "error", err)
		os.Exit(1)
	}

	// Set up graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start the HTTP server in a goroutine
	go func() {
		slog.Info("Starting HTTP server", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	slog.Info("Server started", "addr", httpServer.Addr)
	slog.Info("MCP endpoint available at http://localhost:8080/mcp")

	// Wait for shutdown signal
	<-stop
	slog.Info("Shutting down...")

	// Create a deadline for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown the HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	// Stop the MCP server
	mcpServer.Stop(shutdownCtx)

	slog.Info("Server stopped")
}
