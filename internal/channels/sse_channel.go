package channels

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// SSEChannel represents an SSE channel for sending events to clients
type SSEChannel struct {
	Done chan struct{} // Bidirectional channel for internal use
	w    http.ResponseWriter
	r    *http.Request
}

// NewSSEChannel creates a new SSE channel from an HTTP response writer and request
func NewSSEChannel(w http.ResponseWriter, r *http.Request, sessionId string) *SSEChannel {
	// Set a secure, HTTP-only cookie containing the session ID so reconnect handlers can retrieve it.
	http.SetCookie(w, &http.Cookie{
		Name:     "mcp_session_id",
		Value:    sessionId,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

	// Write the HTTP status before any data is written
	w.WriteHeader(http.StatusOK)

	return &SSEChannel{
		w:    w,
		r:    r,
		Done: make(chan struct{}),
	}
}

// GetDoneChannel returns a receive-only channel that signals when the SSE connection is closed
func (c *SSEChannel) GetDoneChannel() <-chan struct{} {
	return c.Done
}

// Send sends an event with the given event type and data
func (c *SSEChannel) Send(eventType string, data interface{}) error {
	// Marshal the data to JSON if it's not already a string
	var dataStr string
	switch d := data.(type) {
	case string:
		dataStr = d
	default:
		jsonData, err := json.Marshal(d)
		if err != nil {
			return fmt.Errorf("error marshaling event data: %w", err)
		}
		dataStr = string(jsonData)
	}

	// Format the event according to SSE specification
	// If eventType is provided, include the event field
	if eventType != "" {
		_, err := fmt.Fprintf(c.w, "event: %s\n", eventType)
		if err != nil {
			return fmt.Errorf("error writing event type: %w", err)
		}
	}

	// Write the data field
	_, err := fmt.Fprintf(c.w, "data: %s\n\n", dataStr)
	if err != nil {
		return fmt.Errorf("error writing event data: %w", err)
	}

	// Flush the response
	if flusher, ok := c.w.(http.Flusher); ok {
		flusher.Flush()
		return nil
	}

	return fmt.Errorf("response writer does not support flushing")
}

// SendEndpoint sends an endpoint event with the given endpoint URL
func (c *SSEChannel) SendEndpoint(endpoint string) error {
	return c.Send("endpoint", endpoint)
}

func (c *SSEChannel) Close() {
	close(c.Done)
}

var _ OneWayChannel = (*SSEChannel)(nil)
