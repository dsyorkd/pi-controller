package defaults

// WebSocket server defaults
const (
	// WebSocketHost is the default WebSocket server bind address
	WebSocketHost = "0.0.0.0"

	// WebSocketPort is the default WebSocket server port
	WebSocketPort = 8081

	// WebSocketPath is the default WebSocket path
	WebSocketPath = "/ws"

	// WebSocketReadBufferSize is the default read buffer size
	WebSocketReadBufferSize = 1024

	// WebSocketWriteBufferSize is the default write buffer size
	WebSocketWriteBufferSize = 1024

	// WebSocketCheckOrigin indicates whether origin checking is enabled
	WebSocketCheckOrigin = false
)
