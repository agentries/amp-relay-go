package protocol

import (
	"time"

	cbor "github.com/fxamacker/cbor/v2"
)

// MessageType represents the type of AMP message
type MessageType string

const (
	MessageTypeRequest  MessageType = "request"
	MessageTypeResponse MessageType = "response"
	MessageTypeError    MessageType = "error"
	MessageTypeEvent    MessageType = "event"
)

// Message represents the basic structure of an AMP v5.0 message
type Message struct {
	ID            string            `cbor:"1,keyasint" json:"id"`
	Type          MessageType       `cbor:"2,keyasint" json:"type"`
	Timestamp     time.Time         `cbor:"3,keyasint" json:"timestamp"`
	Source        string            `cbor:"4,keyasint" json:"source"`
	Destination   string            `cbor:"5,keyasint" json:"destination"`
	Action        string            `cbor:"6,keyasint" json:"action"`
	Payload       []byte            `cbor:"7,keyasint" json:"payload"`
	Metadata      map[string]string `cbor:"8,keyasint" json:"metadata,omitempty"`
	CorrelationID string            `cbor:"9,keyasint" json:"correlation_id,omitempty"`
	Version       string            `cbor:"10,keyasint" json:"version"`
	Signature     []byte            `cbor:"11,keyasint" json:"signature,omitempty"`
	TTL           int64             `cbor:"12,keyasint" json:"ttl,omitempty"` // Time-to-live in seconds
}

// NewMessage creates a new AMP message with defaults
func NewMessage(msgType MessageType, source, destination, action string, payload []byte) *Message {
	return &Message{
		ID:          generateID(),
		Type:        msgType,
		Timestamp:   time.Now().UTC(),
		Source:      source,
		Destination: destination,
		Action:      action,
		Payload:     payload,
		Metadata:    make(map[string]string),
		Version:     "5.0",
	}
}

// AddMetadata adds metadata to the message
func (m *Message) AddMetadata(key, value string) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]string)
	}
	m.Metadata[key] = value
}

// SetTTL sets the time-to-live for the message in seconds
func (m *Message) SetTTL(seconds int64) {
	m.TTL = seconds
}

// IsExpired checks if the message has expired based on TTL
func (m *Message) IsExpired() bool {
	if m.TTL <= 0 {
		return false // No TTL means no expiration
	}

	expirationTime := m.Timestamp.Add(time.Duration(m.TTL) * time.Second)
	return time.Now().After(expirationTime)
}

// CBORMarshal encodes the message using CBOR
func (m *Message) CBORMarshal() ([]byte, error) {
	return cbor.Marshal(m)
}

// CBORUnmarshal decodes the message from CBOR
func (m *Message) CBORUnmarshal(data []byte) error {
	return cbor.Unmarshal(data, m)
}

// generateID generates a unique ID for the message
func generateID() string {
	// In a real implementation, we'd use a proper UUID generator
	// For now, we'll use timestamp + random component
	return "msg_" + time.Now().UTC().Format("20060102150405") + "_" + getRandomString(8)
}

// getRandomString generates a random string of specified length
func getRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)[:length]
}
