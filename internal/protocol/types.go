package protocol

// Message types
const (
	TypeHello   = "hello"
	TypeCommand = "command"
	TypeAck     = "ack"
	TypeEvent   = "event"
	TypePing    = "ping"
	TypePong    = "pong"
)

// HelloPayload is sent by clients during initial connection
type HelloPayload struct {
	Token        string   `json:"token"`
	ClawID       string   `json:"claw_id"`
	Capabilities []string `json:"capabilities"`
	Version      string   `json:"version"`
}

// CommandPayload contains command details to execute
type CommandPayload struct {
	Cmd  string                 `json:"cmd"`
	Args map[string]interface{} `json:"args,omitempty"`
}

// AckPayload contains command execution result
type AckPayload struct {
	RefID  string                 `json:"ref_id,omitempty"` // references the command envelope ID
	Status string                 `json:"status"`           // "ok" or "error"
	Result map[string]interface{} `json:"result,omitempty"`
	Error  string                 `json:"error,omitempty"`
}
