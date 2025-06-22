package channels

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSSEChannel_NewSSEChannel tests the creation of a new SSE channel
func TestSSEChannel_NewSSEChannel(t *testing.T) {
	// Create a test HTTP response recorder and request
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/events", nil)
	require.NoError(t, err)

	// Create a new SSE channel
	channel := NewSSEChannel(w, r, "test-session")
	require.NotNil(t, channel)

	// Verify the headers are set correctly
	headers := w.Header()
	assert.Equal(t, "text/event-stream", headers.Get("Content-Type"))
	assert.Equal(t, "no-cache", headers.Get("Cache-Control"))
	assert.Equal(t, "keep-alive", headers.Get("Connection"))
	assert.Equal(t, "Content-Type", headers.Get("Access-Control-Expose-Headers"))

	// Verify the status code is set correctly
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify the Done channel is initialized
	assert.NotNil(t, channel.Done)
}

// TestSSEChannel_GetDoneChannel tests the GetDoneChannel method
func TestSSEChannel_GetDoneChannel(t *testing.T) {
	// Create a test HTTP response recorder and request
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/events", nil)
	require.NoError(t, err)

	// Create a new SSE channel
	channel := NewSSEChannel(w, r, "test-session")
	require.NotNil(t, channel)

	// Get the done channel
	doneChannel := channel.GetDoneChannel()
	require.NotNil(t, doneChannel)

	// Close the channel and verify the done channel is closed
	channel.Close()
	_, ok := <-doneChannel
	assert.False(t, ok, "Done channel should be closed")
}

// TestSSEChannel_Send_String tests sending a string event
func TestSSEChannel_Send_String(t *testing.T) {
	// Create a test HTTP response recorder and request
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/events", nil)
	require.NoError(t, err)

	// Create a new SSE channel
	channel := NewSSEChannel(w, r, "test-session")
	require.NotNil(t, channel)

	// Send a string event
	err = channel.Send("test-event", "Hello, World!")
	require.NoError(t, err)

	// Verify the response body contains the expected event
	body := w.Body.String()
	assert.Contains(t, body, "event: test-event")
	assert.Contains(t, body, "data: Hello, World!")
	assert.Contains(t, body, "\n\n") // Make sure there's a blank line at the end
}

// TestSSEChannel_Send_Object tests sending an object event
func TestSSEChannel_Send_Object(t *testing.T) {
	// Create a test HTTP response recorder and request
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/events", nil)
	require.NoError(t, err)

	// Create a new SSE channel
	channel := NewSSEChannel(w, r, "test-session")
	require.NotNil(t, channel)

	// Create a test object
	testObj := map[string]interface{}{
		"message": "Hello, World!",
		"count":   42,
	}

	// Send an object event
	err = channel.Send("test-event", testObj)
	require.NoError(t, err)

	// Verify the response body contains the expected event
	body := w.Body.String()
	assert.Contains(t, body, "event: test-event")

	// Verify the JSON data
	expectedJSON, err := json.Marshal(testObj)
	require.NoError(t, err)
	assert.Contains(t, body, "data: "+string(expectedJSON))
	assert.Contains(t, body, "\n\n") // Make sure there's a blank line at the end
}

// TestSSEChannel_Send_NoEventType tests sending an event without an event type
func TestSSEChannel_Send_NoEventType(t *testing.T) {
	// Create a test HTTP response recorder and request
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/events", nil)
	require.NoError(t, err)

	// Create a new SSE channel
	channel := NewSSEChannel(w, r, "test-session")
	require.NotNil(t, channel)

	// Send an event without an event type
	err = channel.Send("", "Hello, World!")
	require.NoError(t, err)

	// Verify the response body contains the expected event
	body := w.Body.String()
	assert.NotContains(t, body, "event:")
	assert.Contains(t, body, "data: Hello, World!")
	assert.Contains(t, body, "\n\n") // Make sure there's a blank line at the end
}

// TestSSEChannel_SendEndpoint tests sending an endpoint event
func TestSSEChannel_SendEndpoint(t *testing.T) {
	// Create a test HTTP response recorder and request
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/events", nil)
	require.NoError(t, err)

	// Create a new SSE channel
	channel := NewSSEChannel(w, r, "test-session")
	require.NotNil(t, channel)

	// Send an endpoint event
	err = channel.SendEndpoint("https://example.com/api")
	require.NoError(t, err)

	// Verify the response body contains the expected event
	body := w.Body.String()
	assert.Contains(t, body, "event: endpoint")
	assert.Contains(t, body, "data: https://example.com/api")
	assert.Contains(t, body, "\n\n") // Make sure there's a blank line at the end
}

// TestSSEChannel_Send_MarshalError tests handling a marshal error
func TestSSEChannel_Send_MarshalError(t *testing.T) {
	// Create a test HTTP response recorder and request
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/events", nil)
	require.NoError(t, err)

	// Create a new SSE channel
	channel := NewSSEChannel(w, r, "test-session")
	require.NotNil(t, channel)

	// Create an object that will cause a marshal error (circular reference)
	type CircularRef struct {
		Self *CircularRef
	}
	circular := &CircularRef{}
	circular.Self = circular

	// Send the object and expect an error
	err = channel.Send("test-event", circular)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error marshaling event data")
}

// TestSSEChannel_Interface tests that SSEChannel implements the OneWayChannel interface
func TestSSEChannel_Interface(t *testing.T) {
	// Create a test HTTP response recorder and request
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/events", nil)
	require.NoError(t, err)

	// Create a new SSE channel
	channel := NewSSEChannel(w, r, "test-session")
	require.NotNil(t, channel)

	// Verify that the channel implements the OneWayChannel interface
	var _ OneWayChannel = channel
}

// MockResponseWriter is a mock implementation of http.ResponseWriter that doesn't support flushing
type MockResponseWriter struct {
	headers http.Header
	body    []byte
	status  int
}

func NewMockResponseWriter() *MockResponseWriter {
	return &MockResponseWriter{
		headers: make(http.Header),
	}
}

func (w *MockResponseWriter) Header() http.Header {
	return w.headers
}

func (w *MockResponseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return len(b), nil
}

func (w *MockResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}

// TestSSEChannel_Send_NoFlush tests sending an event to a response writer that doesn't support flushing
func TestSSEChannel_Send_NoFlush(t *testing.T) {
	// Create a mock response writer that doesn't implement http.Flusher
	w := NewMockResponseWriter()
	r, err := http.NewRequest("GET", "/events", nil)
	require.NoError(t, err)

	// Create a new SSE channel
	channel := &SSEChannel{
		w:    w,
		r:    r,
		Done: make(chan struct{}),
	}

	// Set headers manually since we're bypassing NewSSEChannel
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)

	// Send an event and expect an error
	err = channel.Send("test-event", "Hello, World!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support flushing")
}

// TestSSEChannel_Send_WriteError tests handling a write error
func TestSSEChannel_Send_WriteError(t *testing.T) {
	// Create a mock response writer that returns an error on write
	w := &MockErrorWriter{}
	r, err := http.NewRequest("GET", "/events", nil)
	require.NoError(t, err)

	// Create a new SSE channel
	channel := &SSEChannel{
		w:    w,
		r:    r,
		Done: make(chan struct{}),
	}

	// Send an event and expect an error
	err = channel.Send("test-event", "Hello, World!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error writing event type")

	// Test with no event type
	w.Reset()
	err = channel.Send("", "Hello, World!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error writing event data")
}

// MockErrorWriter is a mock implementation of http.ResponseWriter that returns an error on write
type MockErrorWriter struct {
	headers http.Header
}

func (w *MockErrorWriter) Header() http.Header {
	if w.headers == nil {
		w.headers = make(http.Header)
	}
	return w.headers
}

func (w *MockErrorWriter) Write([]byte) (int, error) {
	return 0, assert.AnError
}

func (w *MockErrorWriter) WriteHeader(statusCode int) {}

func (w *MockErrorWriter) Reset() {
	w.headers = make(http.Header)
}
