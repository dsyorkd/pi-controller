package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dsyorkd/pi-controller/internal/config"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/storage"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupWebSocketTest(t *testing.T) (*Server, *storage.Database, func()) {
	log := logger.Default()
	db, err := storage.NewForTest(log)
	require.NoError(t, err)

	cfg := &config.WebSocketConfig{
		Host:            "localhost",
		Port:            0, // Dynamic port
		Path:            "/ws",
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     false,
	}

	server := New(cfg, log, db)

	cleanup := func() {
		// Close the shutdown channel to stop the server
		close(server.shutdown)
		// Give time for goroutines to finish
		time.Sleep(20 * time.Millisecond)
		db.Close()
	}

	return server, db, cleanup
}

// drainWelcomeMessage reads and discards the welcome message sent when a client connects
func drainWelcomeMessage(t *testing.T, client *Client) {
	select {
	case <-client.send:
		// Welcome message received and discarded
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Welcome message not received")
	}
}

func TestNew(t *testing.T) {
	log := logger.Default()
	db, err := storage.NewForTest(log)
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.WebSocketConfig{
		Host:            "localhost",
		Port:            8080,
		Path:            "/ws",
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	server := New(cfg, log, db)

	assert.NotNil(t, server)
	assert.NotNil(t, server.config)
	assert.NotNil(t, server.logger)
	assert.NotNil(t, server.database)
	assert.NotNil(t, server.upgrader)
	assert.NotNil(t, server.clients)
	assert.NotNil(t, server.broadcast)
	assert.NotNil(t, server.register)
	assert.NotNil(t, server.unregister)
	assert.NotNil(t, server.shutdown)
}

func TestServer_ClientRegistration(t *testing.T) {
	server, _, cleanup := setupWebSocketTest(t)
	defer cleanup()

	// Start the hub
	go server.run()

	// Create a mock client
	client := &Client{
		server:        server,
		send:          make(chan []byte, 256),
		id:            "test-client",
		subscriptions: make(map[string]bool),
	}

	// Register client
	server.register <- client
	time.Sleep(10 * time.Millisecond)
	drainWelcomeMessage(t, client)

	// Allow some time for processing
	time.Sleep(10 * time.Millisecond)

	// Verify client is registered
	server.clientsMux.RLock()
	_, exists := server.clients[client]
	server.clientsMux.RUnlock()

	assert.True(t, exists)

	// Unregister client
	server.unregister <- client

	// Allow some time for processing
	time.Sleep(10 * time.Millisecond)

	// Verify client is unregistered
	server.clientsMux.RLock()
	_, exists = server.clients[client]
	server.clientsMux.RUnlock()

	assert.False(t, exists)
}

func TestServer_BroadcastMessage(t *testing.T) {
	server, _, cleanup := setupWebSocketTest(t)
	defer cleanup()

	go server.run()

	// Create mock client
	client := &Client{
		server:        server,
		send:          make(chan []byte, 256),
		id:            "test-client",
		subscriptions: make(map[string]bool),
	}

	server.register <- client
	time.Sleep(10 * time.Millisecond)
	drainWelcomeMessage(t, client)

	// Broadcast a message
	msg := Message{
		Type:      MessageTypePing,
		Timestamp: time.Now(),
	}

	server.BroadcastMessage(msg)
	time.Sleep(10 * time.Millisecond)

	// Check if message was received
	select {
	case received := <-client.send:
		var receivedMsg Message
		err := json.Unmarshal(received, &receivedMsg)
		assert.NoError(t, err)
		assert.Equal(t, MessageTypePing, receivedMsg.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Message not received within timeout")
	}
}

func TestServer_BroadcastToTopic(t *testing.T) {
	server, _, cleanup := setupWebSocketTest(t)
	defer cleanup()

	go server.run()

	// Create client subscribed to "gpio" topic
	client1 := &Client{
		server:        server,
		send:          make(chan []byte, 256),
		id:            "client1",
		subscriptions: map[string]bool{"gpio": true},
	}

	// Create client NOT subscribed to "gpio" topic
	client2 := &Client{
		server:        server,
		send:          make(chan []byte, 256),
		id:            "client2",
		subscriptions: make(map[string]bool),
	}

	server.register <- client1
	server.register <- client2
	time.Sleep(10 * time.Millisecond)
	drainWelcomeMessage(t, client1)
	drainWelcomeMessage(t, client2)

	// Broadcast to "gpio" topic
	msg := Message{
		Type:      MessageTypeGPIOReading,
		Timestamp: time.Now(),
	}

	server.BroadcastToTopic("gpio", msg)
	time.Sleep(10 * time.Millisecond)

	// Check client1 received message
	select {
	case received := <-client1.send:
		var receivedMsg Message
		err := json.Unmarshal(received, &receivedMsg)
		assert.NoError(t, err)
		assert.Equal(t, MessageTypeGPIOReading, receivedMsg.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Client1 did not receive message")
	}

	// Check client2 did NOT receive message
	select {
	case <-client2.send:
		t.Fatal("Client2 should not have received message")
	case <-time.After(50 * time.Millisecond):
		// Expected - client2 is not subscribed
	}
}

func TestServer_BroadcastGPIOReading(t *testing.T) {
	server, _, cleanup := setupWebSocketTest(t)
	defer cleanup()

	go server.run()

	client := &Client{
		server:        server,
		send:          make(chan []byte, 256),
		id:            "test-client",
		subscriptions: map[string]bool{"gpio": true},
	}

	server.register <- client
	time.Sleep(10 * time.Millisecond)
	drainWelcomeMessage(t, client)

	reading := GPIOReadingMessage{
		DeviceID:  1,
		NodeID:    1,
		Pin:       17,
		Value:     1.0,
		Timestamp: time.Now(),
	}

	server.BroadcastGPIOReading(reading)
	time.Sleep(10 * time.Millisecond)

	select {
	case received := <-client.send:
		var msg Message
		err := json.Unmarshal(received, &msg)
		assert.NoError(t, err)
		assert.Equal(t, MessageTypeGPIOReading, msg.Type)

		var receivedReading GPIOReadingMessage
		err = json.Unmarshal(msg.Payload, &receivedReading)
		assert.NoError(t, err)
		assert.Equal(t, reading.DeviceID, receivedReading.DeviceID)
		assert.Equal(t, reading.Pin, receivedReading.Pin)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Message not received")
	}
}

func TestServer_BroadcastNodeStatus(t *testing.T) {
	server, _, cleanup := setupWebSocketTest(t)
	defer cleanup()

	go server.run()

	client := &Client{
		server:        server,
		send:          make(chan []byte, 256),
		id:            "test-client",
		subscriptions: map[string]bool{"nodes": true},
	}

	server.register <- client
	time.Sleep(10 * time.Millisecond)
	drainWelcomeMessage(t, client)

	status := NodeStatusMessage{
		NodeID:    1,
		Name:      "test-node",
		Status:    "ready",
		IPAddress: "192.168.1.100",
		Timestamp: time.Now(),
	}

	server.BroadcastNodeStatus(status)
	time.Sleep(10 * time.Millisecond)

	select {
	case received := <-client.send:
		var msg Message
		err := json.Unmarshal(received, &msg)
		assert.NoError(t, err)
		assert.Equal(t, MessageTypeNodeStatus, msg.Type)

		var receivedStatus NodeStatusMessage
		err = json.Unmarshal(msg.Payload, &receivedStatus)
		assert.NoError(t, err)
		assert.Equal(t, status.NodeID, receivedStatus.NodeID)
		assert.Equal(t, status.Name, receivedStatus.Name)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Message not received")
	}
}

func TestServer_BroadcastClusterStatus(t *testing.T) {
	server, _, cleanup := setupWebSocketTest(t)
	defer cleanup()

	go server.run()

	client := &Client{
		server:        server,
		send:          make(chan []byte, 256),
		id:            "test-client",
		subscriptions: map[string]bool{"clusters": true},
	}

	server.register <- client
	time.Sleep(10 * time.Millisecond)
	drainWelcomeMessage(t, client)

	status := ClusterStatusMessage{
		ClusterID:  1,
		Name:       "test-cluster",
		Status:     "healthy",
		NodesReady: 3,
		NodesTotal: 5,
		Timestamp:  time.Now(),
	}

	server.BroadcastClusterStatus(status)
	time.Sleep(10 * time.Millisecond)

	select {
	case received := <-client.send:
		var msg Message
		err := json.Unmarshal(received, &msg)
		assert.NoError(t, err)
		assert.Equal(t, MessageTypeClusterStatus, msg.Type)

		var receivedStatus ClusterStatusMessage
		err = json.Unmarshal(msg.Payload, &receivedStatus)
		assert.NoError(t, err)
		assert.Equal(t, status.ClusterID, receivedStatus.ClusterID)
		assert.Equal(t, status.NodesReady, receivedStatus.NodesReady)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Message not received")
	}
}

func TestServer_BroadcastSystemMetrics(t *testing.T) {
	server, _, cleanup := setupWebSocketTest(t)
	defer cleanup()

	go server.run()

	client := &Client{
		server:        server,
		send:          make(chan []byte, 256),
		id:            "test-client",
		subscriptions: map[string]bool{"system": true},
	}

	server.register <- client
	time.Sleep(10 * time.Millisecond)
	drainWelcomeMessage(t, client)

	metrics := SystemMetricsMessage{
		CPUUsage:    45.5,
		MemoryUsage: 62.3,
		Goroutines:  50,
		Timestamp:   time.Now(),
	}

	server.BroadcastSystemMetrics(metrics)
	time.Sleep(10 * time.Millisecond)

	select {
	case received := <-client.send:
		var msg Message
		err := json.Unmarshal(received, &msg)
		assert.NoError(t, err)
		assert.Equal(t, MessageTypeSystemMetrics, msg.Type)

		var receivedMetrics SystemMetricsMessage
		err = json.Unmarshal(msg.Payload, &receivedMetrics)
		assert.NoError(t, err)
		assert.Equal(t, metrics.CPUUsage, receivedMetrics.CPUUsage)
		assert.Equal(t, metrics.Goroutines, receivedMetrics.Goroutines)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Message not received")
	}
}

func TestClient_SubscribeUnsubscribe(t *testing.T) {
	server, _, cleanup := setupWebSocketTest(t)
	defer cleanup()

	client := &Client{
		server:        server,
		send:          make(chan []byte, 256),
		id:            "test-client",
		subscriptions: make(map[string]bool),
	}

	// Subscribe to topic
	client.subscribe("gpio")
	assert.True(t, client.isSubscribedTo("gpio"))

	// Unsubscribe from topic
	client.unsubscribe("gpio")
	assert.False(t, client.isSubscribedTo("gpio"))
}

func TestClient_SendError(t *testing.T) {
	server, _, cleanup := setupWebSocketTest(t)
	defer cleanup()

	go server.run()

	client := &Client{
		server:        server,
		send:          make(chan []byte, 256),
		id:            "test-client",
		subscriptions: make(map[string]bool),
	}

	server.register <- client
	time.Sleep(10 * time.Millisecond)
	drainWelcomeMessage(t, client)

	client.sendError(400, "Test error")
	time.Sleep(10 * time.Millisecond)

	select {
	case received := <-client.send:
		var msg Message
		err := json.Unmarshal(received, &msg)
		assert.NoError(t, err)
		assert.Equal(t, MessageTypeError, msg.Type)

		var errMsg ErrorMessage
		err = json.Unmarshal(msg.Payload, &errMsg)
		assert.NoError(t, err)
		assert.Equal(t, 400, errMsg.Code)
		assert.Equal(t, "Test error", errMsg.Message)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Error message not received")
	}
}

func TestClient_HandleMessage_Subscribe(t *testing.T) {
	server, _, cleanup := setupWebSocketTest(t)
	defer cleanup()

	client := &Client{
		server:        server,
		send:          make(chan []byte, 256),
		id:            "test-client",
		subscriptions: make(map[string]bool),
	}

	subMsg := SubscribeMessage{Topic: "gpio"}
	payload, _ := json.Marshal(subMsg)

	msg := Message{
		Type:    MessageTypeSubscribe,
		Payload: payload,
	}

	client.handleMessage(msg)

	assert.True(t, client.isSubscribedTo("gpio"))
}

func TestClient_HandleMessage_Unsubscribe(t *testing.T) {
	server, _, cleanup := setupWebSocketTest(t)
	defer cleanup()

	client := &Client{
		server:        server,
		send:          make(chan []byte, 256),
		id:            "test-client",
		subscriptions: map[string]bool{"gpio": true},
	}

	unsubMsg := SubscribeMessage{Topic: "gpio"}
	payload, _ := json.Marshal(unsubMsg)

	msg := Message{
		Type:    MessageTypeUnsubscribe,
		Payload: payload,
	}

	client.handleMessage(msg)

	assert.False(t, client.isSubscribedTo("gpio"))
}

func TestClient_HandleMessage_Ping(t *testing.T) {
	server, _, cleanup := setupWebSocketTest(t)
	defer cleanup()

	go server.run()

	client := &Client{
		server:        server,
		send:          make(chan []byte, 256),
		id:            "test-client",
		subscriptions: make(map[string]bool),
	}

	server.register <- client
	time.Sleep(10 * time.Millisecond)
	drainWelcomeMessage(t, client)

	msg := Message{
		Type:      MessageTypePing,
		RequestID: "test-request",
	}

	client.handleMessage(msg)
	time.Sleep(10 * time.Millisecond)

	// Should receive pong response
	select {
	case received := <-client.send:
		var pongMsg Message
		err := json.Unmarshal(received, &pongMsg)
		assert.NoError(t, err)
		assert.Equal(t, MessageTypePong, pongMsg.Type)
		assert.Equal(t, "test-request", pongMsg.RequestID)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Pong not received")
	}
}

func TestClient_HandleMessage_InvalidType(t *testing.T) {
	server, _, cleanup := setupWebSocketTest(t)
	defer cleanup()

	go server.run()

	client := &Client{
		server:        server,
		send:          make(chan []byte, 256),
		id:            "test-client",
		subscriptions: make(map[string]bool),
	}

	server.register <- client
	time.Sleep(10 * time.Millisecond)
	drainWelcomeMessage(t, client)

	msg := Message{
		Type: "invalid",
	}

	client.handleMessage(msg)
	time.Sleep(10 * time.Millisecond)

	// Should receive error response
	select {
	case received := <-client.send:
		var errMsg Message
		err := json.Unmarshal(received, &errMsg)
		assert.NoError(t, err)
		assert.Equal(t, MessageTypeError, errMsg.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Error not received")
	}
}

func TestGenerateClientID(t *testing.T) {
	id1 := generateClientID()
	id2 := generateClientID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	// IDs should be different (though not guaranteed due to time resolution)
	assert.Contains(t, id1, "-")
	assert.Contains(t, id2, "-")
}

func TestGenerateRandomString(t *testing.T) {
	s1 := generateRandomString(10)
	s2 := generateRandomString(10)

	assert.Len(t, s1, 10)
	assert.Len(t, s2, 10)
	assert.NotEmpty(t, s1)
	assert.NotEmpty(t, s2)
}

// Integration test with actual WebSocket connection
func TestWebSocketIntegration(t *testing.T) {
	server, _, cleanup := setupWebSocketTest(t)
	defer cleanup()

	go server.run()

	// Create test HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", server.handleWebSocket)
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Connect WebSocket client
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Wait for connection and read welcome message (pong)
	ws.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	var welcome Message
	err = ws.ReadJSON(&welcome)
	assert.NoError(t, err)
	assert.Equal(t, MessageTypePong, welcome.Type)

	// Send subscribe message
	subMsg := SubscribeMessage{Topic: "gpio"}
	payload, _ := json.Marshal(subMsg)
	msg := Message{
		Type:      MessageTypeSubscribe,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	err = ws.WriteJSON(msg)
	assert.NoError(t, err)

	// Wait for subscription to be processed
	time.Sleep(50 * time.Millisecond)

	// Broadcast GPIO reading
	reading := GPIOReadingMessage{
		DeviceID: 1,
		Pin:      17,
		Value:    1.0,
	}
	server.BroadcastGPIOReading(reading)

	// Read the GPIO reading with timeout
	ws.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	var received Message
	err = ws.ReadJSON(&received)
	assert.NoError(t, err)
	assert.Equal(t, MessageTypeGPIOReading, received.Type)
}
