package actors

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tochemey/goakt/v3/actor"
	"github.com/traego/scaled-mcp/pkg/config"
	mcpcontext "github.com/traego/scaled-mcp/pkg/context"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/resources"
)

func TestElicitationIntegration(t *testing.T) {
	ctx := context.Background()

	callbackCalled := false
	var capturedContext config.McpContext

	sessionStartupCallback := func(mcpCtx config.McpContext) error {
		callbackCalled = true
		capturedContext = mcpCtx

		sessionID := mcpCtx.GetSessionID()
		assert.NotEmpty(t, sessionID)

		return nil
	}

	serverInfo := &TestServerInfo{
		serverCaps: protocol.ServerCapabilities{
			Tools: &protocol.ToolsServerCapability{
				ListChanged: true,
			},
			Elicitation: &protocol.ElicitationCapability{},
			Experimental: map[string]interface{}{
				"version": "1.0.0",
			},
		},
		serverConfig: &config.ServerConfig{
			ProtocolVersion: protocol.ProtocolVersion20250326,
			Session: config.SessionConfig{
				TTL: time.Minute,
			},
		},
		executors:              NewTestExecutor(),
		registry:               resources.FeatureRegistry{},
		sessionStartupCallback: sessionStartupCallback,
	}

	actorSystem, err := actor.NewActorSystem("test-elicitation-system")
	require.NoError(t, err)

	// Start the actor system
	err = actorSystem.Start(ctx)
	require.NoError(t, err)

	defer func() {
		err := actorSystem.Stop(ctx)
		assert.NoError(t, err)
	}()

	// Create session actor
	sessionID := "test-session-elicitation"
	sessionActor, err := actorSystem.Spawn(ctx, sessionID, NewMcpSessionStateMachine(
		serverInfo,
		sessionID,
	))
	require.NoError(t, err)

	initParams := protocol.InitializeParams{
		ProtocolVersion: protocol.ProtocolVersion20250326,
		ClientInfo: protocol.ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
		Capabilities: protocol.ClientCapabilities{},
	}

	paramsJSON, err := json.Marshal(initParams)
	require.NoError(t, err)

	initRequest := &mcppb.JsonRpcRequest{
		Jsonrpc:    "2.0",
		Method:     "initialize",
		ParamsJson: string(paramsJSON),
		Id:         &mcppb.JsonRpcRequest_StringId{StringId: "init-1"},
	}

	wrappedRequest := &mcppb.WrappedRequest{
		Request:               initRequest,
		IsAsk:                 true,
		RespondToConnectionId: "test-connection",
	}

	// Send initialize request
	response, err := sessionActor.Ask(ctx, sessionActor, wrappedRequest, time.Second*5)
	require.NoError(t, err)

	// Verify response
	jsonRpcResp, ok := response.(*mcppb.JsonRpcResponse)
	require.True(t, ok)
	assert.Equal(t, "2.0", jsonRpcResp.Jsonrpc)
	assert.Equal(t, "init-1", jsonRpcResp.GetStringId())

	var initResult protocol.InitializeResult
	err = json.Unmarshal([]byte(jsonRpcResp.GetResultJson()), &initResult)
	require.NoError(t, err)

	assert.True(t, callbackCalled, "Session startup callback should have been called")
	assert.NotNil(t, capturedContext, "Captured context should not be nil")
	assert.Equal(t, sessionID, capturedContext.GetSessionID())

	assert.NotNil(t, initResult.Capabilities.Elicitation, "Elicitation capability should be present")

	elicitRequest := protocol.ElicitationRequest{
		Message: "Please provide your name",
		RequestedSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []string{"name"},
		},
	}

	elicitParamsJSON, err := json.Marshal(elicitRequest)
	require.NoError(t, err)

	elicitJsonRpcReq := &mcppb.JsonRpcRequest{
		Jsonrpc:    "2.0",
		Method:     "elicitation/create",
		ParamsJson: string(elicitParamsJSON),
		Id:         &mcppb.JsonRpcRequest_StringId{StringId: "elicit-1"},
	}

	wrappedElicitRequest := &mcppb.WrappedRequest{
		Request:               elicitJsonRpcReq,
		IsAsk:                 true,
		RespondToConnectionId: "test-connection",
	}

	elicitResponse, err := sessionActor.Ask(ctx, sessionActor, wrappedElicitRequest, time.Second*5)
	require.NoError(t, err)

	elicitJsonRpcResp, ok := elicitResponse.(*mcppb.JsonRpcResponse)
	require.True(t, ok)
	assert.Equal(t, "2.0", elicitJsonRpcResp.Jsonrpc)
	assert.Equal(t, "elicit-1", elicitJsonRpcResp.GetStringId())

	var elicitResult protocol.ElicitationResponse
	err = json.Unmarshal([]byte(elicitJsonRpcResp.GetResultJson()), &elicitResult)
	require.NoError(t, err)

	assert.Equal(t, protocol.ElicitationActionAccept, elicitResult.Action)
	assert.NotNil(t, elicitResult.Content)
	assert.Contains(t, elicitResult.Content, "mock")
}

func TestMcpContextImplementation(t *testing.T) {
	sessionID := "test-context-session"
	ctx := context.Background()

	mcpCtx := mcpcontext.NewMcpContext(sessionID, nil, ctx)

	assert.Equal(t, sessionID, mcpCtx.GetSessionID())

	assert.Equal(t, ctx, mcpCtx.GetContext())

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"test": map[string]interface{}{
				"type": "string",
			},
		},
	}

	_, err := mcpCtx.Elicit("Test message", schema)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "elicitation not yet implemented")
}
