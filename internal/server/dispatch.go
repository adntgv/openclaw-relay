package server

import (
	"sync"
	"time"
)

const commandTimeout = 30 * time.Second

// PendingCommand represents a command waiting for an ack
type PendingCommand struct {
	ID       string
	ClawID   string
	Cmd      string
	Response chan *AckResult
}

// AckResult contains the response from a command execution
type AckResult struct {
	Status string                 `json:"status"`
	Result map[string]interface{} `json:"result,omitempty"`
	Error  string                 `json:"error,omitempty"`
}

// Dispatcher manages pending commands and their response channels
type Dispatcher struct {
	mu       sync.RWMutex
	pending  map[string]*PendingCommand // indexed by envelope ID
}

// NewDispatcher creates a new Dispatcher
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		pending: make(map[string]*PendingCommand),
	}
}

// Register creates a pending command and returns it
func (d *Dispatcher) Register(id, clawID, cmd string) *PendingCommand {
	pc := &PendingCommand{
		ID:       id,
		ClawID:   clawID,
		Cmd:      cmd,
		Response: make(chan *AckResult, 1),
	}

	d.mu.Lock()
	d.pending[id] = pc
	d.mu.Unlock()

	return pc
}

// Resolve delivers an ack result for a pending command
func (d *Dispatcher) Resolve(id string, result *AckResult) bool {
	d.mu.Lock()
	pc, ok := d.pending[id]
	if ok {
		delete(d.pending, id)
	}
	d.mu.Unlock()

	if !ok {
		return false
	}

	select {
	case pc.Response <- result:
	default:
	}
	return true
}

// Remove cleans up a pending command without resolving it
func (d *Dispatcher) Remove(id string) {
	d.mu.Lock()
	delete(d.pending, id)
	d.mu.Unlock()
}
