package resources

import "github.com/traego/scaled-mcp/pkg/protocol"

// ToolBuilder is a builder for creating resources
type ToolBuilder struct {
	tool protocol.Tool
}

// ParameterBuilder is a builder for creating tool parameters
type ParameterBuilder struct {
	name     string
	property protocol.SchemaProperty
	tool     *ToolBuilder
}

// ToolInput represents a single input parameter for a tool
type ToolInput struct {
	Name        string
	Type        string
	Description string
	Required    bool
	Default     interface{}
}

// NewTool creates a new tool builder
func NewTool(name string) *ToolBuilder {
	return &ToolBuilder{
		tool: protocol.Tool{
			Name: name,
			InputSchema: protocol.InputSchema{
				Type:       "object",
				Properties: make(map[string]protocol.SchemaProperty),
				Required:   []string{},
			},
		},
	}
}

// WithDescription sets the description of the tool
func (b *ToolBuilder) WithDescription(description string) *ToolBuilder {
	b.tool.Description = description
	return b
}

// WithInputs adds multiple input parameters to the tool at once
func (b *ToolBuilder) WithInputs(inputs []ToolInput) *ToolBuilder {
	for _, input := range inputs {
		property := protocol.SchemaProperty{
			Type:        input.Type,
			Description: input.Description,
		}

		if input.Default != nil {
			property.Default = input.Default
		}

		b.tool.InputSchema.Properties[input.Name] = property

		if input.Required {
			b.tool.InputSchema.Required = append(b.tool.InputSchema.Required, input.Name)
		}
	}

	return b
}

// WithString adds a string parameter to the tool
func (b *ToolBuilder) WithString(name string) *ParameterBuilder {
	return &ParameterBuilder{
		name: name,
		property: protocol.SchemaProperty{
			Type: "string",
		},
		tool: b,
	}
}

// WithInteger adds an integer parameter to the tool
func (b *ToolBuilder) WithInteger(name string) *ParameterBuilder {
	return &ParameterBuilder{
		name: name,
		property: protocol.SchemaProperty{
			Type: "integer",
		},
		tool: b,
	}
}

// WithBoolean adds a boolean parameter to the tool
func (b *ToolBuilder) WithBoolean(name string) *ParameterBuilder {
	return &ParameterBuilder{
		name: name,
		property: protocol.SchemaProperty{
			Type: "boolean",
		},
		tool: b,
	}
}

// WithObject adds an object parameter to the tool
func (b *ToolBuilder) WithObject(name string) *ParameterBuilder {
	return &ParameterBuilder{
		name: name,
		property: protocol.SchemaProperty{
			Type: "object",
		},
		tool: b,
	}
}

// WithArray adds an array parameter to the tool
func (b *ToolBuilder) WithArray(name string) *ParameterBuilder {
	return &ParameterBuilder{
		name: name,
		property: protocol.SchemaProperty{
			Type: "array",
		},
		tool: b,
	}
}

// Build builds the tool
func (b *ToolBuilder) Build() protocol.Tool {
	return b.tool
}

// Required marks the parameter as required
func (b *ParameterBuilder) Required() *ParameterBuilder {
	b.tool.tool.InputSchema.Required = append(b.tool.tool.InputSchema.Required, b.name)
	return b
}

// Description sets the description of the parameter
func (b *ParameterBuilder) Description(description string) *ParameterBuilder {
	b.property.Description = description
	return b
}

// Default sets the default value of the parameter
func (b *ParameterBuilder) Default(value interface{}) *ParameterBuilder {
	b.property.Default = value
	return b
}

// Add adds the parameter to the tool and returns the tool builder
func (b *ParameterBuilder) Add() *ToolBuilder {
	b.tool.tool.InputSchema.Properties[b.name] = b.property
	return b.tool
}
