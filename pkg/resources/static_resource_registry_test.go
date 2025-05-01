package resources

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticResourceRegistry(t *testing.T) {
	registry := NewStaticResourceRegistry()

	// Test registering a resource
	resourceURI := "test/resource/1"
	resource := Resource{
		URI:         resourceURI,
		Name:        "Test Resource",
		Description: "A test resource",
		MimeType:    "text/plain",
		Size:        100,
	}

	// Create a provider function
	providerFunc := func(ctx context.Context, uri string) ([]ResourceContents, error) {
		if uri != resourceURI {
			return nil, errors.New("resource not found")
		}
		return []ResourceContents{
			ResourceContentText{
				URI:      uri,
				MimeType: "text/plain",
				Text:     "This is test content",
			},
		}, nil
	}

	// Register the resource
	err := registry.RegisterResource(resource, providerFunc)
	if err != nil {
		t.Fatalf("Failed to register resource: %v", err)
	}

	// Test registering a resource template
	templateURI := "test/template/{id}"
	template := ResourceTemplate{
		URITemplate: templateURI,
		Name:        "Test Template",
		Description: "A test resource template",
		MimeType:    "text/plain",
	}

	err = registry.RegisterResourceTemplate(template)
	if err != nil {
		t.Fatalf("Failed to register resource template: %v", err)
	}

	// Test ListResources
	t.Run("ListResources", func(t *testing.T) {
		ctx := context.Background()

		result := registry.ListResources(ctx, ResourceListOptions{})

		if len(result.Resources) != 1 {
			t.Fatalf("Expected 1 resource, got %d", len(result.Resources))
		}

		if result.Resources[0].URI != resourceURI {
			t.Errorf("Expected resource URI %q, got %q", resourceURI, result.Resources[0].URI)
		}

		if result.Resources[0].Name != resource.Name {
			t.Errorf("Expected resource name %q, got %q", resource.Name, result.Resources[0].Name)
		}

		// Empty next cursor for a single page
		if result.NextCursor != "" {
			t.Errorf("Expected empty next cursor, got %q", result.NextCursor)
		}
	})

	// Test ReadResource
	t.Run("ReadResource", func(t *testing.T) {
		ctx := context.Background()

		// Read an existing resource
		contents, err := registry.ReadResource(ctx, resourceURI)
		_, ok := contents[0].(ResourceContentText)
		assert.True(t, ok)

		if err != nil {
			t.Fatalf("Failed to read resource: %v", err)
		}

		if len(contents) != 1 {
			t.Fatalf("Expected 1 content item, got %d", len(contents))
		}

		if contents[0].GetURI() != resourceURI {
			t.Errorf("Expected content URI %q, got %q", resourceURI, contents[0].GetURI())
		}

		if contents[0].GetText() != "This is test content" {
			t.Errorf("Expected content %q, got %q", "This is test content", contents[0].GetText())
		}

		// Read a non-existent resource
		_, err = registry.ReadResource(ctx, "non-existent-resource")
		if err == nil {
			t.Error("Expected error for non-existent resource")
		}
		if !errors.Is(err, ErrResourceNotFound) {
			t.Errorf("Expected ErrResourceNotFound, got %v", err)
		}
	})

	// Test ListResourceTemplates
	t.Run("ListResourceTemplates", func(t *testing.T) {
		ctx := context.Background()

		result := registry.ListResourceTemplates(ctx, ResourceTemplateListOptions{})

		if len(result.ResourceTemplates) != 1 {
			t.Fatalf("Expected 1 resource template, got %d", len(result.ResourceTemplates))
		}

		if result.ResourceTemplates[0].URITemplate != templateURI {
			t.Errorf("Expected template URI %q, got %q", templateURI, result.ResourceTemplates[0].URITemplate)
		}

		if result.ResourceTemplates[0].Name != template.Name {
			t.Errorf("Expected template name %q, got %q", template.Name, result.ResourceTemplates[0].Name)
		}

		// Empty next cursor for a single page
		if result.NextCursor != "" {
			t.Errorf("Expected empty next cursor, got %q", result.NextCursor)
		}
	})

	// Test SubscribeResource and GetSubscribers
	t.Run("SubscribeResource", func(t *testing.T) {
		// Create a context with subscriber ID
		subscriberID := "test-subscriber"
		ctx := context.WithValue(context.Background(), SubscriberIDKey, subscriberID)

		// Subscribe to a resource
		err := registry.SubscribeResource(ctx, resourceURI)
		if err != nil {
			t.Fatalf("Failed to subscribe to resource: %v", err)
		}

		// Check subscribers
		subscribers := registry.GetSubscribers(resourceURI)
		if len(subscribers) != 1 {
			t.Fatalf("Expected 1 subscriber, got %d", len(subscribers))
		}

		if subscribers[0] != subscriberID {
			t.Errorf("Expected subscriber ID %q, got %q", subscriberID, subscribers[0])
		}

		// Subscribe to a non-existent resource
		err = registry.SubscribeResource(ctx, "non-existent-resource")
		if err == nil {
			t.Error("Expected error for non-existent resource")
		}
		if !errors.Is(err, ErrResourceNotFound) {
			t.Errorf("Expected ErrResourceNotFound, got %v", err)
		}
	})

	// Test UnsubscribeResource
	t.Run("UnsubscribeResource", func(t *testing.T) {
		// Create a context with subscriber ID
		subscriberID := "test-subscriber"
		ctx := context.WithValue(context.Background(), SubscriberIDKey, subscriberID)

		// Unsubscribe from a resource
		err := registry.UnsubscribeResource(ctx, resourceURI)
		if err != nil {
			t.Fatalf("Failed to unsubscribe from resource: %v", err)
		}

		// Check subscribers
		subscribers := registry.GetSubscribers(resourceURI)
		if len(subscribers) != 0 {
			t.Fatalf("Expected 0 subscribers, got %d", len(subscribers))
		}

		// Unsubscribe from a non-existent resource
		err = registry.UnsubscribeResource(ctx, "non-existent-resource")
		if err == nil {
			t.Error("Expected error for non-existent resource")
		}
		if !errors.Is(err, ErrResourceNotFound) {
			t.Errorf("Expected ErrResourceNotFound, got %v", err)
		}
	})

	// Test error cases
	t.Run("ErrorCases", func(t *testing.T) {
		// Register resource with empty URI
		err := registry.RegisterResource(Resource{}, nil)
		if err == nil {
			t.Error("Expected error for empty resource URI")
		}

		// Register template with empty URI
		err = registry.RegisterResourceTemplate(ResourceTemplate{})
		if err == nil {
			t.Error("Expected error for empty template URI")
		}
	})
}

func TestStaticResourceRegistry_Pagination(t *testing.T) {
	registry := NewStaticResourceRegistry()

	// Register multiple resources for pagination testing
	for i := 1; i <= 30; i++ {
		uri := fmt.Sprintf("test/resource/%d", i)
		resource := Resource{
			URI:         uri,
			Name:        fmt.Sprintf("Test Resource %d", i),
			Description: fmt.Sprintf("Description for resource %d", i),
			MimeType:    "text/plain",
			Size:        int64(i * 100),
		}
		err := registry.RegisterResource(resource, nil)
		require.NoError(t, err)
	}

	// Test resource pagination
	t.Run("ResourcePagination", func(t *testing.T) {
		ctx := context.Background()

		// First page
		result := registry.ListResources(ctx, ResourceListOptions{})
		assert.Len(t, result.Resources, 20, "First page should have 20 resources")
		assert.NotEmpty(t, result.NextCursor, "Next cursor should not be empty")

		// Second page using the cursor
		result = registry.ListResources(ctx, ResourceListOptions{Cursor: result.NextCursor})
		assert.Len(t, result.Resources, 10, "Second page should have 10 resources")
		assert.Empty(t, result.NextCursor, "Next cursor should be empty for last page")

		// Using a cursor that doesn't exist returns the first page
		// This is the actual behavior of the implementation
		result = registry.ListResources(ctx, ResourceListOptions{Cursor: "test/resource/999"})
		assert.Len(t, result.Resources, 20, "Non-existent cursor should return first page")
	})

	// Register multiple templates for pagination testing
	for i := 1; i <= 30; i++ {
		uri := fmt.Sprintf("test/template/{id}/%d", i)
		template := ResourceTemplate{
			URITemplate: uri,
			Name:        fmt.Sprintf("Test Template %d", i),
			Description: fmt.Sprintf("Description for template %d", i),
			MimeType:    "text/plain",
		}
		err := registry.RegisterResourceTemplate(template)
		require.NoError(t, err)
	}

	// Test template pagination
	t.Run("TemplatesPagination", func(t *testing.T) {
		ctx := context.Background()

		// First page
		result := registry.ListResourceTemplates(ctx, ResourceTemplateListOptions{})
		assert.Len(t, result.ResourceTemplates, 20, "First page should have 20 templates")
		assert.NotEmpty(t, result.NextCursor, "Next cursor should not be empty")

		// Second page using the cursor
		result = registry.ListResourceTemplates(ctx, ResourceTemplateListOptions{Cursor: result.NextCursor})
		assert.Len(t, result.ResourceTemplates, 10, "Second page should have 10 templates")
		assert.Empty(t, result.NextCursor, "Next cursor should be empty for last page")

		// Using a cursor that doesn't exist returns the first page
		// This is the actual behavior of the implementation
		result = registry.ListResourceTemplates(ctx, ResourceTemplateListOptions{Cursor: "test/template/{id}/999"})
		assert.Len(t, result.ResourceTemplates, 20, "Non-existent cursor should return first page")
	})
}

func TestStaticResourceRegistry_MultipleSubscribers(t *testing.T) {
	registry := NewStaticResourceRegistry()

	// Register a resource
	resourceURI := "test/resource/subscribers"
	resource := Resource{
		URI:         resourceURI,
		Name:        "Subscriber Test Resource",
		Description: "A resource for testing multiple subscribers",
		MimeType:    "text/plain",
	}
	err := registry.RegisterResource(resource, nil)
	require.NoError(t, err)

	// Subscribe multiple subscribers
	subscribers := []string{"subscriber1", "subscriber2", "subscriber3"}
	for _, id := range subscribers {
		ctx := context.WithValue(context.Background(), SubscriberIDKey, id)
		err := registry.SubscribeResource(ctx, resourceURI)
		require.NoError(t, err)
	}

	// Verify all subscribers are registered
	t.Run("AllSubscribersRegistered", func(t *testing.T) {
		registeredSubscribers := registry.GetSubscribers(resourceURI)
		assert.Len(t, registeredSubscribers, len(subscribers))

		// Check each subscriber is in the list
		for _, id := range subscribers {
			found := false
			for _, registeredID := range registeredSubscribers {
				if registeredID == id {
					found = true
					break
				}
			}
			assert.True(t, found, "Subscriber %s should be registered", id)
		}
	})

	// Unsubscribe one subscriber
	t.Run("UnsubscribeOne", func(t *testing.T) {
		unsubscribeID := "subscriber2"
		ctx := context.WithValue(context.Background(), SubscriberIDKey, unsubscribeID)
		err := registry.UnsubscribeResource(ctx, resourceURI)
		require.NoError(t, err)

		// Verify the subscriber was removed
		registeredSubscribers := registry.GetSubscribers(resourceURI)
		assert.Len(t, registeredSubscribers, len(subscribers)-1)

		// Check the unsubscribed ID is not in the list
		for _, id := range registeredSubscribers {
			assert.NotEqual(t, unsubscribeID, id, "Unsubscribed ID should not be in the list")
		}
	})

	// Test default subscriber ID
	t.Run("DefaultSubscriberID", func(t *testing.T) {
		// Context without subscriber ID
		ctx := context.Background()
		err := registry.SubscribeResource(ctx, resourceURI)
		require.NoError(t, err)

		// Verify the default subscriber was added
		registeredSubscribers := registry.GetSubscribers(resourceURI)
		defaultFound := false
		for _, id := range registeredSubscribers {
			if id == "default" {
				defaultFound = true
				break
			}
		}
		assert.True(t, defaultFound, "Default subscriber should be registered")

		// Unsubscribe the default subscriber
		err = registry.UnsubscribeResource(ctx, resourceURI)
		require.NoError(t, err)
	})
}

func TestStaticResourceRegistry_ResourceProvider(t *testing.T) {
	registry := NewStaticResourceRegistry()

	// Register a resource with a provider that returns multiple contents
	resourceURI := "test/resource/multi-content"
	resource := Resource{
		URI:         resourceURI,
		Name:        "Multi-Content Resource",
		Description: "A resource with multiple content items",
		MimeType:    "application/json",
	}

	// Provider that returns multiple content items
	providerFunc := func(ctx context.Context, uri string) ([]ResourceContents, error) {
		if uri != resourceURI {
			return nil, errors.New("resource not found")
		}
		return []ResourceContents{
			ResourceContentText{
				URI:      uri + "/part1",
				MimeType: "application/json",
				Text:     `{"part": 1, "data": "first part"}`,
			},
			ResourceContentText{
				URI:      uri + "/part2",
				MimeType: "application/json",
				Text:     `{"part": 2, "data": "second part"}`,
			},
		}, nil
	}

	err := registry.RegisterResource(resource, providerFunc)
	require.NoError(t, err)

	// Test reading the resource with multiple contents
	t.Run("MultipleContents", func(t *testing.T) {
		ctx := context.Background()
		contents, err := registry.ReadResource(ctx, resourceURI)
		c, ok := contents[0].(ResourceContentText)
		assert.True(t, ok)

		require.NoError(t, err)
		assert.Len(t, contents, 2, "Should return 2 content items")

		// Check first content
		assert.Equal(t, resourceURI+"/part1", c.GetURI())
		assert.Equal(t, "application/json", c.MimeType)
		assert.Equal(t, `{"part": 1, "data": "first part"}`, c.Text)

		// Check second content
		assert.Equal(t, resourceURI+"/part2", contents[1].GetURI())
		assert.Equal(t, "application/json", contents[1].GetMimeType())
		assert.Equal(t, `{"part": 2, "data": "second part"}`, contents[1].GetText())
	})

	// Test resource with no provider
	t.Run("NoProvider", func(t *testing.T) {
		noProviderURI := "test/resource/no-provider"
		noProviderResource := Resource{
			URI:         noProviderURI,
			Name:        "No Provider Resource",
			Description: "A resource without a provider",
			MimeType:    "text/plain",
		}

		err := registry.RegisterResource(noProviderResource, nil)
		require.NoError(t, err)

		// Try to read the resource
		ctx := context.Background()
		_, err = registry.ReadResource(ctx, noProviderURI)
		assert.Error(t, err, "Reading a resource without a provider should fail")
		assert.Contains(t, err.Error(), "no provider available")
	})

	// Test provider that returns an error
	t.Run("ProviderError", func(t *testing.T) {
		errorURI := "test/resource/error"
		errorResource := Resource{
			URI:         errorURI,
			Name:        "Error Resource",
			Description: "A resource with a provider that returns an error",
			MimeType:    "text/plain",
		}

		expectedError := errors.New("provider error")
		errorProviderFunc := func(ctx context.Context, uri string) ([]ResourceContents, error) {
			return nil, expectedError
		}

		err := registry.RegisterResource(errorResource, errorProviderFunc)
		require.NoError(t, err)

		// Try to read the resource
		ctx := context.Background()
		_, err = registry.ReadResource(ctx, errorURI)
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
	})
}
