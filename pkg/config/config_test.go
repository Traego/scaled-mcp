package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/traego/scaled-mcp/pkg/protocol"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "0.0.0.0", cfg.HTTP.Host)
	assert.Equal(t, 8080, cfg.HTTP.Port)
	assert.Equal(t, "/mcp", cfg.HTTP.MCPPath)
	assert.Equal(t, "/sse", cfg.HTTP.SSEPath)
	assert.Equal(t, "/messages", cfg.HTTP.MessagePath)
	assert.False(t, cfg.HTTP.TLS.Enable)
	assert.False(t, cfg.HTTP.CORS.Enable)
	assert.Equal(t, []string{"*"}, cfg.HTTP.CORS.AllowedOrigins)
	assert.Equal(t, 10, cfg.Actor.NumWorkers)
	assert.False(t, cfg.Actor.UseRemoteActors)
	assert.Equal(t, "localhost", cfg.Actor.RemoteConfig.Host)
	assert.Equal(t, 8090, cfg.Actor.RemoteConfig.Port)
	assert.Equal(t, "MCP Server", cfg.ServerInfo.Name)
	assert.Equal(t, "1.0.0", cfg.ServerInfo.Version)
	assert.Equal(t, protocol.ProtocolVersion("1.0.0"), cfg.ProtocolVersion)
	assert.True(t, cfg.EnableSSE)
	assert.False(t, cfg.EnableWebSockets)
	assert.True(t, cfg.BackwardCompatible20241105)
	assert.NotNil(t, cfg.ServerCapabilities.Tools)
	assert.NotNil(t, cfg.ServerCapabilities.Prompts)
	assert.NotNil(t, cfg.ServerCapabilities.Resources)
}

func TestTestConfig(t *testing.T) {
	cfg := TestConfig()

	assert.NotNil(t, cfg)
	assert.Nil(t, cfg.Redis)
	assert.True(t, cfg.Session.UseInMemory)

	defaultCfg := DefaultConfig()
	assert.Equal(t, defaultCfg.HTTP, cfg.HTTP)
	assert.Equal(t, defaultCfg.Actor, cfg.Actor)
	assert.Equal(t, defaultCfg.ServerInfo, cfg.ServerInfo)
	assert.Equal(t, defaultCfg.ProtocolVersion, cfg.ProtocolVersion)
	assert.Equal(t, defaultCfg.EnableSSE, cfg.EnableSSE)
	assert.Equal(t, defaultCfg.EnableWebSockets, cfg.EnableWebSockets)
	assert.Equal(t, defaultCfg.BackwardCompatible20241105, cfg.BackwardCompatible20241105)
	assert.Equal(t, defaultCfg.ServerCapabilities, cfg.ServerCapabilities)
}
