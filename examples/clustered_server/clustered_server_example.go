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

// Simple tool provider for testing
type SimpleToolProvider struct {
	tools map[string]resources.Tool
}

func NewSimpleToolProvider() *SimpleToolProvider {
	provider := &SimpleToolProvider{
		tools: make(map[string]resources.Tool),
	}

	// Register a simple echo tool
	provider.tools["echo"] = resources.NewTool("echo").
		WithDescription("Echo back the input").
		WithString("message").
		Required().
		Description("Message to echo").
		Add().
		Build()

	return provider
}

func (p *SimpleToolProvider) GetTool(ctx context.Context, name string) (resources.Tool, bool) {
	tool, found := p.tools[name]
	return tool, found
}

func (p *SimpleToolProvider) ListTools(ctx context.Context, cursor string) ([]resources.Tool, string) {
	tools := make([]resources.Tool, 0, len(p.tools))
	for _, tool := range p.tools {
		tools = append(tools, tool)
	}
	return tools, ""
}

func (p *SimpleToolProvider) HandleToolInvocation(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
	if name == "echo" {
		message, ok := params["message"].(string)
		if !ok {
			return nil, fmt.Errorf("%w: message parameter must be a string", resources.ErrInvalidParams)
		}
		return map[string]interface{}{
			"message": message,
			"server":  os.Getenv("SERVER_NAME"),
		}, nil
	}
	return nil, resources.ErrToolNotFound
}

func startServer(ctx context.Context, cfg *config.ServerConfig) (*server.McpServer, error) {
	// Create a tool provider
	toolProvider := NewSimpleToolProvider()
	registry := resources.NewDynamicToolRegistry(toolProvider)

	// Create server with the tool registry
	mcpServer, err := server.NewMcpServer(cfg,
		server.WithToolRegistry(registry),
		server.WithServerInfo(cfg.ServerInfo.Name, cfg.ServerInfo.Version),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	// Start the server in a goroutine
	go func() {
		if err := mcpServer.Start(ctx); err != nil && err != http.ErrServerClosed {
			slog.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("Server started",
		"name", cfg.ServerInfo.Name,
		"host", cfg.HTTP.Host,
		"port", cfg.HTTP.Port,
		"actor_port", cfg.Actor.RemoteConfig.Port)

	return mcpServer, nil
}

func main() {
	// Configure logging
	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(logHandler))

	// Create context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create configuration for the first server
	cfg1 := config.DefaultConfig()
	cfg1.HTTP.Port = 8080
	cfg1.ServerInfo.Name = "Server1"
	cfg1.BackwardCompatible20241105 = true
	err := os.Setenv("SERVER_NAME", "Server1")
	if err != nil {
		panic(err)
	}

	// Enable remote actors for clustering
	cfg1.Actor.UseRemoteActors = true
	cfg1.Actor.RemoteConfig.Host = "localhost"
	cfg1.Actor.RemoteConfig.Port = 9090

	cfg1.Clustering.NodeHost = "localhost"
	cfg1.Clustering.RemotingPort = 7010
	cfg1.Clustering.GossipPort = 8010
	cfg1.Clustering.PeersPort = 9090

	cfg1.Clustering.StaticHosts = []string{"localhost:8010", "localhost:8011"}
	cfg1.Clustering.Type = config.ClusteringTypeStatic

	//// Use Redis for session management to share sessions between servers
	//cfg1.Redis = &config.RedisConfig{
	//	Addresses: []string{"localhost:6379"},
	//	Password:  "",
	//	DB:        0,
	//}
	//cfg1.Session.UseInMemory = false

	// Start the first server
	server1, err := startServer(ctx, cfg1)
	if err != nil {
		slog.Error("Failed to start server 1", "error", err)
		os.Exit(1)
	}

	// Create configuration for the second server
	cfg2 := config.DefaultConfig()
	cfg2.HTTP.Port = 8081
	cfg2.ServerInfo.Name = "Server2"
	cfg2.BackwardCompatible20241105 = true
	err = os.Setenv("SERVER_NAME", "Server2")
	if err != nil {
		panic(err)
	}

	// Enable remote actors for clustering
	cfg2.Actor.UseRemoteActors = true
	cfg2.Actor.RemoteConfig.Host = "localhost"
	cfg2.Actor.RemoteConfig.Port = 9091

	cfg2.Clustering.NodeHost = "localhost"
	cfg2.Clustering.RemotingPort = 7011
	cfg2.Clustering.GossipPort = 8011
	cfg2.Clustering.PeersPort = 9091
	cfg2.Clustering.StaticHosts = []string{"localhost:8010", "localhost:8011"}
	cfg2.Clustering.Type = config.ClusteringTypeStatic

	// Use Redis for session management to share sessions between servers
	//cfg2.Redis = &config.RedisConfig{
	//	Addresses: []string{"localhost:6379"},
	//	Password:  "",
	//	DB:        0,
	//}
	//cfg2.Session.UseInMemory = false

	// Start the second server
	server2, err := startServer(ctx, cfg2)
	if err != nil {
		slog.Error("Failed to start server 2", "error", err)
		os.Exit(1)
	}

	slog.Info("Both servers started successfully")
	slog.Info("To test clustering:")
	slog.Info("1. Connect to Server1 at http://localhost:8080/mcp")
	slog.Info("2. Initialize a session and get a session ID")
	slog.Info("3. Use that same session ID to send requests to Server2 at http://localhost:8081/mcp")
	slog.Info("4. The session should be shared between both servers")

	// Wait for termination signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	// Shutdown the servers
	slog.Info("Shutting down servers...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	server1.Stop(shutdownCtx)
	server2.Stop(shutdownCtx)

	slog.Info("Servers stopped")
}
