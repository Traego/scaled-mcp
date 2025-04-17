package config

import (
	"github.com/traego/scaled-mcp/pkg/protocol"
	"time"
)

// ServerConfig holds the configuration for the MCP server
type ServerConfig struct {
	// HTTP server configuration
	HTTP HTTPConfig `json:"http"`

	// Redis configuration for distributed session management (optional)
	Redis *RedisConfig `json:"redis,omitempty"`

	// Session configuration
	Session SessionConfig `json:"session"`

	Clustering ClusteringConfig `json:"clustering"`

	// Actor configuration
	Actor ActorConfig `json:"actor"`

	// Server information
	ServerInfo ServerInfo `json:"server_info"`

	// Protocol version to use
	ProtocolVersion string `json:"protocol_version"`

	// Whether to enable SSE transport
	EnableSSE bool `json:"enable_sse"`

	EnableWebSockets bool `json:"enable_websockets"`

	// Whether to support backward compatibility with older MCP versions
	BackwardCompatible20241105 bool `json:"backward_compatible_2024_11_05"`

	// Feature flags
	//EnablePrompts   bool `json:"enable_prompts"`
	//EnableResources bool `json:"enable_resources"`
	//EnableTools     bool `json:"enable_tools"`

	ServerCapabilities protocol.ServerCapabilities `json:"server_capabilities"`

	RequestTimeout time.Duration `json:"request_timeout"`
}

// ServerInfo holds information about the server
type ServerInfo struct {
	// Server name
	Name string `json:"name"`

	// Server version
	Version string `json:"version"`
}

// HTTPConfig holds the HTTP server configuration
type HTTPConfig struct {
	// Host to bind to
	Host string `json:"host"`

	// Port to listen on
	Port int `json:"port"`

	// Path for the MCP endpoint
	MCPPath string `json:"mcp_path"`

	// Path for the backward compatible SSE endpoint
	SSEPath string `json:"sse_path"`

	// Path for the backward compatible POST endpoint
	MessagePath string `json:"message_path"`

	// TLS configuration
	TLS TLSConfig `json:"tls"`

	// CORS configuration
	CORS CORSConfig `json:"cors"`
}

// TLSConfig holds the TLS configuration
type TLSConfig struct {
	// Whether to enable TLS
	Enable bool `json:"enable"`

	// Path to the certificate file
	CertFile string `json:"cert_file"`

	// Path to the key file
	KeyFile string `json:"key_file"`
}

// CORSConfig holds the CORS configuration
type CORSConfig struct {
	// Whether to enable CORS
	Enable bool `json:"enable"`

	// Allowed origins
	AllowedOrigins []string `json:"allowed_origins"`

	// Allowed headers
	AllowedHeaders []string `json:"allowed_headers"`

	// Exposed headers
	ExposedHeaders []string `json:"exposed_headers"`

	// Allow credentials
	AllowCredentials bool `json:"allow_credentials"`

	// Max age
	MaxAge time.Duration `json:"max_age"`
}

type ClusteringType = string

const (
	ClusteringTypeK8S    ClusteringType = "k8s"
	ClusteringTypeStatic ClusteringType = "static"
)

type ClusteringConfig struct {
	GossipPort   int            `json:"gossip_port"`
	PeersPort    int            `json:"peers_port"`
	RemotingPort int            `json:"remoting_port"`
	Type         ClusteringType `json:"type"`
	StaticHosts  []string       `json:"static_hosts"`
	NodeHost     string         `json:"node_host"`
}

// SessionConfig holds the session configuration
type SessionConfig struct {
	InitializeTimeout time.Duration `json:"initialize_timeout"`

	// Session TTL (time to live)
	TTL time.Duration `json:"ttl"`

	// Whether to use in-memory session store (for testing)
	UseInMemory bool `json:"use_in_memory"`

	// Key prefix for session storage
	KeyPrefix string `json:"key_prefix"`
}

// RedisConfig holds the Redis configuration
type RedisConfig struct {
	// Redis addresses (can be multiple for cluster)
	Addresses []string `json:"addresses"`

	// Redis password
	Password string `json:"password"`

	// Redis database
	DB int `json:"db"`
}

// ActorConfig holds the actor system configuration
type ActorConfig struct {
	// Number of workers for handling actor messages
	NumWorkers int `json:"num_workers"`

	// Whether to use remote clients
	UseRemoteActors bool `json:"use_remote_actors"`

	// Remote actor configuration
	RemoteConfig RemoteActorConfig `json:"remote_config"`
}

// RemoteActorConfig holds the remote actor configuration
type RemoteActorConfig struct {
	// Host for the remote actor system
	Host string `json:"host"`

	// Port for the remote actor system
	Port int `json:"port"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *ServerConfig {
	return &ServerConfig{
		RequestTimeout: 30 * time.Second,
		HTTP: HTTPConfig{
			Host:        "0.0.0.0",
			Port:        8080,
			MCPPath:     "/mcp",
			SSEPath:     "/sse",
			MessagePath: "/messages",
			TLS: TLSConfig{
				Enable: false,
			},
			CORS: CORSConfig{
				Enable:           false,
				AllowedOrigins:   []string{"*"},
				AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
				ExposedHeaders:   []string{},
				AllowCredentials: false,
				MaxAge:           300 * time.Second,
			},
		},
		Session: SessionConfig{
			InitializeTimeout: 10 * time.Second,
			TTL:               5 * time.Minute,
			UseInMemory:       true,
			KeyPrefix:         "mcp:session:",
		},
		Actor: ActorConfig{
			NumWorkers:      10,
			UseRemoteActors: false,
			RemoteConfig: RemoteActorConfig{
				Host: "localhost",
				Port: 8090,
			},
		},
		ServerInfo: ServerInfo{
			Name:    "MCP Server",
			Version: "1.0.0",
		},
		ProtocolVersion:            "1.0.0",
		EnableSSE:                  true,
		EnableWebSockets:           false,
		BackwardCompatible20241105: true,
		ServerCapabilities: protocol.ServerCapabilities{
			Tools:     &protocol.ToolsServerCapability{},
			Prompts:   &protocol.PromptsServerCapability{},
			Resources: &protocol.ResourcesServerCapability{},
		},
	}
}

// TestConfig returns a configuration suitable for testing
func TestConfig() *ServerConfig {
	config := DefaultConfig()
	config.Redis = nil // No Redis for testing
	config.Session.UseInMemory = true
	return config
}
