package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// adminAuth middleware checks X-Admin-Token header
func (s *Server) adminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Admin-Token")
		if token != s.adminToken {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// handleTokenIssue handles POST /token
func (s *Server) handleTokenIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ClawID   string   `json:"claw_id"`
		Scopes   []string `json:"scopes"`
		TTLHours int      `json:"ttl_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.ClawID == "" {
		http.Error(w, `{"error":"claw_id is required"}`, http.StatusBadRequest)
		return
	}
	if req.TTLHours <= 0 {
		req.TTLHours = 24
	}

	tokenStr, err := s.tokenManager.Issue(req.ClawID, req.Scopes, req.TTLHours)
	if err != nil {
		http.Error(w, `{"error":"failed to issue token"}`, http.StatusInternalServerError)
		return
	}

	s.audit.Log("token.issue", req.ClawID, "scopes="+strings.Join(req.Scopes, ","))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": tokenStr})
}

// handleTokenRevoke handles DELETE /token/{jti}
func (s *Server) handleTokenRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Extract JTI from path: /token/{jti}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/token/"), "/")
	jti := parts[0]
	if jti == "" {
		http.Error(w, `{"error":"jti is required"}`, http.StatusBadRequest)
		return
	}

	s.tokenManager.Revoke(jti)
	s.audit.Log("token.revoke", "", "jti="+jti)

	w.WriteHeader(http.StatusNoContent)
}

// handleClients handles GET /clients
func (s *Server) handleClients(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	clients := s.hub.List()
	type clientInfo struct {
		ClawID       string   `json:"claw_id"`
		Capabilities []string `json:"caps"`
		ConnectedAt  string   `json:"connected_at"`
	}

	result := make([]clientInfo, len(clients))
	for i, c := range clients {
		result[i] = clientInfo{
			ClawID:       c.ClawID,
			Capabilities: c.Capabilities,
			ConnectedAt:  c.ConnectedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleCommand handles POST /command (stub for Phase 2, full in Phase 3)
func (s *Server) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ClawID string                 `json:"claw_id"`
		Cmd    string                 `json:"cmd"`
		Args   map[string]interface{} `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.ClawID == "" || req.Cmd == "" {
		http.Error(w, `{"error":"claw_id and cmd are required"}`, http.StatusBadRequest)
		return
	}

	// Check if client is connected
	client, ok := s.hub.Get(req.ClawID)
	if !ok {
		http.Error(w, `{"error":"client not connected"}`, http.StatusNotFound)
		return
	}

	s.audit.Log("command.dispatch", req.ClawID, "cmd="+req.Cmd)

	// Full dispatch implemented in Phase 3
	_ = client
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "dispatched", "result": "stub - full dispatch in Phase 3"})
}

// handleAudit handles GET /audit
func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	clawID := r.URL.Query().Get("claw_id")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	entries := s.audit.Query(clawID, limit)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}
