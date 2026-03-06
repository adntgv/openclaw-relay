package server

import (
	"sync"
	"time"
)

// Client represents a connected WebSocket client
type Client struct {
	ClawID       string
	Capabilities []string
	Conn         *Connection
	ConnectedAt  time.Time
}

// Hub manages all connected clients
type Hub struct {
	mu      sync.RWMutex
	clients map[string]*Client // indexed by claw_id
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]*Client),
	}
}

// Register adds a client to the hub
func (h *Hub) Register(clawID string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[clawID] = client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(clawID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, clawID)
}

// Get retrieves a client by claw_id
func (h *Hub) Get(clawID string) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	client, ok := h.clients[clawID]
	return client, ok
}

// List returns all connected clients
func (h *Hub) List() []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := make([]*Client, 0, len(h.clients))
	for _, client := range h.clients {
		clients = append(clients, client)
	}
	return clients
}

// Count returns the number of connected clients
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
