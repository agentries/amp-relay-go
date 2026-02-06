// Package auth 提供Agent身份验证和消息认证功能
package auth

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/agentries/amp-relay-go/pkg/protocol"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
)

// Authenticator 身份验证器接口
type Authenticator interface {
	// Authenticate 验证Agent身份
	Authenticate(ctx context.Context, did string) error
	
	// GetPublicKey 获取DID对应的公钥
	GetPublicKey(ctx context.Context, did string) (ed25519.PublicKey, error)
	
	// DIDDocument 获取DID文档
	DIDDocument(ctx context.Context, did string) (*DIDDocument, error)
}

// DIDDocument DID文档
type DIDDocument struct {
	ID                   string                 `json:"id"`
	Context              []string               `json:"@context"`
	VerificationMethod   []VerificationMethod   `json:"verificationMethod"`
	Authentication       []string               `json:"authentication"`
	AssertionMethod      []string               `json:"assertionMethod"`
	Service              []Service              `json:"service"`
	Created              time.Time              `json:"created"`
	Updated              time.Time              `json:"updated"`
}

// VerificationMethod 验证方法
type VerificationMethod struct {
	ID                 string `json:"id"`
	Type               string `json:"type"`
	Controller         string `json:"controller"`
	PublicKeyMultibase string `json:"publicKeyMultibase,omitempty"`
	PublicKeyJwk       map[string]interface{} `json:"publicKeyJwk,omitempty"`
}

// Service 服务端点
type Service struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	ServiceEndpoint string `json:"serviceEndpoint"`
}

// MessageAuthenticator 消息认证器
type MessageAuthenticator struct {
	authenticator Authenticator
	privateKey    ed25519.PrivateKey
	publicKey     ed25519.PublicKey
	did           string
}

// NewMessageAuthenticator 创建消息认证器
func NewMessageAuthenticator(authenticator Authenticator, privateKey ed25519.PrivateKey, did string) *MessageAuthenticator {
	return &MessageAuthenticator{
		authenticator: authenticator,
		privateKey:    privateKey,
		publicKey:     privateKey.Public().(ed25519.PublicKey),
		did:           did,
	}
}

// SignMessage 对消息进行签名
func (ma *MessageAuthenticator) SignMessage(msg *protocol.Message) error {
	// 设置消息头
	if msg.Headers == nil {
		msg.Headers = make(map[string]string)
	}
	msg.Headers["x-amp-signer"] = ma.did
	msg.Headers["x-amp-alg"] = "EdDSA"
	msg.Headers["x-amp-key-id"] = ma.did + "#key1"
	
	// 创建JWS头
	headers := jws.NewHeaders()
	headers.Set("alg", jwa.EdDSA)
	headers.Set("typ", "JWS")
	headers.Set("kid", ma.did+"#key1")
	
	// 创建JWK
	jwkKey, err := jwk.FromRaw(ma.publicKey)
	if err != nil {
		return fmt.Errorf("failed to create JWK: %w", err)
	}
	jwkKey.Set(jwk.KeyIDKey, ma.did+"#key1")
	jwkKey.Set(jwk.AlgorithmKey, jwa.EdDSA)
	
	// 序列化消息
	payload, err := msg.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	// 签名
	signed, err := jws.Sign(payload, jws.WithKey(jwa.EdDSA, ma.privateKey, jws.WithProtectedHeaders(headers)))
	if err != nil {
		return fmt.Errorf("failed to sign message: %w", err)
	}
	
	msg.Signature = string(signed)
	return nil
}

// VerifyMessage 验证消息签名
func (ma *MessageAuthenticator) VerifyMessage(msg *protocol.Message) error {
	if msg.Signature == "" {
		return fmt.Errorf("message has no signature")
	}
	
	// 获取签名者DID
	signerDID := msg.Headers["x-amp-signer"]
	if signerDID == "" {
		return fmt.Errorf("missing signer information")
	}
	
	// 验证签名者身份
	if err := ma.authenticator.Authenticate(context.Background(), signerDID); err != nil {
		return fmt.Errorf("failed to authenticate signer: %w", err)
	}
	
	// 获取签名者公钥
	publicKey, err := ma.authenticator.GetPublicKey(context.Background(), signerDID)
	if err != nil {
		return fmt.Errorf("failed to get signer's public key: %w", err)
	}
	
	// 验证签名
	_, err = jws.Verify([]byte(msg.Signature), jws.WithKey(jwa.EdDSA, publicKey))
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}
	
	return nil
}

// DIDAuthenticator DID身份验证器
type DIDAuthenticator struct {
	resolver DIDResolver
	cache    *DIDCache
}

// DIDResolver DID解析器接口
type DIDResolver interface {
	Resolve(ctx context.Context, did string) (*DIDDocument, error)
}

// NewDIDAuthenticator 创建DID身份验证器
func NewDIDAuthenticator(resolver DIDResolver) *DIDAuthenticator {
	return &DIDAuthenticator{
		resolver: resolver,
		cache:    NewDIDCache(5 * time.Minute),
	}
}

// Authenticate 验证DID
func (da *DIDAuthenticator) Authenticate(ctx context.Context, did string) error {
	_, err := da.DIDDocument(ctx, did)
	return err
}

// GetPublicKey 获取DID对应的公钥
func (da *DIDAuthenticator) GetPublicKey(ctx context.Context, did string) (ed25519.PublicKey, error) {
	doc, err := da.DIDDocument(ctx, did)
	if err != nil {
		return nil, err
	}
	
	if len(doc.VerificationMethod) == 0 {
		return nil, fmt.Errorf("no verification methods found in DID document")
	}
	
	// 找到Ed25519验证方法
	for _, vm := range doc.VerificationMethod {
		if vm.Type == "Ed25519VerificationKey2020" || vm.Type == "Ed25519VerificationKey2018" {
			if vm.PublicKeyMultibase != "" {
				// 解析multibase编码的公钥
				return parseMultibasePublicKey(vm.PublicKeyMultibase)
			}
			if vm.PublicKeyJwk != nil {
				// 解析JWK格式的公钥
				return parseJWKPublicKey(vm.PublicKeyJwk)
			}
		}
	}
	
	return nil, fmt.Errorf("no Ed25519 public key found in DID document")
}

// DIDDocument 获取DID文档
func (da *DIDAuthenticator) DIDDocument(ctx context.Context, did string) (*DIDDocument, error) {
	// 检查缓存
	if cached := da.cache.Get(did); cached != nil {
		return cached, nil
	}
	
	// 解析DID
	doc, err := da.resolver.Resolve(ctx, did)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve DID: %w", err)
	}
	
	// 缓存结果
	da.cache.Set(did, doc)
	
	return doc, nil
}

// CapabilityValidator 能力验证器
type CapabilityValidator struct {
	manifest *protocol.CapabilityManifest
}

// NewCapabilityValidator 创建能力验证器
func NewCapabilityValidator(manifest *protocol.CapabilityManifest) *CapabilityValidator {
	return &CapabilityValidator{manifest: manifest}
}

// Validate 验证是否具备指定能力
func (cv *CapabilityValidator) Validate(capability protocol.Capability) bool {
	// 检查是否明确声明缺失
	for _, absent := range cv.manifest.Absent {
		if absent.String() == capability.String() {
			return false
		}
	}
	
	// 检查是否具备该能力
	for _, present := range cv.manifest.Present {
		if present.String() == capability.String() {
			return true
		}
	}
	
	return false
}

// ValidateBatch 批量验证能力
func (cv *CapabilityValidator) ValidateBatch(capabilities []protocol.Capability) []bool {
	results := make([]bool, len(capabilities))
	for i, cap := range capabilities {
		results[i] = cv.Validate(cap)
	}
	return results
}