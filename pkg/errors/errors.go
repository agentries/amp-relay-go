package errors

import (
	"fmt"
	"time"

	"github.com/agentries/amp-relay-go/pkg/protocol"
)

// 标准错误码定义
const (
	CodeCapabilityNotFound       = 1001
	CodeCapabilityUnavailable    = 1002
	CodeTransportError           = 2001
	CodeSignatureVerifyFailed    = 3001
	CodeDecryptionFailed         = 3002
	CodeAuthenticationFailed     = 3003
	CodeInternalError            = 5000
)

// AMPError AMP协议错误
type AMPError struct {
	Code      int
	Name      string
	Message   string
	Details   map[string]interface{}
	RequestID string
	Retryable bool
}

func (e *AMPError) Error() string {
	return fmt.Sprintf("AMP Error [%d] %s: %s", e.Code, e.Name, e.Message)
}

// NewAMPError 创建新的AMP错误
func NewAMPError(code int, message string) *AMPError {
	name := "UNKNOWN_ERROR"
	retryable := false
	
	switch code {
	case CodeCapabilityNotFound:
		name = "CAPABILITY_NOT_FOUND"
	case CodeCapabilityUnavailable:
		name = "CAPABILITY_UNAVAILABLE"
		retryable = true
	case CodeTransportError:
		name = "TRANSPORT_ERROR"
		retryable = true
	case CodeSignatureVerifyFailed:
		name = "SIGNATURE_VERIFICATION_FAILED"
	case CodeDecryptionFailed:
		name = "DECRYPTION_FAILED"
	case CodeAuthenticationFailed:
		name = "AUTHENTICATION_FAILED"
	case CodeInternalError:
		name = "INTERNAL_ERROR"
	}
	
	return &AMPError{
		Code:      code,
		Name:      name,
		Message:   message,
		Retryable: retryable,
	}
}

// ToMessage 将错误转换为协议消息
func (e *AMPError) ToMessage(requestID string) *protocol.Message {
	errPayload := protocol.ErrorMessage{
		Error: protocol.ErrorDetail{
			Code:    e.Code,
			Name:    e.Name,
			Message: e.Message,
			Details: e.Details,
		},
		RequestID: requestID,
	}
	
	payload, _ := protocol.MarshalJSON(errPayload)
	
	return &protocol.Message{
		Type:      protocol.MessageTypeError,
		Version:   protocol.CurrentVersion,
		Timestamp: time.Now(),
		Payload:   payload,
	}
}

// ErrorHandler 错误处理器
type ErrorHandler struct {
	logger Logger
}

// Logger 日志接口
type Logger interface {
	Error(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
}

func NewErrorHandler(logger Logger) *ErrorHandler {
	return &ErrorHandler{logger: logger}
}

// Handle 处理并分类错误
func (h *ErrorHandler) Handle(err error) *AMPError {
	if ampErr, ok := err.(*AMPError); ok {
		h.logError(ampErr)
		return ampErr
	}
	
	// 封装普通错误
	ampErr := NewAMPError(CodeInternalError, err.Error())
	h.logError(ampErr)
	return ampErr
}

func (h *ErrorHandler) logError(e *AMPError) {
	if e.Code >= 3000 {
		h.logger.Error("AMP Security Error: %d %s - %s", e.Code, e.Name, e.Message)
	} else if e.Code >= 2000 {
		h.logger.Warn("AMP Transport Error: %d %s - %s", e.Code, e.Name, e.Message)
	} else {
		h.logger.Info("AMP Logic Error: %d %s - %s", e.Code, e.Name, e.Message)
	}
}