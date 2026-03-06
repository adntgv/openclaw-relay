package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHub(t *testing.T) {
	hub := NewHub()
	if hub.Count() != 0 {
		t.Fatalf("NewHub count = %v, want 0", hub.Count())
	}

	c1 := &Client{ClawID: "c1", Capabilities: []string{"shell"}, ConnectedAt: time.Now()}
	hub.Register("c1", c1)
	if hub.Count() != 1 {
		t.Fatalf("count after register = %v, want 1", hub.Count())
	}

	got, ok := hub.Get("c1")
	if !ok || got.ClawID != "c1" {
		t.Fatalf("Get c1 failed")
	}
	if _, ok := hub.Get("nope"); ok {
		t.Fatal("Get nope should fail")
	}

	hub.Register("c2", &Client{ClawID: "c2", ConnectedAt: time.Now()})
	if len(hub.List()) != 2 {
		t.Fatalf("List len = %d, want 2", len(hub.List()))
	}

	hub.Unregister("c1")
	if hub.Count() != 1 {
		t.Fatalf("count after unregister = %v, want 1", hub.Count())
	}
}

func TestHubConcurrency(t *testing.T) {
	hub := NewHub()
	done := make(chan bool, 3)
	for _, fn := range []func(){
		func() { for i := 0; i < 100; i++ { hub.Register(string(rune(i)), &Client{ConnectedAt: time.Now()}) }; done <- true },
		func() { for i := 0; i < 100; i++ { hub.Unregister(string(rune(i))) }; done <- true },
		func() { for i := 0; i < 100; i++ { hub.List() }; done <- true },
	} {
		go fn()
	}
	<-done; <-done; <-done
}

// Helper to create a test server
func newTestServer() *Server {
	return New(Config{
		Host:       "127.0.0.1",
		Port:       0,
		AdminToken: "test-admin",
		JWTSecret:  "test-jwt-secret-32bytes!!!!!!!!!",
	})
}

func TestHealthEndpoint(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("health status = %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Fatalf("health status = %v", resp["status"])
	}
}

func TestAdminAuthRequired(t *testing.T) {
	srv := newTestServer()

	for _, path := range []string{"/clients", "/command", "/audit", "/token"} {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s without auth: got %d, want 401", path, w.Code)
		}
	}
}

func TestTokenIssueAndRevoke(t *testing.T) {
	srv := newTestServer()

	// Issue token
	body := `{"claw_id":"test-claw","scopes":["shell"],"ttl_hours":1}`
	req := httptest.NewRequest("POST", "/token", strings.NewReader(body))
	req.Header.Set("X-Admin-Token", "test-admin")
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("token issue status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	tok := resp["token"]
	if tok == "" {
		t.Fatal("no token returned")
	}

	// Validate the token
	claims, err := srv.tokenManager.Validate(tok)
	if err != nil {
		t.Fatalf("token validation failed: %v", err)
	}

	// Revoke it
	req = httptest.NewRequest("DELETE", "/token/"+claims.ID, nil)
	req.Header.Set("X-Admin-Token", "test-admin")
	w = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != 204 {
		t.Fatalf("revoke status = %d", w.Code)
	}

	// Verify revoked
	if _, err := srv.tokenManager.Validate(tok); err == nil {
		t.Fatal("revoked token should fail validation")
	}
}

func TestClientsEndpoint(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest("GET", "/clients", nil)
	req.Header.Set("X-Admin-Token", "test-admin")
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("clients status = %d", w.Code)
	}

	var clients []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&clients)
	if len(clients) != 0 {
		t.Fatalf("expected 0 clients, got %d", len(clients))
	}
}

func TestAuditEndpoint(t *testing.T) {
	srv := newTestServer()
	srv.audit.Log("test.action", "test-claw", "detail")

	req := httptest.NewRequest("GET", "/audit?limit=10", nil)
	req.Header.Set("X-Admin-Token", "test-admin")
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("audit status = %d", w.Code)
	}

	body, _ := io.ReadAll(w.Body)
	if !bytes.Contains(body, []byte("test.action")) {
		t.Fatalf("audit missing entry: %s", body)
	}
}

func TestCommandClientNotFound(t *testing.T) {
	srv := newTestServer()

	body := `{"claw_id":"nonexistent","cmd":"shell.exec"}`
	req := httptest.NewRequest("POST", "/command", strings.NewReader(body))
	req.Header.Set("X-Admin-Token", "test-admin")
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("command to missing client: got %d, want 404", w.Code)
	}
}
