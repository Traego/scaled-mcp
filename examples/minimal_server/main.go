package main

import (
	"context"
	"fmt"
	"github.com/traego/scaled-mcp/pkg/resources"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/server"
)

type HelloInput struct {
	Name string `mcp:"name,Your name for the server to greet you,required"`
}

type HelloOutput struct {
	Message string `mcp:"result,A special greeting from the server,required" json:"result"`
}

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
	cfg.HTTP.Port = 9985

	toolRegistry := resources.NewStaticToolRegistry()
	resources.MustRegisterTool(toolRegistry, "hello_world", "A tool to allow the server to greet you", HelloHandler)

	// Create the MCP server with default options
	// This will create a new HTTP server internally
	mcpServer, err := server.NewMcpServer(cfg, server.WithToolRegistry(toolRegistry))
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

func HelloHandler(ctx context.Context, input *HelloInput) (*HelloOutput, error) {
	msg := fmt.Sprintf("Hello, %s!", input.Name)
	return &HelloOutput{Message: msg}, nil
}
