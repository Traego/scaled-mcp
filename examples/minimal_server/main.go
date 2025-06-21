package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/server"
)

func main() {
	ctx, cancelAll := context.WithCancel(context.Background())

	// Configure logging
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// Create a server configuration
	cfg := config.DefaultConfig()
	cfg.BackwardCompatible20241105 = true

	// Customize configuration if needed
	port := 9985
	if portEnv := os.Getenv("PORT"); portEnv != "" {
		if p, err := strconv.Atoi(portEnv); err == nil {
			port = p
		}
	}
	cfg.HTTP.Port = port

	// Create the MCP server with default options
	// This will create a new HTTP server internally
	mcpServer, err := server.NewMcpServer(cfg)
	if err != nil {
		slog.Error("Failed to create MCP server", "error", err)
		os.Exit(1)
	}

	// Start the server
	if err := mcpServer.Start(ctx); err != nil {
		slog.Error("Failed to start MCP server", "error", err)
		os.Exit(1)
	}

	slog.Info("MCP server started", "port", cfg.HTTP.Port)
	slog.Info("OpenTelemetry telemetry configuration:")
	slog.Info("  OTEL_ENABLED=true (default) - Enable/disable telemetry")
	slog.Info("  OTEL_EXPORTER_TYPE=stdout (default) - Exporter type: stdout, otlp")
	slog.Info("  OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 (default) - OTLP endpoint")
	slog.Info("  OTEL_SERVICE_NAME=scaled-mcp-server (default) - Service name")

	// Wait for termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down MCP server...")

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cancelAll()

	mcpServer.Stop(ctx)

	slog.Info("MCP server stopped")
}
