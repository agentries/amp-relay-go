package server

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/openclaw/amp-relay-go/internal/auth"
	"github.com/openclaw/amp-relay-go/internal/protocol"
	"github.com/openclaw/amp-relay-go/internal/storage"
)

// getFreePort asks the OS for a free TCP port on localhost.
func getFreePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}

// TestNewRelayServer verifies that NewRelayServer returns a non-nil server
// with all fields properly initialized.
func TestNewRelayServer(t *testing.T) {
	cfg := DefaultConfig()
	srv := NewRelayServer(cfg)

	if srv == nil {
		t.Fatal("NewRelayServer returned nil")
	}
	if srv.config != cfg {
		t.Error("config field not set correctly")
	}
	if srv.store == nil {
		t.Error("store field is nil; expected config.Storage")
	}
	if srv.store != cfg.Storage {
		t.Error("store does not match config.Storage")
	}
	if srv.clients == nil {
		t.Error("clients map is nil")
	}
	if srv.routes == nil {
		t.Error("routes map is nil")
	}
	if srv.ctx == nil {
		t.Error("ctx is nil")
	}
	if srv.cancel == nil {
		t.Error("cancel func is nil")
	}
	if srv.running.Load() {
		t.Error("server should not be running immediately after creation")
	}
}

// TestDefaultConfig verifies that DefaultConfig returns sensible defaults.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if cfg.ListenAddr != ":8080" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, ":8080")
	}
	if cfg.AllowedOrigins != nil {
		t.Errorf("AllowedOrigins = %v, want nil (allow all in dev mode)", cfg.AllowedOrigins)
	}
	if cfg.Authenticator == nil {
		t.Error("Authenticator is nil, want NoOpAuthenticator")
	}
	// Verify it is a NoOpAuthenticator by type assertion
	if _, ok := cfg.Authenticator.(*auth.NoOpAuthenticator); !ok {
		t.Errorf("Authenticator type = %T, want *auth.NoOpAuthenticator", cfg.Authenticator)
	}
	if cfg.Storage == nil {
		t.Error("Storage is nil, want MemoryStore")
	}
	if _, ok := cfg.Storage.(*storage.MemoryStore); !ok {
		t.Errorf("Storage type = %T, want *storage.MemoryStore", cfg.Storage)
	}
	if cfg.DefaultTTL != 5*time.Minute {
		t.Errorf("DefaultTTL = %v, want %v", cfg.DefaultTTL, 5*time.Minute)
	}
	expectedMaxPayload := int64(512 * 1024)
	if cfg.MaxPayloadSize != expectedMaxPayload {
		t.Errorf("MaxPayloadSize = %d, want %d", cfg.MaxPayloadSize, expectedMaxPayload)
	}
	if cfg.RateLimitPerMinute != 60 {
		t.Errorf("RateLimitPerMinute = %d, want 60", cfg.RateLimitPerMinute)
	}
}

// TestRelayServer_StartStop verifies that the server can be started and
// stopped cleanly, and that the running state reflects the lifecycle.
func TestRelayServer_StartStop(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ListenAddr = getFreePort(t)

	srv := NewRelayServer(cfg)

	if srv.running.Load() {
		t.Fatal("server should not be running before Start()")
	}

	if err := srv.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	if !srv.running.Load() {
		t.Error("server should be running after Start()")
	}

	// Give the HTTP server a moment to bind
	time.Sleep(50 * time.Millisecond)

	if err := srv.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}

	if srv.running.Load() {
		t.Error("server should not be running after Stop()")
	}
}

// TestRelayServer_Start_AlreadyRunning verifies that calling Start() on
// an already-running server returns an error.
func TestRelayServer_Start_AlreadyRunning(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ListenAddr = getFreePort(t)

	srv := NewRelayServer(cfg)

	if err := srv.Start(); err != nil {
		t.Fatalf("first Start() error: %v", err)
	}
	defer srv.Stop()

	err := srv.Start()
	if err == nil {
		t.Fatal("second Start() should return an error, got nil")
	}
	expectedMsg := "server already running"
	if err.Error() != expectedMsg {
		t.Errorf("error message = %q, want %q", err.Error(), expectedMsg)
	}
}

// TestRelayServer_Stop_NotRunning verifies that Stop() on a server that
// was never started is a no-op (returns nil error).
func TestRelayServer_Stop_NotRunning(t *testing.T) {
	cfg := DefaultConfig()
	srv := NewRelayServer(cfg)

	err := srv.Stop()
	if err != nil {
		t.Errorf("Stop() on non-running server returned error: %v", err)
	}
}

// TestRelayServer_RegisterRoute verifies that RegisterRoute stores the handler
// and that it can be retrieved from the routes map.
func TestRelayServer_RegisterRoute(t *testing.T) {
	cfg := DefaultConfig()
	srv := NewRelayServer(cfg)

	called := false
	handler := func(msg *protocol.Message) (*protocol.Message, error) {
		called = true
		return nil, nil
	}

	action := "test.action"
	srv.RegisterRoute(action, handler)

	// Verify the route exists by reading the internal map
	srv.routesMu.RLock()
	h, exists := srv.routes[action]
	srv.routesMu.RUnlock()

	if !exists {
		t.Fatalf("route %q not found after RegisterRoute", action)
	}
	if h == nil {
		t.Fatal("handler is nil")
	}

	// Invoke the handler to confirm it is the one we registered
	_, _ = h(protocol.NewMessage(protocol.MessageTypeRequest, "from", "to", nil))
	if !called {
		t.Error("registered handler was not invoked")
	}

	// Verify UnregisterRoute removes it
	srv.UnregisterRoute(action)
	srv.routesMu.RLock()
	_, exists = srv.routes[action]
	srv.routesMu.RUnlock()

	if exists {
		t.Errorf("route %q still present after UnregisterRoute", action)
	}
}

// TestRelayServer_RegisterRoute_Multiple verifies that multiple routes can
// coexist and be independently managed.
func TestRelayServer_RegisterRoute_Multiple(t *testing.T) {
	cfg := DefaultConfig()
	srv := NewRelayServer(cfg)

	actions := []string{"action.a", "action.b", "action.c"}
	for _, a := range actions {
		a := a
		srv.RegisterRoute(a, func(msg *protocol.Message) (*protocol.Message, error) {
			return nil, fmt.Errorf("handler-%s", a)
		})
	}

	srv.routesMu.RLock()
	routeCount := len(srv.routes)
	srv.routesMu.RUnlock()

	if routeCount != len(actions) {
		t.Errorf("route count = %d, want %d", routeCount, len(actions))
	}

	// Remove one and check that others remain
	srv.UnregisterRoute("action.b")

	srv.routesMu.RLock()
	_, bExists := srv.routes["action.b"]
	_, aExists := srv.routes["action.a"]
	_, cExists := srv.routes["action.c"]
	srv.routesMu.RUnlock()

	if bExists {
		t.Error("action.b should have been removed")
	}
	if !aExists {
		t.Error("action.a should still exist")
	}
	if !cExists {
		t.Error("action.c should still exist")
	}
}

// TestRelayServer_GetStats verifies that GetStats reflects the current state
// of the server.
func TestRelayServer_GetStats(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ListenAddr = ":9999"
	srv := NewRelayServer(cfg)

	// Before starting, stats should show not running
	stats := srv.GetStats()
	if stats.Running {
		t.Error("Running should be false before Start()")
	}
	if stats.Address != ":9999" {
		t.Errorf("Address = %q, want %q", stats.Address, ":9999")
	}
	if stats.ConnectedClients != 0 {
		t.Errorf("ConnectedClients = %d, want 0", stats.ConnectedClients)
	}

	// Simulate adding clients to the internal map (without actually starting)
	srv.clientsMu.Lock()
	srv.clients["client-1"] = &ClientInfo{
		ID:           "client-1",
		DID:          "did:example:1",
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		Metadata:     make(map[string]string),
	}
	srv.clients["client-2"] = &ClientInfo{
		ID:           "client-2",
		DID:          "did:example:2",
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		Metadata:     make(map[string]string),
	}
	srv.clientsMu.Unlock()

	stats = srv.GetStats()
	if stats.ConnectedClients != 2 {
		t.Errorf("ConnectedClients = %d, want 2", stats.ConnectedClients)
	}

	// Simulate running state
	srv.running.Store(true)
	stats = srv.GetStats()
	if !stats.Running {
		t.Error("Running should be true after setting running flag")
	}
}

// TestExtractAction tests the unexported extractAction helper with various
// body types.
func TestExtractAction(t *testing.T) {
	tests := []struct {
		name     string
		body     interface{}
		expected string
	}{
		{
			name:     "map[string]interface{} with action",
			body:     map[string]interface{}{"action": "relay.forward", "payload": "data"},
			expected: "relay.forward",
		},
		{
			name:     "map[string]interface{} without action key",
			body:     map[string]interface{}{"type": "something"},
			expected: "",
		},
		{
			name:     "map[string]interface{} with non-string action",
			body:     map[string]interface{}{"action": 42},
			expected: "",
		},
		{
			name:     "map[interface{}]interface{} with action",
			body:     map[interface{}]interface{}{"action": "relay.broadcast"},
			expected: "relay.broadcast",
		},
		{
			name:     "map[interface{}]interface{} without action key",
			body:     map[interface{}]interface{}{"type": "other"},
			expected: "",
		},
		{
			name:     "map[interface{}]interface{} with non-string action value",
			body:     map[interface{}]interface{}{"action": 99},
			expected: "",
		},
		{
			name:     "nil body",
			body:     nil,
			expected: "",
		},
		{
			name:     "string body (non-map)",
			body:     "just a string",
			expected: "",
		},
		{
			name:     "int body (non-map)",
			body:     12345,
			expected: "",
		},
		{
			name:     "slice body (non-map)",
			body:     []string{"a", "b"},
			expected: "",
		},
		{
			name:     "empty map[string]interface{}",
			body:     map[string]interface{}{},
			expected: "",
		},
		{
			name:     "empty map[interface{}]interface{}",
			body:     map[interface{}]interface{}{},
			expected: "",
		},
		{
			name:     "map[string]interface{} with empty string action",
			body:     map[string]interface{}{"action": ""},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := &protocol.Message{Body: tc.body}
			result := extractAction(msg)
			if result != tc.expected {
				t.Errorf("extractAction() = %q, want %q", result, tc.expected)
			}
		})
	}
}
