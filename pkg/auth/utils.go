package auth

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

// DIDCache DID文档缓存
type DIDCache struct {
	data  map[string]*cacheEntry
	ttl   time.Duration
}

type cacheEntry struct {
	document *DIDDocument
	expiry   time.Time
}

// NewDIDCache 创建DID缓存
func NewDIDCache(ttl time.Duration) *DIDCache {
	return &DIDCache{
		data: make(map[string]*cacheEntry),
		ttl:  ttl,
	}
}

// Get 获取缓存的DID文档
func (c *DIDCache) Get(did string) *DIDDocument {
	entry, exists := c.data[did]
	if !exists || time.Now().After(entry.expiry) {
		delete(c.data, did)
		return nil
	}
	return entry.document
}

// Set 设置DID文档缓存
func (c *DIDCache) Set(did string, doc *DIDDocument) {
	c.data[did] = &cacheEntry{
		document: doc,
		expiry:   time.Now().Add(c.ttl),
	}
}

// parseMultibasePublicKey 解析multibase编码的公钥
func parseMultibasePublicKey(multibase string) ([]byte, error) {
	// 简单的multibase解析，支持base58btc编码
	if strings.HasPrefix(multibase, "z") {
		// base58btc编码
		decoded, err := base58Decode(multibase[1:])
		if err != nil {
			return nil, fmt.Errorf("failed to decode base58: %w", err)
		}
		return decoded, nil
	}
	
	return nil, fmt.Errorf("unsupported multibase encoding")
}

// parseJWKPublicKey 解析JWK格式的公钥
func parseJWKPublicKey(jwkData map[string]interface{}) ([]byte, error) {
	// 从JWK数据中提取Ed25519公钥
	kty, ok := jwkData["kty"].(string)
	if !ok || kty != "OKP" {
		return nil, fmt.Errorf("unsupported key type: %v", kty)
	}
	
	crv, ok := jwkData["crv"].(string)
	if !ok || crv != "Ed25519" {
		return nil, fmt.Errorf("unsupported curve: %v", crv)
	}
	
	x, ok := jwkData["x"].(string)
	if !ok {
		return nil, fmt.Errorf("missing x coordinate in JWK")
	}
	
	// Base64 URL解码
	publicKey, err := base64UrlDecode(x)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}
	
	return publicKey, nil
}

// base64UrlDecode Base64 URL解码
func base64UrlDecode(s string) ([]byte, error) {
	// 添加填充
	padding := 4 - len(s)%4
	if padding != 4 {
		for i := 0; i < padding; i++ {
			s += "="
		}
	}
	return base64.URLEncoding.DecodeString(s)
}

// base58Decode 简单的base58解码实现
func base58Decode(input string) ([]byte, error) {
	alphabet := "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	base := len(alphabet)
	
	// 将base58字符串转换为十进制大数
	var num uint64
	for _, char := range input {
		index := strings.IndexRune(alphabet, char)
		if index == -1 {
			return nil, fmt.Errorf("invalid base58 character: %c", char)
		}
		num = num*uint64(base) + uint64(index)
	}
	
	// 将大数转换为字节数组
	result := make([]byte, 0, 32)
	for num > 0 {
		result = append([]byte{byte(num & 0xff)}, result...)
		num >>= 8
	}
	
	return result, nil
}

// GenerateKeyPair 生成Ed25519密钥对
func GenerateKeyPair() (ed25519.PrivateKey, ed25519.PublicKey, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate key pair: %w", err)
	}
	return privateKey, publicKey, nil
}

// DIDWebResolver did:web解析器
type DIDWebResolver struct {
	baseURL string
}

// NewDIDWebResolver 创建did:web解析器
func NewDIDWebResolver(baseURL string) *DIDWebResolver {
	return &DIDWebResolver{baseURL: baseURL}
}

// Resolve 解析did:web
func (r *DIDWebResolver) Resolve(ctx context.Context, did string) (*DIDDocument, error) {
	if !strings.HasPrefix(did, "did:web:") {
		return nil, fmt.Errorf("invalid did:web format")
	}
	
	// 将did:web转换为URL路径
	parts := strings.Split(did, ":")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid did:web format")
	}
	
	domain := parts[2]
	path := strings.Join(parts[3:], "/")
	
	didURL := fmt.Sprintf("https://%s/%s/did.json", domain, path)
	
	// 这里应该实现HTTP GET请求获取DID文档
	// 简化实现，返回模拟数据
	_ = didURL // 标记为已使用，避免编译错误
	return &DIDDocument{
		ID:      did,
		Context: []string{"https://www.w3.org/ns/did/v1"},
		VerificationMethod: []VerificationMethod{
			{
				ID:                 did + "#key1",
				Type:               "Ed25519VerificationKey2020",
				Controller:         did,
				PublicKeyMultibase: "z6Mkq...", // 模拟公钥
			},
		},
		Authentication:  []string{did + "#key1"},
		AssertionMethod: []string{did + "#key1"},
		Service: []Service{
			{
				ID:              did + "#amp",
				Type:            "AgentMessagingProtocol",
				ServiceEndpoint: fmt.Sprintf("https://%s/amp", domain),
			},
		},
		Created: time.Now(),
		Updated: time.Now(),
	}, nil
}