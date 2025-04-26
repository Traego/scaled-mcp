package executors

import (
	"context"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/proto/mcppb"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"strings"
)

// TODO This actually wants to be pluggable, this is where we'd plug in new fancy stuff

type Executors struct {
	Tools        config.MethodHandler
	Prompts      config.MethodHandler
	Resources    config.MethodHandler
	Utilities    config.MethodHandler
	Experimental config.MethodHandler
}

func DefaultExecutors(serverInfo config.McpServerInfo, experimental config.MethodHandler) *Executors {
	return &Executors{
		Tools:        NewToolExecutor(serverInfo),
		Prompts:      NewPromptExecutor(serverInfo),
		Resources:    NewResourceExecutor(serverInfo),
		Utilities:    NewUtilitiesExecutor(serverInfo),
		Experimental: experimental,
	}
}

func (e *Executors) CanHandleMethod(method string) bool {
	if e.Tools != nil && e.Tools.CanHandleMethod(method) {
		return true
	} else if e.Prompts != nil && e.Prompts.CanHandleMethod(method) {
		return true
	} else if e.Resources != nil && e.Resources.CanHandleMethod(method) {
		return true
	} else if e.Utilities != nil && e.Utilities.CanHandleMethod(method) {
		return true
	} else if e.Experimental != nil && e.Experimental.CanHandleMethod(method) {
		return true
	}
	return false
}

func (e *Executors) HandleMethod(ctx context.Context, method string, req *mcppb.JsonRpcRequest) (*mcppb.JsonRpcResponse, error) {
	ms := strings.Split(method, "/")
	if len(ms) >= 2 {
		switch ms[0] {
		case "tools":
			if e.Tools.CanHandleMethod(method) {
				return e.Tools.HandleMethod(ctx, method, req)
			}
		case "resources":
			if e.Resources.CanHandleMethod(method) {
				return e.Resources.HandleMethod(ctx, method, req)
			}
		case "prompts":
			if e.Prompts.CanHandleMethod(method) {
				return e.Prompts.HandleMethod(ctx, method, req)
			}
		}
	}

	// Handle utility methods that don't have a prefix (like ping)
	if e.Utilities != nil && e.Utilities.CanHandleMethod(method) {
		return e.Utilities.HandleMethod(ctx, method, req)
	}

	if e.Experimental.CanHandleMethod(method) {
		return e.Experimental.HandleMethod(ctx, method, req)
	}

	return nil, protocol.NewMethodNotFoundError(method, req.Id)
}
