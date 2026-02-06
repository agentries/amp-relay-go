// Package protocol 定义AMP v5.13协议核心类型和接口
package protocol

import (
	"context"
	"encoding/json"
	"time"
)

// AMPVersion 定义协议版本
type AMPVersion string

const (
	AMPVersion5_13 AMPVersion = "5.13"
	CurrentVersion AMPVersion = AMPVersion5_13
)

// MessageType 定义消息类型
type MessageType string

const (
	MessageTypeCapabilityRequest MessageType = "capability.request"
	MessageTypeCapabilityResponse MessageType = "capability.response"
	MessageTypeCapabilityError MessageType = "capability.error"
	MessageTypeData MessageType = "data"
	MessageTypePing MessageType = "ping"
	MessageTypePong MessageType = "pong"
	MessageTypeError MessageType = "error"
)

// Capability 定义Agent能力
type Capability struct {
	Domain      string            `json:"domain"`      // 能力域，如 "messaging", "storage", "crypto"
	Type        string            `json:"type"`        // 能力类型，如 "email", "ipfs", "eth"
	Version     string            `json:"version"`     // 版本，如 "v5.13"
	Constraints map[string]string `json:"constraints,omitempty"` // 约束条件
}

// String 返回能力的字符串表示
func (c Capability) String() string {
	return c.Domain + ":" + c.Type + ":" + c.Version
}

// CapabilityManifest Agent能力清单
type CapabilityManifest struct {
	AgentDID    string       `json:"agent_did"`           // Agent DID
	Version     AMPVersion   `json:"version"`             // 协议版本
	IssuedAt    time.Time    `json:"issued_at"`           // 签发时间
	ExpiresAt   time.Time    `json:"expires_at"`          // 过期时间
	Present     []Capability `json:"present"`             // 具备的能力
	Absent      []Capability `json:"absent,omitempty"`    // 缺失的能力 (capability absence > prohibition)
	Constraints map[string]interface{} `json:"constraints,omitempty"` // 全局约束
}

// Message AMP协议消息
type Message struct {
	ID          string                 `json:"id"`                    // 消息ID
	Type        MessageType            `json:"type"`                  // 消息类型
	Version     AMPVersion             `json:"version"`               // 协议版本
	From        string                 `json:"from"`                  // 发送方DID
	To          string                 `json:"to"`                    // 接收方DID
	Timestamp   time.Time              `json:"timestamp"`             // 时间戳
	Payload     json.RawMessage        `json:"payload,omitempty"`     // 消息负载
	Headers     map[string]string      `json:"headers,omitempty"`     // 消息头
	Signature   string                 `json:"signature,omitempty"`   // JWS签名
	Encryption  string                 `json:"encryption,omitempty"`  // JWE加密信息
}

// CapabilityRequest 能力请求
type CapabilityRequest struct {
	Requested []Capability `json:"requested"` // 请求的能力列表
	Context   string       `json:"context,omitempty"` // 请求上下文
}

// CapabilityResponse 能力响应
type CapabilityResponse struct {
	Manifest CapabilityManifest `json:"manifest"` // 能力清单
	Status   string             `json:"status"`   // 状态: "available", "partial", "unavailable"
}

// ErrorDetail 错误详情
type ErrorDetail struct {
	Code    int                    `json:"code"`    // 错误码
	Name    string                 `json:"name"`    // 错误名称
	Message string                 `json:"message"` // 错误消息
	Details map[string]interface{} `json:"details,omitempty"` // 额外详情
}

// ErrorMessage 错误消息
type ErrorMessage struct {
	Error     ErrorDetail `json:"error"`      // 错误详情
	RequestID string      `json:"request_id"` // 关联的请求ID
}

// MarshalJSON 实现JSON序列化
func (m *Message) MarshalJSON() ([]byte, error) {
	type Alias Message
	return json.Marshal(&struct {
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Timestamp: m.Timestamp.Format(time.RFC3339),
		Alias:     (*Alias)(m),
	})
}

// MarshalJSON 通用JSON序列化辅助函数
func MarshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Transport 传输层接口
type Transport interface {
	// Send 发送消息
	Send(ctx context.Context, msg *Message) error
	
	// Receive 接收消息
	Receive(ctx context.Context) (*Message, error)
	
	// Close 关闭传输
	Close() error
	
	// LocalDID 获取本地DID
	LocalDID() string
	
	// RemoteDID 获取远端DID
	RemoteDID() string
}

// SecureTransport 安全传输接口
type SecureTransport interface {
	Transport
	
	// SignMessage 对消息进行签名
	SignMessage(msg *Message) error
	
	// EncryptMessage 对消息进行加密
	EncryptMessage(msg *Message, recipientDID string) error
	
	// VerifyMessage 验证消息签名
	VerifyMessage(msg *Message) error
	
	// DecryptMessage 解密消息
	DecryptMessage(msg *Message) error
}

// Handler 消息处理器接口
type Handler interface {
	// HandleMessage 处理消息
	HandleMessage(ctx context.Context, msg *Message) error
	
	// SupportedTypes 返回支持的消息类型
	SupportedTypes() []MessageType
}

// Stream 流传输接口
type Stream interface {
	// Read 读取数据
	Read(p []byte) (n int, err error)
	
	// Write 写入数据
	Write(p []byte) (n int, err error)
	
	// Close 关闭流
	Close() error
	
	// SetDeadline 设置读写截止时间
	SetDeadline(t time.Time) error
	
	// LocalAddr 本地地址
	LocalAddr() string
	
	// RemoteAddr 远端地址
	RemoteAddr() string
}

// StreamTransport 流传输接口
type StreamTransport interface {
	// CreateStream 创建新的流
	CreateStream(ctx context.Context) (Stream, error)
	
	// AcceptStream 接受新的流
	AcceptStream(ctx context.Context) (Stream, error)
	
	// Close 关闭流传输
	Close() error
}