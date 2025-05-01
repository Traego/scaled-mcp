package resources

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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

func TestResourceTemplateBuilder_WithProvider(t *testing.T) {
	uriTemplate := "test/template/{id}"
	name := "Test Template"

	// Define a test provider function that generates resource contents based on the URI
	providerFunc := func(ctx context.Context, uri string) ([]ResourceContents, error) {
		// Extract id from URI (format: test/template/123)
		parts := strings.Split(uri, "/")
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid URI format: %s", uri)
		}
		id := parts[2]

		return []ResourceContents{
			NewTextResourceContents(uri, "application/json", fmt.Sprintf(`{"id": "%s", "name": "Resource %s"}`, id, id)),
		}, nil
	}

	builder := NewResourceTemplate(uriTemplate, name)
	builder = builder.WithProvider(providerFunc)

	// Build and retrieve both template and provider
	template, provider := builder.BuildWithProvider()

	// Verify template properties
	assert.Equal(t, uriTemplate, template.URITemplate, "Template URI should match")
	assert.Equal(t, name, template.Name, "Template name should match")

	// Verify we have a provider function
	assert.NotNil(t, provider, "Provider function should not be nil")
}

func TestResourceTemplateProviderInvocation(t *testing.T) {
	uriTemplate := "test/template/{id}"
	name := "Test Template"
	mimeType := "application/json"

	// Define a test provider function that generates different content based on resource ID
	providerFunc := func(ctx context.Context, uri string) ([]ResourceContents, error) {
		// Extract id from URI (format: test/template/123)
		parts := strings.Split(uri, "/")
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid URI format: %s", uri)
		}
		id := parts[2]

		// Convert ID to int for testing error case
		idNum, err := strconv.Atoi(id)
		if err != nil {
			return nil, fmt.Errorf("invalid ID: %s", id)
		}

		// Return error for specific ID to test error handling
		if idNum == 999 {
			return nil, fmt.Errorf("resource not available")
		}

		return []ResourceContents{
			NewTextResourceContents(uri, mimeType, fmt.Sprintf(`{"id": "%s", "name": "Resource %s"}`, id, id)),
		}, nil
	}

	// Build the template with provider
	template, provider := NewResourceTemplate(uriTemplate, name).
		WithMimeType(mimeType).
		WithProvider(providerFunc).
		BuildWithProvider()

	// Verify template properties
	assert.Equal(t, uriTemplate, template.URITemplate)
	assert.Equal(t, name, template.Name)
	assert.Equal(t, mimeType, template.MimeType)
	assert.NotNil(t, provider)

	// Test 1: Valid URI
	testUri := "test/template/123"
	contents, err := provider(context.Background(), testUri)
	c, ok := contents[0].(*ResourceContentText)
	assert.True(t, ok)

	assert.NoError(t, err)
	assert.Len(t, contents, 1)
	assert.False(t, contents[0].IsBinary())
	assert.True(t, contents[0].IsText())
	assert.Equal(t, testUri, c.URI)
	assert.Equal(t, mimeType, c.MimeType)
	assert.Equal(t, `{"id": "123", "name": "Resource 123"}`, c.Text)
	assert.True(t, contents[0].IsText())
	assert.False(t, contents[0].IsBinary())

	// Test 2: Error case - provider returns an error
	errUri := "test/template/999"
	contents, err = provider(context.Background(), errUri)
	assert.Error(t, err)
	assert.Nil(t, contents)
	assert.Equal(t, "resource not available", err.Error())

	// Test 3: Invalid URI format
	invalidUri := "invalid/format"
	contents, err = provider(context.Background(), invalidUri)
	assert.Error(t, err)
	assert.Nil(t, contents)
}

func TestBinaryResourceContents(t *testing.T) {
	uriTemplate := "test/template/{id}"
	name := "Binary Template"
	mimeType := "image/png"

	// Create a provider that returns binary content
	providerFunc := func(ctx context.Context, uri string) ([]ResourceContents, error) {
		// Mock binary data
		binaryData := []byte{0x89, 0x50, 0x4E, 0x47} // PNG file header

		return []ResourceContents{
			NewBinaryResourceContents(uri, mimeType, binaryData),
		}, nil
	}

	// Build the template with provider
	_, provider := NewResourceTemplate(uriTemplate, name).
		WithMimeType(mimeType).
		WithProvider(providerFunc).
		BuildWithProvider()

	// Test binary content
	testUri := "test/template/image"
	contents, err := provider(context.Background(), testUri)

	c, ok := contents[0].(ResourceContentBinary)
	assert.True(t, ok)

	assert.NoError(t, err)
	assert.Len(t, contents, 1)
	assert.True(t, contents[0].IsBinary())
	assert.False(t, contents[0].IsText())
	assert.Equal(t, testUri, c.URI)
	assert.Equal(t, mimeType, c.MimeType)
	assert.Equal(t, []byte{0x89, 0x50, 0x4E, 0x47}, c.Blob)
	assert.False(t, contents[0].IsText())
	assert.True(t, contents[0].IsBinary())
}
