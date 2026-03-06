package server

import (
	"sync"
	"time"
)

// AuditEntry represents an audit log entry
type AuditEntry struct {
	Timestamp time.Time `json:"ts"`
	Action    string    `json:"action"`
	ClawID    string    `json:"claw_id,omitempty"`
	Detail    string    `json:"detail,omitempty"`
}

// AuditLog is an append-only in-memory audit log
type AuditLog struct {
	mu      sync.RWMutex
	entries []AuditEntry
}

// NewAuditLog creates a new AuditLog
func NewAuditLog() *AuditLog {
	return &AuditLog{}
}

// Log appends an entry
func (a *AuditLog) Log(action, clawID, detail string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, AuditEntry{
		Timestamp: time.Now(),
		Action:    action,
		ClawID:    clawID,
		Detail:    detail,
	})
}

// Query returns entries filtered by optional clawID, limited to limit
func (a *AuditLog) Query(clawID string, limit int) []AuditEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	var result []AuditEntry
	// Iterate in reverse (newest first)
	for i := len(a.entries) - 1; i >= 0 && len(result) < limit; i-- {
		e := a.entries[i]
		if clawID == "" || e.ClawID == clawID {
			result = append(result, e)
		}
	}
	return result
}
