package server

//
//import (
//	"bytes"
//	"encoding/json"
//	"github.com/traego/scaled-mcp/pkg/protocol"
//	"net/http"
//	"net/http/httptest"
//	"testing"
//
//	"github.com/stretchr/testify/assert"
//	"github.com/stretchr/testify/require"
//	"github.com/traego/scaled-mcp/pkg/config"
//)
//
//func TestInitializeRequest(t *testing.T) {
//	// Create a new MCP server with test configuration
//	cfg := config.DefaultConfig()
//	server, err := NewMcpServer(cfg, WithServerInfo("TestServer", "1.0.0"))
//	require.NoError(t, err)
//
//	// Create a test router
//	err = server.setupRoutes()
//	require.NoError(t, err)
//
//	// Create a test server
//	ts := httptest.NewServer(server.GetRouter())
//	defer ts.Close()
//
//	// Test cases
//	testCases := []struct {
//		name           string
//		request        protocol.JSONRPCMessage
//		expectedStatus int
//		validateFunc   func(*testing.T, *http.Response, []byte)
//	}{
//		{
//			name: "Valid Initialize Request",
//			request: protocol.JSONRPCMessage{
//				JSONRPC: "2.0",
//				ID:      1,
//				Method:  "initialize",
//				Params: protocol.InitializeParams{
//					ProtocolVersion: "2024-11-05",
//					Capabilities: protocol.ClientCapabilities{
//						Roots: &protocol.RootsClientCapability{
//							ListChanged: true,
//						},
//						Sampling: &protocol.SamplingClientCapability{},
//					},
//					ClientInfo: protocol.ClientInfo{
//						Name:    "TestClient",
//						Version: "1.0.0",
//					},
//				},
//			},
//			expectedStatus: http.StatusOK,
//			validateFunc: func(t *testing.T, resp *http.Response, body []byte) {
//				// Parse the response
//				var response protocol.JSONRPCMessage
//				err := json.Unmarshal(body, &response)
//				require.NoError(t, err)
//
//				// Validate the response
//				assert.Equal(t, "2.0", response.JSONRPC)
//				assert.Equal(t, float64(1), response.ID)
//				assert.Nil(t, response.Error)
//				assert.NotNil(t, response.Result)
//
//				// Parse the result
//				resultMap, ok := response.Result.(map[string]interface{})
//				require.True(t, ok)
//
//				// Validate the result fields
//				assert.Equal(t, "2024-11-05", resultMap["protocolVersion"])
//				assert.NotNil(t, resultMap["capabilities"])
//
//				serverInfo, ok := resultMap["serverInfo"].(map[string]interface{})
//				require.True(t, ok)
//				assert.Equal(t, "TestServer", serverInfo["name"])
//				assert.Equal(t, "1.0.0", serverInfo["version"])
//			},
//		},
//		{
//			name: "Unsupported Protocol Version",
//			request: protocol.JSONRPCMessage{
//				JSONRPC: "2.0",
//				ID:      2,
//				Method:  "initialize",
//				Params: protocol.InitializeParams{
//					ProtocolVersion: "unsupported-version",
//					Capabilities: protocol.ClientCapabilities{
//						Roots: &protocol.RootsClientCapability{
//							ListChanged: true,
//						},
//					},
//					ClientInfo: protocol.ClientInfo{
//						Name:    "TestClient",
//						Version: "1.0.0",
//					},
//				},
//			},
//			expectedStatus: http.StatusOK,
//			validateFunc: func(t *testing.T, resp *http.Response, body []byte) {
//				// Parse the response
//				var response protocol.JSONRPCMessage
//				err := json.Unmarshal(body, &response)
//				require.NoError(t, err)
//
//				// Validate the response
//				assert.Equal(t, "2.0", response.JSONRPC)
//				assert.Equal(t, float64(2), response.ID)
//				assert.NotNil(t, response.Error)
//				assert.Nil(t, response.Result)
//
//				// Parse the error
//				errorMap, ok := response.Error.(map[string]interface{})
//				require.True(t, ok)
//
//				// Validate the error fields
//				assert.Equal(t, float64(-32602), errorMap["code"])
//				assert.Equal(t, "Unsupported protocol version", errorMap["message"])
//
//				// Check that supported versions are included
//				data, ok := errorMap["data"].(map[string]interface{})
//				require.True(t, ok)
//				versions, ok := data["supportedVersions"].([]interface{})
//				require.True(t, ok)
//				assert.Contains(t, versions, "2024-11-05")
//				assert.Contains(t, versions, "2025-03")
//			},
//		},
//	}
//
//	// Run test cases
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			// Serialize the request
//			requestBody, err := json.Marshal(tc.request)
//			require.NoError(t, err)
//
//			// Create a new request
//			req, err := http.NewRequest("POST", ts.URL+"/mcp", bytes.NewBuffer(requestBody))
//			require.NoError(t, err)
//			req.Header.Set("Content-Type", "application/json")
//			req.Header.Set("Accept", "application/json, text/event-stream")
//
//			// Send the request
//			client := &http.Client{}
//			resp, err := client.Do(req)
//			require.NoError(t, err)
//			defer resp.Body.Close()
//
//			// Check the status code
//			assert.Equal(t, tc.expectedStatus, resp.StatusCode)
//
//			// Read the response body
//			var body bytes.Buffer
//			_, err = body.ReadFrom(resp.Body)
//			require.NoError(t, err)
//
//			// Validate the response
//			tc.validateFunc(t, resp, body.Bytes())
//		})
//	}
//}
//
//func TestBatchRequests(t *testing.T) {
//	// Create a new MCP server with test configuration
//	cfg := config.DefaultConfig()
//	server, err := NewMcpServer(cfg)
//	require.NoError(t, err)
//
//	// Create a test router
//	err = server.setupRoutes()
//	require.NoError(t, err)
//
//	// Create a test server
//	ts := httptest.NewServer(server.GetRouter())
//	defer ts.Close()
//
//	// Create a batch request with initialize and another request
//	requests := []protocol.JSONRPCMessage{
//		{
//			JSONRPC: "2.0",
//			ID:      1,
//			Method:  "initialize",
//			Params: protocol.InitializeParams{
//				ProtocolVersion: "2024-11-05",
//				Capabilities: protocol.ClientCapabilities{
//					Roots: &protocol.RootsClientCapability{
//						ListChanged: true,
//					},
//				},
//				ClientInfo: protocol.ClientInfo{
//					Name:    "TestClient",
//					Version: "1.0.0",
//				},
//			},
//		},
//		{
//			JSONRPC: "2.0",
//			ID:      2,
//			Method:  "someOtherMethod",
//			Params:  map[string]interface{}{"param1": "value1"},
//		},
//	}
//
//	// Serialize the request
//	requestBody, err := json.Marshal(requests)
//	require.NoError(t, err)
//
//	// Create a new request
//	req, err := http.NewRequest("POST", ts.URL+"/mcp", bytes.NewBuffer(requestBody))
//	require.NoError(t, err)
//	req.Header.Set("Content-Type", "application/json")
//	req.Header.Set("Accept", "application/json")
//
//	// Send the request
//	client := &http.Client{}
//	resp, err := client.Do(req)
//	require.NoError(t, err)
//	defer resp.Body.Close()
//
//	// Check the status code
//	assert.Equal(t, http.StatusOK, resp.StatusCode)
//
//	// Read the response body
//	var body bytes.Buffer
//	_, err = body.ReadFrom(resp.Body)
//	require.NoError(t, err)
//
//	// Parse the response
//	var responses []protocol.JSONRPCMessage
//	err = json.Unmarshal(body.Bytes(), &responses)
//	require.NoError(t, err)
//
//	// Validate that we got two responses
//	assert.Len(t, responses, 2)
//
//	// Validate the first response (initialize)
//	assert.Equal(t, "2.0", responses[0].JSONRPC)
//	assert.Equal(t, float64(1), responses[0].ID)
//	assert.Nil(t, responses[0].Error)
//	assert.NotNil(t, responses[0].Result)
//
//	// Validate the second response
//	assert.Equal(t, "2.0", responses[1].JSONRPC)
//	assert.Equal(t, float64(2), responses[1].ID)
//	assert.Nil(t, responses[1].Error)
//	assert.NotNil(t, responses[1].Result)
//}
//
//func TestNotificationsOnly(t *testing.T) {
//	// Create a new MCP server with test configuration
//	cfg := config.DefaultConfig()
//	server, err := NewMcpServer(cfg)
//	require.NoError(t, err)
//
//	// Create a test router
//	err = server.setupRoutes()
//	require.NoError(t, err)
//
//	// Create a test server
//	ts := httptest.NewServer(server.GetRouter())
//	defer ts.Close()
//
//	// Create a notification (no ID)
//	notification := protocol.JSONRPCMessage{
//		JSONRPC: "2.0",
//		Method:  "someNotification",
//		Params:  map[string]interface{}{"param1": "value1"},
//	}
//
//	// Serialize the request
//	requestBody, err := json.Marshal(notification)
//	require.NoError(t, err)
//
//	// Create a new request
//	req, err := http.NewRequest("POST", ts.URL+"/mcp", bytes.NewBuffer(requestBody))
//	require.NoError(t, err)
//	req.Header.Set("Content-Type", "application/json")
//	req.Header.Set("Accept", "application/json")
//
//	// Send the request
//	client := &http.Client{}
//	resp, err := client.Do(req)
//	require.NoError(t, err)
//	defer resp.Body.Close()
//
//	// For notifications, we should get a 202 Accepted with no body
//	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
//
//	// Read the response body - should be empty
//	var body bytes.Buffer
//	_, err = body.ReadFrom(resp.Body)
//	require.NoError(t, err)
//	assert.Empty(t, body.String())
//}
//
//func TestSSEPreference(t *testing.T) {
//	// Test cases for SSE preference
//	testCases := []struct {
//		name           string
//		preferSSE      bool
//		expectedHeader string
//	}{
//		{
//			name:           "Prefer JSON Response",
//			preferSSE:      false,
//			expectedHeader: "application/json",
//		},
//		{
//			name:           "Prefer SSE Response",
//			preferSSE:      true,
//			expectedHeader: "text/event-stream",
//		},
//	}
//
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			// Create a new MCP server with test configuration
//			cfg := config.DefaultConfig()
//			cfg.EnableSSE = true // Enable SSE for these tests
//			server, err := NewMcpServer(cfg, WithPreferSSE(tc.preferSSE))
//			require.NoError(t, err)
//
//			// Create a test router
//			err = server.setupRoutes()
//			require.NoError(t, err)
//
//			// Create a test server
//			ts := httptest.NewServer(server.GetRouter())
//			defer ts.Close()
//
//			// Create a request
//			request := protocol.JSONRPCMessage{
//				JSONRPC: "2.0",
//				ID:      1,
//				Method:  "initialize",
//				Params: protocol.InitializeParams{
//					ProtocolVersion: "2025-03-26",
//					Capabilities: protocol.ClientCapabilities{
//						Roots: &protocol.RootsClientCapability{
//							ListChanged: true,
//						},
//					},
//					ClientInfo: protocol.ClientInfo{
//						Name:    "TestClient",
//						Version: "1.0.0",
//					},
//				},
//			}
//
//			// Serialize the request
//			requestBody, err := json.Marshal(request)
//			require.NoError(t, err)
//
//			// Create a new request
//			req, err := http.NewRequest("POST", ts.URL+"/mcp", bytes.NewBuffer(requestBody))
//			require.NoError(t, err)
//			req.Header.Set("Content-Type", "application/json")
//			req.Header.Set("Accept", "application/json, text/event-stream")
//
//			// Send the request
//			client := &http.Client{}
//			resp, err := client.Do(req)
//			require.NoError(t, err)
//			defer resp.Body.Close()
//
//			// Check the Content-Type header
//			contentType := resp.Header.Get("Content-Type")
//			assert.Contains(t, contentType, tc.expectedHeader)
//
//			// If it's SSE, we need to close the connection manually
//			if tc.preferSSE {
//				// Just read a bit of the response to verify it's SSE
//				var body bytes.Buffer
//				_, err = body.ReadFrom(resp.Body)
//				require.NoError(t, err)
//				assert.Contains(t, body.String(), "event: connected")
//			}
//		})
//	}
//}
