// Package transport provides RFC-002 compliant WebSocket transport for AMP Relay
package transport

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// AuthFrame represents the authentication frame (RFC-002 §3.1)
type AuthFrame struct {
	// Message type: always "auth"
	Type string `json:"type"`

	// Agent's DID
	DID string `json:"did"`

	// Signature of the connection nonce (or timestamp)
	Signature string `json:"signature"`

	// Signature algorithm (e.g., "ed25519")
	Algorithm string `json:"algorithm"`

	// Timestamp of the auth request
	Timestamp int64 `json:"timestamp"`

	// max_msg_size declaration (RFC-002 §3.3)
	MaxMsgSize int `json:"max_msg_size,omitempty"`

	// Nonce for replay protection
	Nonce string `json:"nonce,omitempty"`
}

// AuthResponse represents the authentication response (RFC-002 §3.1)
type AuthResponse struct {
	// Response type: "auth_ok" or "auth_fail"
	Type string `json:"type"`

	// Server's DID (optional, for mutual auth)
	ServerDID string `json:"server_did,omitempty"`

	// Error message if auth_fail
	Error string `json:"error,omitempty"`

	// Error code
	ErrorCode string `json:"error_code,omitempty"`

	// Negotiated max_msg_size (min of client and server values)
	MaxMsgSize int `json:"max_msg_size,omitempty"`

	// Server timestamp
	Timestamp int64 `json:"timestamp"`
}

// AuthenticatedClient extends Client with auth state
type AuthenticatedClient struct {
	*Client
	Authenticated bool
	DID           string
	MaxMsgSize    int // Negotiated max message size
	AuthTime      time.Time
}

// WebSocketAuthHandler handles RFC-002 authentication
type WebSocketAuthHandler struct {
	// Authenticator interface
	Authenticator interface {
		Verify(did string, signature []byte, nonce string) (bool, error)
	}

	// Server's own DID (for mutual authentication)
	ServerDID string

	// Default max message size (1 MiB as per RFC-002)
	DefaultMaxMsgSize int

	// Auth timeout (RFC-002: must auth within reasonable time)
	AuthTimeout time.Duration
}

// NewWebSocketAuthHandler creates a new auth handler
func NewWebSocketAuthHandler() *WebSocketAuthHandler {
	return &WebSocketAuthHandler{
		DefaultMaxMsgSize: 1024 * 1024, // 1 MiB
		AuthTimeout:       30 * time.Second,
	}
}

// HandleAuth processes the authentication frame
func (h *WebSocketAuthHandler) HandleAuth(client *Client, frame []byte) (*AuthResponse, error) {
	var authFrame AuthFrame
	if err := json.Unmarshal(frame, &authFrame); err != nil {
		return &AuthResponse{
			Type:      "auth_fail",
			Error:     "invalid auth frame format",
			ErrorCode: "invalid_format",
			Timestamp: time.Now().Unix(),
		}, fmt.Errorf("unmarshal auth frame: %w", err)
	}

	// Validate frame type
	if authFrame.Type != "auth" {
		return &AuthResponse{
			Type:      "auth_fail",
			Error:     "expected auth frame",
			ErrorCode: "invalid_type",
			Timestamp: time.Now().Unix(),
		}, fmt.Errorf("expected auth frame, got: %s", authFrame.Type)
	}

	// Validate DID format (basic check)
	if authFrame.DID == "" {
		return &AuthResponse{
			Type:      "auth_fail",
			Error:     "DID cannot be empty",
			ErrorCode: "invalid_did",
			Timestamp: time.Now().Unix(),
		}, fmt.Errorf("empty DID")
	}

	// Check timestamp for replay protection (±5 minutes)
	now := time.Now().Unix()
	if authFrame.Timestamp < now-300 || authFrame.Timestamp > now+300 {
		return &AuthResponse{
			Type:      "auth_fail",
			Error:     "timestamp out of acceptable range",
			ErrorCode: "invalid_timestamp",
			Timestamp: now,
		}, fmt.Errorf("timestamp out of range")
	}

	// TODO: Real signature verification
	// For now, placeholder accepts any DID
	log.Printf("[AUTH] Authenticating DID: %s", authFrame.DID)

	// Negotiate max_msg_size
	negotiatedMax := h.DefaultMaxMsgSize
	if authFrame.MaxMsgSize > 0 && authFrame.MaxMsgSize < negotiatedMax {
		negotiatedMax = authFrame.MaxMsgSize
	}

	// Success
	return &AuthResponse{
		Type:       "auth_ok",
		ServerDID:  h.ServerDID,
		MaxMsgSize: negotiatedMax,
		Timestamp:  now,
	}, nil
}

// SendAuthFailure sends an auth failure response and closes connection
func SendAuthFailure(client *Client, error string, errorCode string) error {
	resp := AuthResponse{
		Type:      "auth_fail",
		Error:     error,
		ErrorCode: errorCode,
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return client.Conn.WriteMessage(websocket.BinaryMessage, data)
}

// SendAuthSuccess sends an auth success response
func SendAuthSuccess(client *Client, serverDID string, maxMsgSize int) error {
	resp := AuthResponse{
		Type:       "auth_ok",
		ServerDID:  serverDID,
		MaxMsgSize: maxMsgSize,
		Timestamp:  time.Now().Unix(),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return client.Conn.WriteMessage(websocket.BinaryMessage, data)
}

// RFC002Constants defines RFC-002 protocol constants
var RFC002Constants = struct {
	// Message types
	MsgTypeAuth    string
	MsgTypeAuthOK  string
	MsgTypeAuthFail string
	
	// Timing
	DefaultPingInterval  time.Duration
	DefaultPongTimeout   time.Duration
	DefaultAuthTimeout   time.Duration
	
	// Size limits
	MinMaxMsgSize        int // 1 MiB
	DefaultMaxMsgSize    int
}{
	MsgTypeAuth:         "auth",
	MsgTypeAuthOK:       "auth_ok",
	MsgTypeAuthFail:     "auth_fail",
	DefaultPingInterval: 30 * time.Second,
	DefaultPongTimeout:  90 * time.Second,
	DefaultAuthTimeout:  30 * time.Second,
	MinMaxMsgSize:       1024 * 1024,
	DefaultMaxMsgSize:   1024 * 1024,
}
