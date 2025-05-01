package resources

import (
	"context"
	"errors"
)

// Common errors
var (
	ErrResourceNotFound = errors.New("resource not found")
)

// ResourceContents defines the interface for resource content, which can be text or binary.
type ResourceContents interface {
	GetURI() string
	GetMimeType() string
	IsText() bool
	IsBinary() bool
	// GetText returns the text content. Returns an empty string if it's binary content.
	GetText() string
	// GetBlob returns the binary content. Returns nil if it's text content.
	GetBlob() []byte
}

// ResourceContentText holds text-based resource content.
type ResourceContentText struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text"`
}

// GetURI implements the ResourceContents interface.
func (t ResourceContentText) GetURI() string { return t.URI }

// GetMimeType implements the ResourceContents interface.
func (t ResourceContentText) GetMimeType() string { return t.MimeType }

// IsText implements the ResourceContents interface.
func (t ResourceContentText) IsText() bool { return true }

// IsBinary implements the ResourceContents interface.
func (t ResourceContentText) IsBinary() bool { return false }

// GetText implements the ResourceContents interface.
func (t ResourceContentText) GetText() string { return t.Text }

// GetBlob implements the ResourceContents interface.
func (t ResourceContentText) GetBlob() []byte { return nil }

// ResourceContentBinary holds binary resource content.
type ResourceContentBinary struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Blob     []byte `json:"blob"`
}

// GetURI implements the ResourceContents interface.
func (b ResourceContentBinary) GetURI() string { return b.URI }

// GetMimeType implements the ResourceContents interface.
func (b ResourceContentBinary) GetMimeType() string { return b.MimeType }

// IsText implements the ResourceContents interface.
func (b ResourceContentBinary) IsText() bool { return false }

// IsBinary implements the ResourceContents interface.
func (b ResourceContentBinary) IsBinary() bool { return true }

// GetText implements the ResourceContents interface.
func (b ResourceContentBinary) GetText() string { return "" }

// GetBlob implements the ResourceContents interface.
func (b ResourceContentBinary) GetBlob() []byte { return b.Blob }

// NewTextResourceContents creates a new text resource contents object.
func NewTextResourceContents(uri string, mimeType string, text string) ResourceContents {
	return &ResourceContentText{
		URI:      uri,
		MimeType: mimeType,
		Text:     text,
	}
}

// NewBinaryResourceContents creates a new binary resource contents object.
func NewBinaryResourceContents(uri string, mimeType string, blob []byte) ResourceContents {
	return ResourceContentBinary{
		URI:      uri,
		MimeType: mimeType,
		Blob:     blob,
	}
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
