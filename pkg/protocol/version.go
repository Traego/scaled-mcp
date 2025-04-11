package protocol

// ProtocolVersion represents the MCP protocol version to use.
type ProtocolVersion string

const (
	// ProtocolVersion20241105 represents the 2024-11-05 MCP specification.
	ProtocolVersion20241105 ProtocolVersion = "2024-11-05"

	// ProtocolVersion20250326 represents the 2025-03-26 MCP specification.
	ProtocolVersion20250326 ProtocolVersion = "2025-03-26"

	// ProtocolVersionAuto will automatically detect and use the highest supported version.
	ProtocolVersionAuto ProtocolVersion = "auto"
)
