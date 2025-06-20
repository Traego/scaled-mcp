package channels

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOneWayChannelInterface(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/events", nil)
	require.NoError(t, err)

	channel := NewSSEChannel(w, r, "test-session")
	require.NotNil(t, channel)

	var oneWayChannel OneWayChannel = channel

	err = oneWayChannel.Send("test", "data")
	assert.NoError(t, err)

	err = oneWayChannel.SendEndpoint("endpoint")
	assert.NoError(t, err)

	oneWayChannel.Close()

	_, ok := <-channel.Done
	assert.False(t, ok, "Channel should be closed")
}

func TestSSEChannelImplementation(t *testing.T) {
	var _ OneWayChannel = (*SSEChannel)(nil)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/events", nil)
	require.NoError(t, err)

	channel := NewSSEChannel(w, r, "test-session")
	require.NotNil(t, channel)

	type TestObject struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	testObj := TestObject{
		Name:  "test",
		Value: 42,
	}

	err = channel.Send("complex", testObj)
	assert.NoError(t, err)

	body := w.Body.String()
	assert.Contains(t, body, "event: complex")
	assert.Contains(t, body, `"name":"test"`)
	assert.Contains(t, body, `"value":42`)
}
