package client

import (
	"encoding/json"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/adntgv/openclaw-relay/internal/config"
	"github.com/adntgv/openclaw-relay/internal/protocol"
	"github.com/gorilla/websocket"
)

const (
	maxBackoff    = 60 * time.Second
	initialDelay  = 1 * time.Second
	noMessageTimeout = 90 * time.Second
)

// Client is the relay WebSocket client
type Client struct {
	cfg     *config.ClientConfig
	handler *Handler
	ws      *websocket.Conn
	mu      sync.Mutex
	done    chan struct{}
}

// New creates a new Client
func New(cfg *config.ClientConfig) *Client {
	handler := NewHandler(
		cfg.AllowedCmds,
		cfg.Shell.AllowedBinaries,
		cfg.Shell.Timeout,
		cfg.HooksDir,
	)
	return &Client{
		cfg:     cfg,
		handler: handler,
		done:    make(chan struct{}),
	}
}

// Run starts the client with reconnect loop
func (c *Client) Run() error {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	attempt := 0
	for {
		select {
		case <-stop:
			slog.Info("shutting down client...")
			c.close()
			return nil
		default:
		}

		err := c.connect()
		if err != nil {
			delay := backoff(attempt)
			slog.Warn("connection failed, retrying", "error", err, "delay", delay)
			time.Sleep(delay)
			attempt++
			continue
		}

		attempt = 0 // Reset on successful connection
		c.readLoop()
		slog.Info("disconnected, reconnecting...")
	}
}

func (c *Client) connect() error {
	ws, _, err := websocket.DefaultDialer.Dial(c.cfg.URL, nil)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.ws = ws
	c.mu.Unlock()

	// Send hello
	hello, err := protocol.New(protocol.TypeHello, protocol.HelloPayload{
		Token:        c.cfg.Token,
		ClawID:       c.cfg.ClawID,
		Capabilities: c.cfg.Capabilities,
		Version:      "1.0.0",
	})
	if err != nil {
		ws.Close()
		return err
	}

	data, _ := hello.Marshal()
	if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
		ws.Close()
		return err
	}

	// Read ack
	ws.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, msg, err := ws.ReadMessage()
	if err != nil {
		ws.Close()
		return err
	}
	ws.SetReadDeadline(time.Time{})

	env, err := protocol.Unmarshal(msg)
	if err != nil || env.Type != protocol.TypeAck {
		ws.Close()
		return err
	}

	var ack protocol.AckPayload
	env.UnmarshalPayload(&ack)
	if ack.Status != "ok" {
		ws.Close()
		return &AuthError{Message: ack.Error}
	}

	slog.Info("connected", "claw_id", c.cfg.ClawID)
	return nil
}

func (c *Client) readLoop() {
	for {
		c.ws.SetReadDeadline(time.Now().Add(noMessageTimeout))
		_, msg, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Warn("read error", "error", err)
			}
			return
		}

		env, err := protocol.Unmarshal(msg)
		if err != nil {
			slog.Warn("invalid message", "error", err)
			continue
		}

		switch env.Type {
		case protocol.TypeCommand:
			go c.handleCommand(env)
		case protocol.TypePing:
			c.sendPong(env)
		}
	}
}

func (c *Client) handleCommand(env *protocol.Envelope) {
	var payload protocol.CommandPayload
	if err := env.UnmarshalPayload(&payload); err != nil {
		c.sendAck(env.ID, "error", nil, "invalid command payload")
		return
	}

	slog.Info("executing command", "cmd", payload.Cmd)

	result, err := c.handler.Execute(payload.Cmd, payload.Args)
	if err != nil {
		c.sendAck(env.ID, "error", nil, err.Error())
		return
	}

	c.sendAck(env.ID, "ok", result, "")
}

func (c *Client) sendAck(refID, status string, result map[string]interface{}, errMsg string) {
	ack := protocol.AckPayload{
		RefID:  refID,
		Status: status,
		Result: result,
		Error:  errMsg,
	}
	env, _ := protocol.New(protocol.TypeAck, ack)
	data, _ := env.Marshal()

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ws != nil {
		c.ws.WriteMessage(websocket.TextMessage, data)
	}
}

func (c *Client) sendPong(ping *protocol.Envelope) {
	pong, _ := protocol.New(protocol.TypePong, json.RawMessage(`{}`))
	data, _ := pong.Marshal()

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ws != nil {
		c.ws.WriteMessage(websocket.TextMessage, data)
	}
}

func (c *Client) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ws != nil {
		c.ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		time.Sleep(time.Second)
		c.ws.Close()
		c.ws = nil
	}
}

func backoff(attempt int) time.Duration {
	if attempt > 6 { // 2^6 = 64s > maxBackoff
		return maxBackoff
	}
	delay := time.Duration(math.Pow(2, float64(attempt))) * initialDelay
	if delay > maxBackoff {
		delay = maxBackoff
	}
	return delay
}

// AuthError represents an authentication failure
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return "auth failed: " + e.Message
}
