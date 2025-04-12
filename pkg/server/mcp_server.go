package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/tochemey/goakt/v3/actor"
	"github.com/tochemey/goakt/v3/discovery/static"
	"github.com/tochemey/goakt/v3/remote"

	"github.com/traego/scaled-mcp/internal/logger"
	"github.com/traego/scaled-mcp/pkg/actors"
	"github.com/traego/scaled-mcp/pkg/config"
	"github.com/traego/scaled-mcp/pkg/executors"
	"github.com/traego/scaled-mcp/pkg/protocol"
	"github.com/traego/scaled-mcp/pkg/resources"
	"github.com/traego/scaled-mcp/pkg/utils"

	"github.com/traego/scaled-mcp/pkg/server/httphandlers"
)

// McpServer represents an MCP server
type McpServer struct {
	// Configuration
	config             *config.ServerConfig
	actorSystem        actor.ActorSystem
	actorMutex         sync.Mutex
	serverCapabilities protocol.ServerCapabilities
	enableSSE          bool
	httpServer         *http.Server
	mcpHandler         *httphandlers.MCPHandler
	featureRegistry    resources.FeatureRegistry
	executors          *executors.Executors
}

func (s *McpServer) GetExecutors() config.MethodHandler {
	return s.executors
}

func (s *McpServer) GetServerConfig() *config.ServerConfig {
	return s.config
}

func (s *McpServer) GetFeatureRegistry() resources.FeatureRegistry {
	return s.featureRegistry
}

func (s *McpServer) GetServerCapabilities() protocol.ServerCapabilities {
	return s.serverCapabilities
}

var _ config.McpServerInfo = (*McpServer)(nil)

// McpServerOption represents an option for the MCP server
type McpServerOption func(*McpServer)

// WithServerInfo sets the server info
func WithServerInfo(name, version string) McpServerOption {
	return func(s *McpServer) {
		s.serverCapabilities = protocol.ServerCapabilities{
			Prompts:   &protocol.PromptsServerCapability{},
			Resources: &protocol.ResourcesServerCapability{},
			Tools:     &protocol.ToolsServerCapability{},
			Logging:   &protocol.LoggingServerCapability{},
		}
	}
}

// WithPreferSSE sets whether to prefer SSE over JSON
func WithEnableSSE(enableSSE bool) McpServerOption {
	return func(s *McpServer) {
		s.enableSSE = enableSSE
	}
}

// WithToolRegistry sets the tool registry for the server
func WithToolRegistry(registry resources.ToolRegistry) McpServerOption {
	return func(s *McpServer) {
		s.featureRegistry.ToolRegistry = registry
	}
}

// WithPromptRegistry sets the prompt registry for the server
func WithPromptRegistry(registry resources.PromptRegistry) McpServerOption {
	return func(s *McpServer) {
		s.featureRegistry.PromptRegistry = registry
	}
}

// WithResourceRegistry sets the resource registry for the server
func WithResourceRegistry(registry resources.ResourceRegistry) McpServerOption {
	return func(s *McpServer) {
		s.featureRegistry.ResourceRegistry = registry
	}
}

func WithExecutors(executors *executors.Executors) McpServerOption {
	return func(s *McpServer) {
		s.executors = executors
	}
}

// NewMcpServer creates a new MCP server
func NewMcpServer(cfg *config.ServerConfig, options ...McpServerOption) (*McpServer, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	opts := make([]actor.Option, 0)
	switch cfg.Clustering.Type {
	case config.ClusteringTypeK8S:
	case config.ClusteringTypeStatic:
		if len(cfg.Clustering.StaticHosts) == 0 {
			return nil, fmt.Errorf("there must be at least one static host")
		}

		// define the discovery options
		discoConfig := static.Config{
			Hosts: cfg.Clustering.StaticHosts,
		}
		// instantiate the dnssd discovery provider
		disco := static.NewDiscovery(&discoConfig)
		clusterConfig := actor.
			NewClusterConfig().
			WithDiscovery(disco).
			WithPartitionCount(19).
			WithKinds(
				&actors.DeathWatcher{},
				&utils.StateMachineActor{},
				&actors.ClientConnectionActor{},
			).
			WithDiscoveryPort(cfg.Clustering.GossipPort).
			WithPeersPort(cfg.Clustering.PeersPort)

		//WithDiscoveryPort(config.GossipPort).
		//WithPeersPort(config.PeersPort).
		//WithKinds(new(actors.AccountEntity))

		opts = append(opts, actor.WithCluster(clusterConfig))
		opts = append(opts, actor.WithRemote(remote.NewConfig(cfg.Clustering.NodeHost, cfg.Clustering.RemotingPort)))
	}

	opts = append(opts, actor.WithLogger(logger.NewSlog(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}).WithGroup("mcp"))))
	opts = append(opts, actor.WithPassivationDisabled())

	// Create the actor system
	actorSystem, err := actor.NewActorSystem(cfg.Actor.SystemName, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create actor system: %w", err)
	}

	// Create the server
	server := &McpServer{
		config:             cfg,
		actorSystem:        actorSystem,
		enableSSE:          true, // Default to prefer SSE when available
		serverCapabilities: cfg.ServerCapabilities,
	}

	// Apply options
	for _, opt := range options {
		opt(server)
	}

	if server.executors == nil {
		server.executors = executors.DefaultExecutors(server, nil)
	}

	// Create a default static tool registry if none provided
	if server.featureRegistry.ToolRegistry == nil {
		server.featureRegistry.ToolRegistry = resources.NewStaticToolRegistry()
		slog.Info("Using default static tool registry")
	}

	// Create a default static prompt registry if none provided
	if server.featureRegistry.PromptRegistry == nil {
		server.featureRegistry.PromptRegistry = resources.NewStaticPromptRegistry()
		slog.Info("Using default static prompt registry")
	}

	// Create a default static resource registry if none provided
	if server.featureRegistry.ResourceRegistry == nil {
		server.featureRegistry.ResourceRegistry = resources.NewStaticResourceRegistry()
		slog.Info("Using default static resource registry")
	}

	// Create the MCP handler with an adapter for the session store
	server.mcpHandler = httphandlers.NewMCPHandler(cfg, actorSystem, server)

	return server, nil
}

// Start starts the MCP server
func (s *McpServer) Start(ctx context.Context) error {
	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.HTTP.Host, s.config.HTTP.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.createHTTPHandler(),
	}

	s.actorMutex.Lock()
	err := s.actorSystem.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start MCP actor system: %w", err)
	}
	s.actorMutex.Unlock()

	supervisor := actor.NewSupervisor(actor.WithAnyErrorDirective(actor.RestartDirective))
	_, err = s.actorSystem.Spawn(ctx, "root", actors.NewRootActor(), actor.WithLongLived(), actor.WithSupervisor(supervisor))
	if err != nil {
		return fmt.Errorf("failed to start root actor: %w", err)
	}

	// Start HTTP server
	go func() {
		var err error
		if s.config.HTTP.TLS.Enable {
			err = s.httpServer.ListenAndServeTLS(s.config.HTTP.TLS.CertFile, s.config.HTTP.TLS.KeyFile)
		} else {
			err = s.httpServer.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			slog.ErrorContext(ctx, "HTTP server error", "error", err)
		}
	}()

	slog.InfoContext(ctx, "MCP server started", "address", addr)
	return nil
}

// Stop stops the MCP server
func (s *McpServer) Stop(ctx context.Context) {
	// Stop HTTP server
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Stop actor system - in goakt v3.2.0 we need to use a different approach
	// since Shutdown is not directly available
	slog.InfoContext(ctx, "Stopping actor system")
	if s.actorSystem != nil {
		s.actorMutex.Lock()
		if err := s.actorSystem.Stop(ctx); err != nil {
			slog.Error("Failed to shutdown actor system", "err", err)
		}
		s.actorMutex.Unlock()
	}

	slog.InfoContext(ctx, "Stopping MCP Server")
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			slog.Error("Failed to shutdown HTTP server", "err", err)
		}
	}
}

// createHTTPHandler creates the HTTP handler for the MCP server
func (s *McpServer) createHTTPHandler() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(s.loggingMiddleware)
	r.Use(s.jsonRpcErrorMiddleware)
	r.Use(middleware.Recoverer) // Recover from panics

	// CORS middleware if needed
	if s.config.HTTP.CORS.Enable {
		corsOptions := cors.Options{
			AllowedOrigins:   s.config.HTTP.CORS.AllowedOrigins,
			AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
			AllowedHeaders:   s.config.HTTP.CORS.AllowedHeaders,
			ExposedHeaders:   s.config.HTTP.CORS.ExposedHeaders,
			AllowCredentials: s.config.HTTP.CORS.AllowCredentials,
			MaxAge:           int(s.config.HTTP.CORS.MaxAge.Seconds()),
		}
		r.Use(cors.Handler(corsOptions))
	}

	// Main MCP endpoint - handles both POST (for new sessions) and GET (for resuming sessions)
	r.Route(s.config.HTTP.MCPPath, func(r chi.Router) {
		r.Post("/", s.mcpHandler.HandleMCPPost)
		r.Get("/", s.mcpHandler.HandleMCPGet)
	})

	// Optional /messages endpoint for 2024 version client negotiation
	if s.config.BackwardCompatible20241105 {
		r.Get(s.config.HTTP.SSEPath, s.mcpHandler.HandleSSEGet)
		r.Post(s.config.HTTP.MessagePath, s.mcpHandler.HandleMessagePost)
	}

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	return r
}

// loggingMiddleware logs HTTP requests
func (s *McpServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		// Call the next handler
		next.ServeHTTP(ww, r)

		// Log the request
		ctx := r.Context()
		latency := time.Since(start)
		slog.InfoContext(ctx, "HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"latency", latency.String(),
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)
	})
}

// jsonRpcErrorMiddleware converts panics and other errors to JSON-RPC errors
func (s *McpServer) jsonRpcErrorMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a response writer that can capture the response
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		// Use a panic recovery function that converts panics to JSON-RPC errors
		defer func() {
			if err := recover(); err != nil {
				// Only handle panics for MCP endpoints
				if !strings.HasPrefix(r.URL.Path, s.config.HTTP.MCPPath) {
					// For non-MCP endpoints, let the standard Recoverer middleware handle it
					panic(err)
				}

				// Log the panic
				slog.Error("Panic in handler", "error", err, "stack", string(debug.Stack()))

				// Convert to JSON-RPC error
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)

				// Create a generic JSON-RPC internal error
				// Since we don't have the request ID here, we'll use nil
				internalError := protocol.NewInternalError(fmt.Sprintf("Internal server error: %v", err), nil)
				response := internalError.ToResponse()

				responseJSON, marshalErr := json.Marshal(response)
				if marshalErr != nil {
					// If we can't marshal the error response, fall back to a generic JSON-RPC server error
					fallbackError := protocol.NewServerError(protocol.ErrServer, "Internal server error", nil, nil)
					fallbackJSON, _ := json.Marshal(fallbackError.ToResponse())
					_, _ = w.Write(fallbackJSON)
					return
				}

				_, _ = w.Write(responseJSON)
			}
		}()

		// Call the next handler
		next.ServeHTTP(ww, r)
	})
}

//// GetToolRegistry returns the tool resources for the server
//func (s *McpServer) GetToolRegistry() resources.ToolRegistry {
//	return s.toolRegistry
//}

//// RegisterTool registers a tool with the static tool resources
//// This is a convenience method that only works if the server is using a StaticToolRegistry
//func (s *McpServer) RegisterTool(tool resources.Tool, handler resources.ToolHandler) error {
//	staticRegistry, ok := s.toolRegistry.(*resources.StaticToolRegistry)
//	if !ok {
//		return fmt.Errorf("cannot register tool: server is not using a static tool resources")
//	}
//	return staticRegistry.RegisterTool(tool, handler)
//}
