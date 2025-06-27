package resources

import (
	"context"
	"fmt"
	"reflect"

	"github.com/traego/scaled-mcp/pkg/protocol"
)

func RegisterStructTool[T any](registry *StaticToolRegistry, name, description string, handler func(ctx context.Context, input *T) (interface{}, error)) error {
	structType := reflect.TypeOf((*T)(nil)).Elem()
	return registry.RegisterStructToolWithHandler(name, description, structType, handler)
}

func MustRegisterStructTool[T any](registry *StaticToolRegistry, name, description string, handler func(ctx context.Context, input *T) (interface{}, error)) {
	if err := RegisterStructTool(registry, name, description, handler); err != nil {
		panic(err)
	}
}

// RegisterStructToolWithTypes registers a tool with typed input and output using generics
func RegisterStructToolWithTypes[TInput, TOutput any](
	registry *StaticToolRegistry,
	name, description string,
	handler func(ctx context.Context, input *TInput) (*TOutput, error),
) error {
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	inputType := reflect.TypeOf((*TInput)(nil)).Elem()
	inputSchema, err := GenerateSchemaFromStruct(inputType)
	if err != nil {
		return fmt.Errorf("failed to generate input schema from struct: %w", err)
	}

	outputType := reflect.TypeOf((*TOutput)(nil)).Elem()
	outputSchema, err := GenerateSchemaFromStruct(outputType)
	if err != nil {
		return fmt.Errorf("failed to generate output schema from struct: %w", err)
	}

	tool := protocol.Tool{
		Name:         name,
		Description:  description,
		InputSchema:  inputSchema,
		OutputSchema: &outputSchema,
	}

	wrappedHandler := func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		inputPtr := reflect.New(inputType)
		if err := UnmarshalParams(params, inputPtr.Interface()); err != nil {
			return nil, fmt.Errorf("%w: failed to unmarshal parameters: %v", ErrInvalidParams, err)
		}

		result, err := handler(ctx, inputPtr.Interface().(*TInput))
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	return registry.RegisterTool(tool, wrappedHandler)
}

func MustRegisterStructToolWithTypes[TInput, TOutput any](
	registry *StaticToolRegistry,
	name, description string,
	handler func(ctx context.Context, input *TInput) (*TOutput, error),
) {
	if err := RegisterStructToolWithTypes(registry, name, description, handler); err != nil {
		panic(err)
	}
}
