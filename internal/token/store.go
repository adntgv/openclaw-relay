package token

import (
	"encoding/json"
	"log/slog"
	"os"
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

// Store manages tokens in memory with optional file persistence
type Store struct {
	mu        sync.RWMutex
	tokens    map[string]*Token // indexed by JTI
	revoked   map[string]bool   // revoked JTIs
	byClawID  map[string][]*Token
	filePath  string // optional file path for persistence
}

// NewStore creates a new token store
func NewStore() *Store {
	return &Store{
		tokens:   make(map[string]*Token),
		revoked:  make(map[string]bool),
		byClawID: make(map[string][]*Token),
	}
}

// NewStoreWithFile creates a token store with file-backed persistence
func NewStoreWithFile(filePath string) (*Store, error) {
	s := &Store{
		tokens:   make(map[string]*Token),
		revoked:  make(map[string]bool),
		byClawID: make(map[string][]*Token),
		filePath: filePath,
	}
	
	// Load existing tokens from file
	if err := s.Load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	
	return s, nil
}

// persistedStore represents the JSON structure for persistence
type persistedStore struct {
	Tokens  []*Token `json:"tokens"`
	Revoked []string `json:"revoked"`
}

// Load reads tokens from the file
func (s *Store) Load() error {
	if s.filePath == "" {
		return nil // no persistence configured
	}
	
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	
	var ps persistedStore
	if err := json.Unmarshal(data, &ps); err != nil {
		return err
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Rebuild in-memory structures
	for _, token := range ps.Tokens {
		s.tokens[token.JTI] = token
		s.byClawID[token.ClawID] = append(s.byClawID[token.ClawID], token)
	}
	
	for _, jti := range ps.Revoked {
		s.revoked[jti] = true
	}
	
	slog.Info("loaded tokens from file", "path", s.filePath, "count", len(ps.Tokens))
	return nil
}

// Save writes tokens to the file
func (s *Store) Save() error {
	if s.filePath == "" {
		return nil // no persistence configured
	}
	
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Collect all tokens and revoked JTIs
	tokens := make([]*Token, 0, len(s.tokens))
	for _, token := range s.tokens {
		tokens = append(tokens, token)
	}
	
	revoked := make([]string, 0, len(s.revoked))
	for jti := range s.revoked {
		revoked = append(revoked, jti)
	}
	
	ps := persistedStore{
		Tokens:  tokens,
		Revoked: revoked,
	}
	
	data, err := json.MarshalIndent(ps, "", "  ")
	if err != nil {
		return err
	}
	
	// Write atomically via temp file
	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return err
	}
	
	if err := os.Rename(tmpPath, s.filePath); err != nil {
		return err
	}
	
	return nil
}

// Add adds a token to the store
func (s *Store) Add(token Token) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens[token.JTI] = &token
	s.byClawID[token.ClawID] = append(s.byClawID[token.ClawID], &token)
	
	// Persist if configured
	s.mu.Unlock()
	s.Save()
	s.mu.Lock()
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
	
	// Persist if configured
	s.mu.Unlock()
	s.Save()
	s.mu.Lock()

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
