package main

//
//import (
//	"bytes"
//	"encoding/json"
//	"fmt"
//	"io"
//	"log"
//	"net/http"
//	"os"
//	"time"
//)
//
//// JSON-RPC request structure
//type JsonRpcRequest struct {
//	Jsonrpc string      `json:"jsonrpc"`
//	Method  string      `json:"method"`
//	Params  interface{} `json:"params,omitempty"`
//	ID      string      `json:"id"`
//}
//
//// JSON-RPC response structure
//type JsonRpcResponse struct {
//	Jsonrpc string          `json:"jsonrpc"`
//	Result  json.RawMessage `json:"result,omitempty"`
//	Error   *JsonRpcError   `json:"error,omitempty"`
//	ID      string          `json:"id"`
//}
//
//// JSON-RPC error structure
//type JsonRpcError struct {
//	Code    int         `json:"code"`
//	Message string      `json:"message"`
//	Data    interface{} `json:"data,omitempty"`
//}
//
//// Initialize params
//type InitializeParams struct {
//	ClientInfo struct {
//		Name    string `json:"name"`
//		Version string `json:"version"`
//	} `json:"clientInfo"`
//}
//
//// Tool invocation params
//type InvokeToolParams struct {
//	Name   string                 `json:"name"`
//	Params map[string]interface{} `json:"params"`
//}
//
//// Send a request to the MCP server
//func sendRequest(url string, sessionID string, request JsonRpcRequest) (*JsonRpcResponse, error) {
//	// Marshal the request to JSON
//	requestJSON, err := json.Marshal(request)
//	if err != nil {
//		return nil, fmt.Errorf("failed to marshal request: %w", err)
//	}
//
//	// Create a new HTTP request
//	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestJSON))
//	if err != nil {
//		return nil, fmt.Errorf("failed to create request: %w", err)
//	}
//
//	// Set headers
//	req.Header.Set("Content-Type", "application/json")
//	if sessionID != "" {
//		req.Header.Set("Mcp-Session-Id", sessionID)
//	}
//
//	// Send the request
//	client := &http.Client{}
//	resp, err := client.Do(req)
//	if err != nil {
//		return nil, fmt.Errorf("failed to send request: %w", err)
//	}
//	defer resp.Body.Close()
//
//	// Read the response body
//	body, err := io.ReadAll(resp.Body)
//	if err != nil {
//		return nil, fmt.Errorf("failed to read response: %w", err)
//	}
//
//	// Check if the response is successful
//	if resp.StatusCode != http.StatusOK {
//		return nil, fmt.Errorf("server returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
//	}
//
//	// Parse the response
//	var response JsonRpcResponse
//	if err := json.Unmarshal(body, &response); err != nil {
//		return nil, fmt.Errorf("failed to unmarshal response: %w, body: %s", err, string(body))
//	}
//
//	return &response, nil
//}
//
//// Initialize a session with the MCP server
//func initialize(url string) (string, error) {
//	// Create initialize request
//	params := InitializeParams{}
//	params.ClientInfo.Name = "TestClient"
//	params.ClientInfo.Version = "1.0.0"
//
//	request := JsonRpcRequest{
//		Jsonrpc: "2.0",
//		Method:  "initialize",
//		Params:  params,
//		ID:      "init-1",
//	}
//
//	// Send the request
//	response, err := sendRequest(url, "", request)
//	if err != nil {
//		return "", fmt.Errorf("initialize failed: %w", err)
//	}
//
//	// Check for errors
//	if response.Error != nil {
//		return "", fmt.Errorf("initialize error: %s", response.Error.Message)
//	}
//
//	// Get the session ID from the response headers
//	// In a real client, you would extract this from the response headers
//	// For this example, we'll parse it from the result
//	var result map[string]interface{}
//	if err := json.Unmarshal(response.Result, &result); err != nil {
//		return "", fmt.Errorf("failed to parse initialize result: %w", err)
//	}
//
//	// For testing purposes, we'll use the server's session ID
//	// In a real implementation, you'd get this from the response headers
//	sessionID, ok := result["sessionId"].(string)
//	if !ok {
//		return "", fmt.Errorf("session ID not found in response")
//	}
//
//	return sessionID, nil
//}
//
//// Invoke a tool on the MCP server
//func invokeTool(url string, sessionID string, toolName string, toolParams map[string]interface{}) (json.RawMessage, error) {
//	// Create invoke request
//	params := InvokeToolParams{
//		Name:   toolName,
//		Params: toolParams,
//	}
//
//	request := JsonRpcRequest{
//		Jsonrpc: "2.0",
//		Method:  "invoke",
//		Params:  params,
//		ID:      fmt.Sprintf("invoke-%d", time.Now().UnixNano()),
//	}
//
//	// Send the request
//	response, err := sendRequest(url, sessionID, request)
//	if err != nil {
//		return nil, fmt.Errorf("invoke failed: %w", err)
//	}
//
//	// Check for errors
//	if response.Error != nil {
//		return nil, fmt.Errorf("invoke error: %s", response.Error.Message)
//	}
//
//	return response.Result, nil
//}
//
//func main() {
//	// URLs for the two servers
//	server1URL := "http://localhost:8080/mcp"
//	server2URL := "http://localhost:8081/mcp"
//
//	// Initialize a session with server 1
//	fmt.Println("Initializing session with Server 1...")
//	sessionID, err := initialize(server1URL)
//	if err != nil {
//		log.Fatalf("Failed to initialize session: %v", err)
//	}
//	fmt.Printf("Session initialized with ID: %s\n", sessionID)
//
//	// Invoke the echo tool on server 1
//	fmt.Println("\nInvoking 'echo' tool on Server 1...")
//	result1, err := invokeTool(server1URL, sessionID, "echo", map[string]interface{}{
//		"message": "Hello from client to Server 1",
//	})
//	if err != nil {
//		log.Fatalf("Failed to invoke tool on server 1: %v", err)
//	}
//
//	// Print the result
//	var echoResult1 map[string]interface{}
//	if err := json.Unmarshal(result1, &echoResult1); err != nil {
//		log.Fatalf("Failed to parse result from server 1: %v", err)
//	}
//	fmt.Printf("Server 1 response: %s (from %s)\n", echoResult1["message"], echoResult1["server"])
//
//	// Wait a moment to ensure session replication
//	fmt.Println("\nWaiting for session replication...")
//	time.Sleep(2 * time.Second)
//
//	// Now use the same session ID to invoke the tool on server 2
//	fmt.Println("\nInvoking 'echo' tool on Server 2 with the same session ID...")
//	result2, err := invokeTool(server2URL, sessionID, "echo", map[string]interface{}{
//		"message": "Hello from client to Server 2",
//	})
//	if err != nil {
//		log.Fatalf("Failed to invoke tool on server 2: %v", err)
//	}
//
//	// Print the result
//	var echoResult2 map[string]interface{}
//	if err := json.Unmarshal(result2, &echoResult2); err != nil {
//		log.Fatalf("Failed to parse result from server 2: %v", err)
//	}
//	fmt.Printf("Server 2 response: %s (from %s)\n", echoResult2["message"], echoResult2["server"])
//
//	// Success!
//	fmt.Println("\nSuccess! The session was successfully shared between both servers.")
//	fmt.Println("This demonstrates that the MCP servers are properly clustered.")
//
//	os.Exit(0)
//}
