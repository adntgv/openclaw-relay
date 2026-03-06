package token

import (
	"testing"
	"time"
)

func TestStoreAddGet(t *testing.T) {
	s := NewStore()
	s.Add(Token{JTI: "j1", ClawID: "c1", Scopes: []string{"shell"}, IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)})

	tok, ok := s.Get("j1")
	if !ok || tok.ClawID != "c1" {
		t.Fatalf("Get j1 failed: ok=%v", ok)
	}
	if _, ok := s.Get("missing"); ok {
		t.Fatal("Get missing should return false")
	}
}

func TestStoreRevoke(t *testing.T) {
	s := NewStore()
	s.Add(Token{JTI: "j1", ClawID: "c1", ExpiresAt: time.Now().Add(time.Hour)})

	if s.IsRevoked("j1") {
		t.Fatal("should not be revoked yet")
	}
	s.Revoke("j1")
	if !s.IsRevoked("j1") {
		t.Fatal("should be revoked")
	}
}

func TestStoreCleanup(t *testing.T) {
	s := NewStore()
	s.Add(Token{JTI: "expired", ClawID: "c1", ExpiresAt: time.Now().Add(-time.Hour)})
	s.Add(Token{JTI: "valid", ClawID: "c1", ExpiresAt: time.Now().Add(time.Hour)})

	removed := s.Cleanup()
	if removed != 1 {
		t.Fatalf("Cleanup removed %d, want 1", removed)
	}
	if _, ok := s.Get("expired"); ok {
		t.Fatal("expired token should be gone")
	}
	if _, ok := s.Get("valid"); !ok {
		t.Fatal("valid token should remain")
	}
}

func TestManagerIssueValidate(t *testing.T) {
	s := NewStore()
	m := NewManager("test-secret-key-32bytes!!!!!!!!", s)

	tokenStr, err := m.Issue("my-claw", []string{"shell"}, 1)
	if err != nil {
		t.Fatalf("Issue error: %v", err)
	}

	claims, err := m.Validate(tokenStr)
	if err != nil {
		t.Fatalf("Validate error: %v", err)
	}
	if claims.ClawID != "my-claw" {
		t.Errorf("ClawID = %v, want my-claw", claims.ClawID)
	}
	if len(claims.Scopes) != 1 || claims.Scopes[0] != "shell" {
		t.Errorf("Scopes = %v, want [shell]", claims.Scopes)
	}
}

func TestManagerValidateInvalid(t *testing.T) {
	s := NewStore()
	m := NewManager("test-secret", s)

	if _, err := m.Validate("garbage"); err == nil {
		t.Fatal("should reject garbage token")
	}

	// Wrong secret
	m2 := NewManager("other-secret", NewStore())
	tok, _ := m2.Issue("c1", nil, 1)
	if _, err := m.Validate(tok); err == nil {
		t.Fatal("should reject token signed with different secret")
	}
}

func TestManagerRevoke(t *testing.T) {
	s := NewStore()
	m := NewManager("test-secret", s)

	tok, _ := m.Issue("c1", nil, 1)
	claims, _ := m.Validate(tok)

	m.Revoke(claims.ID)

	if _, err := m.Validate(tok); err != ErrTokenRevoked {
		t.Fatalf("expected ErrTokenRevoked, got %v", err)
	}
}
