package harness

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/traego/scaled-mcp/pkg/protocol"
)

// MCPTestClient represents a test client for the MCP server
type MCPTestClient struct {
	// Configuration
	baseURL     string
	sessionID   string
	useSSE      bool
	is2024Spec  bool
	httpClient  *http.Client
	sseMessages chan protocol.JSONRPCMessage
	sseCtx      context.Context
	sseCancel   context.CancelFunc
	sseDone     chan struct{}
	mu          sync.Mutex // Protects sessionID and other state
}

// MCPTestClientOption represents an option for the MCP test client
type MCPTestClientOption func(*MCPTestClient)

// WithBaseURL sets the base URL for the MCP server
func WithBaseURL(baseURL string) MCPTestClientOption {
	return func(c *MCPTestClient) {
		c.baseURL = baseURL
	}
}

// WithUseSSE sets whether to use SSE for receiving messages
func WithUseSSE(useSSE bool) MCPTestClientOption {
	return func(c *MCPTestClient) {
		c.useSSE = useSSE
	}
}

// With2024Spec sets whether to use the 2024 spec endpoints
func With2024Spec(is2024Spec bool) MCPTestClientOption {
	return func(c *MCPTestClient) {
		c.is2024Spec = is2024Spec
	}
}

// WithHTTPClient sets the HTTP client to use
func WithHTTPClient(httpClient *http.Client) MCPTestClientOption {
	return func(c *MCPTestClient) {
		c.httpClient = httpClient
	}
}

// NewMCPTestClient creates a new MCP test client
func NewMCPTestClient(options ...MCPTestClientOption) *MCPTestClient {
	client := &MCPTestClient{
		baseURL:     "http://localhost:8080",
		useSSE:      true,
		is2024Spec:  false,
		httpClient:  &http.Client{},
		sseMessages: make(chan protocol.JSONRPCMessage, 100),
		sseDone:     make(chan struct{}),
	}

	// Apply options
	for _, opt := range options {
		opt(client)
	}

	return client
}

// Initialize initializes a session with the MCP server
func (c *MCPTestClient) Initialize(ctx context.Context, clientInfo map[string]interface{}) (*protocol.JSONRPCMessage, error) {
	// Create initialize request
	initRequest := protocol.JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      "init-" + time.Now().Format(time.RFC3339Nano),
		Method:  "initialize",
		Params: map[string]interface{}{
			"client_info": clientInfo,
		},
	}

	// Send the request
	resp, err := c.sendRequest(ctx, initRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize session: %w", err)
	}

	// Extract session ID from response
	resultMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid initialize response: result is not a map")
	}

	sessionID, ok := resultMap["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid initialize response: session_id is not a string")
	}

	// Store session ID
	c.mu.Lock()
	c.sessionID = sessionID
	c.mu.Unlock()

	// Start SSE connection if enabled
	if c.useSSE {
		err = c.startSSE(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to start SSE connection: %w", err)
		}
	}

	return resp, nil
}

// SendMessage sends a message to the MCP server
func (c *MCPTestClient) SendMessage(ctx context.Context, message string) (*protocol.JSONRPCMessage, error) {
	c.mu.Lock()
	sessionID := c.sessionID
	c.mu.Unlock()

	if sessionID == "" {
		return nil, fmt.Errorf("no active session, call Initialize first")
	}

	// Create message request
	msgRequest := protocol.JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      "msg-" + time.Now().Format(time.RFC3339Nano),
		Method:  "message",
		Params: map[string]interface{}{
			"session_id": sessionID,
			"message":    message,
		},
	}

	// Send the request
	return c.sendRequest(ctx, msgRequest)
}

// SendBatch sends a batch of requests to the MCP server
func (c *MCPTestClient) SendBatch(ctx context.Context, messages []string) ([]protocol.JSONRPCMessage, error) {
	c.mu.Lock()
	sessionID := c.sessionID
	c.mu.Unlock()

	if sessionID == "" {
		return nil, fmt.Errorf("no active session, call Initialize first")
	}

	// Create batch request
	batch := make([]protocol.JSONRPCMessage, len(messages))
	for i, msg := range messages {
		batch[i] = protocol.JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      fmt.Sprintf("batch-%d-%s", i, time.Now().Format(time.RFC3339Nano)),
			Method:  "message",
			Params: map[string]interface{}{
				"session_id": sessionID,
				"message":    msg,
			},
		}
	}

	// Send the batch request
	batchBody, err := json.Marshal(batch)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch request: %w", err)
	}

	endpoint := c.getEndpoint()
	resp, err := c.httpClient.Post(endpoint, "application/json", bytes.NewReader(batchBody))
	if err != nil {
		return nil, fmt.Errorf("failed to send batch request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Parse the response
	var batchResult []protocol.JSONRPCMessage
	err = json.NewDecoder(resp.Body).Decode(&batchResult)
	if err != nil {
		return nil, fmt.Errorf("failed to decode batch response: %w", err)
	}

	return batchResult, nil
}

// ReceiveSSEMessage receives a message from the SSE connection
// This will block until a message is received or the context is canceled
func (c *MCPTestClient) ReceiveSSEMessage(ctx context.Context) (*protocol.JSONRPCMessage, error) {
	if !c.useSSE {
		return nil, fmt.Errorf("SSE is not enabled")
	}

	select {
	case msg := <-c.sseMessages:
		return &msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.sseDone:
		return nil, fmt.Errorf("SSE connection closed")
	}
}

// Close closes the MCP test client
func (c *MCPTestClient) Close() {
	if c.sseCancel != nil {
		c.sseCancel()
	}
}

// sendRequest sends a JSON-RPC request to the MCP server
func (c *MCPTestClient) sendRequest(ctx context.Context, request protocol.JSONRPCMessage) (*protocol.JSONRPCMessage, error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := c.getEndpoint()
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Parse the response
	var result protocol.JSONRPCMessage
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for JSON-RPC error
	if result.Error != nil {
		errorObj, ok := result.Error.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid error response: error is not a map")
		}

		code, _ := errorObj["code"].(float64)
		message, _ := errorObj["message"].(string)
		return nil, fmt.Errorf("JSON-RPC error: code=%v, message=%s", code, message)
	}

	return &result, nil
}

// startSSE starts an SSE connection to the MCP server
func (c *MCPTestClient) startSSE(ctx context.Context) error {
	c.mu.Lock()
	sessionID := c.sessionID
	c.mu.Unlock()

	if sessionID == "" {
		return fmt.Errorf("no active session, call Initialize first")
	}

	// Create a new context for the SSE connection
	c.sseCtx, c.sseCancel = context.WithCancel(context.Background())

	// Determine the SSE endpoint
	var sseEndpoint string
	if c.is2024Spec {
		sseEndpoint = fmt.Sprintf("%s/events?session_id=%s", c.baseURL, sessionID)
	} else {
		sseEndpoint = fmt.Sprintf("%s/mcp?session_id=%s", c.baseURL, sessionID)
	}

	// Create the SSE request
	req, err := http.NewRequestWithContext(c.sseCtx, "GET", sseEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// Start the SSE connection in a goroutine
	go func() {
		defer close(c.sseDone)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			slog.Error("Failed to connect to SSE", "error", err)
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			slog.Error("SSE connection failed", "status", resp.StatusCode)
			return
		}

		// Read the SSE stream
		reader := bufio.NewReader(resp.Body)
		for {
			// Check if the context is canceled
			select {
			case <-c.sseCtx.Done():
				return
			default:
				// Continue reading
			}

			// Read a line
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					slog.Info("SSE connection closed by server")
				} else {
					slog.Error("Failed to read SSE stream", "error", err)
				}
				return
			}

			// Parse the SSE event
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				var message protocol.JSONRPCMessage
				err := json.Unmarshal([]byte(data), &message)
				if err != nil {
					slog.Error("Failed to parse SSE message", "error", err, "data", data)
					continue
				}

				// Send the message to the channel
				select {
				case c.sseMessages <- message:
					// Message sent successfully
				default:
					// Channel is full, log and continue
					slog.Warn("SSE message channel is full, dropping message")
				}
			}
		}
	}()

	return nil
}

// getEndpoint returns the appropriate endpoint based on the spec version
func (c *MCPTestClient) getEndpoint() string {
	if c.is2024Spec {
		return fmt.Sprintf("%s/messages", c.baseURL)
	}
	return fmt.Sprintf("%s/mcp", c.baseURL)
}
