package token

import (
	"sync"
	"time"
)

// Token represents a token metadata
type Token struct {
	JTI       string
	ClawID    string
	Scopes    []string
	IssuedAt  time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// Store manages tokens in memory
type Store struct {
	mu       sync.RWMutex
	tokens   map[string]*Token // indexed by JTI
	revoked  map[string]bool   // revoked JTIs
	byClawID map[string][]*Token
}

// NewStore creates a new token store
func NewStore() *Store {
	return &Store{
		tokens:   make(map[string]*Token),
		revoked:  make(map[string]bool),
		byClawID: make(map[string][]*Token),
	}
}

// Add adds a token to the store
func (s *Store) Add(token Token) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens[token.JTI] = &token
	s.byClawID[token.ClawID] = append(s.byClawID[token.ClawID], &token)
}

// Get retrieves a token by JTI
func (s *Store) Get(jti string) (*Token, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	token, ok := s.tokens[jti]
	return token, ok
}

// GetByClawID retrieves all tokens for a claw ID
func (s *Store) GetByClawID(clawID string) []*Token {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tokens := s.byClawID[clawID]
	result := make([]*Token, len(tokens))
	copy(result, tokens)
	return result
}

// Revoke marks a token as revoked
func (s *Store) Revoke(jti string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, ok := s.tokens[jti]
	if !ok {
		return nil // Already doesn't exist
	}

	now := time.Now()
	token.RevokedAt = &now
	s.revoked[jti] = true

	return nil
}

// IsRevoked checks if a token is revoked
func (s *Store) IsRevoked(jti string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.revoked[jti]
}

// List returns all tokens
func (s *Store) List() []*Token {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tokens := make([]*Token, 0, len(s.tokens))
	for _, token := range s.tokens {
		tokens = append(tokens, token)
	}
	return tokens
}

// Cleanup removes expired tokens
func (s *Store) Cleanup() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	removed := 0

	for jti, token := range s.tokens {
		if token.ExpiresAt.Before(now) {
			delete(s.tokens, jti)
			delete(s.revoked, jti)
			
			// Remove from byClawID
			clawTokens := s.byClawID[token.ClawID]
			for i, t := range clawTokens {
				if t.JTI == jti {
					s.byClawID[token.ClawID] = append(clawTokens[:i], clawTokens[i+1:]...)
					break
				}
			}
			
			removed++
		}
	}

	return removed
}
