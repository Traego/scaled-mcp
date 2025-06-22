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

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Configure logging
	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(logHandler))

	// Create registries
	toolRegistry := resources.NewStaticToolRegistry()
	promptRegistry := resources.NewStaticPromptRegistry()
	resourceRegistry := resources.NewStaticResourceRegistry()

	// Register tools with the server
	registerEchoTool(toolRegistry)
	registerGreetingTool(toolRegistry)

	// Register prompts with the server
	registerWelcomePrompt(promptRegistry)
	registerCodeExamplePrompt(promptRegistry)

	// Register resources with the server
	registerTextResource(resourceRegistry)
	registerCodeResource(resourceRegistry)
	registerResourceTemplates(resourceRegistry)

	// Create server with the registries
	cfg := config.DefaultConfig()
	cfg.BackwardCompatible20241105 = true
	cfg.HTTP.Port = 9985
	cfg.HTTP.CORS.Enable = true
	cfg.HTTP.CORS.AllowCredentials = true
	cfg.HTTP.CORS.AllowedOrigins = []string{"*"}

	mcpServer, err := server.NewMcpServer(cfg,
		server.WithToolRegistry(toolRegistry),
		server.WithPromptRegistry(promptRegistry),
		server.WithResourceRegistry(resourceRegistry),
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

	slog.Info("Example static registries server is available")
	slog.Info("Tools available: echo, greeting")
	slog.Info("Prompts available: welcome, code-client_example")
	slog.Info("Resources available: sample-text, sample-code")
	slog.Info("Resource templates available: file/{path}, api/{endpoint}, docs/{topic}")
	slog.Info("Template resources available: file/client_example.txt, api/users, docs/getting-started")
	slog.Info("Utilities available: ping")

	// Wait for termination signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	sctx, c2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer c2()

	// Shutdown the server
	mcpServer.Stop(sctx)
}

// registerEchoTool registers an echo tool with the server
func registerEchoTool(registry *resources.StaticToolRegistry) {
	// Create the echo tool
	echoTool := resources.NewTool("echo").
		WithDescription("Echo back the input message").
		WithInputs([]resources.ToolInput{
			{
				Name:        "message",
				Type:        "string",
				Description: "The message to echo back",
				Required:    true,
			},
		}).
		Build()

	// Register the tool with its handler
	err := registry.RegisterTool(echoTool, func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		message, ok := params["message"].(string)
		if !ok {
			return nil, fmt.Errorf("message parameter must be a string")
		}

		return map[string]interface{}{
			"echo": message,
		}, nil
	})

	if err != nil {
		slog.Error("Failed to register echo tool", "error", err)
	}
}

// registerGreetingTool registers a greeting tool with the server
func registerGreetingTool(registry *resources.StaticToolRegistry) {
	// Create the greeting tool
	greetingTool := resources.NewTool("greeting").
		WithDescription("Generate a greeting for a person").
		WithInputs([]resources.ToolInput{
			{
				Name:        "name",
				Type:        "string",
				Description: "The name of the person to greet",
				Required:    true,
			},
			{
				Name:        "language",
				Type:        "string",
				Description: "The language for the greeting (en, es, fr)",
				Default:     "en",
			},
		}).
		Build()

	// Register the tool with its handler
	err := registry.RegisterTool(greetingTool, func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		name, ok := params["name"].(string)
		if !ok {
			return nil, fmt.Errorf("name parameter must be a string")
		}

		language, _ := params["language"].(string)
		if language == "" {
			language = "en" // Default
		}

		var greeting string
		switch language {
		case "en":
			greeting = fmt.Sprintf("Hello, %s!", name)
		case "es":
			greeting = fmt.Sprintf("¡Hola, %s!", name)
		case "fr":
			greeting = fmt.Sprintf("Bonjour, %s!", name)
		default:
			greeting = fmt.Sprintf("Hello, %s!", name)
		}

		return map[string]interface{}{
			"greeting": greeting,
			"language": language,
		}, nil
	})

	if err != nil {
		slog.Error("Failed to register greeting tool", "error", err)
	}
}

// registerWelcomePrompt registers a welcome prompt with the server
func registerWelcomePrompt(registry *resources.StaticPromptRegistry) {
	// Create the welcome prompt
	welcomePrompt := resources.NewPrompt("welcome").
		WithDescription("A customizable welcome message").
		WithArgument("name").
		Required().
		Description("The name of the person to welcome").
		Add().
		WithArgument("project").
		Description("The project name").
		Add().
		WithUserMessage("I'd like to get started with {{.project}}").
		WithAssistantMessage("Welcome to {{.project}}, {{.name}}! How can I help you today?").
		Build()

	// Register the prompt
	err := registry.RegisterPrompt(welcomePrompt)
	if err != nil {
		slog.Error("Failed to register welcome prompt", "error", err)
	}
}

// registerCodeExamplePrompt registers a code client_example prompt with the server
func registerCodeExamplePrompt(registry *resources.StaticPromptRegistry) {
	// Create the code client_example prompt
	codePrompt := resources.NewPrompt("code-client_example").
		WithDescription("Provides a code client_example in a specified language").
		WithArgument("language").
		Required().
		Description("The programming language for the client_example").
		Add().
		WithArgument("concept").
		Required().
		Description("The programming concept to demonstrate").
		Add().
		WithUserMessage("Show me a {{.language}} client_example of {{.concept}}").
		WithAssistantMessage("Here's an client_example of {{.concept}} in {{.language}}:").
		Build()

	// Register the prompt
	err := registry.RegisterPrompt(codePrompt)
	if err != nil {
		slog.Error("Failed to register code client_example prompt", "error", err)
	}
}

// registerTextResource registers a simple text resource with the server
func registerTextResource(registry *resources.StaticResourceRegistry) {
	// Create the text resource
	textResource := resources.NewResource("sample-text", "Sample Text Resource").
		WithDescription("A sample text resource for demonstration").
		WithMimeType("text/plain").
		WithSize(int64(len("This is a sample text resource for the MCP server."))).
		Build()

	// Register the resource with its provider
	err := registry.RegisterResource(textResource, func(ctx context.Context, uri string) ([]resources.ResourceContents, error) {
		return []resources.ResourceContents{
			&resources.ResourceContentText{
				URI:      uri,
				MimeType: "text/plain",
				Text:     "This is a sample text resource for the MCP server.",
			},
		}, nil
	})

	if err != nil {
		slog.Error("Failed to register text resource", "error", err)
	}
}

// registerCodeResource registers a code resource with the server
func registerCodeResource(registry *resources.StaticResourceRegistry) {
	// Sample Go code
	sampleCode := `package main

import (
	"fmt"
)

func main() {
	fmt.Println("Hello, MCP!")
}
`

	// Create the code resource
	codeResource := resources.NewResource("sample-code", "Sample Code Resource").
		WithDescription("A sample Go code resource").
		WithMimeType("text/x-go").
		WithSize(int64(len(sampleCode))).
		Build()

	// Register the resource with its provider
	err := registry.RegisterResource(codeResource, func(ctx context.Context, uri string) ([]resources.ResourceContents, error) {
		return []resources.ResourceContents{
			&resources.ResourceContentText{
				URI:      uri,
				MimeType: "text/x-go",
				Text:     sampleCode,
			},
		}, nil
	})

	if err != nil {
		slog.Error("Failed to register code resource", "error", err)
	}
}

// registerResourceTemplates registers resource templates with the server
func registerResourceTemplates(registry *resources.StaticResourceRegistry) {
	// Register a file resource template
	fileTemplate := resources.NewResourceTemplate("file/{path}", "File Resource Template").
		WithDescription("Access files via path parameter").
		WithMimeType("application/octet-stream").
		Build()

	// Register the template
	err := registry.RegisterResourceTemplate(fileTemplate)
	if err != nil {
		slog.Error("Failed to register file resource template", "error", err)
	}

	// Create and register a sample dynamic file resource that follows the template pattern
	fileResource := resources.NewResource("file/client_example.txt", "Example Text File").
		WithDescription("An client_example text file resource").
		WithMimeType("text/plain").
		WithSize(int64(len("This is an client_example text file content."))).
		Build()

	// Register the resource with its provider
	err = registry.RegisterResource(fileResource, func(ctx context.Context, uri string) ([]resources.ResourceContents, error) {
		return []resources.ResourceContents{
			&resources.ResourceContentText{
				URI:      uri,
				MimeType: "text/plain",
				Text:     "This is an client_example text file content.",
			},
		}, nil
	})
	if err != nil {
		slog.Error("Failed to register file resource", "error", err)
	}

	// Register an API resource template
	apiTemplate := resources.NewResourceTemplate("api/{endpoint}", "API Resource Template").
		WithDescription("Access API endpoints via endpoint parameter").
		WithMimeType("application/json").
		Build()

	err = registry.RegisterResourceTemplate(apiTemplate)
	if err != nil {
		slog.Error("Failed to register API resource template", "error", err)
	}

	// Create and register a sample API resource that follows the template pattern
	apiResource := resources.NewResource("api/users", "Users API").
		WithDescription("API endpoint for user data").
		WithMimeType("application/json").
		WithSize(int64(len(`{"users": [{"id": 1, "name": "John Doe"}, {"id": 2, "name": "Jane Smith"}]}`))).
		Build()

	// Register the resource with its provider
	err = registry.RegisterResource(apiResource, func(ctx context.Context, uri string) ([]resources.ResourceContents, error) {
		return []resources.ResourceContents{
			&resources.ResourceContentText{
				URI:      uri,
				MimeType: "application/json",
				Text:     `{"users": [{"id": 1, "name": "John Doe"}, {"id": 2, "name": "Jane Smith"}]}`,
			},
		}, nil
	})
	if err != nil {
		slog.Error("Failed to register API resource", "error", err)
	}

	// Register a documentation resource template
	docsTemplate := resources.NewResourceTemplate("docs/{topic}", "Documentation Resource Template").
		WithDescription("Access documentation by topic").
		WithMimeType("text/markdown").
		Build()

	err = registry.RegisterResourceTemplate(docsTemplate)
	if err != nil {
		slog.Error("Failed to register documentation resource template", "error", err)
	}

	// Create and register a sample documentation resource that follows the template pattern
	docsResource := resources.NewResource("docs/getting-started", "Getting Started Guide").
		WithDescription("Documentation for getting started with the MCP server").
		WithMimeType("text/markdown").
		WithSize(int64(len("# Getting Started with MCP Server\n\nThis guide will help you get started with the MCP server.\n\n## Installation\n\n```go\ngo get github.com/traego/scaled-mcp\n```\n\n## Basic Usage\n\nCreate a new server with default configuration:\n\n```go\nserver, err := server.NewMcpServer(config.DefaultConfig())\n```"))).
		Build()

	// Register the resource with its provider
	err = registry.RegisterResource(docsResource, func(ctx context.Context, uri string) ([]resources.ResourceContents, error) {
		return []resources.ResourceContents{
			&resources.ResourceContentText{
				URI:      uri,
				MimeType: "text/markdown",
				Text:     "# Getting Started with MCP Server\n\nThis guide will help you get started with the MCP server.\n\n## Installation\n\n```go\ngo get github.com/traego/scaled-mcp\n```\n\n## Basic Usage\n\nCreate a new server with default configuration:\n\n```go\nserver, err := server.NewMcpServer(config.DefaultConfig())\n```",
			},
		}, nil
	})
	if err != nil {
		slog.Error("Failed to register documentation resource", "error", err)
	}

	slog.Info("Registered resource templates", "templates", []string{"file/{path}", "api/{endpoint}", "docs/{topic}"})
	slog.Info("Registered template resources", "resources", []string{"file/client_example.txt", "api/users", "docs/getting-started"})
}
