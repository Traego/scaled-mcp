package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := config.DefaultConfig()
	cfg.HTTP.Port = 9986
	cfg.Session.UseInMemory = true

	sessionStartupCallback := func(ctx config.McpContext) error {
		slog.Info("Session started, demonstrating elicitation capability", "session_id", ctx.GetSessionID())

		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Your name",
				},
				"role": map[string]interface{}{
					"type":        "string",
					"description": "Your role",
				},
			},
			"required": []string{"name"},
		}

		slog.Info("Would elicit user data with schema", "schema", schema)

		return nil
	}

	mcpServer, err := server.NewMcpServer(cfg,
		server.WithServerInfo("Elicitation Example Server", "1.0.0"),
		server.WithSessionStartupCallback(sessionStartupCallback),
	)
	if err != nil {
		slog.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := mcpServer.Start(ctx); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}

	slog.Info("Elicitation example server started", "port", cfg.HTTP.Port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	slog.Info("Shutting down server...")
	mcpServer.Stop(ctx)
}
