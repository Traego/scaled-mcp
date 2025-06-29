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

type CalculatorInput struct {
	Operation string  `mcp:"operation,The operation to perform (add subtract multiply divide),required"`
	A         float64 `mcp:"a,First operand,required"`
	B         float64 `mcp:"b,Second operand,required"`
}

type CalculatorOutput struct {
	Result    float64 `mcp:"result,The calculation result,required"`
	Operation string  `mcp:"operation,The operation performed,required"`
	A         float64 `mcp:"a,First operand,required"`
	B         float64 `mcp:"b,Second operand,required"`
}

type GreetingInput struct {
	Name     string `mcp:"name,The name of the person to greet,required"`
	Language string `mcp:"language,The language for the greeting (en es fr),default=en"`
	Formal   bool   `mcp:"formal,Whether to use formal greeting"`
}

type GreetingOutput struct {
	Greeting string `mcp:"greeting,The generated greeting,required"`
	Language string `mcp:"language,The language used,required"`
	Formal   bool   `mcp:"formal,Whether formal greeting was used,required"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(logHandler))

	toolRegistry := resources.NewStaticToolRegistry()

	if err := registerGenericsTools(toolRegistry); err != nil {
		slog.Error("Failed to register generics tools", "error", err)
		os.Exit(1)
	}

	cfg := config.DefaultConfig()
	cfg.BackwardCompatible20241105 = true
	cfg.HTTP.Port = 9988
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

	slog.Info("Generics-based tools server is available on port 9988")
	slog.Info("Tools available: calculator_typed, greeting_typed")
	slog.Info("All tools use generics-based registration with compile-time type safety")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	sctx, c2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer c2()

	mcpServer.Stop(sctx)
}

func registerGenericsTools(registry *resources.StaticToolRegistry) error {
	calculatorHandler := func(ctx context.Context, input *CalculatorInput) (*CalculatorOutput, error) {
		var result float64
		switch input.Operation {
		case "add":
			result = input.A + input.B
		case "subtract":
			result = input.A - input.B
		case "multiply":
			result = input.A * input.B
		case "divide":
			if input.B == 0 {
				return nil, fmt.Errorf("%w: division by zero", resources.ErrInvalidParams)
			}
			result = input.A / input.B
		default:
			return nil, fmt.Errorf("%w: unknown operation %s", resources.ErrInvalidParams, input.Operation)
		}

		return &CalculatorOutput{
			Result:    result,
			Operation: input.Operation,
			A:         input.A,
			B:         input.B,
		}, nil
	}

	greetingHandler := func(ctx context.Context, input *GreetingInput) (*GreetingOutput, error) {
		var greeting string

		if input.Formal {
			switch input.Language {
			case "en":
				greeting = fmt.Sprintf("Good day, %s", input.Name)
			case "es":
				greeting = fmt.Sprintf("Buenos días, %s", input.Name)
			case "fr":
				greeting = fmt.Sprintf("Bonjour, %s", input.Name)
			default:
				greeting = fmt.Sprintf("Good day, %s", input.Name)
			}
		} else {
			switch input.Language {
			case "en":
				greeting = fmt.Sprintf("Hello, %s!", input.Name)
			case "es":
				greeting = fmt.Sprintf("¡Hola, %s!", input.Name)
			case "fr":
				greeting = fmt.Sprintf("Salut, %s!", input.Name)
			default:
				greeting = fmt.Sprintf("Hello, %s!", input.Name)
			}
		}

		return &GreetingOutput{
			Greeting: greeting,
			Language: input.Language,
			Formal:   input.Formal,
		}, nil
	}

	if err := resources.RegisterTool(registry, "calculator_typed", "Performs arithmetic with typed output", calculatorHandler); err != nil {
		return fmt.Errorf("failed to register typed calculator tool: %w", err)
	}

	if err := resources.RegisterTool(registry, "greeting_typed", "Generate a greeting with typed output", greetingHandler); err != nil {
		return fmt.Errorf("failed to register typed greeting tool: %w", err)
	}

	return nil
}
