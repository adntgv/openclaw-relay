package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/adntgv/openclaw-relay/internal/protocol"
	"github.com/adntgv/openclaw-relay/internal/token"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Server is the main relay server
type Server struct {
	hub          *Hub
	httpServer   *http.Server
	adminToken   string
	tokenManager *token.Manager
	tokenStore   *token.Store
	audit        *AuditLog
	dispatcher   *Dispatcher
	rateLimiter  *RateLimiter
	startTime    time.Time
	logger       *slog.Logger
	// Metrics
	totalConnections atomic.Int64
	totalCommands    atomic.Int64
}

// Config holds server configuration
type Config struct {
	Host           string
	Port           int
	AdminToken     string
	JWTSecret      string
	TokenStorePath string // optional path for token persistence
}

// New creates a new Server instance
func New(cfg Config) *Server {
	hub := NewHub()
	
	var store *token.Store
	var err error
	if cfg.TokenStorePath != "" {
		store, err = token.NewStoreWithFile(cfg.TokenStorePath)
		if err != nil {
			slog.Error("failed to load token store", "error", err)
			store = token.NewStore() // fallback to in-memory
		}
	} else {
		store = token.NewStore()
	}
	
	mgr := token.NewManager(cfg.JWTSecret, store)
	audit := NewAuditLog()

	logger := slog.Default()

	srv := &Server{
		hub:          hub,
		adminToken:   cfg.AdminToken,
		tokenManager: mgr,
		tokenStore:   store,
		audit:        audit,
		dispatcher:   NewDispatcher(),
		rateLimiter:  NewRateLimiter(10, time.Second, 30), // 10 req/s, burst 30
		startTime:    time.Now(),
		logger:       logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.handleRoot)
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/ws", srv.handleWebSocket)
	mux.HandleFunc("/token", srv.adminAuth(srv.handleTokenRoute))
	mux.HandleFunc("/token/", srv.adminAuth(srv.handleTokenRoute))
	mux.HandleFunc("/clients", srv.adminAuth(srv.handleClients))
	mux.HandleFunc("/command", srv.adminAuth(srv.handleCommand))
	mux.HandleFunc("/audit", srv.adminAuth(srv.handleAudit))
	mux.HandleFunc("/metrics", srv.handleMetrics)

	srv.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler: srv.rateLimiter.Middleware(mux),
	}

	return srv
}

// handleTokenRoute dispatches POST /token vs DELETE /token/{jti}
func (s *Server) handleTokenRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/token")
	path = strings.TrimPrefix(path, "/")

	if r.Method == http.MethodPost && path == "" {
		s.handleTokenIssue(w, r)
		return
	}
	if r.Method == http.MethodDelete && path != "" {
		// path is the JTI
		s.tokenManager.Revoke(path)
		s.audit.Log("token.revoke", "", "jti="+path)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("starting relay server", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down server, draining connections...")
	
	// Close all WebSocket connections first
	s.hub.CloseAll()
	
	// Then shut down the HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}
	
	s.logger.Info("server shutdown complete")
	return nil
}

// handleRoot serves the landing page with instructions
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>OpenClaw Relay</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, monospace;
         background: #0a0a0a; color: #e0e0e0; padding: 2rem; line-height: 1.6; }
  .container { max-width: 720px; margin: 0 auto; }
  h1 { color: #4fc3f7; font-size: 2rem; margin-bottom: 0.5rem; }
  h2 { color: #81d4fa; font-size: 1.2rem; margin-top: 2rem; margin-bottom: 0.5rem; }
  .subtitle { color: #888; margin-bottom: 2rem; }
  pre { background: #1a1a1a; border: 1px solid #333; border-radius: 6px;
        padding: 1rem; overflow-x: auto; margin: 0.5rem 0 1rem; font-size: 0.85rem; }
  code { color: #4fc3f7; }
  .endpoint { display: flex; gap: 0.5rem; align-items: center; margin: 0.3rem 0; }
  .method { background: #1b5e20; color: #a5d6a7; padding: 2px 8px; border-radius: 3px;
            font-size: 0.75rem; font-weight: bold; min-width: 52px; text-align: center; }
  .method.post { background: #e65100; color: #ffcc80; }
  .method.delete { background: #b71c1c; color: #ef9a9a; }
  .path { color: #ccc; font-family: monospace; }
  .auth { color: #666; font-size: 0.8rem; }
  a { color: #4fc3f7; }
  .status { display: inline-block; background: #1b5e20; color: #a5d6a7;
            padding: 2px 10px; border-radius: 12px; font-size: 0.8rem; }
</style>
</head>
<body>
<div class="container">
  <h1>🦀 OpenClaw Relay</h1>
  <p class="subtitle">WebSocket command relay for OpenClaw agents <span class="status">● running</span></p>

  <h2>Quick Start</h2>
  <p>1. Issue a client token:</p>
  <pre>curl -X POST https://relay.adntgv.com/token \
  -H "X-Admin-Token: YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"claw_id": "my-node", "scopes": ["shell"], "ttl_hours": 720}'</pre>

  <p>2. Connect a client:</p>
  <pre>relay client \
  --url wss://relay.adntgv.com/ws \
  --token TOKEN_FROM_STEP_1 \
  --claw-id my-node</pre>

  <p>3. Send a command:</p>
  <pre>curl -X POST https://relay.adntgv.com/command \
  -H "X-Admin-Token: YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"claw_id": "my-node", "cmd": "shell.exec", "args": {"command": "echo hello"}}'</pre>

  <h2>API Endpoints</h2>
  <div class="endpoint"><span class="method">GET</span><span class="path">/health</span></div>
  <div class="endpoint"><span class="method">GET</span><span class="path">/metrics</span></div>
  <div class="endpoint"><span class="method">GET</span><span class="path">/ws</span><span class="auth">— WebSocket (JWT in hello)</span></div>
  <div class="endpoint"><span class="method post">POST</span><span class="path">/token</span><span class="auth">— Issue JWT token 🔒</span></div>
  <div class="endpoint"><span class="method delete">DELETE</span><span class="path">/token/{jti}</span><span class="auth">— Revoke token 🔒</span></div>
  <div class="endpoint"><span class="method">GET</span><span class="path">/clients</span><span class="auth">— List connected clients 🔒</span></div>
  <div class="endpoint"><span class="method post">POST</span><span class="path">/command</span><span class="auth">— Dispatch command 🔒</span></div>
  <div class="endpoint"><span class="method">GET</span><span class="path">/audit</span><span class="auth">— Audit log 🔒</span></div>
  <p style="color:#666;font-size:0.8rem;margin-top:0.5rem;">🔒 = requires <code>X-Admin-Token</code> header</p>

  <h2>Install Client</h2>
  <pre># From source
go install github.com/adntgv/openclaw-relay/cmd/relay@latest

# Or download binary from GitHub Releases
# https://github.com/adntgv/openclaw-relay/releases</pre>

  <h2>Links</h2>
  <p>
    <a href="https://github.com/adntgv/openclaw-relay">GitHub</a> ·
    <a href="/health">Health Check</a> ·
    <a href="/metrics">Metrics</a> ·
    <a href="https://docs.openclaw.ai">OpenClaw Docs</a>
  </p>
</div>
</body>
</html>`)
}

// handleHealth returns server health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"clients": s.hub.Count(),
		"uptime":  time.Since(s.startTime).Seconds(),
	})
}

// handleWebSocket handles WebSocket upgrade
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("websocket upgrade error", "error", err)
		return
	}

	s.totalConnections.Add(1)
	conn := NewConnection(ws, s.hub)
	conn.SetMessageHandler(func(env *protocol.Envelope) {
		s.handleMessage(conn, env)
	})
	conn.Start()
}

// handleMetrics returns basic metrics in JSON format
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"connected_clients":  s.hub.Count(),
		"total_connections": s.totalConnections.Load(),
		"total_commands":    s.totalCommands.Load(),
		"uptime_seconds":    time.Since(s.startTime).Seconds(),
	})
}

// handleMessage processes incoming messages
func (s *Server) handleMessage(conn *Connection, env *protocol.Envelope) {
	switch env.Type {
	case protocol.TypeHello:
		s.handleHello(conn, env)
	case protocol.TypePong:
		// alive
	case protocol.TypeAck:
		s.handleAckMessage(conn, env)
	default:
		s.logger.Warn("unknown message type", "type", env.Type)
	}
}

// handleHello authenticates and registers clients
func (s *Server) handleHello(conn *Connection, env *protocol.Envelope) {
	var payload protocol.HelloPayload
	if err := env.UnmarshalPayload(&payload); err != nil {
		s.logger.Warn("invalid hello payload", "error", err)
		s.sendError(conn, env.ID, "invalid hello payload")
		conn.Close()
		return
	}

	if payload.ClawID == "" {
		s.sendError(conn, env.ID, "claw_id is required")
		conn.Close()
		return
	}

	// Validate JWT token
	claims, err := s.tokenManager.Validate(payload.Token)
	if err != nil {
		s.logger.Warn("auth failed", "claw_id", payload.ClawID, "error", err)
		s.sendError(conn, env.ID, "authentication failed: "+err.Error())
		conn.Close()
		return
	}

	// Verify claw_id matches token
	if claims.ClawID != payload.ClawID {
		s.logger.Warn("claw_id mismatch", "token_claw_id", claims.ClawID, "payload_claw_id", payload.ClawID)
		s.sendError(conn, env.ID, "claw_id does not match token")
		conn.Close()
		return
	}

	// Register client
	conn.SetClawID(payload.ClawID)
	client := &Client{
		ClawID:       payload.ClawID,
		Capabilities: payload.Capabilities,
		Conn:         conn,
		ConnectedAt:  time.Now(),
	}
	s.hub.Register(payload.ClawID, client)
	s.audit.Log("client.connect", payload.ClawID, "caps="+strings.Join(payload.Capabilities, ","))

	// Send ack
	ack := protocol.AckPayload{Status: "ok"}
	ackEnv, _ := protocol.New(protocol.TypeAck, ack)
	conn.Send(ackEnv)

	s.logger.Info("client authenticated", "claw_id", payload.ClawID)
}

// handleAckMessage routes ack messages to the dispatcher
func (s *Server) handleAckMessage(conn *Connection, env *protocol.Envelope) {
	var payload protocol.AckPayload
	if err := env.UnmarshalPayload(&payload); err != nil {
		s.logger.Warn("invalid ack payload", "error", err)
		return
	}

	// Use RefID to find the original command, fall back to envelope ID
	refID := payload.RefID
	if refID == "" {
		refID = env.ID
	}

	result := &AckResult{
		Status: payload.Status,
		Result: payload.Result,
		Error:  payload.Error,
	}

	if !s.dispatcher.Resolve(refID, result) {
		s.logger.Debug("no pending command for ack", "ref_id", refID)
	}
}

// sendError sends an error ack to a connection
func (s *Server) sendError(conn *Connection, refID string, msg string) {
	ack := protocol.AckPayload{Status: "error", Error: msg}
	env, _ := protocol.New(protocol.TypeAck, ack)
	conn.Send(env)
	s.logger.Warn("sent error ack", "ref_id", refID, "error", msg)
}
