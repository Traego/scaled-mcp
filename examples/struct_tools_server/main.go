package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"reflect"
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

type GreetingInput struct {
	Name     string `mcp:"name,The name of the person to greet,required"`
	Language string `mcp:"language,The language for the greeting (en es fr),default=en"`
	Formal   bool   `mcp:"formal,Whether to use formal greeting"`
}

type WeatherInput struct {
	Location string `mcp:"location,The location to get weather for,required"`
	Units    string `mcp:"units,Temperature units (celsius fahrenheit),default=celsius"`
	Days     *int   `mcp:"days,Number of forecast days (1-7)"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(logHandler))

	toolRegistry := resources.NewStaticToolRegistry()

	if err := registerStructTools(toolRegistry); err != nil {
		slog.Error("Failed to register struct tools", "error", err)
		os.Exit(1)
	}

	cfg := config.DefaultConfig()
	cfg.BackwardCompatible20241105 = true
	cfg.HTTP.Port = 9986
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

	slog.Info("Struct-based tools server is available")
	slog.Info("Tools available: calculator, greeting, weather")
	slog.Info("All tools use struct-based reflection for schema generation")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	sctx, c2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer c2()

	mcpServer.Stop(sctx)
}

func registerStructTools(registry *resources.StaticToolRegistry) error {
	calculatorType := reflect.TypeOf(CalculatorInput{})
	if err := registry.RegisterStructToolWithHandler("calculator", "Performs basic arithmetic operations", calculatorType, calculatorHandler); err != nil {
		return fmt.Errorf("failed to register calculator tool: %w", err)
	}

	greetingType := reflect.TypeOf(GreetingInput{})
	if err := registry.RegisterStructToolWithHandler("greeting", "Generate a greeting for a person", greetingType, greetingHandler); err != nil {
		return fmt.Errorf("failed to register greeting tool: %w", err)
	}

	weatherType := reflect.TypeOf(WeatherInput{})
	if err := registry.RegisterStructToolWithHandler("weather", "Get weather information for a location", weatherType, weatherHandler); err != nil {
		return fmt.Errorf("failed to register weather tool: %w", err)
	}

	return nil
}

func calculatorHandler(ctx context.Context, input *CalculatorInput) (interface{}, error) {
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

	return map[string]interface{}{
		"operation": input.Operation,
		"a":         input.A,
		"b":         input.B,
		"result":    result,
	}, nil
}

func greetingHandler(ctx context.Context, input *GreetingInput) (interface{}, error) {
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

	return map[string]interface{}{
		"greeting": greeting,
		"language": input.Language,
		"formal":   input.Formal,
	}, nil
}

func weatherHandler(ctx context.Context, input *WeatherInput) (interface{}, error) {
	temperature := 22.0
	if input.Units == "fahrenheit" {
		temperature = 71.6
	}

	result := map[string]interface{}{
		"location":    input.Location,
		"temperature": temperature,
		"units":       input.Units,
		"conditions":  "Sunny",
		"humidity":    45,
	}

	if input.Days != nil && *input.Days > 1 {
		forecast := make([]map[string]interface{}, *input.Days)
		for i := 0; i < *input.Days; i++ {
			dayTemp := temperature + float64(i-1)*2
			forecast[i] = map[string]interface{}{
				"day":         i + 1,
				"temperature": dayTemp,
				"conditions":  "Partly cloudy",
			}
		}
		result["forecast"] = forecast
	}

	return result, nil
}
