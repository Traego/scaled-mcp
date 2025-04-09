package resources

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
)

// ResourceProvider defines a function that provides the contents of a resource
type ResourceProvider func(ctx context.Context, uri string) ([]ResourceContents, error)

// StaticResourceRegistry is a registry that holds a fixed set of resources
type StaticResourceRegistry struct {
	mu                sync.RWMutex
	resources         map[string]Resource
	resourceTemplates map[string]ResourceTemplate
	providers         map[string]ResourceProvider
	subscribers       map[string]map[string]bool // uri -> set of subscriber IDs
}

// NewStaticResourceRegistry creates a new static resource registry
func NewStaticResourceRegistry() *StaticResourceRegistry {
	return &StaticResourceRegistry{
		resources:         make(map[string]Resource),
		resourceTemplates: make(map[string]ResourceTemplate),
		providers:         make(map[string]ResourceProvider),
		subscribers:       make(map[string]map[string]bool),
	}
}

// RegisterResource registers a resource with the registry
func (r *StaticResourceRegistry) RegisterResource(resource Resource, provider ResourceProvider) error {
	if resource.URI == "" {
		return fmt.Errorf("resource URI cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.resources[resource.URI] = resource
	if provider != nil {
		r.providers[resource.URI] = provider
	}

	slog.Info("Registered resource", "uri", resource.URI, "name", resource.Name)
	return nil
}

// RegisterResourceTemplate registers a resource template with the registry
func (r *StaticResourceRegistry) RegisterResourceTemplate(template ResourceTemplate) error {
	if template.URITemplate == "" {
		return fmt.Errorf("resource template URI cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.resourceTemplates[template.URITemplate] = template
	slog.Info("Registered resource template", "uriTemplate", template.URITemplate, "name", template.Name)
	return nil
}

// ListResources returns a paginated list of resources
func (r *StaticResourceRegistry) ListResources(ctx context.Context, opts ResourceListOptions) ResourceListResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get all resource URIs and sort them for consistent pagination
	uris := make([]string, 0, len(r.resources))
	for uri := range r.resources {
		uris = append(uris, uri)
	}
	sort.Strings(uris)

	// Find the starting position based on cursor
	startPos := 0
	if opts.Cursor != "" {
		for i, uri := range uris {
			if uri == opts.Cursor {
				startPos = i + 1 // Start after the cursor
				break
			}
		}
	}

	// Calculate end position
	endPos := startPos + 20
	if endPos > len(uris) {
		endPos = len(uris)
	}

	// Extract the resources for this page
	var result ResourceListResult

	// No resources or cursor beyond the end
	if startPos >= len(uris) {
		return result
	}

	// Get the resources for this page
	result.Resources = make([]Resource, 0, endPos-startPos)
	for i := startPos; i < endPos; i++ {
		result.Resources = append(result.Resources, r.resources[uris[i]])
	}

	// Set next cursor if there are more resources
	if endPos < len(uris) {
		result.NextCursor = uris[endPos-1]
	}

	return result
}

// ReadResource reads a resource by URI
func (r *StaticResourceRegistry) ReadResource(ctx context.Context, uri string) ([]ResourceContents, error) {
	r.mu.RLock()
	provider, providerExists := r.providers[uri]
	_, resourceExists := r.resources[uri]
	r.mu.RUnlock()

	if !resourceExists {
		return nil, ErrResourceNotFound
	}

	if !providerExists {
		return nil, fmt.Errorf("no provider available for resource: %s", uri)
	}

	return provider(ctx, uri)
}

// SubscribeResource subscribes to updates for a resource
func (r *StaticResourceRegistry) SubscribeResource(ctx context.Context, uri string) error {
	// Extract subscriber ID from context
	subscriberID, ok := ctx.Value("subscriber_id").(string)
	if !ok || subscriberID == "" {
		subscriberID = "default" // Use a default ID if none provided
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if resource exists
	if _, exists := r.resources[uri]; !exists {
		return ErrResourceNotFound
	}

	// Create subscriber set if it doesn't exist
	if _, exists := r.subscribers[uri]; !exists {
		r.subscribers[uri] = make(map[string]bool)
	}

	// Add subscriber
	r.subscribers[uri][subscriberID] = true
	slog.Info("Subscribed to resource", "uri", uri, "subscriberID", subscriberID)
	return nil
}

// UnsubscribeResource unsubscribes from updates for a resource
func (r *StaticResourceRegistry) UnsubscribeResource(ctx context.Context, uri string) error {
	// Extract subscriber ID from context
	subscriberID, ok := ctx.Value("subscriber_id").(string)
	if !ok || subscriberID == "" {
		subscriberID = "default" // Use a default ID if none provided
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if resource exists
	if _, exists := r.resources[uri]; !exists {
		return ErrResourceNotFound
	}

	// Check if subscriber set exists
	subscribers, exists := r.subscribers[uri]
	if !exists {
		return nil // Nothing to unsubscribe
	}

	// Remove subscriber
	delete(subscribers, subscriberID)
	slog.Info("Unsubscribed from resource", "uri", uri, "subscriberID", subscriberID)
	return nil
}

// ListResourceTemplates returns a paginated list of resource templates
func (r *StaticResourceRegistry) ListResourceTemplates(ctx context.Context, opts ResourceTemplateListOptions) ResourceTemplateListResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get all template URIs and sort them for consistent pagination
	uris := make([]string, 0, len(r.resourceTemplates))
	for uri := range r.resourceTemplates {
		uris = append(uris, uri)
	}
	sort.Strings(uris)

	// Find the starting position based on cursor
	startPos := 0
	if opts.Cursor != "" {
		for i, uri := range uris {
			if uri == opts.Cursor {
				startPos = i + 1 // Start after the cursor
				break
			}
		}
	}

	// Calculate end position
	endPos := startPos + 20
	if endPos > len(uris) {
		endPos = len(uris)
	}

	// Extract the templates for this page
	var result ResourceTemplateListResult

	// No templates or cursor beyond the end
	if startPos >= len(uris) {
		return result
	}

	// Get the templates for this page
	result.ResourceTemplates = make([]ResourceTemplate, 0, endPos-startPos)
	for i := startPos; i < endPos; i++ {
		result.ResourceTemplates = append(result.ResourceTemplates, r.resourceTemplates[uris[i]])
	}

	// Set next cursor if there are more templates
	if endPos < len(uris) {
		result.NextCursor = uris[endPos-1]
	}

	return result
}

// GetSubscribers returns the subscribers for a resource
func (r *StaticResourceRegistry) GetSubscribers(uri string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	subscribers, exists := r.subscribers[uri]
	if !exists {
		return nil
	}

	result := make([]string, 0, len(subscribers))
	for id := range subscribers {
		result = append(result, id)
	}
	return result
}

// Ensure StaticResourceRegistry implements ResourceRegistry
var _ ResourceRegistry = (*StaticResourceRegistry)(nil)
