package resources

import (
	"context"
)

// ResourceBuilder is a builder for creating resources
type ResourceBuilder struct {
	resource Resource
}

// ResourceTemplateBuilder is a builder for creating resource templates
type ResourceTemplateBuilder struct {
	template ResourceTemplate
	provider ResourceTemplateProvider
}

// ResourceTemplateProvider is a function that provides resource contents for a given URI that matches a template
type ResourceTemplateProvider func(ctx context.Context, uri string) ([]ResourceContents, error)

// NewResource creates a new resource builder
func NewResource(uri string, name string) *ResourceBuilder {
	return &ResourceBuilder{
		resource: Resource{
			URI:  uri,
			Name: name,
		},
	}
}

// WithDescription sets the description of the resource
func (b *ResourceBuilder) WithDescription(description string) *ResourceBuilder {
	b.resource.Description = description
	return b
}

// WithMimeType sets the MIME type of the resource
func (b *ResourceBuilder) WithMimeType(mimeType string) *ResourceBuilder {
	b.resource.MimeType = mimeType
	return b
}

// WithSize sets the size of the resource
func (b *ResourceBuilder) WithSize(size int64) *ResourceBuilder {
	b.resource.Size = size
	return b
}

// Build builds the resource
func (b *ResourceBuilder) Build() Resource {
	return b.resource
}

// NewResourceTemplate creates a new resource template builder
func NewResourceTemplate(uriTemplate string, name string) *ResourceTemplateBuilder {
	return &ResourceTemplateBuilder{
		template: ResourceTemplate{
			URITemplate: uriTemplate,
			Name:        name,
		},
	}
}

// WithDescription sets the description of the resource template
func (b *ResourceTemplateBuilder) WithDescription(description string) *ResourceTemplateBuilder {
	b.template.Description = description
	return b
}

// WithMimeType sets the MIME type of the resource template
func (b *ResourceTemplateBuilder) WithMimeType(mimeType string) *ResourceTemplateBuilder {
	b.template.MimeType = mimeType
	return b
}

// WithProvider sets the provider function for the resource template
func (b *ResourceTemplateBuilder) WithProvider(provider ResourceTemplateProvider) *ResourceTemplateBuilder {
	b.provider = provider
	return b
}

// Build builds the resource template
// This method is kept for backward compatibility
func (b *ResourceTemplateBuilder) Build() ResourceTemplate {
	return b.template
}

// BuildWithProvider builds the resource template and returns both the template and its provider function
func (b *ResourceTemplateBuilder) BuildWithProvider() (ResourceTemplate, ResourceTemplateProvider) {
	return b.template, b.provider
}
