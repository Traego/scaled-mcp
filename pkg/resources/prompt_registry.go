package resources

import (
	"context"
	"errors"
)

// Common errors
var (
	ErrPromptNotFound = errors.New("prompt not found")
)

// PromptMessage represents a message in a prompt
type PromptMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// Prompt represents an MCP prompt definition
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
	Messages    []PromptMessage  `json:"messages,omitempty"`
}

// PromptArgument represents an argument for a prompt template
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
}

// PromptListOptions provides pagination options for listing prompts
type PromptListOptions struct {
	Cursor string // Cursor for pagination
}

// PromptListResult represents a paginated list of prompts
type PromptListResult struct {
	Prompts    []Prompt `json:"prompts"`
	NextCursor string   `json:"nextCursor,omitempty"` // Cursor for the next page, empty if no more pages
}

// PromptRegistry defines the interface for a prompt registry
type PromptRegistry interface {
	// GetPrompt returns a prompt by name
	GetPrompt(ctx context.Context, name string) (Prompt, bool)

	// ListPrompts returns a paginated list of prompts
	ListPrompts(ctx context.Context, opts PromptListOptions) PromptListResult

	// ProcessPrompt processes a prompt template with the given arguments
	ProcessPrompt(ctx context.Context, name string, arguments map[string]string) ([]PromptMessage, error)
}
