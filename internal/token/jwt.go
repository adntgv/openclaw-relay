package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenRevoked = errors.New("token has been revoked")
)

// Claims represents JWT claims
type Claims struct {
	ClawID string   `json:"claw_id"`
	Scopes []string `json:"scopes"`
	jwt.RegisteredClaims
}

// Manager handles JWT creation and validation
type Manager struct {
	secret []byte
	store  *Store
}

// NewManager creates a new JWT manager
func NewManager(secret string, store *Store) *Manager {
	return &Manager{
		secret: []byte(secret),
		store:  store,
	}
}

// Issue creates a new JWT token
func (m *Manager) Issue(clawID string, scopes []string, ttlHours int) (string, error) {
	jti := uuid.New().String()
	now := time.Now()
	expiresAt := now.Add(time.Hour * time.Duration(ttlHours))

	claims := Claims{
		ClawID: clawID,
		Scopes: scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Subject:   clawID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secret)
	if err != nil {
		return "", err
	}

	// Store token metadata
	m.store.Add(Token{
		JTI:       jti,
		ClawID:    clawID,
		Scopes:    scopes,
		IssuedAt:  now,
		ExpiresAt: expiresAt,
	})

	return tokenString, nil
}

// Validate validates a JWT token and returns its claims
func (m *Manager) Validate(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Check if token is revoked
	if m.store.IsRevoked(claims.ID) {
		return nil, ErrTokenRevoked
	}

	return claims, nil
}

// Revoke revokes a token by its JTI
func (m *Manager) Revoke(jti string) error {
	return m.store.Revoke(jti)
}
