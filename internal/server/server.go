package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/adntgv/openclaw-relay/internal/protocol"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// Server is the main relay server
type Server struct {
	hub        *Hub
	httpServer *http.Server
	adminToken string
	startTime  time.Time
}

// Config holds server configuration
type Config struct {
	Host       string
	Port       int
	AdminToken string
}

// New creates a new Server instance
func New(cfg Config) *Server {
	hub := NewHub()

	srv := &Server{
		hub:        hub,
		adminToken: cfg.AdminToken,
		startTime:  time.Now(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/ws", srv.handleWebSocket)

	srv.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler: mux,
	}

	return srv
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
	uptime := time.Since(s.startTime).Seconds()

	response := map[string]interface{}{
		"status":  "ok",
		"clients": s.hub.Count(),
		"uptime":  uptime,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleWebSocket handles WebSocket upgrade and client connection
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}

	conn := NewConnection(ws, s.hub)
	
	// Set message handler for authentication
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
		// Pong received, connection is alive
	case protocol.TypeAck:
		// Command acknowledgment (handled in Phase 3)
	default:
		log.Printf("unknown message type: %s", env.Type)
	}
}

// handleHello processes hello messages and authenticates clients
func (s *Server) handleHello(conn *Connection, env *protocol.Envelope) {
	var payload protocol.HelloPayload
	if err := env.UnmarshalPayload(&payload); err != nil {
		log.Printf("invalid hello payload: %v", err)
		conn.Close()
		return
	}

	// Basic validation (full JWT auth in Phase 2)
	if payload.ClawID == "" {
		log.Printf("hello missing claw_id")
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

	// Send ack
	ackPayload := protocol.AckPayload{
		Status: "ok",
	}
	ackEnv, err := protocol.New(protocol.TypeAck, ackPayload)
	if err != nil {
		log.Printf("failed to create ack: %v", err)
		return
	}

	if err := conn.Send(ackEnv); err != nil {
		log.Printf("failed to send ack: %v", err)
	}

	log.Printf("client connected: %s (capabilities: %v)", payload.ClawID, payload.Capabilities)
}
