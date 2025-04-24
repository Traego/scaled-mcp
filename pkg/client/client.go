package client

// Package client provides a robust MCP client implementation that supports both
// the 2024-11-05 and 2025-03-26 MCP specifications.

import (
	"context"
	"net/http"

	"github.com/traego/scaled-mcp/pkg/protocol"
)

// ConnectionMethod represents the transport method used for the MCP connection.
type ConnectionMethod string

const (
	// ConnectionMethodSSE represents a connection using Server-Sent Events.
	ConnectionMethodSSE ConnectionMethod = "sse"

	// ConnectionMethodHTTP represents a connection using direct HTTP requests.
	ConnectionMethodHTTP ConnectionMethod = "http"
)

// ClientOptions contains configuration options for the MCP client.
type ClientOptions struct {
	// ProtocolVersion specifies which MCP protocol version to use.
	// If set to ProtocolVersionAuto, the client will negotiate the highest supported version.
	ProtocolVersion protocol.ProtocolVersion

	// HTTPClient allows providing a custom HTTP client for the transport layer.
	HTTPClient *http.Client

	// ClientInfo contains information about the client to send during initialization.
	ClientInfo ClientInfo

	// Capabilities defines the client's capabilities to negotiate with the server.
	Capabilities Capabilities

	// UseSSEForEvents controls whether the client should establish an SSE connection
	// for events even when using the 2025 protocol (which supports both HTTP and SSE).
	// Setting this to true provides a fallback for event delivery.
	// Default is true.
	UseSSEForEvents bool
}

// ClientInfo contains information about the client to send during initialization.
type ClientInfo struct {
	// Name is the name of the client.
	Name string

	// Version is the version of the client.
	Version string
}

// Capabilities defines the client's capabilities to negotiate with the server.
type Capabilities struct {
	// Roots indicates whether the client supports the roots feature.
	Roots RootsCapabilities

	// Sampling indicates whether the client supports the sampling feature.
	Sampling SamplingCapabilities
}

// RootsCapabilities defines the client's capabilities for the roots feature.
type RootsCapabilities struct {
	// Enabled indicates whether the roots feature is enabled.
	Enabled bool

	// ListChanged indicates whether the client supports list changed notifications.
	ListChanged bool
}

// SamplingCapabilities defines the client's capabilities for the sampling feature.
type SamplingCapabilities struct {
	// Enabled indicates whether the sampling feature is enabled.
	Enabled bool
}

// McpClient is the interface for an MCP client.
type McpClient interface {
	// Connect establishes a connection with the server and performs protocol initialization.
	Connect(ctx context.Context) error

	// Close closes the client connection.
	Close(ctx context.Context) error

	// IsInitialized returns whether the client has been initialized.
	IsInitialized() bool

	// GetSessionID returns the current session ID, if any.
	GetSessionID() string

	// GetProtocolVersion returns the negotiated protocol version.
	GetProtocolVersion() protocol.ProtocolVersion

	// GetConnectionMethod returns the connection method being used (SSE or HTTP).
	GetConnectionMethod() ConnectionMethod

	// SendRequest sends a request to the server and returns the response.
	SendRequest(ctx context.Context, method string, params interface{}) (*protocol.JSONRPCMessage, error)

	// SendNotification sends a notification to the server.
	SendNotification(ctx context.Context, method string, params interface{}) error

	// AddEventHandler adds an event handler for server-sent events.
	AddEventHandler(handler EventHandler)

	// RemoveEventHandler removes an event handler.
	RemoveEventHandler(handler EventHandler)

	// ListTools retrieves the list of available tools from the server.
	ListTools(ctx context.Context) (*protocol.ToolListResult, error)

	// FindTool searches for a tool by name in the tools list.
	FindTool(ctx context.Context, toolName string) (*protocol.Tool, error)

	// CallTool calls a specific tool with the given parameters.
	CallTool(ctx context.Context, toolName string, params interface{}) (*protocol.JSONRPCMessage, error)
}

// EventHandler is the interface for handling server-sent events.
type EventHandler interface {
	// HandleEvent handles a server-sent event.
	HandleEvent(event *protocol.JSONRPCMessage)
}

// EventHandlerFunc is a function that implements the EventHandler interface.
type EventHandlerFunc func(event *protocol.JSONRPCMessage)

// HandleEvent calls the function.
func (f EventHandlerFunc) HandleEvent(event *protocol.JSONRPCMessage) {
	f(event)
}

// DefaultClientOptions returns the default client options.
func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		ProtocolVersion: protocol.ProtocolVersionAuto,
		HTTPClient:      http.DefaultClient,
		ClientInfo: ClientInfo{
			Name:    "scaled-mcp-client",
			Version: "1.0.0",
		},
		Capabilities: Capabilities{
			Roots: RootsCapabilities{
				Enabled:     true,
				ListChanged: true,
			},
			Sampling: SamplingCapabilities{
				Enabled: true,
			},
		},
		UseSSEForEvents: false,
	}
}

// NewMcpClient creates a new MCP client with the given server URL and options.
func NewMcpClient(serverURL string, options ClientOptions) (McpClient, error) {
	return NewHTTPClient(serverURL, options)
}
