package resources

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"text/template"
)

// StaticPromptRegistry is a registry that holds a fixed set of prompts
type StaticPromptRegistry struct {
	mu      sync.RWMutex
	prompts map[string]Prompt
}

// NewStaticPromptRegistry creates a new static prompt registry
func NewStaticPromptRegistry() *StaticPromptRegistry {
	return &StaticPromptRegistry{
		prompts: make(map[string]Prompt),
	}
}

// RegisterPrompt registers a prompt with the registry
func (r *StaticPromptRegistry) RegisterPrompt(prompt Prompt) error {
	if prompt.Name == "" {
		return fmt.Errorf("prompt name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.prompts[prompt.Name] = prompt
	slog.Info("Registered prompt", "name", prompt.Name)
	return nil
}

// GetPrompt returns a prompt by name
func (r *StaticPromptRegistry) GetPrompt(ctx context.Context, name string) (Prompt, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	prompt, ok := r.prompts[name]
	return prompt, ok
}

// ListPrompts returns a paginated list of prompts
func (r *StaticPromptRegistry) ListPrompts(ctx context.Context, opts PromptListOptions) PromptListResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get all prompt names and sort them for consistent pagination
	names := make([]string, 0, len(r.prompts))
	for name := range r.prompts {
		names = append(names, name)
	}
	sort.Strings(names)

	// Find the starting position based on cursor
	startPos := 0
	if opts.Cursor != "" {
		for i, name := range names {
			if name == opts.Cursor {
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

	// Extract the prompts for this page
	var result PromptListResult

	// No prompts or cursor beyond the end
	if startPos >= len(names) {
		return result
	}

	// Get the prompts for this page
	result.Prompts = make([]Prompt, 0, endPos-startPos)
	for i := startPos; i < endPos; i++ {
		result.Prompts = append(result.Prompts, r.prompts[names[i]])
	}

	// Set next cursor if there are more prompts
	if endPos < len(names) {
		result.NextCursor = names[endPos-1]
	}

	return result
}

// ProcessPrompt processes a prompt template with the given arguments
func (r *StaticPromptRegistry) ProcessPrompt(ctx context.Context, name string, arguments map[string]string) ([]PromptMessage, error) {
	r.mu.RLock()
	prompt, exists := r.prompts[name]
	r.mu.RUnlock()

	if !exists {
		return nil, ErrPromptNotFound
	}

	// Validate required arguments
	for _, arg := range prompt.Arguments {
		if arg.Required {
			if _, exists := arguments[arg.Name]; !exists {
				return nil, fmt.Errorf("%w: missing required argument %s", ErrInvalidParams, arg.Name)
			}
		}
	}

	// Process each message
	processedMessages := make([]PromptMessage, 0, len(prompt.Messages))
	for _, msg := range prompt.Messages {
		// Process content if it's a string
		if contentStr, ok := msg.Content.(string); ok {
			// Create a template
			tmpl, err := template.New(name).Parse(contentStr)
			if err != nil {
				return nil, fmt.Errorf("error parsing template: %w", err)
			}

			// Execute the template
			var buf strings.Builder
			if err := tmpl.Execute(&buf, arguments); err != nil {
				return nil, fmt.Errorf("error executing template: %w", err)
			}

			// Create a new message with the processed content
			processedMsg := PromptMessage{
				Role:    msg.Role,
				Content: buf.String(),
			}
			processedMessages = append(processedMessages, processedMsg)
		} else {
			// Non-string content, just copy the message
			processedMessages = append(processedMessages, msg)
		}
	}

	return processedMessages, nil
}

// Ensure StaticPromptRegistry implements PromptRegistry
var _ PromptRegistry = (*StaticPromptRegistry)(nil)
