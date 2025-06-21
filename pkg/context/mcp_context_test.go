package context

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tochemey/goakt/v3/actor"
)

func TestNewMcpContext(t *testing.T) {
	sessionID := "test-session-123"
	ctx := context.Background()
	
	mcpCtx := NewMcpContext(sessionID, nil, ctx)
	
	assert.NotNil(t, mcpCtx)
	assert.Equal(t, sessionID, mcpCtx.sessionID)
	assert.Equal(t, ctx, mcpCtx.ctx)
	assert.Nil(t, mcpCtx.actorContext)
}

func TestMcpContext_GetSessionID(t *testing.T) {
	sessionID := "test-session-456"
	ctx := context.Background()
	
	mcpCtx := NewMcpContext(sessionID, nil, ctx)
	
	result := mcpCtx.GetSessionID()
	assert.Equal(t, sessionID, result)
}

func TestMcpContext_GetContext(t *testing.T) {
	sessionID := "test-session-789"
	ctx := context.Background()
	
	mcpCtx := NewMcpContext(sessionID, nil, ctx)
	
	result := mcpCtx.GetContext()
	assert.Equal(t, ctx, result)
}

func TestMcpContext_Elicit_NotImplemented(t *testing.T) {
	sessionID := "test-session-elicit"
	ctx := context.Background()
	
	mcpCtx := NewMcpContext(sessionID, nil, ctx)
	
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
		},
	}
	
	response, err := mcpCtx.Elicit("Test message", schema)
	
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "elicitation not yet implemented")
}

func TestMcpContext_WithActorContext(t *testing.T) {
	sessionID := "test-session-actor"
	ctx := context.Background()
	
	var actorCtx *actor.ReceiveContext = nil
	
	mcpCtx := NewMcpContext(sessionID, actorCtx, ctx)
	
	assert.NotNil(t, mcpCtx)
	assert.Equal(t, sessionID, mcpCtx.sessionID)
	assert.Equal(t, ctx, mcpCtx.ctx)
	assert.Equal(t, actorCtx, mcpCtx.actorContext)
}

func TestMcpContext_EmptySessionID(t *testing.T) {
	sessionID := ""
	ctx := context.Background()
	
	mcpCtx := NewMcpContext(sessionID, nil, ctx)
	
	assert.NotNil(t, mcpCtx)
	assert.Equal(t, "", mcpCtx.GetSessionID())
}

func TestMcpContext_NilContext(t *testing.T) {
	sessionID := "test-session-nil-ctx"
	
	mcpCtx := NewMcpContext(sessionID, nil, nil)
	
	assert.NotNil(t, mcpCtx)
	assert.Equal(t, sessionID, mcpCtx.GetSessionID())
	assert.Nil(t, mcpCtx.GetContext())
}
