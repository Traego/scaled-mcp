package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	protocolVersion := c.options.ProtocolVersion
	if protocolVersion == protocol.ProtocolVersionAuto {
		// Try to detect the server's protocol version
		req, err := http.NewRequestWithContext(ctx, "GET", c.serverURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to connect to server: %w", err)
		}
		defer resp.Body.Close()

		// Check if the server advertises MCP support
		mcpHeader := resp.Header.Get("Mcp-Version")
		switch mcpHeader {
		case string(protocol.ProtocolVersion20250326):
			protocolVersion = protocol.ProtocolVersion20250326
		case string(protocol.ProtocolVersion20241105):
			protocolVersion = protocol.ProtocolVersion20241105
		case "":
			protocolVersion = protocol.ProtocolVersion20250326
		default:
			protocolVersion = protocol.ProtocolVersion20250326
		}
	}

	// Set the protocol version
	c.protocolMutex.Lock()
	c.protocolVersion = protocolVersion
	c.protocolMutex.Unlock()

	// Connect using the appropriate protocol version
	var err error
	if protocolVersion == protocol.ProtocolVersion20241105 {
		err = c.connect2024(ctx)
	} else {
		err = c.connect2025(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to connect using %s spec: %w", protocolVersion, err)
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

	resp, err := c.SendRequest(ctx, "initialize", initParams)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	if resp.Error != nil {
		// Handle error response - Error is an interface{} so we need to extract the message
		if errObj, ok := resp.Error.(map[string]interface{}); ok {
			if msg, ok := errObj["message"].(string); ok {
				return fmt.Errorf("initialize failed: %s", msg)
			}
		}
		return fmt.Errorf("initialize failed: %v", resp.Error)
	}

	c.initialized = true
	return nil
}

// connect2024 establishes a connection using the 2024 MCP spec.
func (c *httpClient) connect2024(ctx context.Context) error {
	// Set default message endpoint
	c.messageEndpoint = c.serverURL + "/messages"
	c.sseEndpoint = c.serverURL + "/sse"

	// Set the protocol version and connection method
	c.protocolMutex.Lock()
	c.protocolVersion = protocol.ProtocolVersion20241105
	c.connectionMethod = ConnectionMethodSSE
	c.protocolMutex.Unlock()

	// Create a context with cancel for the SSE connection
	sseCtx, cancel := context.WithCancel(ctx)
	c.cancelSSE = cancel

	// Create a request for the SSE endpoint
	req, err := http.NewRequestWithContext(sseCtx, http.MethodGet, c.sseEndpoint, nil)
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

// connect2025 establishes a connection using the 2025 MCP spec.
func (c *httpClient) connect2025(ctx context.Context) error {
	// For 2025 spec, we use direct HTTP requests
	// Set the message endpoint
	c.mcpEndpoint = c.serverURL + "/mcp"
	c.sseEndpoint = c.serverURL + "/mcp"
	c.messageEndpoint = c.serverURL + "/mcp"

	// Try to make a 2025-style initialize call first
	initializeReq := &protocol.JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": protocol.ProtocolVersion20250326,
			"clientInfo": map[string]string{
				"name":    c.options.ClientInfo.Name,
				"version": c.options.ClientInfo.Version,
			},
			"capabilities": map[string]interface{}{
				"roots": map[string]interface{}{
					"listChanged": c.options.Capabilities.Roots.ListChanged,
				},
			},
		},
	}

	// Convert the request to JSON
	reqBody, err := json.Marshal(initializeReq)
	if err != nil {
		return fmt.Errorf("failed to marshal initialize request: %w", err)
	}

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.mcpEndpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set the content type
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	// If we get a 404, the server might be a 2024 server
	if resp.StatusCode == http.StatusNotFound {
		// Fall back to 2024 protocol
		c.protocolMutex.Lock()
		c.protocolVersion = protocol.ProtocolVersion20241105
		c.protocolMutex.Unlock()
		return c.connect2024(ctx)
	}

	// For any other error status, return an error
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned unexpected status: %d", resp.StatusCode)
	}

	sessionID := resp.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		return fmt.Errorf("mcp session ID header not returned from server")
	}

	c.sessionIdMutex.Lock()
	c.sessionID = sessionID
	c.sessionIdMutex.Unlock()

	// Read the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the response
	var initResp protocol.JSONRPCMessage
	if err := json.Unmarshal(respBody, &initResp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for errors
	if initResp.Error != nil {
		return fmt.Errorf("server returned error: %v", initResp.Error)
	}

	// Successfully connected using 2025 protocol
	c.protocolMutex.Lock()
	c.protocolVersion = protocol.ProtocolVersion20250326
	c.connectionMethod = ConnectionMethodHTTP
	c.protocolMutex.Unlock()

	return nil
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
			// Check if the response contains an error
			if response.Error != nil {
				// Try to convert the error to a JsonRpcError
				errData, err := json.Marshal(response.Error)
				if err == nil {
					var jsonRpcErr protocol.JsonRpcError
					if err := json.Unmarshal(errData, &jsonRpcErr); err == nil {
						// Successfully converted to JsonRpcError
						jsonRpcErr.ID = response.ID
						return response, &jsonRpcErr
					}
				}

				// Fallback if conversion fails
				if errObj, ok := response.Error.(map[string]interface{}); ok {
					if msg, ok := errObj["message"].(string); ok {
						code := -32000 // Default server error code
						if c, ok := errObj["code"].(float64); ok {
							code = int(c)
						}
						return response, protocol.NewError(code, msg, nil, response.ID)
					}
				}
				return response, fmt.Errorf("JSON-RPC error: %v", response.Error)
			}
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

			// Check if the response contains an error
			if response.Error != nil {
				// Try to convert the error to a JsonRpcError
				errData, err := json.Marshal(response.Error)
				if err == nil {
					var jsonRpcErr protocol.JsonRpcError
					if err := json.Unmarshal(errData, &jsonRpcErr); err == nil {
						// Successfully converted to JsonRpcError
						jsonRpcErr.ID = response.ID
						return &response, &jsonRpcErr
					}
				}

				// Fallback if conversion fails
				if errObj, ok := response.Error.(map[string]interface{}); ok {
					if msg, ok := errObj["message"].(string); ok {
						code := -32000 // Default server error code
						if c, ok := errObj["code"].(float64); ok {
							code = int(c)
						}
						return &response, protocol.NewError(code, msg, nil, response.ID)
					}
				}
				return &response, fmt.Errorf("JSON-RPC error: %v", response.Error)
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
	c.sessionIdMutex.Lock()
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	c.sessionIdMutex.Unlock()

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
