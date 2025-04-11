// Package main provides an client_example of using the MCP client.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/traego/scaled-mcp/pkg/client"
	"github.com/traego/scaled-mcp/pkg/protocol"
)

func main() {
	// Set up logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("Received shutdown signal")
		cancel()
	}()

	// Server URL - replace with your MCP server URL
	serverURL := "http://localhost:8080"

	// Create client options
	options := client.DefaultClientOptions()

	// Use auto protocol version detection to support both 2024 and 2025 specs
	options.ProtocolVersion = protocol.ProtocolVersionAuto

	// Set client info
	options.ClientInfo = client.ClientInfo{
		Name:    "client_example-client",
		Version: "1.0.0",
	}

	// Create a new MCP client
	mcpClient, err := client.NewMcpClient(serverURL, options)
	if err != nil {
		slog.Error("Failed to create MCP client", "error", err)
		os.Exit(1)
	}

	// Add an event handler for server-sent events
	mcpClient.AddEventHandler(client.EventHandlerFunc(func(event *protocol.JSONRPCMessage) {
		slog.Info("Received event", "method", event.Method)
	}))

	// Connect to the server
	slog.Info("Connecting to MCP server...")
	if err := mcpClient.Connect(ctx); err != nil {
		slog.Error("Failed to connect to MCP server", "error", err)
		os.Exit(1)
	}

	defer func() {
		_ = mcpClient.Close(ctx)
	}()

	slog.Info("MCP client connected successfully", "sessionID", mcpClient.GetSessionID())

	// Example: List roots
	slog.Info("Listing roots")
	resp, err := mcpClient.SendRequest(ctx, "roots/list", nil)
	if err != nil {
		slog.Error("Failed to list roots", "error", err)
	} else {
		slog.Info("Roots listed successfully", "response", resp)
	}

	// Example: Send a notification
	slog.Info("Sending notification")
	err = mcpClient.SendNotification(ctx, "notifications/test", map[string]interface{}{
		"message": "test notification",
	})
	if err != nil {
		slog.Error("Failed to send notification", "error", err)
	} else {
		slog.Info("Notification sent successfully")
	}

	// Keep the program running until context is cancelled
	<-ctx.Done()
	slog.Info("Shutting down")
}
