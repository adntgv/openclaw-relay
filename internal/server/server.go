package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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
	startTime    time.Time
}

// Config holds server configuration
type Config struct {
	Host       string
	Port       int
	AdminToken string
	JWTSecret  string
}

// New creates a new Server instance
func New(cfg Config) *Server {
	hub := NewHub()
	store := token.NewStore()
	mgr := token.NewManager(cfg.JWTSecret, store)
	audit := NewAuditLog()

	srv := &Server{
		hub:          hub,
		adminToken:   cfg.AdminToken,
		tokenManager: mgr,
		tokenStore:   store,
		audit:        audit,
		dispatcher:   NewDispatcher(),
		startTime:    time.Now(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/ws", srv.handleWebSocket)
	mux.HandleFunc("/token", srv.adminAuth(srv.handleTokenRoute))
	mux.HandleFunc("/token/", srv.adminAuth(srv.handleTokenRoute))
	mux.HandleFunc("/clients", srv.adminAuth(srv.handleClients))
	mux.HandleFunc("/command", srv.adminAuth(srv.handleCommand))
	mux.HandleFunc("/audit", srv.adminAuth(srv.handleAudit))

	srv.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler: mux,
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
	log.Printf("Starting relay server on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
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
		log.Printf("websocket upgrade error: %v", err)
		return
	}

	conn := NewConnection(ws, s.hub)
	conn.SetMessageHandler(func(env *protocol.Envelope) {
		s.handleMessage(conn, env)
	})
	conn.Start()
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
		log.Printf("unknown message type: %s", env.Type)
	}
}

// handleHello authenticates and registers clients
func (s *Server) handleHello(conn *Connection, env *protocol.Envelope) {
	var payload protocol.HelloPayload
	if err := env.UnmarshalPayload(&payload); err != nil {
		log.Printf("invalid hello payload: %v", err)
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
		log.Printf("auth failed for %s: %v", payload.ClawID, err)
		s.sendError(conn, env.ID, "authentication failed: "+err.Error())
		conn.Close()
		return
	}

	// Verify claw_id matches token
	if claims.ClawID != payload.ClawID {
		log.Printf("claw_id mismatch: token=%s, payload=%s", claims.ClawID, payload.ClawID)
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

	log.Printf("client authenticated: %s", payload.ClawID)
}

// handleAckMessage routes ack messages to the dispatcher
func (s *Server) handleAckMessage(conn *Connection, env *protocol.Envelope) {
	var payload protocol.AckPayload
	if err := env.UnmarshalPayload(&payload); err != nil {
		log.Printf("invalid ack payload: %v", err)
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
		log.Printf("no pending command for ack %s", refID)
	}
}

// sendError sends an error ack to a connection
func (s *Server) sendError(conn *Connection, refID string, msg string) {
	ack := protocol.AckPayload{Status: "error", Error: msg}
	env, _ := protocol.New(protocol.TypeAck, ack)
	conn.Send(env)
}
