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
	"github.com/tmaxmax/go-sse"
	"github.com/traego/scaled-mcp/pkg/protocol"
)

// httpClient implements the McpClient interface for HTTP-based MCP communication.
type httpClient struct {
	serverURL        string
	options          ClientOptions
	httpClient       *http.Client
	sessionIdMutex   sync.Mutex
	sessionID        string
	initialized      bool
	eventHandlers    []EventHandler
	handlersMutex    sync.RWMutex
	sseConnection    *sse.Connection
	mcpEndpoint      string
	messageEndpoint  string
	sseEndpoint      string
	protocolVersion  protocol.ProtocolVersion
	connectionMethod ConnectionMethod
	requestIDCounter int
	requestIDMutex   sync.Mutex
	responseMap      map[string]chan *protocol.JSONRPCMessage
	responseMapMutex sync.RWMutex
	endpointMutex    sync.RWMutex
	protocolMutex    sync.RWMutex
	cancelSSE        context.CancelFunc
}

// newHTTPClient creates a new HTTP-based MCP client.
func newHTTPClient(serverURL string, options ClientOptions) (*httpClient, error) {
	// Use the provided HTTP client or create a default one
	if options.HTTPClient == nil {
		options.HTTPClient = &http.Client{
			Timeout: 30 * time.Second,
		}
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
	// Determine which protocol version to use
	protocolVersion := c.determineProtocolVersion(ctx)

	// Set up the transport based on protocol version
	if protocolVersion == protocol.ProtocolVersion20241105 {
		// For 2024 spec, we need to set up the SSE connection
		slog.Info("Using 2024-11-05 spec")
		c.protocolMutex.Lock()
		c.protocolVersion = protocol.ProtocolVersion20241105
		c.protocolMutex.Unlock()

		// Set the message endpoint
		c.messageEndpoint = c.serverURL + "/messages"

		c.connectionMethod = ConnectionMethodSSE

		// 2024 spec requires special SSE setup
		if err := c.setupSSE(ctx, c.serverURL+"/sse"); err != nil {
			return fmt.Errorf("failed to set up SSE connection: %w", err)
		}
	} else {
		// For 2025 spec, we use the /mcp endpoint
		slog.Info("Using 2025-03-26 spec")
		c.protocolMutex.Lock()
		c.protocolVersion = protocol.ProtocolVersion20250326
		c.protocolMutex.Unlock()

		// Set the endpoint URLs
		c.mcpEndpoint = c.serverURL + "/mcp"
		c.sseEndpoint = c.serverURL + "/mcp"
		c.messageEndpoint = c.serverURL + "/mcp"

		// Determine the connection method - 2025 spec supports direct HTTP
		c.connectionMethod = ConnectionMethodHTTP

		// If UseSSEForEvents is enabled, try to establish an SSE connection as well
		if c.options.UseSSEForEvents {
			slog.Info("Setting up SSE connection for events (2025 protocol)")
			// Try to establish SSE connection - but don't fail if it doesn't work
			// We still have HTTP as fallback
			if err := c.setupSSE(ctx, c.sseEndpoint); err != nil {
				slog.Warn("Failed to set up SSE connection for events, will use HTTP only", "error", err)
				// We don't return error here - 2025 can work fine without SSE
			} else {
				slog.Info("Successfully established SSE connection for events")
			}
		}
	}

	// Now that the transport is set up, send the initialize request
	if err := c.sendInitializeRequest(ctx); err != nil {
		// If initialization fails with a 404 for the 2025 spec, try falling back to 2024
		if protocolVersion == protocol.ProtocolVersion20250326 &&
			isHTTPNotFoundError(err) {

			slog.Info("Failed to initialize with 2025 spec (404 error), falling back to 2024 spec")

			// Reset and try with 2024 spec instead
			c.protocolMutex.Lock()
			c.protocolVersion = protocol.ProtocolVersion20241105
			c.protocolMutex.Unlock()

			// Clean up any existing SSE connection
			if c.cancelSSE != nil {
				c.cancelSSE()
				c.cancelSSE = nil
			}

			// Set up the 2024 transport
			if err := c.setupSSE(ctx, c.serverURL+"/sse"); err != nil {
				return fmt.Errorf("failed to set up SSE connection during fallback: %w", err)
			}

			// Set the message endpoint for 2024 protocol
			c.messageEndpoint = c.serverURL + "/messages"

			// Try initialization again
			if err := c.sendInitializeRequest(ctx); err != nil {
				return err
			}
		} else {
			// For any other error, just return it
			return err
		}
	}

	c.initialized = true
	return nil
}

// isHTTPNotFoundError checks if an error is a 404 Not Found error
func isHTTPNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	// Try to extract HTTP status code from error message
	// This is a simple heuristic and might need to be improved
	return strings.Contains(err.Error(), "404") ||
		strings.Contains(err.Error(), "not found")
}

// determineProtocolVersion tries to detect the server's protocol version
func (c *httpClient) determineProtocolVersion(ctx context.Context) protocol.ProtocolVersion {
	protocolVersion := c.options.ProtocolVersion

	if protocolVersion != protocol.ProtocolVersionAuto {
		return protocolVersion
	}

	// Try to detect the server's protocol version
	req, err := http.NewRequestWithContext(ctx, "GET", c.serverURL, nil)
	if err != nil {
		slog.Error("Failed to create HTTP request for protocol detection", "error", err)
		return protocol.ProtocolVersion20250326 // Default to latest if detection fails
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("Failed to connect to server for protocol detection", "error", err)
		return protocol.ProtocolVersion20250326 // Default to latest if detection fails
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	// Check if the server advertises MCP support
	mcpHeader := resp.Header.Get("Mcp-Version")
	switch mcpHeader {
	case string(protocol.ProtocolVersion20250326):
		return protocol.ProtocolVersion20250326
	case string(protocol.ProtocolVersion20241105):
		return protocol.ProtocolVersion20241105
	default:
		return protocol.ProtocolVersion20250326 // Default to latest if not recognized
	}
}

// sendInitializeRequest sends the initialize request to the server
func (c *httpClient) sendInitializeRequest(ctx context.Context) error {
	slog.Info("Sending initialize request")

	// Create initialization parameters
	initParams := c.createInitializeParams()

	// Send initialize request
	resp, err := c.SendRequest(ctx, "initialize", initParams)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	if resp.Error != nil {
		return c.extractJSONRPCError("initialize failed", resp.Error)
	}

	// Extract and save session ID from result if available
	if resp.Result != nil {
		if resultMap, ok := resp.Result.(map[string]interface{}); ok {
			if sessionID, ok := resultMap["sessionId"].(string); ok && sessionID != "" {
				c.sessionIdMutex.Lock()
				c.sessionID = sessionID
				c.sessionIdMutex.Unlock()
				slog.Info("Received session ID from initialize response", "sessionId", sessionID)
			}
		}
	}

	return nil
}

// createInitializeParams creates the parameters for the initialize request
func (c *httpClient) createInitializeParams() map[string]interface{} {
	return map[string]interface{}{
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
}

// setupSSE establishes an SSE connection to the given endpoint
func (c *httpClient) setupSSE(ctx context.Context, endpoint string) error {
	slog.Info("Setting up SSE connection", "endpoint", endpoint)

	// Create a context with cancel for the SSE connection
	sseCtx, cancel := context.WithCancel(ctx)
	c.cancelSSE = cancel

	// Create a request for the SSE endpoint
	req, err := http.NewRequestWithContext(sseCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create SSE request: %w", err)
	}

	// Create a new SSE connection
	c.sseConnection = sse.NewConnection(req)

	// Set up a channel to signal when connection is established
	connectionEstablished := make(chan struct{})
	connectionError := make(chan error, 1)

	// Subscribe to all events
	c.sseConnection.SubscribeToAll(func(event sse.Event) {
		// Signal that we've received an event (connection established)
		select {
		case connectionEstablished <- struct{}{}:
		default:
			// Already signaled
		}

		// Check if this is the endpoint event
		if event.Type == "endpoint" {
			// The endpoint is a plain string, not JSON
			endpointURL := event.Data

			// Process the endpoint URL safely
			c.endpointMutex.Lock()

			// Check if the URL is empty
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

		// For all other events, parse as JSON-RPC
		var message protocol.JSONRPCMessage
		if err := json.Unmarshal([]byte(event.Data), &message); err != nil {
			slog.Error("Failed to parse SSE event", "error", err)
			return
		}

		// Check if this is a response to a request
		if message.ID != nil {
			requestID := fmt.Sprintf("%v", message.ID)
			c.responseMapMutex.RLock()
			responseChan, ok := c.responseMap[requestID]
			c.responseMapMutex.RUnlock()

			if ok {
				// Try to send the response, but don't block if the channel is full or closed
				select {
				case responseChan <- &message:
					slog.Debug("Sent response to channel", "id", message.ID)
				default:
					slog.Error("Failed to send response to channel", "id", message.ID)
				}
				return
			} else {
				slog.Debug("No response channel found for request", "id", message.ID)
			}
		}

		// Dispatch the event to all registered handlers
		c.dispatchEvent(&message)
	})

	// Start a goroutine to handle the connection
	go func() {
		err := c.sseConnection.Connect()
		if err != nil {
			slog.Error("SSE connection error", "error", err)
			select {
			case connectionError <- err:
			case <-sseCtx.Done():
			default:
				// Channel already closed or full
			}
		}
	}()

	// Wait for the connection to be established with a timeout
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-connectionError:
		return fmt.Errorf("failed to establish SSE connection: %w", err)
	case <-connectionEstablished:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for SSE connection")
	}
}

// Close closes the connection to the server.
func (c *httpClient) Close(ctx context.Context) error {
	if !c.initialized {
		return nil
	}

	// Cancel the SSE context if it exists
	if c.cancelSSE != nil {
		c.cancelSSE()
		c.cancelSSE = nil
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

// IsInitialized returns whether the client is initialized.
func (c *httpClient) IsInitialized() bool {
	return c.initialized
}

// GetSessionID returns the current session ID, if any.
func (c *httpClient) GetSessionID() string {
	return c.sessionID
}

// AddEventHandler adds an event handler to the client.
func (c *httpClient) AddEventHandler(handler EventHandler) {
	c.handlersMutex.Lock()
	defer c.handlersMutex.Unlock()
	c.eventHandlers = append(c.eventHandlers, handler)
}

// RemoveEventHandler removes an event handler from the client.
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
	defer c.handlersMutex.RUnlock()

	for _, handler := range c.eventHandlers {
		go handler.HandleEvent(event)
	}
}

// GetProtocolVersion returns the negotiated protocol version.
func (c *httpClient) GetProtocolVersion() protocol.ProtocolVersion {
	c.protocolMutex.RLock()
	defer c.protocolMutex.RUnlock()
	return c.protocolVersion
}

// GetConnectionMethod returns the connection method used.
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

	// Create a channel for the response and register it
	responseChan, cleanup := c.registerResponseChannel(requestID)
	defer cleanup()

	// Get the message endpoint safely
	endpoint := c.getMessageEndpoint()

	// Create and send the HTTP request
	resp, err := c.sendHTTPRequest(ctx, endpoint, request)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	// Extract session ID from response if this is an initialize request
	if method == "initialize" && resp.Header.Get("Mcp-Session-Id") != "" {
		c.sessionIdMutex.Lock()
		c.sessionID = resp.Header.Get("Mcp-Session-Id")
		c.sessionIdMutex.Unlock()
	}

	// Check if we're using SSE for responses
	usingSse := c.GetConnectionMethod() == ConnectionMethodSSE

	// Process the response based on transport method and status code
	if usingSse && resp.StatusCode == http.StatusAccepted {
		return c.waitForSSEResponse(ctx, responseChan)
	} else if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
		return c.processHTTPResponse(resp, requestID)
	} else {
		// Unexpected status code
		return nil, fmt.Errorf("unexpected response status: %d", resp.StatusCode)
	}
}

// getMessageEndpoint safely retrieves the message endpoint
func (c *httpClient) getMessageEndpoint() string {
	c.endpointMutex.RLock()
	defer c.endpointMutex.RUnlock()
	return c.messageEndpoint
}

// registerResponseChannel creates and registers a response channel
func (c *httpClient) registerResponseChannel(requestID string) (chan *protocol.JSONRPCMessage, func()) {
	responseChan := make(chan *protocol.JSONRPCMessage, 1)

	c.responseMapMutex.Lock()
	c.responseMap[requestID] = responseChan
	c.responseMapMutex.Unlock()

	// Return the channel and a cleanup function
	cleanup := func() {
		c.responseMapMutex.Lock()
		delete(c.responseMap, requestID)
		c.responseMapMutex.Unlock()
		close(responseChan)
	}

	return responseChan, cleanup
}

// sendHTTPRequest creates and sends an HTTP request with the given payload
func (c *httpClient) sendHTTPRequest(ctx context.Context, endpoint string, payload interface{}) (*http.Response, error) {
	// Marshal the request to JSON
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create a new HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	// Add session ID if we have one
	c.sessionIdMutex.Lock()
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	c.sessionIdMutex.Unlock()

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return resp, nil
}

// waitForSSEResponse waits for a response from the SSE channel
func (c *httpClient) waitForSSEResponse(ctx context.Context, responseChan chan *protocol.JSONRPCMessage) (*protocol.JSONRPCMessage, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case response := <-responseChan:
		if response.Error != nil {
			return response, c.extractJSONRPCError("JSON-RPC error", response.Error)
		}
		return response, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

// processHTTPResponse processes a direct HTTP response
func (c *httpClient) processHTTPResponse(resp *http.Response, requestID string) (*protocol.JSONRPCMessage, error) {
	if resp.Header.Get("Content-Type") == "application/json" {
		var response protocol.JSONRPCMessage
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		// Check if the response contains an error
		if response.Error != nil {
			return &response, c.extractJSONRPCError("JSON-RPC error", response.Error)
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

	return nil, fmt.Errorf("unexpected content type: %s", resp.Header.Get("Content-Type"))
}

// SendNotification sends a notification to the server without waiting for a response.
func (c *httpClient) SendNotification(ctx context.Context, method string, params interface{}) error {
	if !c.initialized && method != "notifications/initialized" {
		return fmt.Errorf("client not initialized")
	}

	// Create a JSON-RPC notification message (no ID)
	notification := protocol.JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	// Get the message endpoint safely
	endpoint := c.getMessageEndpoint()

	// Create and send the HTTP request - but set Accept to application/json only
	req, err := c.createNotificationRequest(ctx, endpoint, notification)
	if err != nil {
		return err
	}

	// Send the notification
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Check response status
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response status: %d", resp.StatusCode)
	}

	return nil
}

// createNotificationRequest creates an HTTP request for a notification
func (c *httpClient) createNotificationRequest(ctx context.Context, endpoint string, notification protocol.JSONRPCMessage) (*http.Request, error) {
	// Marshal the notification to JSON
	reqBody, err := json.Marshal(notification)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notification: %w", err)
	}

	// Create a new HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers - notifications only need JSON response
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Add session ID if we have one
	c.sessionIdMutex.Lock()
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	c.sessionIdMutex.Unlock()

	return req, nil
}

// extractJSONRPCError converts a JSON-RPC error to a Go error
func (c *httpClient) extractJSONRPCError(prefix string, jsonRpcErr interface{}) error {
	// Try to convert the error to a structured format
	if errObj, ok := jsonRpcErr.(map[string]interface{}); ok {
		if msg, ok := errObj["message"].(string); ok {
			code := -32000 // Default server error code
			if c, ok := errObj["code"].(float64); ok {
				code = int(c)
			}
			return protocol.NewError(code, msg, nil, nil)
		}
	}

	return fmt.Errorf("%s: %v", prefix, jsonRpcErr)
}
