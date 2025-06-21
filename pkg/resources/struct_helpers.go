package resources

import (
	"context"
	"reflect"
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
