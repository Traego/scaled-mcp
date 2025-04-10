package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/resources"
	"github.com/traego/scaled-mcp/pkg/server"
)

func main() {
	ctx := context.Background()
	// Configure logging
	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(logHandler))

	// Create a custom tool provider
	toolProvider := NewExampleToolProvider()

	// Create a dynamic tool registry with the provider
	registry := resources.NewDynamicToolRegistry(toolProvider)

	// Create server with the dynamic tool registry
	cfg := config.DefaultConfig()
	mcpServer, err := server.NewMcpServer(cfg,
		server.WithToolRegistry(registry),
	)
	if err != nil {
		slog.Error("Failed to create MCP server", "error", err)
		os.Exit(1)
	}

	// Start the server
	go func() {
		if err := mcpServer.Start(ctx); err != nil && err != http.ErrServerClosed {
			slog.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("Server started", "host", cfg.HTTP.Host, "port", cfg.HTTP.Port)
	slog.Info("Example dynamic tool registry is available")
	slog.Info("Tools available: weather, calculator")

	// Wait for termination signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	mcpServer.Stop(ctx)
}

// ExampleToolProvider implements the resources.ToolProvider interface
type ExampleToolProvider struct {
	tools map[string]resources.Tool
}

// NewExampleToolProvider creates a new example tool provider
func NewExampleToolProvider() *ExampleToolProvider {
	provider := &ExampleToolProvider{
		tools: make(map[string]resources.Tool),
	}

	// Register some example tools
	provider.tools["weather"] = resources.NewTool("weather").
		WithDescription("Get weather information for a location").
		WithInputs([]resources.ToolInput{
			{
				Name:        "location",
				Type:        "string",
				Description: "The location to get weather for",
				Required:    true,
			},
			{
				Name:        "units",
				Type:        "string",
				Description: "Temperature units (celsius or fahrenheit)",
				Default:     "celsius",
			},
		}).
		Build()

	provider.tools["calculator"] = resources.NewTool("calculator").
		WithDescription("Perform a calculation").
		WithInputs([]resources.ToolInput{
			{
				Name:        "operation",
				Type:        "string",
				Description: "The operation to perform (add, subtract, multiply, divide)",
				Required:    true,
			},
			{
				Name:        "a",
				Type:        "integer",
				Description: "First operand",
				Required:    true,
			},
			{
				Name:        "b",
				Type:        "integer",
				Description: "Second operand",
				Required:    true,
			},
		}).
		Build()

	return provider
}

// GetTool returns a tool by name
func (p *ExampleToolProvider) GetTool(ctx context.Context, name string) (resources.Tool, bool) {
	tool, found := p.tools[name]
	return tool, found
}

// ListTools returns a list of available tools
func (p *ExampleToolProvider) ListTools(ctx context.Context, cursor string) ([]resources.Tool, string) {
	// Get all tool names and sort them
	names := make([]string, 0, len(p.tools))
	for name := range p.tools {
		names = append(names, name)
	}

	// Find the starting position based on cursor
	startPos := 0
	if cursor != "" {
		for i, name := range names {
			if name == cursor {
				startPos = i + 1 // Start after the cursor
				break
			}
		}
	}

	// Calculate end position
	endPos := startPos + 20
	if endPos > len(names) {
		endPos = len(names)
	}

	// No tools or cursor beyond the end
	if startPos >= len(names) {
		return []resources.Tool{}, ""
	}

	// Get the tools for this page
	result := make([]resources.Tool, 0, endPos-startPos)
	for i := startPos; i < endPos; i++ {
		result = append(result, p.tools[names[i]])
	}

	// Set next cursor if there are more tools
	nextCursor := ""
	if endPos < len(names) {
		nextCursor = names[endPos-1]
	}

	return result, nextCursor
}

// HandleToolInvocation handles a tool invocation
func (p *ExampleToolProvider) HandleToolInvocation(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
	tool, found := p.tools[name]
	if !found {
		return nil, resources.ErrToolNotFound
	}

	// Validate required parameters
	for _, paramName := range tool.InputSchema.Required {
		if _, exists := params[paramName]; !exists {
			return nil, fmt.Errorf("%w: missing required parameter %s", resources.ErrInvalidParams, paramName)
		}
	}

	// Handle the tool invocation based on the name
	switch name {
	case "weather":
		return handleWeatherTool(params)
	case "calculator":
		return handleCalculatorTool(params)
	default:
		return nil, fmt.Errorf("tool handler not implemented for %s", name)
	}
}

// handleWeatherTool handles the weather tool invocation
func handleWeatherTool(params map[string]interface{}) (interface{}, error) {
	location, _ := params["location"].(string)
	units, _ := params["units"].(string)

	if units == "" {
		units = "celsius" // Default
	}

	// In a real implementation, you would call a weather API here
	// For this example, we'll just return mock data
	return map[string]interface{}{
		"location":    location,
		"temperature": 22,
		"units":       units,
		"conditions":  "Sunny",
		"humidity":    45,
	}, nil
}

// handleCalculatorTool handles the calculator tool invocation
func handleCalculatorTool(params map[string]interface{}) (interface{}, error) {
	operation, _ := params["operation"].(string)

	// Convert parameters to integers
	aFloat, ok1 := params["a"].(float64)
	bFloat, ok2 := params["b"].(float64)

	if !ok1 || !ok2 {
		return nil, fmt.Errorf("%w: parameters 'a' and 'b' must be numbers", resources.ErrInvalidParams)
	}

	a, b := int(aFloat), int(bFloat)

	var result int
	switch operation {
	case "add":
		result = a + b
	case "subtract":
		result = a - b
	case "multiply":
		result = a * b
	case "divide":
		if b == 0 {
			return nil, fmt.Errorf("%w: division by zero", resources.ErrInvalidParams)
		}
		result = a / b
	default:
		return nil, fmt.Errorf("%w: invalid operation '%s'", resources.ErrInvalidParams, operation)
	}

	return map[string]interface{}{
		"operation": operation,
		"a":         a,
		"b":         b,
		"result":    result,
	}, nil
}
