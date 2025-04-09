package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/server"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 8080, "HTTP server port")
	redisAddr := flag.String("redis", "localhost:6379", "Redis address")
	flag.Parse()

	// Configure structured logging
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(logHandler))

	// Create server config
	cfg := config.DefaultConfig()
	cfg.HTTP.Port = *port
	cfg.Redis.Addresses = []string{*redisAddr}

	// Create MCP server
	mcpServer, err := server.NewMcpServer(cfg)
	if err != nil {
		slog.Error("Failed to create MCP server", "error", err)
		os.Exit(1)
	}

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the server
	if err := mcpServer.Start(ctx); err != nil {
		slog.Error("Failed to start MCP server", "error", err)
		os.Exit(1)
	}

	slog.Info("MCP server started",
		"host", cfg.HTTP.Host,
		"port", cfg.HTTP.Port,
		"mcp_path", cfg.HTTP.MCPPath,
	)

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	slog.Info("Received shutdown signal, stopping server...")

	// Create a context with timeout for shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	mcpServer.Stop(shutdownCtx)

	slog.Info("Server stopped gracefully")
}
