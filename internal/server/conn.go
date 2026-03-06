package server

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/adntgv/openclaw-relay/internal/protocol"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = 30 * time.Second

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512KB
)

// Connection wraps a WebSocket connection with read/write goroutines
type Connection struct {
	ws      *websocket.Conn
	send    chan []byte
	hub     *Hub
	clawID  string
	ctx     context.Context
	cancel  context.CancelFunc
	onMessage func(*protocol.Envelope)
	closeOnce sync.Once
}

// NewConnection creates a new Connection instance
func NewConnection(ws *websocket.Conn, hub *Hub) *Connection {
	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		ws:     ws,
		send:   make(chan []byte, 256),
		hub:    hub,
		ctx:    ctx,
		cancel: cancel,
	}
}

// SetClawID sets the client's claw ID after authentication
func (c *Connection) SetClawID(clawID string) {
	c.clawID = clawID
}

// SetMessageHandler sets the callback for incoming messages
func (c *Connection) SetMessageHandler(handler func(*protocol.Envelope)) {
	c.onMessage = handler
}

// Start begins the read and write pumps
func (c *Connection) Start() {
	go c.writePump()
	go c.readPump()
}

// Close closes the connection
func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		c.cancel()
		if c.clawID != "" {
			c.hub.Unregister(c.clawID)
		}
		close(c.send)
	})
}

// Send queues a message for sending
func (c *Connection) Send(env *protocol.Envelope) error {
	data, err := env.Marshal()
	if err != nil {
		return err
	}

	select {
	case c.send <- data:
		return nil
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Connection) readPump() {
	defer c.Close()

	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		_, message, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket error: %v", err)
			}
			return
		}

		env, err := protocol.Unmarshal(message)
		if err != nil {
			log.Printf("invalid envelope: %v", err)
			continue
		}

		if c.onMessage != nil {
			c.onMessage(env)
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel closed
				c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.ws.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.ctx.Done():
			return
		}
	}
}

// SendJSON is a helper to send JSON-encoded messages
func (c *Connection) SendJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	select {
	case c.send <- data:
		return nil
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}
