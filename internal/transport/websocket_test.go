package transport

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewWebSocketServer(t *testing.T) {
	server := NewWebSocketServer(":0")
	if server == nil {
		t.Fatal("NewWebSocketServer returned nil")
	}
	if server.clients == nil {
		t.Error("clients map should be initialized")
	}
	if server.register == nil {
		t.Error("register channel should be initialized")
	}
	if server.unregister == nil {
		t.Error("unregister channel should be initialized")
	}
	if server.broadcast == nil {
		t.Error("broadcast channel should be initialized")
	}
	if server.ctx == nil {
		t.Error("context should be initialized")
	}
}

func TestWebSocketServer_SetMessageHandler(t *testing.T) {
	server := NewWebSocketServer(":0")

	handler := func(clientID string, data []byte) error {
		return nil
	}

	server.SetMessageHandler(handler)
	if server.messageHandler == nil {
		t.Error("messageHandler should be set")
	}
}

func TestWebSocketServer_StartStop(t *testing.T) {
	server := NewWebSocketServer(":0")
	err := server.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !server.running.Load() {
		t.Error("Server should be running after Start")
	}

	err = server.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if server.running.Load() {
		t.Error("Server should not be running after Stop")
	}
}

func TestWebSocketServer_Start_AlreadyRunning(t *testing.T) {
	server := NewWebSocketServer(":0")
	err := server.Start()
	if err != nil {
		t.Fatalf("First Start failed: %v", err)
	}
	defer server.Stop()

	// Second start should be no-op (returns nil)
	err = server.Start()
	if err != nil {
		t.Errorf("Second Start should not error: %v", err)
	}
}

func TestWebSocketServer_Stop_NotRunning(t *testing.T) {
	server := NewWebSocketServer(":0")

	// Stop server that was never started
	err := server.Stop()
	if err != nil {
		t.Errorf("Stop on non-running server should not error: %v", err)
	}
}

func TestWebSocketServer_HealthEndpoint(t *testing.T) {
	server := NewWebSocketServer(":0")
	err := server.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop()

	// Get the actual listen address
	addr := server.server.Addr
	if addr == ":0" {
		// Server is listening, but we need the actual address.
		// Use the handler directly for testing.
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		server.handleHealth(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), `"status":"ok"`) {
			t.Errorf("Expected health response, got %s", w.Body.String())
		}
	}
}

func TestWebSocketServer_WebSocketConnection(t *testing.T) {
	// Create test server
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Upgrade failed: %v", err)
		}
		defer conn.Close()

		// Echo back any received message
		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			conn.WriteMessage(mt, message)
		}
	}))
	defer s.Close()

	// Connect WebSocket client
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http") + "/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer ws.Close()

	// Send message
	testMessage := []byte("hello, websocket!")
	err = ws.WriteMessage(websocket.BinaryMessage, testMessage)
	if err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	// Read response
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	mt, received, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	if mt != websocket.BinaryMessage {
		t.Errorf("Expected binary message, got %d", mt)
	}
	if string(received) != string(testMessage) {
		t.Errorf("Expected %s, got %s", testMessage, received)
	}
}

func TestWebSocketServer_Broadcast(t *testing.T) {
	server := NewWebSocketServer(":0")

	// Cannot fully test broadcast without connected clients,
	// but we can verify the method doesn't panic
	server.Broadcast([]byte("test broadcast"))

	// Verify running status
	if server.running.Load() {
		t.Error("Server should not be running yet")
	}
}

func TestWebSocketServer_SendToClient(t *testing.T) {
	server := NewWebSocketServer(":0")

	// Test sending to non-existent client
	sent := server.SendToClient("non-existent", []byte("test"))
	if sent {
		t.Error("SendToClient should return false for non-existent client")
	}
}

func TestWebSocketServer_GetClientCount(t *testing.T) {
	server := NewWebSocketServer(":0")

	count := server.GetClientCount()
	if count != 0 {
		t.Errorf("Expected 0 clients, got %d", count)
	}
}

func TestClient_Close(t *testing.T) {
	// Create test server
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Upgrade failed: %v", err)
		}
		defer conn.Close()

		// Keep connection open until closed
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer s.Close()

	// Connect client
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http") + "/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}

	// Create client struct
	client := &Client{
		ID:       "test-client",
		Conn:     ws,
		SendChan: make(chan []byte, 256),
	}

	// Test IsClosed
	if client.IsClosed() {
		t.Error("Client should not be closed initially")
	}

	// Test Close
	client.Close()

	if !client.IsClosed() {
		t.Error("Client should be closed after Close()")
	}

	// Double close should not panic
	client.Close()
}

func TestWebSocketServer_UpgraderConfiguration(t *testing.T) {
	server := NewWebSocketServer(":0")

	// Check default upgrader configuration
	if server.Upgrader.ReadBufferSize != 1024 {
		t.Errorf("Expected ReadBufferSize 1024, got %d", server.Upgrader.ReadBufferSize)
	}
	if server.Upgrader.WriteBufferSize != 1024 {
		t.Errorf("Expected WriteBufferSize 1024, got %d", server.Upgrader.WriteBufferSize)
	}

	// Check that CheckOrigin is set
	if server.Upgrader.CheckOrigin == nil {
		t.Error("CheckOrigin should be set")
	}

	// Test CheckOrigin allows all origins
	req := httptest.NewRequest("GET", "http://example.com", nil)
	if !server.Upgrader.CheckOrigin(req) {
		t.Error("CheckOrigin should allow all origins")
	}
}

func TestGenerateClientID(t *testing.T) {
	id1 := generateClientID()

	if id1 == "" {
		t.Error("generateClientID should not return empty string")
	}
	if !strings.HasPrefix(id1, "client_") {
		t.Errorf("Expected ID to start with 'client_', got %s", id1)
	}
	// Verify format is correct
	if len(id1) < 10 {
		t.Error("ID should have reasonable length")
	}
}

func TestRandomString(t *testing.T) {
	str := randomString(10)
	if len(str) != 10 {
		t.Errorf("Expected length 10, got %d", len(str))
	}

	// Verify the string contains only valid characters
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	for _, ch := range str {
		found := false
		for _, valid := range charset {
			if ch == valid {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Invalid character in random string: %c", ch)
		}
	}
}

func TestWebSocketServer_ContextCancellation(t *testing.T) {
	server := NewWebSocketServer(":0")

	// Cancel context before starting
	server.cancel()

	// Context should be done
	select {
	case <-server.ctx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled")
	}
}

func TestWebSocketServer_Integration(t *testing.T) {
	server := NewWebSocketServer(":0")
	handler := func(clientID string, data []byte) error {
		return nil
	}
	server.SetMessageHandler(handler)

	err := server.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop()

	if !server.running.Load() {
		t.Error("Server should be running")
	}
	if server.GetClientCount() != 0 {
		t.Errorf("Expected 0 clients, got %d", server.GetClientCount())
	}
}

func BenchmarkWebSocketServer_Broadcast(b *testing.B) {
	server := NewWebSocketServer(":0")
	server.Start()
	defer server.Stop()

	message := []byte("benchmark broadcast message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.Broadcast(message)
	}
}

func BenchmarkGenerateClientID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateClientID()
	}
}
