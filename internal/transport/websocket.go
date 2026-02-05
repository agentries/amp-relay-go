// Package transport provides WebSocket transport layer for AMP Relay Server
package transport

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// MessageHandler is the callback function for handling incoming messages
type MessageHandler func(clientID string, data []byte) error

// Client represents a connected WebSocket client
type Client struct {
	ID       string
	Conn     *websocket.Conn
	Server   *WebSocketServer
	SendChan chan []byte
	mu       sync.RWMutex
	closed   bool
}

// WebSocketServer manages WebSocket connections
type WebSocketServer struct {
	// Server configuration
	Addr     string
	Upgrader websocket.Upgrader

	// Connection management
	clients    map[string]*Client
	clientsMu  sync.RWMutex
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte

	// Lifecycle management
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running atomic.Bool

	// Message handler callback
	messageHandler MessageHandler

	// HTTP server
	server *http.Server
}

// NewWebSocketServer creates a new WebSocket server instance
func NewWebSocketServer(addr string) *WebSocketServer {
	ctx, cancel := context.WithCancel(context.Background())

	return &WebSocketServer{
		Addr: addr,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins in development
				// TODO: Configure allowed origins for production
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// SetMessageHandler sets the callback function for handling messages
func (ws *WebSocketServer) SetMessageHandler(handler MessageHandler) {
	ws.messageHandler = handler
}

// Start starts the WebSocket server
func (ws *WebSocketServer) Start() error {
	if ws.running.Load() {
		return nil
	}

	ws.running.Store(true)

	// Start the hub goroutine for managing connections
	ws.wg.Add(1)
	go ws.runHub()

	// Setup HTTP handlers on a local mux
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", ws.handleWebSocket)
	mux.HandleFunc("/health", ws.handleHealth)

	// Create HTTP server
	ws.server = &http.Server{
		Addr:    ws.Addr,
		Handler: mux,
	}

	log.Printf("WebSocket server starting on %s", ws.Addr)

	// Start listening in a goroutine
	ws.wg.Add(1)
	go func() {
		defer ws.wg.Done()
		if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("WebSocket server error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully stops the WebSocket server
func (ws *WebSocketServer) Stop() error {
	if !ws.running.Load() {
		return nil
	}

	log.Println("Stopping WebSocket server...")

	// Signal all goroutines to stop
	ws.cancel()

	// Close all client connections
	ws.clientsMu.Lock()
	for _, client := range ws.clients {
		client.Close()
	}
	ws.clients = make(map[string]*Client)
	ws.clientsMu.Unlock()

	// Shutdown HTTP server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ws.server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// Wait for all goroutines to finish
	ws.wg.Wait()

	ws.running.Store(false)
	log.Println("WebSocket server stopped")
	return nil
}

// Broadcast sends a message to all connected clients
func (ws *WebSocketServer) Broadcast(data []byte) {
	select {
	case ws.broadcast <- data:
	case <-time.After(100 * time.Millisecond):
		log.Println("Broadcast timeout: channel full")
	}
}

// SendToClient sends a message to a specific client
func (ws *WebSocketServer) SendToClient(clientID string, data []byte) bool {
	ws.clientsMu.RLock()
	client, exists := ws.clients[clientID]
	ws.clientsMu.RUnlock()

	if !exists {
		return false
	}

	select {
	case client.SendChan <- data:
		return true
	case <-time.After(100 * time.Millisecond):
		return false
	}
}

// GetClientCount returns the number of connected clients
func (ws *WebSocketServer) GetClientCount() int {
	ws.clientsMu.RLock()
	defer ws.clientsMu.RUnlock()
	return len(ws.clients)
}

// handleWebSocket handles WebSocket upgrade requests
func (ws *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := ws.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Generate client ID
	clientID := generateClientID()

	// Create client
	client := &Client{
		ID:       clientID,
		Conn:     conn,
		Server:   ws,
		SendChan: make(chan []byte, 256),
	}

	// Register client
	ws.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()

	log.Printf("Client %s connected from %s", clientID, r.RemoteAddr)
}

// handleHealth provides health check endpoint
func (ws *WebSocketServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"status":"ok","clients":%d}`, ws.GetClientCount())))
}

// runHub manages client registration/unregistration and broadcasting
func (ws *WebSocketServer) runHub() {
	defer ws.wg.Done()

	for {
		select {
		case <-ws.ctx.Done():
			return

		case client := <-ws.register:
			ws.clientsMu.Lock()
			ws.clients[client.ID] = client
			ws.clientsMu.Unlock()

		case client := <-ws.unregister:
			ws.clientsMu.Lock()
			if _, exists := ws.clients[client.ID]; exists {
				delete(ws.clients, client.ID)
				close(client.SendChan)
			}
			ws.clientsMu.Unlock()

		case message := <-ws.broadcast:
			ws.clientsMu.RLock()
			clients := make([]*Client, 0, len(ws.clients))
			for _, client := range ws.clients {
				clients = append(clients, client)
			}
			ws.clientsMu.RUnlock()

			// Send to all clients
			for _, client := range clients {
				select {
				case client.SendChan <- message:
				default:
					// Client send buffer full, close connection
					client.Close()
				}
			}
		}
	}
}

// readPump handles incoming messages from client
func (c *Client) readPump() {
	defer func() {
		c.Server.unregister <- c
		c.Conn.Close()
	}()

	// Configure connection
	c.Conn.SetReadLimit(512 * 1024) // 512KB max message size
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for client %s: %v", c.ID, err)
			}
			break
		}

		// Reset read deadline
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// Call message handler if set
		if c.Server.messageHandler != nil {
			if err := c.Server.messageHandler(c.ID, message); err != nil {
				log.Printf("Message handler error for client %s: %v", c.ID, err)
			}
		}
	}
}

// writePump handles outgoing messages to client
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.SendChan:
			if !ok {
				// Channel closed
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				log.Printf("Write error for client %s: %v", c.ID, err)
				return
			}

		case <-ticker.C:
			// Send ping
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.Server.ctx.Done():
			return
		}
	}
}

// Close closes the client connection
func (c *Client) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()

	c.Conn.Close()
}

// IsClosed checks if client connection is closed
func (c *Client) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return "client_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// randomString generates a cryptographically secure random string
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))
	for i := range b {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			// Fallback to less secure but still functional approach
			b[i] = charset[i%len(charset)]
			continue
		}
		b[i] = charset[n.Int64()]
	}
	return string(b)
}
