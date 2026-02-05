package protocol

import (
	"bytes"
	"testing"
	"time"
)

func TestNewMessage(t *testing.T) {
	msg := NewMessage(MessageTypeRequest, "source-did", "dest-did", "test.action", []byte("test payload"))

	if msg == nil {
		t.Fatal("NewMessage returned nil")
	}

	// Check required fields
	if msg.ID == "" {
		t.Error("Message ID should not be empty")
	}
	if msg.Type != MessageTypeRequest {
		t.Errorf("Expected type %s, got %s", MessageTypeRequest, msg.Type)
	}
	if msg.Source != "source-did" {
		t.Errorf("Expected source 'source-did', got %s", msg.Source)
	}
	if msg.Destination != "dest-did" {
		t.Errorf("Expected destination 'dest-did', got %s", msg.Destination)
	}
	if msg.Action != "test.action" {
		t.Errorf("Expected action 'test.action', got %s", msg.Action)
	}
	if !bytes.Equal(msg.Payload, []byte("test payload")) {
		t.Errorf("Expected payload 'test payload', got %s", string(msg.Payload))
	}
	if msg.Version != "5.0" {
		t.Errorf("Expected version '5.0', got %s", msg.Version)
	}
	if msg.Metadata == nil {
		t.Error("Metadata should be initialized")
	}

	// Check timestamp is set
	if msg.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestMessage_AddMetadata(t *testing.T) {
	msg := NewMessage(MessageTypeRequest, "source", "dest", "action", nil)

	// Add metadata
	msg.AddMetadata("key1", "value1")
	msg.AddMetadata("key2", "value2")

	if msg.Metadata["key1"] != "value1" {
		t.Errorf("Expected metadata key1='value1', got %s", msg.Metadata["key1"])
	}
	if msg.Metadata["key2"] != "value2" {
		t.Errorf("Expected metadata key2='value2', got %s", msg.Metadata["key2"])
	}

	// Test nil metadata initialization
	msg.Metadata = nil
	msg.AddMetadata("key3", "value3")
	if msg.Metadata["key3"] != "value3" {
		t.Error("AddMetadata should initialize nil metadata")
	}
}

func TestMessage_SetTTL(t *testing.T) {
	msg := NewMessage(MessageTypeRequest, "source", "dest", "action", nil)

	msg.SetTTL(300) // 5 minutes
	if msg.TTL != 300 {
		t.Errorf("Expected TTL 300, got %d", msg.TTL)
	}
}

func TestMessage_IsExpired(t *testing.T) {
	tests := []struct {
		name       string
		ttl        int64
		modifyTime func(time.Time) time.Time
		want       bool
	}{
		{
			name: "no_ttl_not_expired",
			ttl:  0,
			want: false,
		},
		{
			name: "future_not_expired",
			ttl:  300,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewMessage(MessageTypeRequest, "source", "dest", "action", nil)
			msg.SetTTL(tt.ttl)

			got := msg.IsExpired()
			if got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}

	// Test expired message (negative TTL in past)
	t.Run("expired_message", func(t *testing.T) {
		msg := NewMessage(MessageTypeRequest, "source", "dest", "action", nil)
		// Set timestamp in the past
		msg.Timestamp = time.Now().Add(-10 * time.Minute)
		msg.SetTTL(300) // 5 minutes TTL

		if !msg.IsExpired() {
			t.Error("Expected message to be expired")
		}
	})
}

func TestMessage_CBORMarshalUnmarshal(t *testing.T) {
	original := NewMessage(MessageTypeRequest, "did:example:123", "did:example:456", "test.action", []byte("test data"))
	original.AddMetadata("test-key", "test-value")
	original.SetTTL(600)

	// Marshal
	data, err := original.CBORMarshal()
	if err != nil {
		t.Fatalf("CBORMarshal failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("Marshaled data should not be empty")
	}

	// Unmarshal
	var decoded Message
	err = decoded.CBORUnmarshal(data)
	if err != nil {
		t.Fatalf("CBORUnmarshal failed: %v", err)
	}

	// Verify fields
	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: got %s, want %s", decoded.ID, original.ID)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type mismatch: got %s, want %s", decoded.Type, original.Type)
	}
	if decoded.Source != original.Source {
		t.Errorf("Source mismatch: got %s, want %s", decoded.Source, original.Source)
	}
	if decoded.Destination != original.Destination {
		t.Errorf("Destination mismatch: got %s, want %s", decoded.Destination, original.Destination)
	}
	if decoded.Action != original.Action {
		t.Errorf("Action mismatch: got %s, want %s", decoded.Action, original.Action)
	}
	if !bytes.Equal(decoded.Payload, original.Payload) {
		t.Errorf("Payload mismatch: got %s, want %s", string(decoded.Payload), string(original.Payload))
	}
	if decoded.Version != original.Version {
		t.Errorf("Version mismatch: got %s, want %s", decoded.Version, original.Version)
	}
	if decoded.TTL != original.TTL {
		t.Errorf("TTL mismatch: got %d, want %d", decoded.TTL, original.TTL)
	}
	if decoded.Metadata["test-key"] != "test-value" {
		t.Errorf("Metadata mismatch: got %s, want test-value", decoded.Metadata["test-key"])
	}
}

func TestMessage_CBORUnmarshal_InvalidData(t *testing.T) {
	var msg Message
	err := msg.CBORUnmarshal([]byte("invalid cbor data"))
	if err == nil {
		t.Error("Expected error when unmarshaling invalid CBOR data")
	}
}

func TestMessageTypes(t *testing.T) {
	tests := []struct {
		msgType MessageType
		want    string
	}{
		{MessageTypeRequest, "request"},
		{MessageTypeResponse, "response"},
		{MessageTypeError, "error"},
		{MessageTypeEvent, "event"},
	}

	for _, tt := range tests {
		t.Run(string(tt.msgType), func(t *testing.T) {
			msg := NewMessage(tt.msgType, "source", "dest", "action", nil)
			if msg.Type != tt.msgType {
				t.Errorf("Expected type %s, got %s", tt.msgType, msg.Type)
			}
		})
	}
}

func TestMessage_ComplexPayload(t *testing.T) {
	// Test with various payload types
	testCases := []struct {
		name    string
		payload []byte
	}{
		{
			name:    "json_payload",
			payload: []byte(`{"key":"value","number":123}`),
		},
		{
			name:    "binary_payload",
			payload: []byte{0x00, 0x01, 0x02, 0x03, 0xFF},
		},
		{
			name:    "empty_payload",
			payload: []byte{},
		},
		{
			name:    "large_payload",
			payload: bytes.Repeat([]byte("x"), 1024),
		},
		{
			name:    "unicode_payload",
			payload: []byte("Hello, 世界! 🌍"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := NewMessage(MessageTypeRequest, "source", "dest", "action", tc.payload)

			data, err := original.CBORMarshal()
			if err != nil {
				t.Fatalf("CBORMarshal failed: %v", err)
			}

			var decoded Message
			err = decoded.CBORUnmarshal(data)
			if err != nil {
				t.Fatalf("CBORUnmarshal failed: %v", err)
			}

			if !bytes.Equal(decoded.Payload, tc.payload) {
				t.Errorf("Payload mismatch after round-trip")
			}
		})
	}
}

func TestMessage_CorrelationID(t *testing.T) {
	msg := NewMessage(MessageTypeResponse, "source", "dest", "action", []byte("response"))
	msg.CorrelationID = "request-123"

	data, err := msg.CBORMarshal()
	if err != nil {
		t.Fatalf("CBORMarshal failed: %v", err)
	}

	var decoded Message
	err = decoded.CBORUnmarshal(data)
	if err != nil {
		t.Fatalf("CBORUnmarshal failed: %v", err)
	}

	if decoded.CorrelationID != "request-123" {
		t.Errorf("CorrelationID mismatch: got %s, want request-123", decoded.CorrelationID)
	}
}

func TestMessage_Signature(t *testing.T) {
	msg := NewMessage(MessageTypeRequest, "source", "dest", "action", []byte("data"))
	msg.Signature = []byte("mock-signature-bytes")

	data, err := msg.CBORMarshal()
	if err != nil {
		t.Fatalf("CBORMarshal failed: %v", err)
	}

	var decoded Message
	err = decoded.CBORUnmarshal(data)
	if err != nil {
		t.Fatalf("CBORUnmarshal failed: %v", err)
	}

	if !bytes.Equal(decoded.Signature, msg.Signature) {
		t.Errorf("Signature mismatch after round-trip")
	}
}

func BenchmarkMessage_CBORMarshal(b *testing.B) {
	msg := NewMessage(MessageTypeRequest, "did:example:123", "did:example:456", "benchmark.action", []byte("benchmark payload data"))
	msg.AddMetadata("key1", "value1")
	msg.AddMetadata("key2", "value2")
	msg.SetTTL(300)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := msg.CBORMarshal()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMessage_CBORUnmarshal(b *testing.B) {
	msg := NewMessage(MessageTypeRequest, "did:example:123", "did:example:456", "benchmark.action", []byte("benchmark payload data"))
	data, _ := msg.CBORMarshal()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var decoded Message
		err := decoded.CBORUnmarshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}
