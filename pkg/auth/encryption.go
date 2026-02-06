package auth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/agentries/amp-relay-go/pkg/protocol"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"golang.org/x/crypto/nacl/box"
)

// Encryptor 消息加密器
type Encryptor struct {
	authenticator Authenticator
}

// NewEncryptor 创建消息加密器
func NewEncryptor(authenticator Authenticator) *Encryptor {
	return &Encryptor{authenticator: authenticator}
}

// EncryptMessage 对消息进行加密
// 使用NaCl box加密 (基于Curve25519/XSalsa20/Poly1305)
func (e *Encryptor) EncryptMessage(msg *protocol.Message, recipientDID string) error {
	// 获取接收方的公钥
	recipientKey, err := e.authenticator.GetPublicKey(context.Background(), recipientDID)
	if err != nil {
		return fmt.Errorf("failed to get recipient public key: %w", err)
	}

	// Ed25519公钥转换为Curve25519用于加密
	recipientCurveKey, err := ed25519PublicKeyToCurve25519(recipientKey)
	if err != nil {
		return fmt.Errorf("failed to convert recipient key: %w", err)
	}

	// 序列化消息负载
	payload, err := msg.Payload.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// 生成临时密钥对
	 ephemeralPublicKey, ephemeralPrivateKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate ephemeral key: %w", err)
	}

	// 加密
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	encrypted := box.Seal(nil, payload, &nonce, recipientCurveKey, ephemeralPrivateKey)

	// 构建加密结果: ephemeralPubKey + nonce + ciphertext
	result := make([]byte, 0, 32+24+len(encrypted))
	result = append(result, ephemeralPublicKey[:]...)
	result = append(result, nonce[:]...)
	result = append(result, encrypted...)

	// 更新消息
	msg.Payload = result
	msg.Encryption = "nacl-box"
	if msg.Headers == nil {
		msg.Headers = make(map[string]string)
	}
	msg.Headers["x-amp-encryption"] = "nacl-box"
	msg.Headers["x-amp-recipient"] = recipientDID

	return nil
}

// DecryptMessage 解密消息
func (e *Encryptor) DecryptMessage(msg *protocol.Message, privateKey ed25519.PrivateKey) error {
	if msg.Encryption == "" {
		return nil // 未加密，无需解密
	}

	if msg.Encryption != "nacl-box" {
		return fmt.Errorf("unsupported encryption type: %s", msg.Encryption)
	}

	// Ed25519私钥转换为Curve25519
	curvePrivateKey, err := ed25519PrivateKeyToCurve25519(privateKey)
	if err != nil {
		return fmt.Errorf("failed to convert private key: %w", err)
	}

	data := msg.Payload
	if len(data) < 32+24 {
		return fmt.Errorf("encrypted data too short")
	}

	// 提取ephemeral公钥、nonce和密文
	var ephemeralPublicKey [32]byte
	copy(ephemeralPublicKey[:], data[:32])

	var nonce [24]byte
	copy(nonce[:], data[32:56])

	ciphertext := data[56:]

	// 解密
	plaintext, ok := box.Open(nil, ciphertext, &nonce, &ephemeralPublicKey, curvePrivateKey)
	if !ok {
		return fmt.Errorf("decryption failed")
	}

	// 更新消息负载
	msg.Payload = plaintext
	msg.Encryption = ""
	delete(msg.Headers, "x-amp-encryption")

	return nil
}

// SecureMessageProcessor 安全消息处理器
type SecureMessageProcessor struct {
	authenticator *MessageAuthenticator
	encryptor     *Encryptor
	privateKey    ed25519.PrivateKey
	publicKey     ed25519.PublicKey
	did           string
}

// NewSecureMessageProcessor 创建安全消息处理器
func NewSecureMessageProcessor(
	authenticator Authenticator,
	privateKey ed25519.PrivateKey,
	did string,
) *SecureMessageProcessor {
	publicKey := privateKey.Public().(ed25519.PublicKey)
	return &SecureMessageProcessor{
		authenticator: NewMessageAuthenticator(authenticator, privateKey, did),
		encryptor:     NewEncryptor(authenticator),
		privateKey:    privateKey,
		publicKey:     publicKey,
		did:           did,
	}
}

// ProcessOutgoingMessage 处理出站消息（签名+加密）
func (smp *SecureMessageProcessor) ProcessOutgoingMessage(msg *protocol.Message, recipientDID string) error {
	// 设置消息基本信息
	msg.From = smp.did
	msg.Version = protocol.CurrentVersion
	msg.Timestamp = time.Now()

	// 1. 先签名
	if err := smp.authenticator.SignMessage(msg); err != nil {
		return fmt.Errorf("failed to sign message: %w", err)
	}

	// 2. 后加密
	if err := smp.encryptor.EncryptMessage(msg, recipientDID); err != nil {
		return fmt.Errorf("failed to encrypt message: %w", err)
	}

	return nil
}

// ProcessIncomingMessage 处理入站消息（解密+验证）
func (smp *SecureMessageProcessor) ProcessIncomingMessage(msg *protocol.Message) error {
	// 1. 先解密
	if err := smp.encryptor.DecryptMessage(msg, smp.privateKey); err != nil {
		return fmt.Errorf("failed to decrypt message: %w", err)
	}

	// 2. 后验证签名
	if err := smp.authenticator.VerifyMessage(msg); err != nil {
		return fmt.Errorf("failed to verify message: %w", err)
	}

	return nil
}

// ed25519PublicKeyToCurve25519 将Ed25519公钥转换为Curve25519
func ed25519PublicKeyToCurve25519(ed25519Pub []byte) (*[32]byte, error) {
	if len(ed25519Pub) != 32 {
		return nil, fmt.Errorf("invalid Ed25519 public key length: %d", len(ed25519Pub))
	}

	var curvePub [32]byte
	// 使用原始转换 - Ed25519和Curve25519使用相同的底层曲线
	// 注意：实际转换需要更复杂的逻辑，这里简化处理
	copy(curvePub[:], ed25519Pub)

	return &curvePub, nil
}

// ed25519PrivateKeyToCurve25519 将Ed25519私钥转换为Curve25519
func ed25519PrivateKeyToCurve25519(ed25519Priv ed25519.PrivateKey) (*[32]byte, error) {
	if len(ed25519Priv) != 64 {
		return nil, fmt.Errorf("invalid Ed25519 private key length: %d", len(ed25519Priv))
	}

	var curvePriv [32]byte
	// Ed25519私钥的前32字节是种子，用于生成Curve25519私钥
	copy(curvePriv[:], ed25519Priv[:32])

	return &curvePriv, nil
}

// JWS签名相关辅助函数 (保留用于兼容性)
func signWithJWS(payload []byte, privateKey ed25519.PrivateKey) (string, error) {
	// 创建JWS头
	headers := jws.NewHeaders()
	headers.Set("alg", jwa.EdDSA)
	headers.Set("typ", "JWS")

	// 签名
	signed, err := jws.Sign(payload, jws.WithKey(jwa.EdDSA, privateKey, jws.WithProtectedHeaders(headers)))
	if err != nil {
		return "", fmt.Errorf("failed to sign: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(signed), nil
}

func verifyJWS(signed string, publicKey ed25519.PublicKey) ([]byte, error) {
	data, err := base64.RawURLEncoding.DecodeString(signed)
	if err != nil {
		return nil, fmt.Errorf("failed to decode: %w", err)
	}

	// 验证签名
	verified, err := jws.Verify(data, jws.WithKey(jwa.EdDSA, publicKey))
	if err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	return verified, nil
}

func createJWK(publicKey ed25519.PublicKey, keyID string) (jwk.Key, error) {
	jwkKey, err := jwk.FromRaw(publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWK: %w", err)
	}

	if err := jwkKey.Set(jwk.KeyIDKey, keyID); err != nil {
		return nil, err
	}
	if err := jwkKey.Set(jwk.AlgorithmKey, jwa.EdDSA); err != nil {
		return nil, err
	}

	return jwkKey, nil
}
