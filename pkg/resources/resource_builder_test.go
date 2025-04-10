package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewResource(t *testing.T) {
	uri := "test/resource"
	name := "Test Resource"
	builder := NewResource(uri, name)

	if builder == nil {
		t.Fatal("NewResource returned nil")
	}

	resource := builder.Build()

	assert.Equal(t, uri, resource.URI, "Resource URI should match")
	assert.Equal(t, name, resource.Name, "Resource name should match")
	assert.Empty(t, resource.Description, "Description should be empty")
	assert.Empty(t, resource.MimeType, "MimeType should be empty")
	assert.Zero(t, resource.Size, "Size should be zero")
}

func TestResourceBuilder_WithDescription(t *testing.T) {
	description := "Test resource description"
	resource := NewResource("test/resource", "Test Resource").
		WithDescription(description).
		Build()

	assert.Equal(t, description, resource.Description, "Description should match")
}

func TestResourceBuilder_WithMimeType(t *testing.T) {
	mimeType := "application/json"
	resource := NewResource("test/resource", "Test Resource").
		WithMimeType(mimeType).
		Build()

	assert.Equal(t, mimeType, resource.MimeType, "MimeType should match")
}

func TestResourceBuilder_WithSize(t *testing.T) {
	size := int64(1024)
	resource := NewResource("test/resource", "Test Resource").
		WithSize(size).
		Build()

	assert.Equal(t, size, resource.Size, "Size should match")
}

func TestResourceBuilder_ChainedMethods(t *testing.T) {
	uri := "test/resource"
	name := "Test Resource"
	description := "Test resource description"
	mimeType := "application/json"
	size := int64(1024)

	resource := NewResource(uri, name).
		WithDescription(description).
		WithMimeType(mimeType).
		WithSize(size).
		Build()

	assert.Equal(t, uri, resource.URI, "Resource URI should match")
	assert.Equal(t, name, resource.Name, "Resource name should match")
	assert.Equal(t, description, resource.Description, "Description should match")
	assert.Equal(t, mimeType, resource.MimeType, "MimeType should match")
	assert.Equal(t, size, resource.Size, "Size should match")
}

func TestNewResourceTemplate(t *testing.T) {
	uriTemplate := "test/template/{id}"
	name := "Test Template"
	builder := NewResourceTemplate(uriTemplate, name)

	if builder == nil {
		t.Fatal("NewResourceTemplate returned nil")
	}

	template := builder.Build()

	assert.Equal(t, uriTemplate, template.URITemplate, "Template URI should match")
	assert.Equal(t, name, template.Name, "Template name should match")
	assert.Empty(t, template.Description, "Description should be empty")
	assert.Empty(t, template.MimeType, "MimeType should be empty")
}

func TestResourceTemplateBuilder_WithDescription(t *testing.T) {
	description := "Test template description"
	template := NewResourceTemplate("test/template/{id}", "Test Template").
		WithDescription(description).
		Build()

	assert.Equal(t, description, template.Description, "Description should match")
}

func TestResourceTemplateBuilder_WithMimeType(t *testing.T) {
	mimeType := "application/json"
	template := NewResourceTemplate("test/template/{id}", "Test Template").
		WithMimeType(mimeType).
		Build()

	assert.Equal(t, mimeType, template.MimeType, "MimeType should match")
}

func TestResourceTemplateBuilder_ChainedMethods(t *testing.T) {
	uriTemplate := "test/template/{id}"
	name := "Test Template"
	description := "Test template description"
	mimeType := "application/json"

	template := NewResourceTemplate(uriTemplate, name).
		WithDescription(description).
		WithMimeType(mimeType).
		Build()

	assert.Equal(t, uriTemplate, template.URITemplate, "Template URI should match")
	assert.Equal(t, name, template.Name, "Template name should match")
	assert.Equal(t, description, template.Description, "Description should match")
	assert.Equal(t, mimeType, template.MimeType, "MimeType should match")
}
