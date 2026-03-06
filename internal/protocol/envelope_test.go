package protocol

import (
	"encoding/json"
	"testing"
)

func TestNew(t *testing.T) {
	payload := HelloPayload{
		Token:        "test-token",
		ClawID:       "test-claw",
		Capabilities: []string{"shell"},
		Version:      "1.0.0",
	}

	env, err := New(TypeHello, payload)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if env.ID == "" {
		t.Error("New() envelope ID is empty")
	}
	if env.Type != TypeHello {
		t.Errorf("New() type = %v, want %v", env.Type, TypeHello)
	}
	if env.Timestamp == 0 {
		t.Error("New() timestamp is 0")
	}
	if env.Signature != nil {
		t.Error("New() signature should be nil")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		env     Envelope
		wantErr bool
	}{
		{
			name: "valid envelope",
			env: Envelope{
				ID:        "test-id",
				Type:      TypeHello,
				Timestamp: 1710000000,
				Payload:   json.RawMessage(`{}`),
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			env: Envelope{
				Type:      TypeHello,
				Timestamp: 1710000000,
				Payload:   json.RawMessage(`{}`),
			},
			wantErr: true,
		},
		{
			name: "missing type",
			env: Envelope{
				ID:        "test-id",
				Timestamp: 1710000000,
				Payload:   json.RawMessage(`{}`),
			},
			wantErr: true,
		},
		{
			name: "missing timestamp",
			env: Envelope{
				ID:      "test-id",
				Type:    TypeHello,
				Payload: json.RawMessage(`{}`),
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			env: Envelope{
				ID:        "test-id",
				Type:      "invalid",
				Timestamp: 1710000000,
				Payload:   json.RawMessage(`{}`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.env.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	payload := HelloPayload{
		Token:        "test-token",
		ClawID:       "test-claw",
		Capabilities: []string{"shell", "browser"},
		Version:      "1.0.0",
	}

	env, err := New(TypeHello, payload)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Marshal
	data, err := env.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Unmarshal
	env2, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if env2.ID != env.ID {
		t.Errorf("Unmarshal() ID = %v, want %v", env2.ID, env.ID)
	}
	if env2.Type != env.Type {
		t.Errorf("Unmarshal() Type = %v, want %v", env2.Type, env.Type)
	}

	// Unmarshal payload
	var gotPayload HelloPayload
	if err := env2.UnmarshalPayload(&gotPayload); err != nil {
		t.Fatalf("UnmarshalPayload() error = %v", err)
	}

	if gotPayload.ClawID != payload.ClawID {
		t.Errorf("UnmarshalPayload() ClawID = %v, want %v", gotPayload.ClawID, payload.ClawID)
	}
	if gotPayload.Token != payload.Token {
		t.Errorf("UnmarshalPayload() Token = %v, want %v", gotPayload.Token, payload.Token)
	}
}

func TestUnmarshalInvalid(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "invalid json",
			data: `{invalid}`,
		},
		{
			name: "missing required fields",
			data: `{"id":"test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Unmarshal([]byte(tt.data))
			if err == nil {
				t.Error("Unmarshal() expected error, got nil")
			}
		})
	}
}
