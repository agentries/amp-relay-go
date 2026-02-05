// Package auth provides authentication and authorization for the AMP Relay Server
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Authenticator defines the interface for DID-based authentication
type Authenticator interface {
	// Verify verifies a DID authentication request
	// Returns the verified DID and nil error on success
	Verify(ctx context.Context, did string, proof *AuthenticationProof) (*VerificationResult, error)

	// ValidateToken validates an existing authentication token
	ValidateToken(ctx context.Context, token string) (*TokenClaims, error)

	// RefreshToken refreshes an authentication token
	RefreshToken(ctx context.Context, token string) (string, error)

	// RevokeToken revokes an authentication token
	RevokeToken(ctx context.Context, token string) error
}

// AuthenticationProof represents the proof of authentication
// This could be a signature, JWT, or other proof mechanism
type AuthenticationProof struct {
	// Type of proof (e.g., "jwt", "signature", "challenge-response")
	Type string `json:"type"`

	// The actual proof data
	Data []byte `json:"data"`

	// Timestamp when the proof was created
	Timestamp time.Time `json:"timestamp"`

	// Challenge that was signed (for challenge-response auth)
	Challenge string `json:"challenge,omitempty"`

	// Signature algorithm used
	Algorithm string `json:"algorithm,omitempty"`
}

// VerificationResult contains the result of a successful authentication
type VerificationResult struct {
	// The verified DID
	DID string `json:"did"`

	// Authentication token (if applicable)
	Token string `json:"token,omitempty"`

	// Token expiration time
	ExpiresAt time.Time `json:"expires_at,omitempty"`

	// Additional claims about the identity
	Claims map[string]interface{} `json:"claims,omitempty"`

	// Verification timestamp
	VerifiedAt time.Time `json:"verified_at"`
}

// TokenClaims represents the claims in an authentication token
type TokenClaims struct {
	// The DID
	DID string `json:"did"`

	// Token issued at time
	IssuedAt time.Time `json:"iat"`

	// Token expiration time
	ExpiresAt time.Time `json:"exp"`

	// Token ID (for revocation)
	TokenID string `json:"jti,omitempty"`

	// Additional claims
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// IsExpired checks if the token claims have expired
func (c *TokenClaims) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// AuthError represents an authentication error
type AuthError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("auth error [%s]: %s", e.Code, e.Message)
}

// Common authentication error codes
const (
	ErrCodeInvalidDID         = "invalid_did"
	ErrCodeInvalidProof       = "invalid_proof"
	ErrCodeExpiredToken       = "expired_token"
	ErrCodeInvalidToken       = "invalid_token"
	ErrCodeTokenRevoked       = "token_revoked"
	ErrCodeAuthFailed         = "authentication_failed"
	ErrCodeDIDNotFound        = "did_not_found"
	ErrCodeServiceUnavailable = "service_unavailable"
)

// PlaceholderAuthenticator is a placeholder implementation that always succeeds
// This is used for development and testing before integrating with real Agentries
// TODO: Replace with real Agentries integration
// See: https://docs.agentries.io/
type PlaceholderAuthenticator struct {
	mu sync.RWMutex
	// In-memory token storage for placeholder implementation
	tokens map[string]*TokenClaims
	// Token validity duration
	tokenDuration time.Duration
}

// NewPlaceholderAuthenticator creates a new placeholder authenticator
func NewPlaceholderAuthenticator() *PlaceholderAuthenticator {
	return &PlaceholderAuthenticator{
		tokens:        make(map[string]*TokenClaims),
		tokenDuration: 24 * time.Hour,
	}
}

// Verify implements placeholder DID verification
// Currently always succeeds with mock verification
// TODO: Integrate with Agentries for real DID verification
func (p *PlaceholderAuthenticator) Verify(ctx context.Context, did string, proof *AuthenticationProof) (*VerificationResult, error) {
	// Basic DID format validation
	if did == "" {
		return nil, &AuthError{Code: ErrCodeInvalidDID, Message: "DID cannot be empty"}
	}

	// TODO: Real Agentries integration would:
	// 1. Resolve the DID to a DID document
	// 2. Verify the proof against the public keys in the document
	// 3. Check for revocation status
	// 4. Validate any additional credentials

	// For now, generate a mock token
	tokenID := generateTokenID()
	now := time.Now()
	expiresAt := now.Add(p.tokenDuration)

	// Store token claims
	claims := &TokenClaims{
		DID:       did,
		IssuedAt:  now,
		ExpiresAt: expiresAt,
		TokenID:   tokenID,
		Extra:     make(map[string]interface{}),
	}

	p.mu.Lock()
	p.tokens[tokenID] = claims
	p.mu.Unlock()

	return &VerificationResult{
		DID:        did,
		Token:      tokenID,
		ExpiresAt:  expiresAt,
		VerifiedAt: now,
		Claims: map[string]interface{}{
			"placeholder": true,
			"note":        "This is a placeholder implementation. Integrate with Agentries for production.",
		},
	}, nil
}

// ValidateToken validates a token in the placeholder implementation
func (p *PlaceholderAuthenticator) ValidateToken(ctx context.Context, token string) (*TokenClaims, error) {
	p.mu.RLock()
	claims, exists := p.tokens[token]
	p.mu.RUnlock()

	if !exists {
		return nil, &AuthError{Code: ErrCodeInvalidToken, Message: "token not found"}
	}

	if claims.IsExpired() {
		p.mu.Lock()
		delete(p.tokens, token)
		p.mu.Unlock()
		return nil, &AuthError{Code: ErrCodeExpiredToken, Message: "token has expired"}
	}

	return claims, nil
}

// RefreshToken refreshes a token in the placeholder implementation
func (p *PlaceholderAuthenticator) RefreshToken(ctx context.Context, token string) (string, error) {
	claims, err := p.ValidateToken(ctx, token)
	if err != nil {
		return "", err
	}

	// Create new token
	newTokenID := generateTokenID()
	now := time.Now()
	expiresAt := now.Add(p.tokenDuration)

	newClaims := &TokenClaims{
		DID:       claims.DID,
		IssuedAt:  now,
		ExpiresAt: expiresAt,
		TokenID:   newTokenID,
		Extra:     claims.Extra,
	}

	// Revoke old token and store new one atomically
	p.mu.Lock()
	delete(p.tokens, token)
	p.tokens[newTokenID] = newClaims
	p.mu.Unlock()

	return newTokenID, nil
}

// RevokeToken revokes a token in the placeholder implementation
func (p *PlaceholderAuthenticator) RevokeToken(ctx context.Context, token string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.tokens[token]; !exists {
		return &AuthError{Code: ErrCodeInvalidToken, Message: "token not found"}
	}

	delete(p.tokens, token)
	return nil
}

// SetTokenDuration sets the token validity duration (for testing)
func (p *PlaceholderAuthenticator) SetTokenDuration(duration time.Duration) {
	p.tokenDuration = duration
}

// generateTokenID generates a cryptographically secure unique token ID
func generateTokenID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return "token_" + hex.EncodeToString(b)
}

// NoOpAuthenticator is an authenticator that does no verification
// Use this when authentication is disabled
type NoOpAuthenticator struct{}

// NewNoOpAuthenticator creates a new no-op authenticator
func NewNoOpAuthenticator() *NoOpAuthenticator {
	return &NoOpAuthenticator{}
}

// Verify always succeeds without verification
func (n *NoOpAuthenticator) Verify(ctx context.Context, did string, proof *AuthenticationProof) (*VerificationResult, error) {
	return &VerificationResult{
		DID:        did,
		VerifiedAt: time.Now(),
		Claims: map[string]interface{}{
			"auth_disabled": true,
		},
	}, nil
}

// ValidateToken always succeeds
func (n *NoOpAuthenticator) ValidateToken(ctx context.Context, token string) (*TokenClaims, error) {
	return &TokenClaims{
		DID:       "anonymous",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil
}

// RefreshToken returns the same token
func (n *NoOpAuthenticator) RefreshToken(ctx context.Context, token string) (string, error) {
	return token, nil
}

// RevokeToken does nothing
func (n *NoOpAuthenticator) RevokeToken(ctx context.Context, token string) error {
	return nil
}

// AuthMiddleware provides helper functions for authentication middleware
// This can be integrated with the HTTP/WebSocket server
type AuthMiddleware struct {
	Authenticator Authenticator
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(auth Authenticator) *AuthMiddleware {
	return &AuthMiddleware{Authenticator: auth}
}

type contextKey struct{}

var didContextKey contextKey

// ExtractDIDFromContext extracts the DID from a context
// This is used after authentication middleware has run
func ExtractDIDFromContext(ctx context.Context) (string, bool) {
	did, ok := ctx.Value(didContextKey).(string)
	return did, ok
}

// ContextWithDID adds a DID to a context
func ContextWithDID(ctx context.Context, did string) context.Context {
	return context.WithValue(ctx, didContextKey, did)
}

// IntegrationPoint defines how auth integrates with the server
// This struct is a placeholder for future server integration
type IntegrationPoint struct {
	// EnableAuth enables authentication
	EnableAuth bool

	// Authenticator is the authenticator to use
	Authenticator Authenticator

	// ExemptRoutes are routes that don't require authentication
	ExemptRoutes []string
}

// NewIntegrationPoint creates a new auth integration point for the server
func NewIntegrationPoint(enableAuth bool) *IntegrationPoint {
	var auth Authenticator
	if enableAuth {
		// Use placeholder for now, will be replaced with Agentries
		auth = NewPlaceholderAuthenticator()
	} else {
		auth = NewNoOpAuthenticator()
	}

	return &IntegrationPoint{
		EnableAuth:    enableAuth,
		Authenticator: auth,
		ExemptRoutes:  []string{"/health", "/ws"}, // WebSocket upgrade exempt - auth happens after upgrade
	}
}
