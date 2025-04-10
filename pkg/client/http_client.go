package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/r3labs/sse/v2"
	"github.com/traego/scaled-mcp/pkg/protocol"
)

// httpClient implements the McpClient interface for HTTP-based MCP communication.
type httpClient struct {
	serverURL        string
	options          ClientOptions
	httpClient       *http.Client
	sessionID        string
	initialized      bool
	eventHandlers    []EventHandler
	handlersMutex    sync.RWMutex
	sseClient        *sse.Client
	messageEndpoint  string
	sseEndpoint      string
	protocolVersion  ProtocolVersion
	connectionMethod ConnectionMethod
	requestIDCounter int
	requestIDMutex   sync.Mutex
	responseMap      map[string]chan *protocol.JSONRPCMessage
	responseMapMutex sync.RWMutex
	endpointMutex    sync.RWMutex
	protocolMutex    sync.RWMutex
}

// newHTTPClient creates a new HTTP-based MCP client.
func newHTTPClient(serverURL string, options ClientOptions) (*httpClient, error) {
	if options.HTTPClient == nil {
		options.HTTPClient = http.DefaultClient
	}

	client := &httpClient{
		serverURL:        serverURL,
		options:          options,
		httpClient:       options.HTTPClient,
		eventHandlers:    make([]EventHandler, 0),
		requestIDCounter: 0,
		responseMap:      make(map[string]chan *protocol.JSONRPCMessage),
		connectionMethod: ConnectionMethodHTTP, // Default to HTTP
	}

	return client, nil
}

// Connect establishes a connection with the server and performs protocol initialization.
func (c *httpClient) Connect(ctx context.Context) error {
	// If already initialized, return
	if c.initialized {
		return nil
	}

	// Determine which protocol version to use
	protocolVersion := c.options.ProtocolVersion
	if protocolVersion == ProtocolVersionAuto {
		// Default to the latest version
		protocolVersion = ProtocolVersion20250326
	}

	// For 2024 spec, we need to establish an SSE connection first
	if protocolVersion == ProtocolVersion20241105 {
		if err := c.connect2024(ctx); err != nil {
			// Try 2025 spec if 2024 fails and we're in auto mode
			if c.options.ProtocolVersion == ProtocolVersionAuto {
				slog.Info("Failed to connect using 2024 spec, trying 2025 spec")
				protocolVersion = ProtocolVersion20250326
			} else {
				return fmt.Errorf("failed to connect using 2024 spec: %w", err)
			}
		}
	}

	// For 2025 spec or if 2024 failed in auto mode
	if protocolVersion == ProtocolVersion20250326 {
		if err := c.connect2025(ctx); err != nil {
			return fmt.Errorf("failed to connect using 2025 spec: %w", err)
		}
	}

	// Send initialize request
	initParams := map[string]interface{}{
		"protocolVersion": string(c.protocolVersion),
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": c.options.Capabilities.Roots.ListChanged,
			},
			"sampling": map[string]interface{}{},
		},
		"client_info": map[string]interface{}{
			"name":    c.options.ClientInfo.Name,
			"version": c.options.ClientInfo.Version,
		},
	}

	// Send initialize request
	resp, err := c.SendRequest(ctx, "initialize", initParams)
	if err != nil {
		return fmt.Errorf("failed to send initialize request: %w", err)
	}

	// Check for errors in the response
	if resp.Error != nil {
		return fmt.Errorf("initialize request failed: %v", resp.Error)
	}

	// Send initialized notification
	if err := c.SendNotification(ctx, "notifications/initialized", nil); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	c.initialized = true
	return nil
}

// connect2024 establishes a connection using the 2024 MCP spec.
func (c *httpClient) connect2024(ctx context.Context) error {
	// Create SSE client for the 2024 spec
	c.sseClient = sse.NewClient(c.serverURL + "/sse")

	// Set default message endpoint
	c.messageEndpoint = c.serverURL + "/messages"
	c.sseEndpoint = c.serverURL + "/sse"

	// Set the protocol version and connection method
	c.protocolMutex.Lock()
	c.protocolVersion = ProtocolVersion20241105
	c.connectionMethod = ConnectionMethodSSE
	c.protocolMutex.Unlock()

	// Set up event handler for SSE messages
	connectionEstablished := make(chan struct{})

	c.sseClient.OnConnect(func(client *sse.Client) {
		slog.Info("Connected to SSE endpoint", "url", client.URL)
		// Signal that we've connected to the SSE endpoint
		select {
		case connectionEstablished <- struct{}{}:
		default:
		}
	})

	// Start a goroutine to handle SSE events
	go func() {
		err := c.sseClient.SubscribeWithContext(ctx, "", func(msg *sse.Event) {
			// Check if this is the endpoint event
			if string(msg.Event) == "endpoint" {
				// The endpoint is a plain string, not JSON
				endpointURL := string(msg.Data)

				// Process the endpoint URL safely
				c.endpointMutex.Lock()

				// Check if the URL is absolute or relative
				if endpointURL != "" {
					// Check if it's an absolute URL (starts with http:// or https://)
					if strings.HasPrefix(endpointURL, "http://") || strings.HasPrefix(endpointURL, "https://") {
						c.messageEndpoint = endpointURL
					} else {
						// It's a relative URL, so join it with the server URL
						baseURL, err := url.Parse(c.serverURL)
						if err != nil {
							slog.Error("Failed to parse server URL", "error", err)
							c.endpointMutex.Unlock()
							return
						}

						relURL, err := url.Parse(endpointURL)
						if err != nil {
							slog.Error("Failed to parse endpoint URL", "error", err)
							c.endpointMutex.Unlock()
							return
						}

						c.messageEndpoint = baseURL.ResolveReference(relURL).String()
					}
					slog.Info("Updated message endpoint", "endpoint", c.messageEndpoint)
				}
				c.endpointMutex.Unlock()

				return
			}

			// Parse the event data as a JSON-RPC message
			var event protocol.JSONRPCMessage
			if err := json.Unmarshal(msg.Data, &event); err != nil {
				slog.Error("Failed to parse SSE event", "error", err)
				return
			}

			// Check if this is a response to a request
			if event.ID != nil {
				requestID := fmt.Sprintf("%v", event.ID)
				c.responseMapMutex.RLock()
				responseChan, ok := c.responseMap[requestID]
				c.responseMapMutex.RUnlock()

				if ok {
					// Try to send the response, but don't block if the channel is full or closed
					select {
					case responseChan <- &event:
						slog.Debug("Sent response to channel", "id", event.ID)
					default:
						slog.Error("Failed to send response to channel", "id", event.ID)
					}
					return
				} else {
					slog.Debug("No response channel found for request", "id", event.ID)
				}
			}

			// Dispatch the event to all registered handlers
			c.dispatchEvent(&event)
		})

		if err != nil {
			slog.Error("Failed to subscribe to SSE events", "error", err)
		}
	}()

	// Wait for the connection to be established with a timeout
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-connectionEstablished:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for SSE connection")
	}
}

// connect2025 establishes a connection using the 2025 MCP spec.
func (c *httpClient) connect2025(ctx context.Context) error {
	// For 2025 spec, we use direct HTTP requests
	// Set the message endpoint
	c.messageEndpoint = c.serverURL + "/messages"

	// Make a GET request to the server to check if it's available
	req, err := http.NewRequestWithContext(ctx, "GET", c.serverURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	// Check if the server supports the 2025 spec
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned unexpected status: %d", resp.StatusCode)
	}

	// Check if the server advertises MCP support
	mcpHeader := resp.Header.Get("Mcp-Version")
	if mcpHeader != "" {
		// Server advertises MCP support, check the version
		if mcpHeader == "2025-03-26" {
			c.protocolMutex.Lock()
			c.protocolVersion = ProtocolVersion20250326
			c.protocolMutex.Unlock()
		} else if mcpHeader == "2024-11-05" {
			// Server only supports 2024 spec, but we're trying to use 2025
			if c.options.ProtocolVersion == ProtocolVersion20250326 {
				return fmt.Errorf("server only supports 2024-11-05 spec, but client requires 2025-03-26")
			}
			c.protocolMutex.Lock()
			c.protocolVersion = ProtocolVersion20241105
			c.protocolMutex.Unlock()
			// We need to use the 2024 connection method
			return c.connect2024(ctx)
		} else {
			// Unknown version
			return fmt.Errorf("server advertises unknown MCP version: %s", mcpHeader)
		}
	} else {
		// No MCP header, assume 2025 spec
		c.protocolMutex.Lock()
		c.protocolVersion = ProtocolVersion20250326
		c.protocolMutex.Unlock()
	}

	// Set the connection method
	c.protocolMutex.Lock()
	c.connectionMethod = ConnectionMethodHTTP
	c.protocolMutex.Unlock()

	return nil
}

// Close closes the client connection.
func (c *httpClient) Close(ctx context.Context) error {
	// If not initialized, return
	if !c.initialized {
		return nil
	}

	// Close SSE client if it exists
	if c.sseClient != nil {
		c.sseClient.Unsubscribe(nil)
	}

	// Clear response map
	c.responseMapMutex.Lock()
	for id, ch := range c.responseMap {
		close(ch)
		delete(c.responseMap, id)
	}
	c.responseMapMutex.Unlock()

	c.initialized = false
	return nil
}

// IsInitialized returns whether the client has been initialized.
func (c *httpClient) IsInitialized() bool {
	return c.initialized
}

// GetSessionID returns the current session ID, if any.
func (c *httpClient) GetSessionID() string {
	return c.sessionID
}

// GetProtocolVersion returns the negotiated protocol version.
func (c *httpClient) GetProtocolVersion() ProtocolVersion {
	c.protocolMutex.RLock()
	defer c.protocolMutex.RUnlock()
	return c.protocolVersion
}

// GetConnectionMethod returns the connection method being used (SSE or HTTP).
func (c *httpClient) GetConnectionMethod() ConnectionMethod {
	c.protocolMutex.RLock()
	defer c.protocolMutex.RUnlock()
	return c.connectionMethod
}

// generateRequestID generates a unique request ID.
func (c *httpClient) generateRequestID() string {
	c.requestIDMutex.Lock()
	defer c.requestIDMutex.Unlock()

	c.requestIDCounter++
	return fmt.Sprintf("%s-%d", uuid.New().String()[:8], c.requestIDCounter)
}

// SendRequest sends a request to the server and waits for a response.
func (c *httpClient) SendRequest(ctx context.Context, method string, params interface{}) (*protocol.JSONRPCMessage, error) {
	if !c.initialized && method != "initialize" {
		return nil, fmt.Errorf("client not initialized")
	}

	// Generate a unique request ID
	requestID := c.generateRequestID()

	// Create a JSON-RPC message
	request := protocol.JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      requestID,
		Method:  method,
		Params:  params,
	}

	// Create a channel to receive the response
	responseChan := make(chan *protocol.JSONRPCMessage, 1)

	// Register the response channel in the map BEFORE sending the request
	// to avoid race conditions where the response comes back before we're listening
	c.responseMapMutex.Lock()
	c.responseMap[requestID] = responseChan
	c.responseMapMutex.Unlock()

	// Clean up the response channel when we're done
	defer func() {
		c.responseMapMutex.Lock()
		delete(c.responseMap, requestID)
		c.responseMapMutex.Unlock()
		close(responseChan)
	}()

	// Marshal the request to JSON
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Get the message endpoint safely
	c.endpointMutex.RLock()
	endpoint := c.messageEndpoint
	c.endpointMutex.RUnlock()

	// Create a new HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	// Add session ID if we have one
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check if this is the initialize response and extract session ID if present
	if method == "initialize" && resp.Header.Get("Mcp-Session-Id") != "" {
		c.sessionID = resp.Header.Get("Mcp-Session-Id")
	}

	// Check if we're using SSE for responses
	c.protocolMutex.RLock()
	usingSse := c.connectionMethod == ConnectionMethodSSE
	c.protocolMutex.RUnlock()

	// Handle different response types based on the protocol version and content type
	if usingSse && resp.StatusCode == http.StatusAccepted {
		// For SSE transport, the response will come through the SSE channel
		// We already registered the response channel in the map above
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case response := <-responseChan:
			return response, nil
		case <-time.After(30 * time.Second):
			return nil, fmt.Errorf("timeout waiting for response")
		}
	} else if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
		// For direct HTTP response
		if resp.Header.Get("Content-Type") == "application/json" {
			var response protocol.JSONRPCMessage
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			return &response, nil
		}

		// For 202 Accepted with no body, return a success response
		if resp.StatusCode == http.StatusAccepted {
			return &protocol.JSONRPCMessage{
				JSONRPC: "2.0",
				ID:      requestID,
				Result:  true,
			}, nil
		}
	}

	// If we get here, something went wrong
	return nil, fmt.Errorf("unexpected response: %d", resp.StatusCode)
}

// SendNotification sends a notification to the server without waiting for a response.
func (c *httpClient) SendNotification(ctx context.Context, method string, params interface{}) error {
	if !c.initialized && method != "notifications/initialized" {
		return fmt.Errorf("client not initialized")
	}

	// Create a JSON-RPC message
	notification := protocol.JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	// Marshal the notification to JSON
	reqBody, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	// Get the message endpoint safely
	c.endpointMutex.RLock()
	endpoint := c.messageEndpoint
	c.endpointMutex.RUnlock()

	// Create a new HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Add session ID if we have one
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}

	// Send the notification
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response status: %d", resp.StatusCode)
	}

	return nil
}

// AddEventHandler adds an event handler for server-sent events.
func (c *httpClient) AddEventHandler(handler EventHandler) {
	c.handlersMutex.Lock()
	defer c.handlersMutex.Unlock()
	c.eventHandlers = append(c.eventHandlers, handler)
}

// RemoveEventHandler removes an event handler.
func (c *httpClient) RemoveEventHandler(handler EventHandler) {
	c.handlersMutex.Lock()
	defer c.handlersMutex.Unlock()

	for i, h := range c.eventHandlers {
		if h == handler {
			c.eventHandlers = append(c.eventHandlers[:i], c.eventHandlers[i+1:]...)
			break
		}
	}
}

// dispatchEvent dispatches an event to all registered handlers.
func (c *httpClient) dispatchEvent(event *protocol.JSONRPCMessage) {
	c.handlersMutex.RLock()
	handlers := make([]EventHandler, len(c.eventHandlers))
	copy(handlers, c.eventHandlers)
	c.handlersMutex.RUnlock()

	for _, handler := range handlers {
		go handler.HandleEvent(event)
	}
}
