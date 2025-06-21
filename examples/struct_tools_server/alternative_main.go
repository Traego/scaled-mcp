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
	"github.com/traego/scaled-mcp/pkg/resources"
	"github.com/traego/scaled-mcp/pkg/server"
)

type SimpleInput struct {
	Message string `mcp:"message,The message to echo,required"`
	Count   int    `mcp:"count,Number of times to repeat,default=1"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(logHandler))

	toolRegistry := resources.NewStaticToolRegistry()

	echoHandler := func(ctx context.Context, input *SimpleInput) (interface{}, error) {
		result := make([]string, input.Count)
		for i := 0; i < input.Count; i++ {
			result[i] = input.Message
		}
		return map[string]interface{}{
			"messages": result,
			"count":    input.Count,
		}, nil
	}

	if err := resources.RegisterStructTool(toolRegistry, "echo", "Echo a message multiple times", echoHandler); err != nil {
		slog.Error("Failed to register echo tool", "error", err)
		os.Exit(1)
	}

	cfg := config.DefaultConfig()
	cfg.BackwardCompatible20241105 = true
	cfg.HTTP.Port = 9987
	cfg.HTTP.CORS.Enable = true
	cfg.HTTP.CORS.AllowCredentials = true
	cfg.HTTP.CORS.AllowedOrigins = []string{"*"}

	mcpServer, err := server.NewMcpServer(cfg,
		server.WithToolRegistry(toolRegistry),
	)
	if err != nil {
		slog.Error("Failed to create MCP server", "error", err)
		os.Exit(1)
	}

	go func() {
		if err := mcpServer.Start(ctx); err != nil && err != http.ErrServerClosed {
			slog.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("Alternative struct-based tools server is available on port 9987")
	slog.Info("Tools available: echo")
	slog.Info("This example uses the convenience RegisterStructTool function")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	sctx, c2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer c2()

	mcpServer.Stop(sctx)
}
