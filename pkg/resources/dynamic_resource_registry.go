package resources

import (
	"context"
	"log/slog"
)

// ResourceProvider defines the interface for dynamically providing resources
type ResourceProviderInterface interface {
	// ListResources returns a list of available resources
	ListResources(ctx context.Context, cursor string) ([]Resource, string)

	// ReadResource reads a resource by URI
	ReadResource(ctx context.Context, uri string) ([]ResourceContents, error)

	// SubscribeResource subscribes to updates for a resource
	SubscribeResource(ctx context.Context, uri string) error

	// UnsubscribeResource unsubscribes from updates for a resource
	UnsubscribeResource(ctx context.Context, uri string) error

	// ListResourceTemplates returns a list of available resource templates
	ListResourceTemplates(ctx context.Context, cursor string) ([]ResourceTemplate, string)
}

// DynamicResourceRegistry is a registry that dynamically provides resources through a provider
type DynamicResourceRegistry struct {
	provider ResourceProviderInterface
}

// NewDynamicResourceRegistry creates a new dynamic resource registry
func NewDynamicResourceRegistry(provider ResourceProviderInterface) *DynamicResourceRegistry {
	return &DynamicResourceRegistry{
		provider: provider,
	}
}

// ListResources returns a paginated list of resources
func (r *DynamicResourceRegistry) ListResources(ctx context.Context, opts ResourceListOptions) ResourceListResult {
	resources, nextCursor := r.provider.ListResources(ctx, opts.Cursor)

	return ResourceListResult{
		Resources:  resources,
		NextCursor: nextCursor,
	}
}

// ReadResource reads a resource by URI
func (r *DynamicResourceRegistry) ReadResource(ctx context.Context, uri string) ([]ResourceContents, error) {
	contents, err := r.provider.ReadResource(ctx, uri)
	if err != nil {
		slog.Debug("Error reading resource", "uri", uri, "error", err)
		return nil, err
	}
	return contents, nil
}

// SubscribeResource subscribes to updates for a resource
func (r *DynamicResourceRegistry) SubscribeResource(ctx context.Context, uri string) error {
	err := r.provider.SubscribeResource(ctx, uri)
	if err != nil {
		slog.Debug("Error subscribing to resource", "uri", uri, "error", err)
		return err
	}
	return nil
}

// UnsubscribeResource unsubscribes from updates for a resource
func (r *DynamicResourceRegistry) UnsubscribeResource(ctx context.Context, uri string) error {
	err := r.provider.UnsubscribeResource(ctx, uri)
	if err != nil {
		slog.Debug("Error unsubscribing from resource", "uri", uri, "error", err)
		return err
	}
	return nil
}

// ListResourceTemplates returns a paginated list of resource templates
func (r *DynamicResourceRegistry) ListResourceTemplates(ctx context.Context, opts ResourceTemplateListOptions) ResourceTemplateListResult {
	templates, nextCursor := r.provider.ListResourceTemplates(ctx, opts.Cursor)

	return ResourceTemplateListResult{
		ResourceTemplates: templates,
		NextCursor:        nextCursor,
	}
}

// Ensure DynamicResourceRegistry implements ResourceRegistry
var _ ResourceRegistry = (*DynamicResourceRegistry)(nil)
