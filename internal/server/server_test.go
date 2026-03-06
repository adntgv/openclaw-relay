package server

import (
	"testing"
	"time"
)

func TestHub(t *testing.T) {
	hub := NewHub()

	// Test initial state
	if hub.Count() != 0 {
		t.Errorf("NewHub() count = %v, want 0", hub.Count())
	}

	// Test Register
	client1 := &Client{
		ClawID:       "test-1",
		Capabilities: []string{"shell"},
		ConnectedAt:  time.Now(),
	}
	hub.Register("test-1", client1)

	if hub.Count() != 1 {
		t.Errorf("After Register count = %v, want 1", hub.Count())
	}

	// Test Get
	got, ok := hub.Get("test-1")
	if !ok {
		t.Error("Get() returned false for existing client")
	}
	if got.ClawID != "test-1" {
		t.Errorf("Get() ClawID = %v, want test-1", got.ClawID)
	}

	// Test Get non-existent
	_, ok = hub.Get("non-existent")
	if ok {
		t.Error("Get() returned true for non-existent client")
	}

	// Test Register multiple
	client2 := &Client{
		ClawID:       "test-2",
		Capabilities: []string{"browser"},
		ConnectedAt:  time.Now(),
	}
	hub.Register("test-2", client2)

	if hub.Count() != 2 {
		t.Errorf("After second Register count = %v, want 2", hub.Count())
	}

	// Test List
	clients := hub.List()
	if len(clients) != 2 {
		t.Errorf("List() length = %v, want 2", len(clients))
	}

	// Test Unregister
	hub.Unregister("test-1")
	if hub.Count() != 1 {
		t.Errorf("After Unregister count = %v, want 1", hub.Count())
	}

	_, ok = hub.Get("test-1")
	if ok {
		t.Error("Get() returned true for unregistered client")
	}

	// Verify test-2 still exists
	got, ok = hub.Get("test-2")
	if !ok {
		t.Error("Get() returned false for remaining client")
	}
	if got.ClawID != "test-2" {
		t.Errorf("Get() ClawID = %v, want test-2", got.ClawID)
	}
}

func TestHubConcurrency(t *testing.T) {
	hub := NewHub()

	// Test concurrent Register/Unregister
	done := make(chan bool)

	// Goroutine 1: Register clients
	go func() {
		for i := 0; i < 100; i++ {
			client := &Client{
				ClawID:       string(rune(i)),
				ConnectedAt:  time.Now(),
			}
			hub.Register(string(rune(i)), client)
		}
		done <- true
	}()

	// Goroutine 2: Unregister clients
	go func() {
		for i := 0; i < 100; i++ {
			hub.Unregister(string(rune(i)))
		}
		done <- true
	}()

	// Goroutine 3: List clients
	go func() {
		for i := 0; i < 100; i++ {
			hub.List()
		}
		done <- true
	}()

	// Wait for all goroutines
	<-done
	<-done
	<-done

	// No panic = success
}
