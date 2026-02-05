package protocol

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"time"

	cbor "github.com/fxamacker/cbor/v2"
)

// MessageType represents AMP message type codes per RFC 001 §4.3
type MessageType uint8

const (
	// Control (0x00-0x0F)
	MessageTypePing           MessageType = 0x01
	MessageTypePong           MessageType = 0x02
	MessageTypeACK            MessageType = 0x03
	MessageTypeProcOK         MessageType = 0x04
	MessageTypeProcFail       MessageType = 0x05
	MessageTypeContactRequest MessageType = 0x06
	MessageTypeContactResp    MessageType = 0x07
	MessageTypeContactRevoke  MessageType = 0x08
	MessageTypeProcessing     MessageType = 0x09
	MessageTypeProgress       MessageType = 0x0A
	MessageTypeInputRequired  MessageType = 0x0B
	MessageTypeError          MessageType = 0x0F

	// Message (0x10-0x1F)
	MessageTypeMessage     MessageType = 0x10
	MessageTypeRequest     MessageType = 0x11
	MessageTypeResponse    MessageType = 0x12
	MessageTypeStreamStart MessageType = 0x13
	MessageTypeStreamData  MessageType = 0x14
	MessageTypeStreamEnd   MessageType = 0x15

	// Capability (0x20-0x2F)
	MessageTypeCapQuery   MessageType = 0x20
	MessageTypeCapDeclare MessageType = 0x21
	MessageTypeCapInvoke  MessageType = 0x22
	MessageTypeCapResult  MessageType = 0x23

	// Document (0x30-0x3F)
	MessageTypeDocSend    MessageType = 0x30
	MessageTypeDocRequest MessageType = 0x31

	// Credential (0x40-0x4F)
	MessageTypeCredIssue   MessageType = 0x40
	MessageTypeCredRequest MessageType = 0x41
	MessageTypeCredPresent MessageType = 0x42
	MessageTypeCredVerify  MessageType = 0x43

	// Delegation (0x50-0x5F)
	MessageTypeDelegGrant  MessageType = 0x50
	MessageTypeDelegRevoke MessageType = 0x51
	MessageTypeDelegQuery  MessageType = 0x52

	// Presence (0x60-0x6F)
	MessageTypePresence      MessageType = 0x60
	MessageTypePresenceQuery MessageType = 0x61
	MessageTypePresenceSub   MessageType = 0x62
	MessageTypePresenceUnsub MessageType = 0x63

	// Handshake (0x70-0x7F)
	MessageTypeHello       MessageType = 0x70
	MessageTypeHelloACK    MessageType = 0x71
	MessageTypeHelloReject MessageType = 0x72

	// Extension (0xF0-0xFF)
	MessageTypeExtension MessageType = 0xF0

	// MessageTypeEvent is kept as an alias for backward compat during transition
	MessageTypeEvent MessageType = 0x10
)

// Message represents the base AMP v5.0 message per RFC 001 §4.1
type Message struct {
	V        uint        `cbor:"1,keyasint" json:"v"`                          // Protocol version (1)
	ID       []byte      `cbor:"2,keyasint" json:"id"`                        // Message ID (16 bytes: 8 timestamp + 8 random)
	Type     MessageType `cbor:"3,keyasint" json:"typ"`                       // Message type code
	Ts       uint64      `cbor:"4,keyasint" json:"ts"`                        // Unix timestamp (milliseconds)
	TTL      uint64      `cbor:"5,keyasint" json:"ttl"`                       // Time-to-live (milliseconds)
	From     string      `cbor:"6,keyasint" json:"from"`                      // Sender DID
	To       string      `cbor:"7,keyasint" json:"to"`                        // Recipient DID
	ReplyTo  []byte      `cbor:"8,keyasint,omitempty" json:"reply_to,omitempty"` // Message ID being replied to
	ThreadID []byte      `cbor:"9,keyasint,omitempty" json:"thread_id,omitempty"` // Conversation thread ID
	Sig      []byte      `cbor:"10,keyasint,omitempty" json:"sig,omitempty"`   // Ed25519 signature
	Body     interface{} `cbor:"11,keyasint,omitempty" json:"body,omitempty"`  // Message body (type-dependent)
	Ext      map[string]interface{} `cbor:"12,keyasint,omitempty" json:"ext,omitempty"` // Extension fields (NOT signed)
}

// NewMessage creates a new AMP message with RFC 001 defaults
func NewMessage(msgType MessageType, from, to string, body interface{}) *Message {
	now := time.Now()
	return &Message{
		V:    1,
		ID:   generateID(now),
		Type: msgType,
		Ts:   uint64(now.UnixMilli()),
		TTL:  86400000, // 24 hours in milliseconds (RFC 001 §8.3 default)
		From: from,
		To:   to,
		Body: body,
	}
}

// IDHex returns the message ID as a hex string (for logging/map keys)
func (m *Message) IDHex() string {
	return hex.EncodeToString(m.ID)
}

// IsExpired checks if the message has expired based on TTL per RFC 001 §8.3
func (m *Message) IsExpired() bool {
	if m.TTL == 0 {
		return false // TTL=0 means immediate delivery, not "no expiration"
	}
	return uint64(time.Now().UnixMilli()) > m.Ts+m.TTL
}

// CBORMarshal encodes the message using CBOR
func (m *Message) CBORMarshal() ([]byte, error) {
	return cbor.Marshal(m)
}

// CBORUnmarshal decodes the message from CBOR
func (m *Message) CBORUnmarshal(data []byte) error {
	return cbor.Unmarshal(data, m)
}

// generateID generates a 16-byte message ID per RFC 001 §4.2
// Format: 8 bytes timestamp (milliseconds, big-endian) + 8 bytes random
func generateID(now time.Time) []byte {
	id := make([]byte, 16)
	binary.BigEndian.PutUint64(id[:8], uint64(now.UnixMilli()))
	if _, err := rand.Read(id[8:]); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return id
}
