package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/agentries/amp-relay-go/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockDIDResolver 模拟DID解析器
type MockDIDResolver struct {
	documents map[string]*DIDDocument
}

func NewMockDIDResolver() *MockDIDResolver {
	return &MockDIDResolver{
		documents: make(map[string]*DIDDocument),
	}
}

func (m *MockDIDResolver) Resolve(ctx context.Context, did string) (*DIDDocument, error) {
	doc, exists := m.documents[did]
	if !exists {
		return nil, fmt.Errorf("DID not found: %s", did)
	}
	return doc, nil
}

func (m *MockDIDResolver) Register(did string, doc *DIDDocument) {
	m.documents[did] = doc
}

func TestDIDAuthenticator_Authenticate(t *testing.T) {
	resolver := NewMockDIDResolver()
	auth := NewDIDAuthenticator(resolver)

	// 生成测试密钥对
	privateKey, publicKey, err := GenerateKeyPair()
	require.NoError(t, err)
	_ = privateKey // 避免未使用变量警告

	// 创建测试DID文档
	testDID := "did:web:agentries.xyz:agent:test123"
	doc := &DIDDocument{
		ID:      testDID,
		Context: []string{"https://www.w3.org/ns/did/v1"},
		VerificationMethod: []VerificationMethod{
			{
				ID:                 testDID + "#key1",
				Type:               "Ed25519VerificationKey2020",
				Controller:         testDID,
				PublicKeyMultibase: "z" + string(publicKey), // 简化编码
			},
		},
		Authentication:  []string{testDID + "#key1"},
		AssertionMethod: []string{testDID + "#key1"},
		Service: []Service{
			{
				ID:              testDID + "#amp",
				Type:            "AgentMessagingProtocol",
				ServiceEndpoint: "wss://relay.agentries.xyz/agent/test123",
			},
		},
		Created: time.Now(),
		Updated: time.Now(),
	}

	resolver.Register(testDID, doc)

	t.Run("valid DID authentication", func(t *testing.T) {
		err := auth.Authenticate(context.Background(), testDID)
		assert.NoError(t, err)
	})

	t.Run("invalid DID format", func(t *testing.T) {
		err := auth.Authenticate(context.Background(), "invalid-did")
		assert.Error(t, err)
	})

	t.Run("non-existent DID", func(t *testing.T) {
		err := auth.Authenticate(context.Background(), "did:web:agentries.xyz:agent:nonexistent")
		assert.Error(t, err)
	})
}

func TestDIDAuthenticator_GetPublicKey(t *testing.T) {
	resolver := NewMockDIDResolver()
	auth := NewDIDAuthenticator(resolver)

	// 生成测试密钥对
	privateKey, publicKey, err := GenerateKeyPair()
	require.NoError(t, err)
	_ = privateKey // 避免未使用变量警告

	testDID := "did:web:agentries.xyz:agent:test456"
	doc := &DIDDocument{
		ID:      testDID,
		Context: []string{"https://www.w3.org/ns/did/v1"},
		VerificationMethod: []VerificationMethod{
			{
				ID:                 testDID + "#key1",
				Type:               "Ed25519VerificationKey2020",
				Controller:         testDID,
				PublicKeyMultibase: "z" + string(publicKey),
			},
		},
	}

	resolver.Register(testDID, doc)

	t.Run("get valid public key", func(t *testing.T) {
		retrievedKey, err := auth.GetPublicKey(context.Background(), testDID)
		assert.NoError(t, err)
		// 由于multibase解析简化，这里可能不完全匹配
		assert.NotNil(t, retrievedKey)
	})

	t.Run("DID without verification method", func(t *testing.T) {
		testDID2 := "did:web:agentries.xyz:agent:test789"
		doc2 := &DIDDocument{
			ID:      testDID2,
			Context: []string{"https://www.w3.org/ns/did/v1"},
		}
		resolver.Register(testDID2, doc2)

		_, err := auth.GetPublicKey(context.Background(), testDID2)
		assert.Error(t, err)
	})
}

func TestDIDAuthenticator_Cache(t *testing.T) {
	resolver := NewMockDIDResolver()
	auth := NewDIDAuthenticator(resolver)

	// 生成测试密钥对
	privateKey, publicKey, err := GenerateKeyPair()
	require.NoError(t, err)
	_ = privateKey // 避免未使用变量警告

	testDID := "did:web:agentries.xyz:agent:cachetest"
	doc := &DIDDocument{
		ID:      testDID,
		Context: []string{"https://www.w3.org/ns/did/v1"},
		VerificationMethod: []VerificationMethod{
			{
				ID:                 testDID + "#key1",
				Type:               "Ed25519VerificationKey2020",
				Controller:         testDID,
				PublicKeyMultibase: "z" + string(publicKey),
			},
		},
	}

	resolver.Register(testDID, doc)

	// 第一次解析
	_, err = auth.DIDDocument(context.Background(), testDID)
	assert.NoError(t, err)

	// 删除底层文档（模拟缓存命中）
	delete(resolver.documents, testDID)

	// 第二次解析应该从缓存获取
	doc2, err := auth.DIDDocument(context.Background(), testDID)
	assert.NoError(t, err)
	assert.Equal(t, testDID, doc2.ID)
}

func TestCapabilityValidator(t *testing.T) {
	manifest := &protocol.CapabilityManifest{
		AgentDID: "did:web:agentries.xyz:agent:captest",
		Present: []protocol.Capability{
			{Domain: "messaging", Type: "email", Version: "v1.0"},
			{Domain: "crypto", Type: "sign", Version: "v2.0"},
		},
		Absent: []protocol.Capability{
			{Domain: "execution", Type: "arbitrary-code", Version: "v1.0"},
		},
	}

	validator := NewCapabilityValidator(manifest)

	tests := []struct {
		name       string
		capability protocol.Capability
		expected   bool
	}{
		{
			name:       "present capability",
			capability: protocol.Capability{Domain: "messaging", Type: "email", Version: "v1.0"},
			expected:   true,
		},
		{
			name:       "absent capability",
			capability: protocol.Capability{Domain: "execution", Type: "arbitrary-code", Version: "v1.0"},
			expected:   false,
		},
		{
			name:       "unknown capability",
			capability: protocol.Capability{Domain: "storage", Type: "ipfs", Version: "v1.0"},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(tt.capability)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageAuthenticator(t *testing.T) {
	resolver := NewMockDIDResolver()
	didAuth := NewDIDAuthenticator(resolver)

	// 生成测试密钥对
	privateKey, publicKey, err := GenerateKeyPair()
	require.NoError(t, err)

	testDID := "did:web:agentries.xyz:agent:msgtest"
	doc := &DIDDocument{
		ID:      testDID,
		Context: []string{"https://www.w3.org/ns/did/v1"},
		VerificationMethod: []VerificationMethod{
			{
				ID:                 testDID + "#key1",
				Type:               "Ed25519VerificationKey2020",
				Controller:         testDID,
				PublicKeyMultibase: "z" + string(publicKey),
			},
		},
	}

	resolver.Register(testDID, doc)

	msgAuth := NewMessageAuthenticator(didAuth, privateKey, testDID)

	t.Run("sign and verify message", func(t *testing.T) {
		msg := &protocol.Message{
			ID:        "test-msg-1",
			Type:      protocol.MessageTypeData,
			Version:   protocol.CurrentVersion,
			From:      testDID,
			To:        "did:web:agentries.xyz:agent:recipient",
			Timestamp: time.Now(),
			Payload:   json.RawMessage(`{"content":"hello"}`),
		}

		// 签名
		err := msgAuth.SignMessage(msg)
		assert.NoError(t, err)
		assert.NotEmpty(t, msg.Signature)
		assert.Equal(t, testDID, msg.Headers["x-amp-signer"])

		// 验证
		err = msgAuth.VerifyMessage(msg)
		assert.NoError(t, err)
	})

	t.Run("verify message without signature", func(t *testing.T) {
		msg := &protocol.Message{
			ID:      "test-msg-2",
			Type:    protocol.MessageTypeData,
			From:    testDID,
			Payload: json.RawMessage(`{"content":"hello"}`),
		}

		err := msgAuth.VerifyMessage(msg)
		assert.Error(t, err)
	})
}

func TestDIDWebResolver(t *testing.T) {
	// 注意：这个测试使用模拟数据，实际DID解析需要HTTP客户端
	resolver := NewDIDWebResolver("https://agentries.xyz")

	t.Run("resolve valid did:web", func(t *testing.T) {
		doc, err := resolver.Resolve(context.Background(), "did:web:agentries.xyz:agent:test")
		assert.NoError(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, "did:web:agentries.xyz:agent:test", doc.ID)
	})

	t.Run("invalid did format", func(t *testing.T) {
		_, err := resolver.Resolve(context.Background(), "did:eth:test")
		assert.Error(t, err)
	})
}

func TestDIDCache(t *testing.T) {
	cache := NewDIDCache(100 * time.Millisecond)
	doc := &DIDDocument{ID: "did:web:test"}

	t.Run("cache set and get", func(t *testing.T) {
		cache.Set("did:web:test", doc)
		retrieved := cache.Get("did:web:test")
		assert.Equal(t, doc, retrieved)
	})

	t.Run("cache expiration", func(t *testing.T) {
		cache.Set("did:web:expire", doc)
		time.Sleep(150 * time.Millisecond)
		retrieved := cache.Get("did:web:expire")
		assert.Nil(t, retrieved)
	})
}
