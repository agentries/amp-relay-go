package auth

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// TestNewPlaceholderAuthenticator
// ---------------------------------------------------------------------------

func TestNewPlaceholderAuthenticator(t *testing.T) {
	a := NewPlaceholderAuthenticator()
	if a == nil {
		t.Fatal("NewPlaceholderAuthenticator returned nil")
	}
	if a.tokens == nil {
		t.Fatal("tokens map is nil; expected initialized map")
	}
	if len(a.tokens) != 0 {
		t.Fatalf("tokens map should be empty, got %d entries", len(a.tokens))
	}
	if a.tokenDuration != 24*time.Hour {
		t.Fatalf("expected tokenDuration 24h, got %v", a.tokenDuration)
	}
}

// ---------------------------------------------------------------------------
// TestPlaceholderAuthenticator_Verify
// ---------------------------------------------------------------------------

func TestPlaceholderAuthenticator_Verify(t *testing.T) {
	tests := []struct {
		name        string
		did         string
		wantErr     bool
		wantErrCode string
	}{
		{
			name:    "valid DID returns token",
			did:     "did:example:123",
			wantErr: false,
		},
		{
			name:        "empty DID returns ErrCodeInvalidDID",
			did:         "",
			wantErr:     true,
			wantErrCode: ErrCodeInvalidDID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewPlaceholderAuthenticator()
			ctx := context.Background()

			result, err := a.Verify(ctx, tt.did, nil)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				authErr, ok := err.(*AuthError)
				if !ok {
					t.Fatalf("expected *AuthError, got %T", err)
				}
				if authErr.Code != tt.wantErrCode {
					t.Fatalf("expected error code %q, got %q", tt.wantErrCode, authErr.Code)
				}
				if result != nil {
					t.Fatal("expected nil result on error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if result.DID != tt.did {
				t.Fatalf("expected DID %q, got %q", tt.did, result.DID)
			}
			if result.Token == "" {
				t.Fatal("expected non-empty token")
			}
			if !strings.HasPrefix(result.Token, "token_") {
				t.Fatalf("token should have 'token_' prefix, got %q", result.Token)
			}
			if result.ExpiresAt.IsZero() {
				t.Fatal("expected non-zero ExpiresAt")
			}
			if result.VerifiedAt.IsZero() {
				t.Fatal("expected non-zero VerifiedAt")
			}
			if result.Claims == nil {
				t.Fatal("expected non-nil Claims")
			}
			if result.Claims["placeholder"] != true {
				t.Fatal("expected claims to contain placeholder=true")
			}

			// The token should also be stored internally
			a.mu.RLock()
			_, exists := a.tokens[result.Token]
			a.mu.RUnlock()
			if !exists {
				t.Fatal("token was not stored in internal map")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestPlaceholderAuthenticator_ValidateToken
// ---------------------------------------------------------------------------

func TestPlaceholderAuthenticator_ValidateToken(t *testing.T) {
	t.Run("valid token", func(t *testing.T) {
		a := NewPlaceholderAuthenticator()
		ctx := context.Background()

		result, err := a.Verify(ctx, "did:example:456", nil)
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}

		claims, err := a.ValidateToken(ctx, result.Token)
		if err != nil {
			t.Fatalf("ValidateToken failed: %v", err)
		}
		if claims.DID != "did:example:456" {
			t.Fatalf("expected DID %q, got %q", "did:example:456", claims.DID)
		}
		if claims.TokenID != result.Token {
			t.Fatalf("expected TokenID %q, got %q", result.Token, claims.TokenID)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		a := NewPlaceholderAuthenticator()
		a.SetTokenDuration(-1 * time.Second) // tokens expire immediately
		ctx := context.Background()

		result, err := a.Verify(ctx, "did:example:expired", nil)
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}

		// Token is already expired because duration is negative
		_, err = a.ValidateToken(ctx, result.Token)
		if err == nil {
			t.Fatal("expected error for expired token, got nil")
		}
		authErr, ok := err.(*AuthError)
		if !ok {
			t.Fatalf("expected *AuthError, got %T", err)
		}
		if authErr.Code != ErrCodeExpiredToken {
			t.Fatalf("expected error code %q, got %q", ErrCodeExpiredToken, authErr.Code)
		}

		// Expired token should be cleaned up from the map
		a.mu.RLock()
		_, exists := a.tokens[result.Token]
		a.mu.RUnlock()
		if exists {
			t.Fatal("expired token should have been removed from internal map")
		}
	})

	t.Run("not-found token", func(t *testing.T) {
		a := NewPlaceholderAuthenticator()
		ctx := context.Background()

		_, err := a.ValidateToken(ctx, "token_does_not_exist")
		if err == nil {
			t.Fatal("expected error for non-existent token, got nil")
		}
		authErr, ok := err.(*AuthError)
		if !ok {
			t.Fatalf("expected *AuthError, got %T", err)
		}
		if authErr.Code != ErrCodeInvalidToken {
			t.Fatalf("expected error code %q, got %q", ErrCodeInvalidToken, authErr.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// TestPlaceholderAuthenticator_RefreshToken
// ---------------------------------------------------------------------------

func TestPlaceholderAuthenticator_RefreshToken(t *testing.T) {
	a := NewPlaceholderAuthenticator()
	ctx := context.Background()

	// Create initial token
	result, err := a.Verify(ctx, "did:example:refresh", nil)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	oldToken := result.Token

	// Refresh
	newToken, err := a.RefreshToken(ctx, oldToken)
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}
	if newToken == "" {
		t.Fatal("expected non-empty new token")
	}
	if newToken == oldToken {
		t.Fatal("new token should differ from old token")
	}
	if !strings.HasPrefix(newToken, "token_") {
		t.Fatalf("new token should have 'token_' prefix, got %q", newToken)
	}

	// Old token should be revoked
	_, err = a.ValidateToken(ctx, oldToken)
	if err == nil {
		t.Fatal("old token should be invalid after refresh")
	}
	authErr, ok := err.(*AuthError)
	if !ok {
		t.Fatalf("expected *AuthError, got %T", err)
	}
	if authErr.Code != ErrCodeInvalidToken {
		t.Fatalf("expected error code %q, got %q", ErrCodeInvalidToken, authErr.Code)
	}

	// New token should be valid
	claims, err := a.ValidateToken(ctx, newToken)
	if err != nil {
		t.Fatalf("ValidateToken on new token failed: %v", err)
	}
	if claims.DID != "did:example:refresh" {
		t.Fatalf("expected DID %q, got %q", "did:example:refresh", claims.DID)
	}
}

// ---------------------------------------------------------------------------
// TestPlaceholderAuthenticator_RevokeToken
// ---------------------------------------------------------------------------

func TestPlaceholderAuthenticator_RevokeToken(t *testing.T) {
	t.Run("revoke existing token", func(t *testing.T) {
		a := NewPlaceholderAuthenticator()
		ctx := context.Background()

		result, err := a.Verify(ctx, "did:example:revoke", nil)
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}

		err = a.RevokeToken(ctx, result.Token)
		if err != nil {
			t.Fatalf("RevokeToken failed: %v", err)
		}

		// Token should no longer be valid
		_, err = a.ValidateToken(ctx, result.Token)
		if err == nil {
			t.Fatal("expected error after revocation, got nil")
		}
	})

	t.Run("revoke non-existent token", func(t *testing.T) {
		a := NewPlaceholderAuthenticator()
		ctx := context.Background()

		err := a.RevokeToken(ctx, "token_nonexistent")
		if err == nil {
			t.Fatal("expected error when revoking non-existent token, got nil")
		}
		authErr, ok := err.(*AuthError)
		if !ok {
			t.Fatalf("expected *AuthError, got %T", err)
		}
		if authErr.Code != ErrCodeInvalidToken {
			t.Fatalf("expected error code %q, got %q", ErrCodeInvalidToken, authErr.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// TestPlaceholderAuthenticator_ConcurrentAccess
// ---------------------------------------------------------------------------

func TestPlaceholderAuthenticator_ConcurrentAccess(t *testing.T) {
	a := NewPlaceholderAuthenticator()
	ctx := context.Background()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Collect tokens produced by Verify so we can validate/revoke them
	// concurrently from other goroutines.
	tokenCh := make(chan string, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()

			// Each goroutine verifies a unique DID
			did := "did:example:concurrent_" + string(rune('A'+idx%26))
			result, err := a.Verify(ctx, did, nil)
			if err != nil {
				t.Errorf("Verify failed for %s: %v", did, err)
				return
			}
			tokenCh <- result.Token

			// Validate our own token
			_, err = a.ValidateToken(ctx, result.Token)
			if err != nil {
				// Token may have been revoked by another goroutine already;
				// that is acceptable in a concurrent scenario.
				return
			}
		}(i)
	}

	wg.Wait()
	close(tokenCh)

	// Collect all tokens and revoke them concurrently
	var tokens []string
	for tok := range tokenCh {
		tokens = append(tokens, tok)
	}

	var wg2 sync.WaitGroup
	wg2.Add(len(tokens))
	for _, tok := range tokens {
		go func(token string) {
			defer wg2.Done()
			// Ignore errors; some may already be revoked by concurrent goroutines
			_ = a.RevokeToken(ctx, token)
		}(tok)
	}
	wg2.Wait()

	// After all revocations the token map should be empty
	a.mu.RLock()
	remaining := len(a.tokens)
	a.mu.RUnlock()
	if remaining != 0 {
		t.Fatalf("expected 0 tokens remaining after full revocation, got %d", remaining)
	}
}

// ---------------------------------------------------------------------------
// TestNoOpAuthenticator
// ---------------------------------------------------------------------------

func TestNoOpAuthenticator(t *testing.T) {
	a := NewNoOpAuthenticator()
	ctx := context.Background()

	t.Run("Verify always succeeds", func(t *testing.T) {
		result, err := a.Verify(ctx, "did:example:noop", nil)
		if err != nil {
			t.Fatalf("NoOp Verify should not fail: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.DID != "did:example:noop" {
			t.Fatalf("expected DID %q, got %q", "did:example:noop", result.DID)
		}
		if result.Claims["auth_disabled"] != true {
			t.Fatal("expected auth_disabled=true claim")
		}
	})

	t.Run("ValidateToken always succeeds", func(t *testing.T) {
		claims, err := a.ValidateToken(ctx, "any-token-value")
		if err != nil {
			t.Fatalf("NoOp ValidateToken should not fail: %v", err)
		}
		if claims == nil {
			t.Fatal("expected non-nil claims")
		}
		if claims.DID != "anonymous" {
			t.Fatalf("expected DID %q, got %q", "anonymous", claims.DID)
		}
		if claims.IsExpired() {
			t.Fatal("NoOp token claims should not be expired")
		}
	})

	t.Run("RefreshToken returns same token", func(t *testing.T) {
		newToken, err := a.RefreshToken(ctx, "original-token")
		if err != nil {
			t.Fatalf("NoOp RefreshToken should not fail: %v", err)
		}
		if newToken != "original-token" {
			t.Fatalf("NoOp RefreshToken should return same token, got %q", newToken)
		}
	})

	t.Run("RevokeToken succeeds", func(t *testing.T) {
		err := a.RevokeToken(ctx, "any-token")
		if err != nil {
			t.Fatalf("NoOp RevokeToken should not fail: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestContextWithDID_ExtractDIDFromContext
// ---------------------------------------------------------------------------

func TestContextWithDID_ExtractDIDFromContext(t *testing.T) {
	tests := []struct {
		name string
		did  string
	}{
		{name: "standard DID", did: "did:example:roundtrip"},
		{name: "empty string DID", did: ""},
		{name: "complex DID", did: "did:web:example.com:user:alice"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := ContextWithDID(context.Background(), tt.did)
			got, ok := ExtractDIDFromContext(ctx)
			if !ok {
				t.Fatal("expected ok=true from ExtractDIDFromContext")
			}
			if got != tt.did {
				t.Fatalf("expected DID %q, got %q", tt.did, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestExtractDIDFromContext_Missing
// ---------------------------------------------------------------------------

func TestExtractDIDFromContext_Missing(t *testing.T) {
	ctx := context.Background()
	_, ok := ExtractDIDFromContext(ctx)
	if ok {
		t.Fatal("expected ok=false when no DID is in context")
	}
}

// ---------------------------------------------------------------------------
// TestNewIntegrationPoint
// ---------------------------------------------------------------------------

func TestNewIntegrationPoint(t *testing.T) {
	t.Run("auth enabled uses PlaceholderAuthenticator", func(t *testing.T) {
		ip := NewIntegrationPoint(true)
		if ip == nil {
			t.Fatal("expected non-nil IntegrationPoint")
		}
		if !ip.EnableAuth {
			t.Fatal("expected EnableAuth=true")
		}
		if _, ok := ip.Authenticator.(*PlaceholderAuthenticator); !ok {
			t.Fatalf("expected *PlaceholderAuthenticator, got %T", ip.Authenticator)
		}
		if len(ip.ExemptRoutes) == 0 {
			t.Fatal("expected non-empty ExemptRoutes")
		}
	})

	t.Run("auth disabled uses NoOpAuthenticator", func(t *testing.T) {
		ip := NewIntegrationPoint(false)
		if ip == nil {
			t.Fatal("expected non-nil IntegrationPoint")
		}
		if ip.EnableAuth {
			t.Fatal("expected EnableAuth=false")
		}
		if _, ok := ip.Authenticator.(*NoOpAuthenticator); !ok {
			t.Fatalf("expected *NoOpAuthenticator, got %T", ip.Authenticator)
		}
	})
}

// ---------------------------------------------------------------------------
// TestGenerateTokenID
// ---------------------------------------------------------------------------

func TestGenerateTokenID(t *testing.T) {
	t.Run("has token_ prefix", func(t *testing.T) {
		id := generateTokenID()
		if !strings.HasPrefix(id, "token_") {
			t.Fatalf("expected 'token_' prefix, got %q", id)
		}
	})

	t.Run("unique across calls", func(t *testing.T) {
		const n = 100
		seen := make(map[string]struct{}, n)
		for i := 0; i < n; i++ {
			id := generateTokenID()
			if _, dup := seen[id]; dup {
				t.Fatalf("duplicate token ID generated: %q", id)
			}
			seen[id] = struct{}{}
		}
	})

	t.Run("expected length", func(t *testing.T) {
		id := generateTokenID()
		// "token_" (6 chars) + hex-encoded 16 bytes (32 chars) = 38
		if len(id) != 38 {
			t.Fatalf("expected token ID length 38, got %d (%q)", len(id), id)
		}
	})
}
