package protocol

// SESSION_ID_CONTEXT_KEY is the key used to store and retrieve the session ID from the context
const SESSION_ID_CONTEXT_KEY = "sessionId"

// JSONRPCMessage represents a JSON-RPC message
type JSONRPCMessage struct {
	JSONRPC string              `json:"jsonrpc"`
	ID      interface{}         `json:"id,omitempty"`
	Method  string              `json:"method,omitempty"`
	Params  interface{}         `json:"params,omitempty"`
	Result  interface{}         `json:"result,omitempty"`
	Error   interface{}         `json:"error,omitempty"`
	Headers map[string][]string `json:"headers,omitempty"`
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema InputSchema `json:"inputSchema,omitempty"`
}

// InputSchema represents the schema for tool inputs
type InputSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]SchemaProperty `json:"properties"`
	Required   []string                  `json:"required,omitempty"`
}

// SchemaProperty represents a property in an input schema
type SchemaProperty struct {
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Default     interface{} `json:"default,omitempty"`
}

// ToolListOptions provides pagination options for listing resources
type ToolListOptions struct {
	Cursor string // Cursor for pagination
}

// ToolListResult represents a paginated list of resources
type ToolListResult struct {
	Tools      []Tool `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"` // Cursor for the next page, empty if no more pages
}

// ClientInfo represents information about the client
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerInfo represents information about the server
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeParams represents the parameters for an initialize request
type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
}

// InitializeResult represents the result of an initialize request
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	SessionID       string             `json:"sessionId,omitempty"`
}

// ClientCapabilities represents the capabilities of the client
type ClientCapabilities struct {
	Roots        *RootsClientCapability    `json:"roots,omitempty"`
	Sampling     *SamplingClientCapability `json:"sampling,omitempty"`
	Experimental map[string]interface{}    `json:"experimental,omitempty"`
}

// RootsClientCapability represents the roots capability of the client
type RootsClientCapability struct {
	ListChanged bool `json:"listChanged"`
}

// SamplingClientCapability represents the sampling capability of the client
type SamplingClientCapability struct {
	// Add sampling-specific fields here
}

// ServerCapabilities represents the capabilities of the server
type ServerCapabilities struct {
	Prompts      *PromptsServerCapability   `json:"prompts,omitempty"`
	Resources    *ResourcesServerCapability `json:"resources,omitempty"`
	Tools        *ToolsServerCapability     `json:"tools,omitempty"`
	Logging      *LoggingServerCapability   `json:"logging,omitempty"`
	Experimental map[string]interface{}     `json:"experimental,omitempty"`
}

// PromptsServerCapability represents the prompts capability of the server
type PromptsServerCapability struct {
	ListChanged bool `json:"listChanged"`
}

// ResourcesServerCapability represents the resources capability of the server
type ResourcesServerCapability struct {
	Subscribe   bool `json:"subscribe"`
	ListChanged bool `json:"listChanged"`
}

// ToolsServerCapability represents the resources capability of the server
type ToolsServerCapability struct {
	ListChanged bool `json:"listChanged"`
}

// LoggingServerCapability represents the logging capability of the server
type LoggingServerCapability struct {
	// Empty struct as per the 2025 spec
}
