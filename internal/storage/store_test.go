package storage

import (
	"fmt"
	"testing"
	"time"

	"github.com/openclaw/amp-relay-go/internal/protocol"
)

func newTestMsg(from, to string) *protocol.Message {
	return protocol.NewMessage(protocol.MessageTypeRequest, from, to, []byte("payload"))
}

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
	msg := newTestMsg("source", "dest")

	err := store.Save(msg, 0)
	if err != nil {
		t.Errorf("Save without TTL failed: %v", err)
	}

	msg2 := newTestMsg("source2", "dest2")
	err = store.Save(msg2, 5*time.Minute)
	if err != nil {
		t.Errorf("Save with TTL failed: %v", err)
	}
}

func TestMemoryStore_Get(t *testing.T) {
	store := NewMemoryStore()
	msg := newTestMsg("source", "dest")

	err := store.Save(msg, 5*time.Minute)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	retrieved, err := store.Get(msg.IDHex())
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Get returned nil for existing message")
	}
	if retrieved.IDHex() != msg.IDHex() {
		t.Errorf("Retrieved wrong message: got %s, want %s", retrieved.IDHex(), msg.IDHex())
	}

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
	msg := newTestMsg("source", "dest")

	err := store.Save(msg, 1) // 1 nanosecond
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	retrieved, err := store.Get(msg.IDHex())
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if retrieved != nil {
		t.Error("Get should return nil for expired message")
	}

	// List cleans up expired messages
	store.List()
	store.mutex.RLock()
	_, exists := store.messages[msg.IDHex()]
	store.mutex.RUnlock()
	if exists {
		t.Error("Expired message should be removed from store after List")
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()
	msg := newTestMsg("source", "dest")

	store.Save(msg, 5*time.Minute)

	err := store.Delete(msg.IDHex())
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	retrieved, _ := store.Get(msg.IDHex())
	if retrieved != nil {
		t.Error("Message should be deleted")
	}

	err = store.Delete("non-existent-id")
	if err != nil {
		t.Errorf("Delete non-existent should not error: %v", err)
	}
}

func TestMemoryStore_List(t *testing.T) {
	store := NewMemoryStore()

	for i := 0; i < 5; i++ {
		msg := newTestMsg(fmt.Sprintf("source-%d", i), "dest")
		err := store.Save(msg, 5*time.Minute)
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

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

	msg1 := newTestMsg("source-non-expiring", "dest1")
	msg2 := newTestMsg("source-expiring", "dest2")

	store.Save(msg1, 0) // no expiration
	store.Save(msg2, 1) // 1 nanosecond

	time.Sleep(15 * time.Millisecond)

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

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				msg := newTestMsg(fmt.Sprintf("source-%d-%d", id, j), "dest")
				store.Save(msg, 5*time.Minute)
			}
			done <- true
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				store.List()
			}
			done <- true
		}(i)
	}

	for i := 0; i < numGoroutines*2; i++ {
		<-done
	}

	messages, _ := store.List()
	if len(messages) == 0 {
		t.Error("Expected some messages in store")
	}
}

func TestMemoryStore_Update(t *testing.T) {
	store := NewMemoryStore()
	msg := newTestMsg("source", "dest")
	store.Save(msg, 5*time.Minute)

	updated := newTestMsg("source", "dest")
	updated.ID = msg.ID // Same ID
	updated.Body = []byte("updated")
	store.Save(updated, 5*time.Minute)

	retrieved, _ := store.Get(msg.IDHex())
	if retrieved == nil {
		t.Fatal("Failed to retrieve updated message")
	}
	body, ok := retrieved.Body.([]byte)
	if !ok || string(body) != "updated" {
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
		msg := protocol.NewMessage(msgType, "source", "dest", []byte("payload"))
		err := store.Save(msg, 5*time.Minute)
		if err != nil {
			t.Errorf("Failed to save type 0x%02x message: %v", msgType, err)
		}

		retrieved, _ := store.Get(msg.IDHex())
		if retrieved == nil {
			t.Errorf("Failed to retrieve type 0x%02x message", msgType)
		}
		if retrieved.Type != msgType {
			t.Errorf("Type mismatch for 0x%02x message", msgType)
		}
	}
}

func TestMemoryStore_LargePayload(t *testing.T) {
	store := NewMemoryStore()

	largePayload := make([]byte, 1024*1024)
	for i := range largePayload {
		largePayload[i] = byte(i % 256)
	}

	msg := protocol.NewMessage(protocol.MessageTypeRequest, "source", "dest", largePayload)
	err := store.Save(msg, 5*time.Minute)
	if err != nil {
		t.Fatalf("Failed to save large message: %v", err)
	}

	retrieved, _ := store.Get(msg.IDHex())
	if retrieved == nil {
		t.Fatal("Failed to retrieve large message")
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
		{"no_expiration", 0, 10 * time.Millisecond, false},
		{"short_ttl_expires", 1, 10 * time.Millisecond, true},
		{"long_ttl_no_expire", 1 * time.Hour, 10 * time.Millisecond, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := newTestMsg("source", "dest")
			store.Save(msg, tt.ttl)

			time.Sleep(tt.wait)

			retrieved, _ := store.Get(msg.IDHex())
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
		msg := newTestMsg("source", "dest")
		store.Save(msg, 5*time.Minute)
	}
}

func BenchmarkMemoryStore_Get(b *testing.B) {
	store := NewMemoryStore()
	msg := newTestMsg("source", "dest")
	store.Save(msg, 5*time.Minute)
	key := msg.IDHex()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Get(key)
	}
}

func BenchmarkMemoryStore_List(b *testing.B) {
	store := NewMemoryStore()
	for i := 0; i < 1000; i++ {
		msg := newTestMsg("source", "dest")
		store.Save(msg, 5*time.Minute)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.List()
	}
}
