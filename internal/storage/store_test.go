package storage

import (
	"fmt"
	"testing"
	"time"

	"github.com/openclaw/amp-relay-go/internal/protocol"
)

func TestNewMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	if store == nil {
		t.Fatal("NewMemoryStore returned nil")
	}
	if store.messages == nil {
		t.Error("messages map should be initialized")
	}
}

func TestMemoryStore_Save(t *testing.T) {
	store := NewMemoryStore()
	msg := protocol.NewMessage(protocol.MessageTypeRequest, "source", "dest", "action", []byte("payload"))

	// Test saving without TTL
	err := store.Save(msg, 0)
	if err != nil {
		t.Errorf("Save without TTL failed: %v", err)
	}

	// Test saving with TTL
	msg2 := protocol.NewMessage(protocol.MessageTypeRequest, "source2", "dest2", "action2", []byte("payload2"))
	err = store.Save(msg2, 5*time.Minute)
	if err != nil {
		t.Errorf("Save with TTL failed: %v", err)
	}
}

func TestMemoryStore_Get(t *testing.T) {
	store := NewMemoryStore()
	msg := protocol.NewMessage(protocol.MessageTypeRequest, "source", "dest", "action", []byte("payload"))

	// Save message
	err := store.Save(msg, 5*time.Minute)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Get existing message
	retrieved, err := store.Get(msg.ID)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Get returned nil for existing message")
	}
	if retrieved.ID != msg.ID {
		t.Errorf("Retrieved wrong message: got %s, want %s", retrieved.ID, msg.ID)
	}

	// Get non-existent message
	notFound, err := store.Get("non-existent-id")
	if err != nil {
		t.Errorf("Get for non-existent should not error: %v", err)
	}
	if notFound != nil {
		t.Error("Get for non-existent should return nil")
	}
}

func TestMemoryStore_Get_Expired(t *testing.T) {
	store := NewMemoryStore()
	msg := protocol.NewMessage(protocol.MessageTypeRequest, "source", "dest", "action", []byte("payload"))

	// Save message with very short TTL (1 nanosecond)
	err := store.Save(msg, 1)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Get should return nil and remove expired message
	retrieved, err := store.Get(msg.ID)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if retrieved != nil {
		t.Error("Get should return nil for expired message")
	}

	// Verify message was removed from store
	store.mutex.RLock()
	_, exists := store.messages[msg.ID]
	store.mutex.RUnlock()
	if exists {
		t.Error("Expired message should be removed from store")
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()
	msg := protocol.NewMessage(protocol.MessageTypeRequest, "source", "dest", "action", []byte("payload"))

	// Save and then delete
	err := store.Save(msg, 5*time.Minute)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	err = store.Delete(msg.ID)
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	// Verify deletion
	retrieved, _ := store.Get(msg.ID)
	if retrieved != nil {
		t.Error("Message should be deleted")
	}

	// Delete non-existent should not error
	err = store.Delete("non-existent-id")
	if err != nil {
		t.Errorf("Delete non-existent should not error: %v", err)
	}
}

func TestMemoryStore_List(t *testing.T) {
	store := NewMemoryStore()
	ids := make(map[string]bool)

	// Add multiple messages with unique IDs
	for i := 0; i < 5; i++ {
		msg := protocol.NewMessage(protocol.MessageTypeRequest, "source", "dest", "action", []byte("payload"))
		// Ensure unique ID by waiting if needed
		for ids[msg.ID] {
			time.Sleep(time.Millisecond)
			msg = protocol.NewMessage(protocol.MessageTypeRequest, "source", "dest", "action", []byte("payload"))
		}
		ids[msg.ID] = true
		err := store.Save(msg, 5*time.Minute)
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// List should return all messages
	messages, err := store.List()
	if err != nil {
		t.Errorf("List failed: %v", err)
	}
	if len(messages) != 5 {
		t.Errorf("Expected 5 messages, got %d", len(messages))
	}
}

func TestMemoryStore_List_WithExpired(t *testing.T) {
	store := NewMemoryStore()

	// Ensure we get unique IDs
	var msg1, msg2 *protocol.Message
	for {
		msg1 = protocol.NewMessage(protocol.MessageTypeRequest, "source-non-expiring", "dest1", "action", []byte("payload1"))
		time.Sleep(2 * time.Millisecond)
		msg2 = protocol.NewMessage(protocol.MessageTypeRequest, "source-expiring", "dest2", "action", []byte("payload2"))
		if msg1.ID != msg2.ID {
			break
		}
	}

	// Add non-expiring message
	store.Save(msg1, 0)

	// Add expiring message
	store.Save(msg2, 1) // 1 nanosecond

	// Wait for expiration
	time.Sleep(15 * time.Millisecond)

	// List should only return non-expired messages
	messages, err := store.List()
	if err != nil {
		t.Errorf("List failed: %v", err)
	}
	if len(messages) != 1 {
		t.Errorf("Expected 1 message after filtering expired, got %d", len(messages))
	}
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	store := NewMemoryStore()
	const numGoroutines = 10
	const numOperations = 10

	done := make(chan bool, numGoroutines*2)

	// Concurrent writes with unique IDs per goroutine
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				// Use unique source to ensure different IDs
				source := fmt.Sprintf("source-%d-%d", id, j)
				msg := protocol.NewMessage(protocol.MessageTypeRequest, source, "dest", "action", []byte("payload"))
				store.Save(msg, 5*time.Minute)
			}
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				store.List()
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines*2; i++ {
		<-done
	}

	// Verify store has messages (at least some unique ones)
	messages, _ := store.List()
	if len(messages) == 0 {
		t.Error("Expected some messages in store")
	}
}

func TestMemoryStore_Update(t *testing.T) {
	store := NewMemoryStore()
	msg := protocol.NewMessage(protocol.MessageTypeRequest, "source", "dest", "action", []byte("original"))

	// Save original
	store.Save(msg, 5*time.Minute)

	// Update with same ID (simulating an update)
	updated := protocol.NewMessage(protocol.MessageTypeRequest, "source", "dest", "action", []byte("updated"))
	updated.ID = msg.ID // Same ID
	store.Save(updated, 5*time.Minute)

	// Retrieve and verify
	retrieved, _ := store.Get(msg.ID)
	if string(retrieved.Payload) != "updated" {
		t.Error("Message should be updated")
	}
}

func TestMemoryStore_EmptyList(t *testing.T) {
	store := NewMemoryStore()

	messages, err := store.List()
	if err != nil {
		t.Errorf("List failed: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("Expected empty list, got %d messages", len(messages))
	}
}

func TestMemoryStore_VariousMessageTypes(t *testing.T) {
	store := NewMemoryStore()

	messageTypes := []protocol.MessageType{
		protocol.MessageTypeRequest,
		protocol.MessageTypeResponse,
		protocol.MessageTypeError,
		protocol.MessageTypeEvent,
	}

	for _, msgType := range messageTypes {
		msg := protocol.NewMessage(msgType, "source", "dest", "action", []byte("payload"))
		err := store.Save(msg, 5*time.Minute)
		if err != nil {
			t.Errorf("Failed to save %s message: %v", msgType, err)
		}

		retrieved, _ := store.Get(msg.ID)
		if retrieved == nil {
			t.Errorf("Failed to retrieve %s message", msgType)
		}
		if retrieved.Type != msgType {
			t.Errorf("Type mismatch for %s message", msgType)
		}
	}
}

func TestMemoryStore_LargePayload(t *testing.T) {
	store := NewMemoryStore()

	// Create large payload (1MB)
	largePayload := make([]byte, 1024*1024)
	for i := range largePayload {
		largePayload[i] = byte(i % 256)
	}

	msg := protocol.NewMessage(protocol.MessageTypeRequest, "source", "dest", "action", largePayload)
	err := store.Save(msg, 5*time.Minute)
	if err != nil {
		t.Fatalf("Failed to save large message: %v", err)
	}

	retrieved, _ := store.Get(msg.ID)
	if retrieved == nil {
		t.Fatal("Failed to retrieve large message")
	}

	if len(retrieved.Payload) != len(largePayload) {
		t.Errorf("Payload size mismatch: got %d, want %d", len(retrieved.Payload), len(largePayload))
	}
}

func TestMemoryStore_TTLOperations(t *testing.T) {
	store := NewMemoryStore()

	tests := []struct {
		name    string
		ttl     time.Duration
		wait    time.Duration
		expired bool
	}{
		{
			name:    "no_expiration",
			ttl:     0,
			wait:    10 * time.Millisecond,
			expired: false,
		},
		{
			name:    "short_ttl_expires",
			ttl:     1,
			wait:    10 * time.Millisecond,
			expired: true,
		},
		{
			name:    "long_ttl_no_expire",
			ttl:     1 * time.Hour,
			wait:    10 * time.Millisecond,
			expired: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := protocol.NewMessage(protocol.MessageTypeRequest, "source", "dest", "action", []byte("payload"))
			store.Save(msg, tt.ttl)

			time.Sleep(tt.wait)

			retrieved, _ := store.Get(msg.ID)
			if tt.expired && retrieved != nil {
				t.Error("Expected message to be expired")
			}
			if !tt.expired && retrieved == nil {
				t.Error("Expected message to not be expired")
			}
		})
	}
}

func BenchmarkMemoryStore_Save(b *testing.B) {
	store := NewMemoryStore()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := protocol.NewMessage(protocol.MessageTypeRequest, "source", "dest", "action", []byte("payload"))
		store.Save(msg, 5*time.Minute)
	}
}

func BenchmarkMemoryStore_Get(b *testing.B) {
	store := NewMemoryStore()
	msg := protocol.NewMessage(protocol.MessageTypeRequest, "source", "dest", "action", []byte("payload"))
	store.Save(msg, 5*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Get(msg.ID)
	}
}

func BenchmarkMemoryStore_List(b *testing.B) {
	store := NewMemoryStore()

	// Populate store
	for i := 0; i < 1000; i++ {
		msg := protocol.NewMessage(protocol.MessageTypeRequest, "source", "dest", "action", []byte("payload"))
		store.Save(msg, 5*time.Minute)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.List()
	}
}
