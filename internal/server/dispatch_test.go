package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/adntgv/openclaw-relay/internal/protocol"
	"github.com/gorilla/websocket"
)

func TestFullCommandDispatch(t *testing.T) {
	srv := newTestServer()

	// Start test HTTP server
	ts := httptest.NewServer(srv.httpServer.Handler)
	defer ts.Close()

	// Issue a token
	tok, _ := srv.tokenManager.Issue("test-claw", []string{"shell"}, 1)

	// Connect WebSocket client
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer ws.Close()

	// Send hello
	hello, _ := protocol.New(protocol.TypeHello, protocol.HelloPayload{
		Token:        tok,
		ClawID:       "test-claw",
		Capabilities: []string{"shell"},
		Version:      "1.0.0",
	})
	helloBytes, _ := hello.Marshal()
	ws.WriteMessage(websocket.TextMessage, helloBytes)

	// Read ack
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("read hello ack: %v", err)
	}
	ackEnv, _ := protocol.Unmarshal(msg)
	if ackEnv.Type != protocol.TypeAck {
		t.Fatalf("expected ack, got %s", ackEnv.Type)
	}

	// Give hub time to register
	time.Sleep(50 * time.Millisecond)

	// Dispatch command via admin API in background
	done := make(chan *http.Response, 1)
	go func() {
		body := `{"claw_id":"test-claw","cmd":"shell.exec","args":{"command":"echo hello"}}`
		req, _ := http.NewRequest("POST", ts.URL+"/command", strings.NewReader(body))
		req.Header.Set("X-Admin-Token", "test-admin")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("command request: %v", err)
			return
		}
		done <- resp
	}()

	// Client reads the command
	_, cmdMsg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("read command: %v", err)
	}
	cmdEnv, err := protocol.Unmarshal(cmdMsg)
	if err != nil {
		t.Fatalf("unmarshal command: %v", err)
	}
	if cmdEnv.Type != protocol.TypeCommand {
		t.Fatalf("expected command, got %s", cmdEnv.Type)
	}

	// Client sends ack with ref_id
	ackPayload := protocol.AckPayload{
		RefID:  cmdEnv.ID,
		Status: "ok",
		Result: map[string]interface{}{"output": "hello"},
	}
	ackReply, _ := protocol.New(protocol.TypeAck, ackPayload)
	ackBytes, _ := ackReply.Marshal()
	ws.WriteMessage(websocket.TextMessage, ackBytes)

	// Check HTTP response
	select {
	case resp := <-done:
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("command response status = %d", resp.StatusCode)
		}
		var result AckResult
		json.NewDecoder(resp.Body).Decode(&result)
		if result.Status != "ok" {
			t.Fatalf("result status = %s", result.Status)
		}
		if result.Result["output"] != "hello" {
			t.Fatalf("result output = %v", result.Result["output"])
		}
	case <-time.After(5 * time.Second):
		t.Fatal("command response timeout")
	}
}

func TestCommandTimeout(t *testing.T) {
	// Test with a very short timeout by using the dispatcher directly
	d := NewDispatcher()
	pc := d.Register("cmd-1", "claw-1", "test")

	go func() {
		// Don't resolve - let it timeout
		time.Sleep(2 * time.Second)
		d.Remove("cmd-1")
	}()

	select {
	case <-pc.Response:
		t.Fatal("should not have received response")
	case <-time.After(100 * time.Millisecond):
		// Expected timeout
	}
}
