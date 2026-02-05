package protocol

import (
	"bytes"
	"testing"
	"time"
)

func TestNewMessage(t *testing.T) {
	msg := NewMessage(MessageTypeRequest, "did:web:alice", "did:web:bob", map[string]string{"action": "ping"})

	if msg.V != 1 {
		t.Errorf("Expected V=1, got %d", msg.V)
	}
	if len(msg.ID) != 16 {
		t.Errorf("Expected 16-byte ID, got %d bytes", len(msg.ID))
	}
	if msg.Type != MessageTypeRequest {
		t.Errorf("Expected type 0x%02x, got 0x%02x", MessageTypeRequest, msg.Type)
	}
	if msg.Ts == 0 {
		t.Error("Timestamp should be set")
	}
	if msg.TTL != 86400000 {
		t.Errorf("Expected default TTL 86400000ms, got %d", msg.TTL)
	}
	if msg.From != "did:web:alice" {
		t.Errorf("Expected From 'did:web:alice', got %s", msg.From)
	}
	if msg.To != "did:web:bob" {
		t.Errorf("Expected To 'did:web:bob', got %s", msg.To)
	}
	if msg.Body == nil {
		t.Error("Body should be set")
	}
}

func TestNewMessage_UniqueIDs(t *testing.T) {
	msg1 := NewMessage(MessageTypeRequest, "a", "b", nil)
	msg2 := NewMessage(MessageTypeRequest, "a", "b", nil)

	if bytes.Equal(msg1.ID, msg2.ID) {
		t.Error("Two messages should have different IDs")
	}
}

func TestMessage_IDHex(t *testing.T) {
	msg := NewMessage(MessageTypeRequest, "a", "b", nil)
	hex := msg.IDHex()

	if len(hex) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("Expected 32 hex chars, got %d", len(hex))
	}
}

func TestMessage_IsExpired(t *testing.T) {
	tests := []struct {
		name    string
		ttlMs   uint64
		ageMs   int64 // how old the message is
		expired bool
	}{
		{"TTL=0 never expires", 0, 999999, false},
		{"fresh message", 5000, 0, false},
		{"expired message", 1, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewMessage(MessageTypeRequest, "a", "b", nil)
			msg.TTL = tt.ttlMs
			msg.Ts = uint64(time.Now().Add(-time.Duration(tt.ageMs) * time.Millisecond).UnixMilli())

			if got := msg.IsExpired(); got != tt.expired {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expired)
			}
		})
	}
}

func TestMessage_IsExpired_WithSleep(t *testing.T) {
	msg := NewMessage(MessageTypeRequest, "a", "b", nil)
	msg.TTL = 1 // 1 millisecond

	time.Sleep(10 * time.Millisecond)

	if !msg.IsExpired() {
		t.Error("Message should be expired after sleep")
	}
}

func TestMessage_CBORRoundtrip(t *testing.T) {
	original := NewMessage(MessageTypeRequest, "did:web:alice", "did:web:bob", []byte("hello"))
	original.ReplyTo = []byte{1, 2, 3, 4}
	original.ThreadID = []byte{5, 6, 7, 8}
	original.Sig = []byte{9, 10, 11, 12}
	original.Ext = map[string]interface{}{"custom": "value"}

	data, err := original.CBORMarshal()
	if err != nil {
		t.Fatalf("CBORMarshal failed: %v", err)
	}

	decoded := &Message{}
	if err := decoded.CBORUnmarshal(data); err != nil {
		t.Fatalf("CBORUnmarshal failed: %v", err)
	}

	if decoded.V != original.V {
		t.Errorf("V: got %d, want %d", decoded.V, original.V)
	}
	if !bytes.Equal(decoded.ID, original.ID) {
		t.Errorf("ID mismatch")
	}
	if decoded.Type != original.Type {
		t.Errorf("Type: got 0x%02x, want 0x%02x", decoded.Type, original.Type)
	}
	if decoded.Ts != original.Ts {
		t.Errorf("Ts: got %d, want %d", decoded.Ts, original.Ts)
	}
	if decoded.TTL != original.TTL {
		t.Errorf("TTL: got %d, want %d", decoded.TTL, original.TTL)
	}
	if decoded.From != original.From {
		t.Errorf("From: got %s, want %s", decoded.From, original.From)
	}
	if decoded.To != original.To {
		t.Errorf("To: got %s, want %s", decoded.To, original.To)
	}
	if !bytes.Equal(decoded.ReplyTo, original.ReplyTo) {
		t.Errorf("ReplyTo mismatch")
	}
	if !bytes.Equal(decoded.ThreadID, original.ThreadID) {
		t.Errorf("ThreadID mismatch")
	}
	if !bytes.Equal(decoded.Sig, original.Sig) {
		t.Errorf("Sig mismatch")
	}
}

func TestMessage_CBORRoundtrip_NilBody(t *testing.T) {
	original := NewMessage(MessageTypeResponse, "a", "b", nil)
	data, err := original.CBORMarshal()
	if err != nil {
		t.Fatalf("CBORMarshal failed: %v", err)
	}

	decoded := &Message{}
	if err := decoded.CBORUnmarshal(data); err != nil {
		t.Fatalf("CBORUnmarshal failed: %v", err)
	}

	if decoded.Type != MessageTypeResponse {
		t.Errorf("Type mismatch: got 0x%02x", decoded.Type)
	}
}

func TestMessage_CBORUnmarshal_InvalidData(t *testing.T) {
	msg := &Message{}
	err := msg.CBORUnmarshal([]byte("not cbor"))
	if err == nil {
		t.Error("Expected error for invalid CBOR data")
	}
}

func TestMessageType_Constants(t *testing.T) {
	// Verify type code assignments match RFC 001 §4.3
	tests := []struct {
		name     string
		typ      MessageType
		expected uint8
	}{
		{"Ping", MessageTypePing, 0x01},
		{"Pong", MessageTypePong, 0x02},
		{"ACK", MessageTypeACK, 0x03},
		{"Error", MessageTypeError, 0x0F},
		{"Message", MessageTypeMessage, 0x10},
		{"Request", MessageTypeRequest, 0x11},
		{"Response", MessageTypeResponse, 0x12},
		{"StreamStart", MessageTypeStreamStart, 0x13},
		{"CapQuery", MessageTypeCapQuery, 0x20},
		{"DocSend", MessageTypeDocSend, 0x30},
		{"CredIssue", MessageTypeCredIssue, 0x40},
		{"DelegGrant", MessageTypeDelegGrant, 0x50},
		{"Presence", MessageTypePresence, 0x60},
		{"Hello", MessageTypeHello, 0x70},
		{"Extension", MessageTypeExtension, 0xF0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if uint8(tt.typ) != tt.expected {
				t.Errorf("MessageType%s = 0x%02x, want 0x%02x", tt.name, uint8(tt.typ), tt.expected)
			}
		})
	}
}

func TestGenerateID_Format(t *testing.T) {
	now := time.Now()
	id := generateID(now)

	if len(id) != 16 {
		t.Fatalf("Expected 16 bytes, got %d", len(id))
	}

	// First 8 bytes should be near current time in milliseconds
	tsMs := uint64(now.UnixMilli())
	idTs := uint64(id[0])<<56 | uint64(id[1])<<48 | uint64(id[2])<<40 | uint64(id[3])<<32 |
		uint64(id[4])<<24 | uint64(id[5])<<16 | uint64(id[6])<<8 | uint64(id[7])

	diff := int64(tsMs) - int64(idTs)
	if diff < 0 {
		diff = -diff
	}
	if diff > 1000 {
		t.Errorf("Timestamp in ID too far from now: diff=%dms", diff)
	}
}

func TestGenerateID_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := generateID(time.Now())
		key := string(id)
		if seen[key] {
			t.Fatalf("Duplicate ID at iteration %d", i)
		}
		seen[key] = true
	}
}

func BenchmarkNewMessage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewMessage(MessageTypeRequest, "did:web:alice", "did:web:bob", nil)
	}
}

func BenchmarkMessage_CBORMarshal(b *testing.B) {
	msg := NewMessage(MessageTypeRequest, "did:web:alice", "did:web:bob", []byte("benchmark payload"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg.CBORMarshal()
	}
}

func BenchmarkMessage_CBORUnmarshal(b *testing.B) {
	msg := NewMessage(MessageTypeRequest, "did:web:alice", "did:web:bob", []byte("benchmark payload"))
	data, _ := msg.CBORMarshal()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		decoded := &Message{}
		decoded.CBORUnmarshal(data)
	}
}
