package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"github.com/spenceryork/pi-controller/internal/config"
	"github.com/spenceryork/pi-controller/internal/storage"
)

// Server represents the WebSocket server
type Server struct {
	config   *config.WebSocketConfig
	logger   *logrus.Logger
	database *storage.Database
	upgrader websocket.Upgrader
	
	// Client management
	clients    map[*Client]bool
	clientsMux sync.RWMutex
	
	// Message broadcasting
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	
	// Shutdown
	shutdown chan struct{}
}

// Client represents a WebSocket client connection
type Client struct {
	server *Server
	conn   *websocket.Conn
	send   chan []byte
	id     string
	
	// Subscription management
	subscriptions map[string]bool
	subMux        sync.RWMutex
}

// Message types for WebSocket communication
type MessageType string

const (
	MessageTypeSubscribe     MessageType = "subscribe"
	MessageTypeUnsubscribe   MessageType = "unsubscribe"
	MessageTypeGPIOReading   MessageType = "gpio_reading"
	MessageTypeNodeStatus    MessageType = "node_status"
	MessageTypeClusterStatus MessageType = "cluster_status"
	MessageTypeSystemMetrics MessageType = "system_metrics"
	MessageTypeError         MessageType = "error"
	MessageTypePing          MessageType = "ping"
	MessageTypePong          MessageType = "pong"
)

// Message represents a WebSocket message
type Message struct {
	Type      MessageType     `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	RequestID string          `json:"request_id,omitempty"`
}

// SubscribeMessage represents a subscription request
type SubscribeMessage struct {
	Topic string `json:"topic"`
}

// GPIOReadingMessage represents a GPIO reading event
type GPIOReadingMessage struct {
	DeviceID  uint      `json:"device_id"`
	NodeID    uint      `json:"node_id"`
	Pin       int       `json:"pin"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// NodeStatusMessage represents a node status update
type NodeStatusMessage struct {
	NodeID    uint      `json:"node_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	IPAddress string    `json:"ip_address"`
	Timestamp time.Time `json:"timestamp"`
}

// ClusterStatusMessage represents a cluster status update
type ClusterStatusMessage struct {
	ClusterID uint      `json:"cluster_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	NodesReady int      `json:"nodes_ready"`
	NodesTotal int      `json:"nodes_total"`
	Timestamp time.Time `json:"timestamp"`
}

// SystemMetricsMessage represents system metrics
type SystemMetricsMessage struct {
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	Goroutines  int       `json:"goroutines"`
	Timestamp   time.Time `json:"timestamp"`
}

// ErrorMessage represents an error response
type ErrorMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// New creates a new WebSocket server
func New(cfg *config.WebSocketConfig, logger *logrus.Logger, db *storage.Database) *Server {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  cfg.ReadBufferSize,
		WriteBufferSize: cfg.WriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			return !cfg.CheckOrigin
		},
	}

	return &Server{
		config:     cfg,
		logger:     logger,
		database:   db,
		upgrader:   upgrader,
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		shutdown:   make(chan struct{}),
	}
}

// Start starts the WebSocket server
func (s *Server) Start() error {
	// Start the hub
	go s.run()

	// Create HTTP server for WebSocket upgrades
	mux := http.NewServeMux()
	mux.HandleFunc(s.config.Path, s.handleWebSocket)

	server := &http.Server{
		Addr:    s.config.GetAddress(),
		Handler: mux,
	}

	s.logger.WithFields(logrus.Fields{
		"address": s.config.GetAddress(),
		"path":    s.config.Path,
	}).Info("Starting WebSocket server")

	return server.ListenAndServe()
}

// Stop gracefully stops the WebSocket server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Shutting down WebSocket server")
	close(s.shutdown)
	
	// Close all client connections
	s.clientsMux.RLock()
	for client := range s.clients {
		client.conn.Close()
	}
	s.clientsMux.RUnlock()
	
	return nil
}

// run manages the WebSocket hub
func (s *Server) run() {
	ticker := time.NewTicker(54 * time.Second) // Send ping every 54 seconds
	defer ticker.Stop()

	for {
		select {
		case client := <-s.register:
			s.clientsMux.Lock()
			s.clients[client] = true
			s.clientsMux.Unlock()
			
			s.logger.WithField("client_id", client.id).Debug("Client connected")
			
			// Send welcome message
			welcome := Message{
				Type:      MessageTypePong,
				Timestamp: time.Now(),
			}
			s.sendToClient(client, welcome)

		case client := <-s.unregister:
			s.clientsMux.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.send)
			}
			s.clientsMux.Unlock()
			
			s.logger.WithField("client_id", client.id).Debug("Client disconnected")

		case message := <-s.broadcast:
			s.clientsMux.RLock()
			for client := range s.clients {
				select {
				case client.send <- message:
				default:
					delete(s.clients, client)
					close(client.send)
				}
			}
			s.clientsMux.RUnlock()

		case <-ticker.C:
			// Send ping to all clients
			ping := Message{
				Type:      MessageTypePing,
				Timestamp: time.Now(),
			}
			s.BroadcastMessage(ping)

		case <-s.shutdown:
			return
		}
	}
}

// handleWebSocket handles WebSocket upgrade requests
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}

	client := &Client{
		server:        s,
		conn:          conn,
		send:          make(chan []byte, 256),
		id:            generateClientID(),
		subscriptions: make(map[string]bool),
	}

	s.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// BroadcastMessage broadcasts a message to all connected clients
func (s *Server) BroadcastMessage(msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		s.logger.WithError(err).Error("Failed to marshal broadcast message")
		return
	}

	s.broadcast <- data
}

// BroadcastToTopic broadcasts a message to clients subscribed to a specific topic
func (s *Server) BroadcastToTopic(topic string, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		s.logger.WithError(err).Error("Failed to marshal topic message")
		return
	}

	s.clientsMux.RLock()
	for client := range s.clients {
		if client.isSubscribedTo(topic) {
			select {
			case client.send <- data:
			default:
				delete(s.clients, client)
				close(client.send)
			}
		}
	}
	s.clientsMux.RUnlock()
}

// sendToClient sends a message to a specific client
func (s *Server) sendToClient(client *Client, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		s.logger.WithError(err).Error("Failed to marshal client message")
		return
	}

	select {
	case client.send <- data:
	default:
		s.clientsMux.Lock()
		delete(s.clients, client)
		close(client.send)
		s.clientsMux.Unlock()
	}
}

// BroadcastGPIOReading broadcasts a GPIO reading to subscribed clients
func (s *Server) BroadcastGPIOReading(reading GPIOReadingMessage) {
	payload, _ := json.Marshal(reading)
	msg := Message{
		Type:      MessageTypeGPIOReading,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	
	s.BroadcastToTopic("gpio", msg)
}

// BroadcastNodeStatus broadcasts a node status update to subscribed clients
func (s *Server) BroadcastNodeStatus(status NodeStatusMessage) {
	payload, _ := json.Marshal(status)
	msg := Message{
		Type:      MessageTypeNodeStatus,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	
	s.BroadcastToTopic("nodes", msg)
}

// BroadcastClusterStatus broadcasts a cluster status update to subscribed clients
func (s *Server) BroadcastClusterStatus(status ClusterStatusMessage) {
	payload, _ := json.Marshal(status)
	msg := Message{
		Type:      MessageTypeClusterStatus,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	
	s.BroadcastToTopic("clusters", msg)
}

// BroadcastSystemMetrics broadcasts system metrics to subscribed clients
func (s *Server) BroadcastSystemMetrics(metrics SystemMetricsMessage) {
	payload, _ := json.Marshal(metrics)
	msg := Message{
		Type:      MessageTypeSystemMetrics,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	
	s.BroadcastToTopic("system", msg)
}

// Client methods

// readPump handles reading messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.server.unregister <- c
		c.conn.Close()
	}()

	// Set read deadline and pong handler
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, messageData, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.server.logger.WithError(err).Error("WebSocket read error")
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(messageData, &msg); err != nil {
			c.sendError(400, "Invalid message format")
			continue
		}

		c.handleMessage(msg)
	}
}

// writePump handles writing messages to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming messages from clients
func (c *Client) handleMessage(msg Message) {
	switch msg.Type {
	case MessageTypeSubscribe:
		var sub SubscribeMessage
		if err := json.Unmarshal(msg.Payload, &sub); err != nil {
			c.sendError(400, "Invalid subscribe message")
			return
		}
		c.subscribe(sub.Topic)

	case MessageTypeUnsubscribe:
		var unsub SubscribeMessage
		if err := json.Unmarshal(msg.Payload, &unsub); err != nil {
			c.sendError(400, "Invalid unsubscribe message")
			return
		}
		c.unsubscribe(unsub.Topic)

	case MessageTypePing:
		// Respond with pong
		pong := Message{
			Type:      MessageTypePong,
			Timestamp: time.Now(),
			RequestID: msg.RequestID,
		}
		c.server.sendToClient(c, pong)

	default:
		c.sendError(400, "Unknown message type")
	}
}

// subscribe adds a topic subscription for the client
func (c *Client) subscribe(topic string) {
	c.subMux.Lock()
	c.subscriptions[topic] = true
	c.subMux.Unlock()
	
	c.server.logger.WithFields(logrus.Fields{
		"client_id": c.id,
		"topic":     topic,
	}).Debug("Client subscribed to topic")
}

// unsubscribe removes a topic subscription for the client
func (c *Client) unsubscribe(topic string) {
	c.subMux.Lock()
	delete(c.subscriptions, topic)
	c.subMux.Unlock()
	
	c.server.logger.WithFields(logrus.Fields{
		"client_id": c.id,
		"topic":     topic,
	}).Debug("Client unsubscribed from topic")
}

// isSubscribedTo checks if the client is subscribed to a topic
func (c *Client) isSubscribedTo(topic string) bool {
	c.subMux.RLock()
	defer c.subMux.RUnlock()
	return c.subscriptions[topic]
}

// sendError sends an error message to the client
func (c *Client) sendError(code int, message string) {
	errMsg := ErrorMessage{
		Code:    code,
		Message: message,
	}
	payload, _ := json.Marshal(errMsg)
	
	msg := Message{
		Type:      MessageTypeError,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	
	c.server.sendToClient(c, msg)
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return time.Now().Format("20060102150405") + "-" + generateRandomString(6)
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}