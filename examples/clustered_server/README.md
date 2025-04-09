# Clustered MCP Server Example

This example demonstrates how to set up a horizontally scalable MCP server cluster using Redis for session management and remote actors for communication between nodes.

## Prerequisites

- Go 1.24 or higher
- Redis server running locally on the default port (6379)

## How It Works

The example consists of two parts:

1. **Clustered Server**: Starts two MCP server instances configured to work together in a cluster.
2. **Test Client**: Connects to one server, then sends requests to the other server using the same session ID.

The clustering is achieved through:

- **Redis Session Store**: Both servers use Redis to share session data
- **Remote Actors**: Enables actor communication between server nodes
- **Shared Session IDs**: Allows a client to connect to any node in the cluster

## Running the Example

### 1. Start Redis

Make sure Redis is running locally on the default port:

```bash
# Install Redis if needed
brew install redis  # macOS
apt-get install redis-server  # Ubuntu/Debian

# Start Redis
redis-server
```

### 2. Start the Clustered Server

```bash
go run clustered_server_example.go
```

This will start two MCP server instances:
- Server1: HTTP on port 8080, Actor system on port 9090
- Server2: HTTP on port 8081, Actor system on port 9091

### 3. Run the Test Client

In a separate terminal:

```bash
go run test_clustered_client.go
```

The client will:
1. Initialize a session with Server1
2. Invoke the "echo" tool on Server1
3. Use the same session ID to invoke the "echo" tool on Server2
4. Verify that both servers recognize the same session

## Expected Output

The test client should output something like:

```
Initializing session with Server 1...
Session initialized with ID: abc123...

Invoking 'echo' tool on Server 1...
Server 1 response: Hello from client to Server 1 (from Server1)

Waiting for session replication...

Invoking 'echo' tool on Server 2 with the same session ID...
Server 2 response: Hello from client to Server 2 (from Server2)

Success! The session was successfully shared between both servers.
This demonstrates that the MCP servers are properly clustered.
```

## Key Configuration Settings

The key configuration settings for clustering are:

```go
// Enable remote actors for clustering
cfg.Actor.UseRemoteActors = true
cfg.Actor.RemoteConfig.Host = "localhost"
cfg.Actor.RemoteConfig.Port = 9090  // Different for each server

// Use Redis for session management
cfg.Redis = &config.RedisConfig{
    Addresses: []string{"localhost:6379"},
    Password:  "",
    DB:        0,
}
cfg.Session.UseInMemory = false  // Disable in-memory store
```

## Notes

- In a production environment, you would typically run each server on a separate machine
- You might use a Redis cluster instead of a single Redis instance
- Load balancing would be handled by a reverse proxy like NGINX or a cloud load balancer
