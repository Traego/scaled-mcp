package protocol

//
//import (
//	"github.com/traego/scaled-mcp/pkg/config"
//)
//
//// CreateInitializeResponse creates a JSON-RPC response for an initialize request
//// using the server configuration and session ID
//func CreateInitializeResponse(id interface{}, sessionID string, cfg *config.ServerConfig, protocolVersion string) JSONRPCMessage {
//	// Create the server capabilities based on the configuration
//	capabilities := ServerCapabilities{
//		Logging: &LoggingServerCapability{},
//	}
//
//	// Add prompts capability if enabled
//	if cfg.EnablePrompts {
//		capabilities.Prompts = &PromptsServerCapability{
//			ListChanged: true,
//		}
//	}
//
//	// Add resources capability if enabled
//	if cfg.EnableResources {
//		capabilities.Resources = &ResourcesServerCapability{
//			Subscribe:   true,
//			ListChanged: true,
//		}
//	}
//
//	// Add resources capability if enabled
//	if cfg.EnableTools {
//		capabilities.Tools = &ToolsServerCapability{
//			ListChanged: true,
//		}
//	}
//
//	// Create the initialize result
//	result := InitializeResult{
//		ProtocolVersion: protocolVersion,
//		ServerInfo: ServerInfo{
//			Name:    cfg.ServerInfo.Name,
//			Version: cfg.ServerInfo.Version,
//		},
//		Capabilities: capabilities,
//		SessionID:    sessionID,
//	}
//
//	// Create the JSON-RPC response
//	return JSONRPCMessage{
//		JSONRPC: "2.0",
//		ID:      id,
//		Result:  result,
//	}
//}
