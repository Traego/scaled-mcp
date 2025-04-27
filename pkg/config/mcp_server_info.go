package config

import (
	"context"
	"github.com/traego/scaled-mcp/pkg/auth"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/resources"
	"net/http"
)

type McpServerInfo interface {
	GetFeatureRegistry() resources.FeatureRegistry
	GetServerCapabilities() protocol.ServerCapabilities
	GetServerConfig() *ServerConfig
	GetExecutors() MethodHandler
	GetAuthHandler() AuthHandler
	GetTraceHandler() TraceHandler
}

type AuthHandler interface {
	ExtractAuth(r *http.Request) auth.AuthInfo
	Serialize(auth auth.AuthInfo) ([]byte, error)
	Deserialize(b []byte) (auth.AuthInfo, error)
}

type TraceHandler interface {
	ExtractTraceId(r *http.Request) string
	SetTraceId(ctx context.Context, traceId string) context.Context
}

type MethodHandler interface {
	CanHandleMethod(method string) bool
	HandleMethod(ctx context.Context, method string, req *mcppb.JsonRpcRequest) (*mcppb.JsonRpcResponse, error)
}
