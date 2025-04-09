package resources

import (
	"context"
	"log/slog"
)

// PromptProvider defines the interface for dynamically providing prompts
type PromptProvider interface {
	// GetPrompt returns a prompt by name
	GetPrompt(ctx context.Context, name string) (Prompt, bool)

	// ListPrompts returns a list of available prompts
	ListPrompts(ctx context.Context, cursor string) ([]Prompt, string)

	// ProcessPrompt processes a prompt template with the given arguments
	ProcessPrompt(ctx context.Context, name string, arguments map[string]string) ([]PromptMessage, error)
}

// DynamicPromptRegistry is a registry that dynamically provides prompts through a provider
type DynamicPromptRegistry struct {
	provider PromptProvider
}

// NewDynamicPromptRegistry creates a new dynamic prompt registry
func NewDynamicPromptRegistry(provider PromptProvider) *DynamicPromptRegistry {
	return &DynamicPromptRegistry{
		provider: provider,
	}
}

// GetPrompt returns a prompt by name
func (r *DynamicPromptRegistry) GetPrompt(ctx context.Context, name string) (Prompt, bool) {
	prompt, found := r.provider.GetPrompt(ctx, name)
	if !found {
		slog.Debug("Prompt not found", "name", name)
	}
	return prompt, found
}

// ListPrompts returns a paginated list of prompts
func (r *DynamicPromptRegistry) ListPrompts(ctx context.Context, opts PromptListOptions) PromptListResult {
	prompts, nextCursor := r.provider.ListPrompts(ctx, opts.Cursor)

	return PromptListResult{
		Prompts:    prompts,
		NextCursor: nextCursor,
	}
}

// ProcessPrompt processes a prompt template with the given arguments
func (r *DynamicPromptRegistry) ProcessPrompt(ctx context.Context, name string, arguments map[string]string) ([]PromptMessage, error) {
	return r.provider.ProcessPrompt(ctx, name, arguments)
}

// Ensure DynamicPromptRegistry implements PromptRegistry
var _ PromptRegistry = (*DynamicPromptRegistry)(nil)
