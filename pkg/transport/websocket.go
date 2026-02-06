package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/agentries/amp-relay-go/pkg/protocol"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WSTransport WebSocket传输实现
type WSTransport struct {
	conn       *websocket.Conn
	localDID   string
	remoteDID  string
	logger     *zap.Logger
	mu         sync.Mutex
	isClosed   bool
}

// NewWSTransport 创建WebSocket传输
func NewWSTransport(conn *websocket.Conn, localDID, remoteDID string, logger *zap.Logger) *WSTransport {
	return &WSTransport{
		conn:      conn,
		localDID:  localDID,
		remoteDID: remoteDID,
		logger:    logger,
	}
}

// Send 发送消息
func (t *WSTransport) Send(ctx context.Context, msg *protocol.Message) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.isClosed {
		return fmt.Errorf("transport closed")
	}
	
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	return t.conn.WriteMessage(websocket.TextMessage, data)
}

// Receive 接收消息
func (t *WSTransport) Receive(ctx context.Context) (*protocol.Message, error) {
	_, data, err := t.conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}
	
	var msg protocol.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}
	
	return &msg, nil
}

// Close 关闭连接
func (t *WSTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.isClosed {
		return nil
	}
	
	t.isClosed = true
	return t.conn.Close()
}

func (t *WSTransport) LocalDID() string  { return t.localDID }
func (t *WSTransport) RemoteDID() string { return t.remoteDID }

// MessageRelay 消息中继器
type MessageRelay struct {
	transports map[string]protocol.Transport
	mu         sync.RWMutex
	logger     *zap.Logger
}

func NewMessageRelay(logger *zap.Logger) *MessageRelay {
	return &MessageRelay{
		transports: make(map[string]protocol.Transport),
		logger:     logger,
	}
}

// Register 注册Agent传输
func (r *MessageRelay) Register(did string, t protocol.Transport) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.transports[did] = t
}

// Unregister 注销Agent传输
func (r *MessageRelay) Unregister(did string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.transports, did)
}

// Forward 转发消息
func (r *MessageRelay) Forward(ctx context.Context, msg *protocol.Message) error {
	r.mu.RLock()
	target, exists := r.transports[msg.To]
	r.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("target agent %s not found", msg.To)
	}
	
	return target.Send(ctx, msg)
}

// Start 开始运行中继服务
func (r *MessageRelay) Start(t protocol.Transport) {
	ctx := context.Background()
	defer r.Unregister(t.LocalDID())
	
	r.Register(t.LocalDID(), t)
	
	for {
		msg, err := t.Receive(ctx)
		if err != nil {
			r.logger.Error("failed to receive message", zap.Error(err), zap.String("did", t.LocalDID()))
			break
		}
		
		if err := r.Forward(ctx, msg); err != nil {
			r.logger.Warn("failed to forward message", zap.Error(err), zap.String("from", msg.From), zap.String("to", msg.To))
			// 可选：向发送方返回错误消息
		}
	}
}