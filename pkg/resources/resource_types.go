package resources

import (
	"context"
	"errors"
)

// Common errors
var (
	ErrResourceNotFound = errors.New("resource not found")
)

// ResourceContents represents the contents of a resource
// It can contain either text or binary data as per the MCP specification
type ResourceContents struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	// Content is used for backward compatibility with existing code
	// When serialized to JSON, it maps to the "text" field according to MCP spec
	Content string `json:"text,omitempty"`
	// Text is used for MCP spec alignment and is not used for serialization
	Text    string
	// For binary content (base64 encoded)
	Blob []byte `json:"blob,omitempty"`
}

// NewTextResourceContents creates a new text resource contents
func NewTextResourceContents(uri string, mimeType string, text string) ResourceContents {
	return ResourceContents{
		URI:      uri,
		MimeType: mimeType,
		Content:  text,
		Text:     text,
	}
}

// NewBinaryResourceContents creates a new binary resource contents
func NewBinaryResourceContents(uri string, mimeType string, blob []byte) ResourceContents {
	return ResourceContents{
		URI:      uri,
		MimeType: mimeType,
		Blob:     blob,
	}
}

// IsText returns true if the resource contains text content
func (r ResourceContents) IsText() bool {
	return r.Content != ""
}

// IsBinary returns true if the resource contains binary content
func (r ResourceContents) IsBinary() bool {
	return len(r.Blob) > 0
}

// GetURI returns the URI of the resource
func (r ResourceContents) GetURI() string {
	return r.URI
}

// GetMimeType returns the MIME type of the resource
func (r ResourceContents) GetMimeType() string {
	return r.MimeType
}

// Resource represents an MCP resource definition
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	Size        int64  `json:"size,omitempty"`
}

// ResourceTemplate represents a template for resources
type ResourceTemplate struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ResourceListOptions provides pagination options for listing resources
type ResourceListOptions struct {
	Cursor string // Cursor for pagination
}

// ResourceListResult represents a paginated list of resources
type ResourceListResult struct {
	Resources  []Resource `json:"resources"`
	NextCursor string     `json:"nextCursor,omitempty"` // Cursor for the next page, empty if no more pages
}

// ResourceTemplateListOptions provides pagination options for listing resource templates
type ResourceTemplateListOptions struct {
	Cursor string // Cursor for pagination
}

// ResourceTemplateListResult represents a paginated list of resource templates
type ResourceTemplateListResult struct {
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
	NextCursor        string             `json:"nextCursor,omitempty"` // Cursor for the next page, empty if no more pages
}

// ResourceRegistry defines the interface for a resource registry
type ResourceRegistry interface {
	// ListResources returns a paginated list of resources
	ListResources(ctx context.Context, opts ResourceListOptions) ResourceListResult

	// ReadResource reads a resource by URI
	ReadResource(ctx context.Context, uri string) ([]ResourceContents, error)

	// SubscribeResource subscribes to updates for a resource
	SubscribeResource(ctx context.Context, uri string) error

	// UnsubscribeResource unsubscribes from updates for a resource
	UnsubscribeResource(ctx context.Context, uri string) error

	// ListResourceTemplates returns a paginated list of resource templates
	ListResourceTemplates(ctx context.Context, opts ResourceTemplateListOptions) ResourceTemplateListResult
}
