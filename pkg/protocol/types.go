package protocol

import (
	"encoding/json"
)

// SESSION_ID_CONTEXT_KEY is the key used to store and retrieve the session ID from the context
const SESSION_ID_CONTEXT_KEY = "sessionId"

// JSONRPCMessage represents a JSON-RPC message
type JSONRPCMessage struct {
	JSONRPC string              `json:"jsonrpc"`
	ID      interface{}         `json:"id,omitempty"`
	Method  string              `json:"method,omitempty"`
	Params  interface{}         `json:"params,omitempty"`
	Result  interface{}         `json:"result,omitempty"`
	Error   interface{}         `json:"error,omitempty"`
	Headers map[string][]string `json:"headers,omitempty"`
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema InputSchema `json:"inputSchema,omitempty,omitzero"`
}

// InputSchema represents the schema for tool inputs
type InputSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]SchemaProperty `json:"properties"`
	Required   []string                  `json:"required,omitempty"`
}

// SchemaProperty represents a property in an input schema
type SchemaProperty struct {
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Default     interface{} `json:"default,omitempty"`
}

// ToolListOptions provides pagination options for listing resources
type ToolListOptions struct {
	Cursor string // Cursor for pagination
}

// ToolListResult represents a paginated list of resources
type ToolListResult struct {
	Tools      []Tool `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"` // Cursor for the next page, empty if no more pages
}

// ClientInfo represents information about the client
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerInfo represents information about the server
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeParams represents the parameters for an initialize request
type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
}

// InitializeResult represents the result of an initialize request
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	SessionID       string             `json:"sessionId,omitempty"`
}

// ClientCapabilities represents the capabilities of the client
type ClientCapabilities struct {
	Roots        *RootsClientCapability    `json:"roots,omitempty"`
	Sampling     *SamplingClientCapability `json:"sampling,omitempty"`
	Experimental map[string]interface{}    `json:"experimental,omitempty"`
}

// RootsClientCapability represents the roots capability of the client
type RootsClientCapability struct {
	ListChanged bool `json:"listChanged"`
}

// SamplingClientCapability represents the sampling capability of the client
type SamplingClientCapability struct {
	// Add sampling-specific fields here
}

// ServerCapabilities represents the capabilities of the server
type ServerCapabilities struct {
	Prompts      *PromptsServerCapability   `json:"prompts,omitempty"`
	Resources    *ResourcesServerCapability `json:"resources,omitempty"`
	Tools        *ToolsServerCapability     `json:"tools,omitempty"`
	Logging      *LoggingServerCapability   `json:"logging,omitempty"`
	Experimental map[string]interface{}     `json:"experimental,omitempty"`
}

// PromptsServerCapability represents the prompts capability of the server
type PromptsServerCapability struct {
	ListChanged bool `json:"listChanged"`
}

// ResourcesServerCapability represents the resources capability of the server
type ResourcesServerCapability struct {
	Subscribe   bool `json:"subscribe"`
	ListChanged bool `json:"listChanged"`
}

// ToolsServerCapability represents the resources capability of the server
type ToolsServerCapability struct {
	ListChanged bool `json:"listChanged"`
}

// LoggingServerCapability represents the logging capability of the server
type LoggingServerCapability struct {
	// Empty struct as per the 2025 spec
}

// ToolCallResult represents the result of a tool call
type ToolCallResult struct {
	Content []ToolCallContent `json:"content"`
	IsError bool              `json:"isError,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface for ToolCallResult
func (r ToolCallResult) MarshalJSON() ([]byte, error) {
	// Create a map to hold the marshaled content
	result := map[string]interface{}{
		"isError": r.IsError,
	}

	// Marshal each content item separately
	contentItems := make([]interface{}, len(r.Content))
	for i, item := range r.Content {
		contentItems[i] = item
	}
	result["content"] = contentItems

	// Marshal the result map
	return json.Marshal(result)
}

// ToolCallContent represents a content item in a tool call result
type ToolCallContent interface {
	GetType() string
}

// TextContent represents a text content item in a tool call result
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// GetType returns the type of the content item
func (t TextContent) GetType() string {
	return "text"
}

// ImageContent represents an image content item in a tool call result
type ImageContent struct {
	Type     string `json:"type"`
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
}

// GetType returns the type of the content item
func (i ImageContent) GetType() string {
	return "image"
}

// AudioContent represents an audio content item in a tool call result
type AudioContent struct {
	Type     string `json:"type"`
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
}

// GetType returns the type of the content item
func (a AudioContent) GetType() string {
	return "audio"
}

// ResourceContent represents a resource content item in a tool call result
type ResourceContent struct {
	Type     string      `json:"type"`
	Resource interface{} `json:"resource"`
}

// GetType returns the type of the content item
func (r ResourceContent) GetType() string {
	return "resource"
}

// NewTextContent creates a new text content item
func NewTextContent(text string) TextContent {
	return TextContent{
		Type: "text",
		Text: text,
	}
}

// NewImageContent creates a new image content item
func NewImageContent(data string, mimeType string) ImageContent {
	return ImageContent{
		Type:     "image",
		Data:     data,
		MimeType: mimeType,
	}
}

// NewAudioContent creates a new audio content item
func NewAudioContent(data string, mimeType string) AudioContent {
	return AudioContent{
		Type:     "audio",
		Data:     data,
		MimeType: mimeType,
	}
}

// NewResourceContent creates a new resource content item
func NewResourceContent(resource interface{}) ResourceContent {
	return ResourceContent{
		Type:     "resource",
		Resource: resource,
	}
}

// NewToolCallResult creates a new tool call result with the given content items
func NewToolCallResult(content []ToolCallContent, isError bool) ToolCallResult {
	return ToolCallResult{
		Content: content,
		IsError: isError,
	}
}
