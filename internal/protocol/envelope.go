package protocol

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Envelope wraps all WebSocket messages
type Envelope struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Timestamp int64           `json:"ts"`
	Payload   json.RawMessage `json:"payload"`
	Signature *string         `json:"signature"`
}

// New creates a new envelope with the given type and payload
func New(msgType string, payload interface{}) (*Envelope, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &Envelope{
		ID:        uuid.New().String(),
		Type:      msgType,
		Timestamp: time.Now().Unix(),
		Payload:   payloadBytes,
		Signature: nil,
	}, nil
}

// Validate checks if the envelope has all required fields
func (e *Envelope) Validate() error {
	if e.ID == "" {
		return errors.New("envelope ID is required")
	}
	if e.Type == "" {
		return errors.New("envelope type is required")
	}
	if e.Timestamp == 0 {
		return errors.New("envelope timestamp is required")
	}
	
	// Validate type is one of the known types
	validTypes := map[string]bool{
		TypeHello:   true,
		TypeCommand: true,
		TypeAck:     true,
		TypeEvent:   true,
		TypePing:    true,
		TypePong:    true,
	}
	
	if !validTypes[e.Type] {
		return errors.New("invalid envelope type: " + e.Type)
	}
	
	return nil
}

// UnmarshalPayload unmarshals the payload into the given interface
func (e *Envelope) UnmarshalPayload(v interface{}) error {
	return json.Unmarshal(e.Payload, v)
}

// Marshal converts the envelope to JSON bytes
func (e *Envelope) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// Unmarshal parses JSON bytes into an envelope
func Unmarshal(data []byte) (*Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	
	if err := env.Validate(); err != nil {
		return nil, err
	}
	
	return &env, nil
}
