// Package server provides the core AMP Relay Server implementation
package server

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/agentries/amp-relay-go/internal/protocol"
	"github.com/agentries/amp-relay-go/internal/storage"
	"github.com/agentries/amp-relay-go/internal/transport"
)

// Config holds server configuration
type Config struct {
	// Network configuration
	ListenAddr string

	// Storage configuration
	Storage storage.MessageStore

	// Message handling
	DefaultTTL     time.Duration
	MaxPayloadSize int64

	// Rate limiting
	RateLimitPerMinute int
}

// DefaultConfig returns a default server configuration
func DefaultConfig() *Config {
	return &Config{
		ListenAddr:         ":8080",
		Storage:            storage.NewMemoryStore(),
		DefaultTTL:         5 * time.Minute,
		MaxPayloadSize:     512 * 1024, // 512KB
		RateLimitPerMinute: 60,
	}
}

// RelayServer is the main AMP Relay Server
type RelayServer struct {
	config *Config

	// Transport layer
	wsServer *transport.WebSocketServer

	// Storage
	store storage.MessageStore

	// Client management
	clients   map[string]*ClientInfo
	clientsMu sync.RWMutex

	// Message routing
	routes   map[string]RouteHandler
	routesMu sync.RWMutex

	// Lifecycle
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
}

// ClientInfo holds information about a connected client
type ClientInfo struct {
	ID           string
	DID          string // Decentralized Identifier
	ConnectedAt  time.Time
	LastActivity time.Time
	Metadata     map[string]string
}

// RouteHandler is a function that handles messages for a specific action
type RouteHandler func(msg *protocol.Message) (*protocol.Message, error)

// NewRelayServer creates a new AMP Relay Server instance
func NewRelayServer(config *Config) *RelayServer {
	ctx, cancel := context.WithCancel(context.Background())

	return &RelayServer{
		config:  config,
		store:   config.Storage,
		clients: make(map[string]*ClientInfo),
		routes:  make(map[string]RouteHandler),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start starts the relay server
func (s *RelayServer) Start() error {
	if s.running {
		return fmt.Errorf("server already running")
	}

	// Create WebSocket server
	s.wsServer = transport.NewWebSocketServer(s.config.ListenAddr)
	s.wsServer.SetMessageHandler(s.handleWebSocketMessage)

	// Start WebSocket server
	if err := s.wsServer.Start(); err != nil {
		return fmt.Errorf("failed to start WebSocket server: %w", err)
	}

	s.running = true

	// Start background tasks
	s.wg.Add(1)
	go s.cleanupLoop()

	log.Printf("AMP Relay Server started on %s", s.config.ListenAddr)
	return nil
}

// Stop gracefully stops the relay server
func (s *RelayServer) Stop() error {
	if !s.running {
		return nil
	}

	log.Println("Stopping AMP Relay Server...")

	// Signal shutdown
	s.cancel()

	// Stop WebSocket server
	if s.wsServer != nil {
		if err := s.wsServer.Stop(); err != nil {
			log.Printf("Error stopping WebSocket server: %v", err)
		}
	}

	// Wait for background tasks
	s.wg.Wait()

	s.running = false
	log.Println("AMP Relay Server stopped")
	return nil
}

// RegisterRoute registers a handler for a specific action
func (s *RelayServer) RegisterRoute(action string, handler RouteHandler) {
	s.routesMu.Lock()
	defer s.routesMu.Unlock()
	s.routes[action] = handler
}

// UnregisterRoute removes a route handler
func (s *RelayServer) UnregisterRoute(action string) {
	s.routesMu.Lock()
	defer s.routesMu.Unlock()
	delete(s.routes, action)
}

// GetStats returns server statistics
func (s *RelayServer) GetStats() ServerStats {
	s.clientsMu.RLock()
	clientCount := len(s.clients)
	s.clientsMu.RUnlock()

	return ServerStats{
		ConnectedClients: clientCount,
		Address:          s.config.ListenAddr,
		Running:          s.running,
	}
}

// ServerStats holds server statistics
type ServerStats struct {
	ConnectedClients int
	Address          string
	Running          bool
}

// handleWebSocketMessage processes incoming WebSocket messages
func (s *RelayServer) handleWebSocketMessage(clientID string, data []byte) error {
	// Decode CBOR message
	msg := &protocol.Message{}
	if err := msg.CBORUnmarshal(data); err != nil {
		log.Printf("Failed to decode message from client %s: %v", clientID, err)
		return fmt.Errorf("invalid message format: %w", err)
	}

	// Update client info
	s.updateClientActivity(clientID)

	// Process message based on type
	switch msg.Type {
	case protocol.MessageTypeRequest:
		return s.handleRequest(clientID, msg)
	case protocol.MessageTypeEvent:
		return s.handleEvent(clientID, msg)
	default:
		log.Printf("Unsupported message type from client %s: %s", clientID, msg.Type)
		return fmt.Errorf("unsupported message type: %s", msg.Type)
	}
}

// handleRequest processes request messages
func (s *RelayServer) handleRequest(clientID string, msg *protocol.Message) error {
	// Store the message
	ttl := s.config.DefaultTTL
	if msg.TTL > 0 {
		ttl = time.Duration(msg.TTL) * time.Second
	}

	if err := s.store.Save(msg, ttl); err != nil {
		log.Printf("Failed to store message: %v", err)
		return s.sendErrorResponse(clientID, msg, "storage_error", "Failed to store message")
	}

	// Route the message if a handler exists
	s.routesMu.RLock()
	handler, exists := s.routes[msg.Action]
	s.routesMu.RUnlock()

	if exists {
		response, err := handler(msg)
		if err != nil {
			log.Printf("Route handler error for action %s: %v", msg.Action, err)
			return s.sendErrorResponse(clientID, msg, "handler_error", err.Error())
		}

		if response != nil {
			// Send response back to client
			return s.sendResponse(clientID, msg.ID, response)
		}
	}

	// Forward to destination if specified
	if msg.Destination != "" && msg.Destination != "relay-server" {
		return s.forwardMessage(msg)
	}

	return nil
}

// handleEvent processes event messages
func (s *RelayServer) handleEvent(clientID string, msg *protocol.Message) error {
	// Store event
	ttl := s.config.DefaultTTL
	if msg.TTL > 0 {
		ttl = time.Duration(msg.TTL) * time.Second
	}

	if err := s.store.Save(msg, ttl); err != nil {
		log.Printf("Failed to store event: %v", err)
		return err
	}

	// Broadcast to all clients except sender
	s.clientsMu.RLock()
	clients := make([]string, 0, len(s.clients))
	for id := range s.clients {
		if id != clientID {
			clients = append(clients, id)
		}
	}
	s.clientsMu.RUnlock()

	// Forward to each client
	for _, targetID := range clients {
		if err := s.forwardMessageToClient(targetID, msg); err != nil {
			log.Printf("Failed to forward event to client %s: %v", targetID, err)
		}
	}

	return nil
}

// forwardMessage forwards a message to its destination
func (s *RelayServer) forwardMessage(msg *protocol.Message) error {
	// Try to find the destination client
	s.clientsMu.RLock()
	for clientID, info := range s.clients {
		if info.DID == msg.Destination {
			s.clientsMu.RUnlock()
			return s.forwardMessageToClient(clientID, msg)
		}
	}
	s.clientsMu.RUnlock()

	// Destination not found, message stays in store for later retrieval
	log.Printf("Destination %s not connected, message stored for later delivery", msg.Destination)
	return nil
}

// forwardMessageToClient sends a message to a specific client
func (s *RelayServer) forwardMessageToClient(clientID string, msg *protocol.Message) error {
	data, err := msg.CBORMarshal()
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if !s.wsServer.SendToClient(clientID, data) {
		return fmt.Errorf("failed to send to client %s", clientID)
	}

	return nil
}

// sendResponse sends a response message
func (s *RelayServer) sendResponse(clientID string, requestID string, response *protocol.Message) error {
	response.CorrelationID = requestID
	response.Type = protocol.MessageTypeResponse

	data, err := response.CBORMarshal()
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	if !s.wsServer.SendToClient(clientID, data) {
		return fmt.Errorf("failed to send response to client %s", clientID)
	}

	return nil
}

// sendErrorResponse sends an error response
func (s *RelayServer) sendErrorResponse(clientID string, originalMsg *protocol.Message, code string, message string) error {
	errorMsg := protocol.NewMessage(
		protocol.MessageTypeError,
		"relay-server",
		originalMsg.Source,
		"error",
		[]byte(message),
	)
	errorMsg.CorrelationID = originalMsg.ID
	errorMsg.AddMetadata("error_code", code)

	data, err := errorMsg.CBORMarshal()
	if err != nil {
		return err
	}

	if !s.wsServer.SendToClient(clientID, data) {
		return fmt.Errorf("failed to send error response")
	}

	return nil
}

// updateClientActivity updates client activity timestamp
func (s *RelayServer) updateClientActivity(clientID string) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	if client, exists := s.clients[clientID]; exists {
		client.LastActivity = time.Now()
	} else {
		// New client
		s.clients[clientID] = &ClientInfo{
			ID:           clientID,
			ConnectedAt:  time.Now(),
			LastActivity: time.Now(),
			Metadata:     make(map[string]string),
		}
	}
}

// cleanupLoop runs periodic cleanup tasks
func (s *RelayServer) cleanupLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanupInactiveClients()
		}
	}
}

// cleanupInactiveClients removes clients that haven't been active for a while
func (s *RelayServer) cleanupInactiveClients() {
	cutoff := time.Now().Add(-5 * time.Minute)

	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	for id, client := range s.clients {
		if client.LastActivity.Before(cutoff) {
			delete(s.clients, id)
			log.Printf("Removed inactive client: %s", id)
		}
	}
}
