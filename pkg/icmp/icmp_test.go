package icmp

import (
	"bytes"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
		wantType Type
		wantCode Code
	}{
		{
			name: "valid echo request",
			data: []byte{
				0x08, 0x00, 0x00, 0x00, // Type (8), Code (0), Checksum (will be recalculated)
				0x12, 0x34, 0x00, 0x01, // ID, Sequence
				0x48, 0x65, 0x6c, 0x6c, 0x6f, // "Hello"
			},
			wantErr:  false,
			wantType: TypeEchoRequest,
			wantCode: 0,
		},
		{
			name: "valid echo reply",
			data: []byte{
				0x00, 0x00, 0x00, 0x00,
				0x12, 0x34, 0x00, 0x01,
				0x48, 0x65, 0x6c, 0x6c, 0x6f,
			},
			wantErr:  false,
			wantType: TypeEchoReply,
			wantCode: 0,
		},
		{
			name:    "too short",
			data:    []byte{0x08, 0x00, 0x00},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := Parse(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if msg.Type != tt.wantType {
					t.Errorf("Type = %v, want %v", msg.Type, tt.wantType)
				}
				if msg.Code != tt.wantCode {
					t.Errorf("Code = %v, want %v", msg.Code, tt.wantCode)
				}
			}
		})
	}
}

func TestMessage_Serialize(t *testing.T) {
	msg := NewEchoRequest(0x1234, 1, []byte("Hello, World!"))

	data, err := msg.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	if len(data) < MinHeaderLength {
		t.Errorf("Serialized message too short: %d bytes", len(data))
	}

	// Parse it back
	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if parsed.Type != TypeEchoRequest {
		t.Errorf("Type = %v, want %v", parsed.Type, TypeEchoRequest)
	}
	if parsed.ID != 0x1234 {
		t.Errorf("ID = 0x%04x, want 0x1234", parsed.ID)
	}
	if parsed.Sequence != 1 {
		t.Errorf("Sequence = %d, want 1", parsed.Sequence)
	}
	if !bytes.Equal(parsed.Data, []byte("Hello, World!")) {
		t.Errorf("Data = %v, want %v", parsed.Data, []byte("Hello, World!"))
	}
}

func TestMessage_VerifyChecksum(t *testing.T) {
	msg := NewEchoRequest(0x1234, 1, []byte("test"))

	data, err := msg.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if !parsed.VerifyChecksum() {
		t.Error("VerifyChecksum() = false, want true")
	}

	// Corrupt checksum
	parsed.Checksum = 0xFFFF
	if parsed.VerifyChecksum() {
		t.Error("VerifyChecksum() = true for corrupted checksum, want false")
	}
}

func TestNewEchoRequest(t *testing.T) {
	id := uint16(0x5678)
	seq := uint16(42)
	data := []byte("ping data")

	msg := NewEchoRequest(id, seq, data)

	if msg.Type != TypeEchoRequest {
		t.Errorf("Type = %v, want %v", msg.Type, TypeEchoRequest)
	}
	if msg.Code != 0 {
		t.Errorf("Code = %v, want 0", msg.Code)
	}
	if msg.ID != id {
		t.Errorf("ID = %v, want %v", msg.ID, id)
	}
	if msg.Sequence != seq {
		t.Errorf("Sequence = %v, want %v", msg.Sequence, seq)
	}
	if !bytes.Equal(msg.Data, data) {
		t.Errorf("Data = %v, want %v", msg.Data, data)
	}
}

func TestNewEchoReply(t *testing.T) {
	id := uint16(0x5678)
	seq := uint16(42)
	data := []byte("pong data")

	msg := NewEchoReply(id, seq, data)

	if msg.Type != TypeEchoReply {
		t.Errorf("Type = %v, want %v", msg.Type, TypeEchoReply)
	}
	if msg.Code != 0 {
		t.Errorf("Code = %v, want 0", msg.Code)
	}
	if msg.ID != id {
		t.Errorf("ID = %v, want %v", msg.ID, id)
	}
	if msg.Sequence != seq {
		t.Errorf("Sequence = %v, want %v", msg.Sequence, seq)
	}
	if !bytes.Equal(msg.Data, data) {
		t.Errorf("Data = %v, want %v", msg.Data, data)
	}
}

func TestMessage_IsEchoRequest(t *testing.T) {
	msg := NewEchoRequest(1, 1, nil)
	if !msg.IsEchoRequest() {
		t.Error("IsEchoRequest() = false, want true")
	}

	msg.Type = TypeEchoReply
	if msg.IsEchoRequest() {
		t.Error("IsEchoRequest() = true for echo reply, want false")
	}
}

func TestMessage_IsEchoReply(t *testing.T) {
	msg := NewEchoReply(1, 1, nil)
	if !msg.IsEchoReply() {
		t.Error("IsEchoReply() = false, want true")
	}

	msg.Type = TypeEchoRequest
	if msg.IsEchoReply() {
		t.Error("IsEchoReply() = true for echo request, want false")
	}
}

func TestMessage_IsError(t *testing.T) {
	tests := []struct {
		name string
		typ  Type
		want bool
	}{
		{"destination unreachable", TypeDestinationUnreachable, true},
		{"time exceeded", TypeTimeExceeded, true},
		{"echo request", TypeEchoRequest, false},
		{"echo reply", TypeEchoReply, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &Message{Type: tt.typ}
			if got := msg.IsError(); got != tt.want {
				t.Errorf("IsError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestType_String(t *testing.T) {
	tests := []struct {
		typ  Type
		want string
	}{
		{TypeEchoRequest, "EchoRequest"},
		{TypeEchoReply, "EchoReply"},
		{TypeDestinationUnreachable, "DestinationUnreachable"},
		{TypeTimeExceeded, "TimeExceeded"},
		{Type(99), "Unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.typ.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewDestinationUnreachable(t *testing.T) {
	data := []byte("original packet data")
	msg := NewDestinationUnreachable(CodeHostUnreachable, data)

	if msg.Type != TypeDestinationUnreachable {
		t.Errorf("Type = %v, want %v", msg.Type, TypeDestinationUnreachable)
	}
	if msg.Code != CodeHostUnreachable {
		t.Errorf("Code = %v, want %v", msg.Code, CodeHostUnreachable)
	}
	if !bytes.Equal(msg.Data, data) {
		t.Errorf("Data = %v, want %v", msg.Data, data)
	}
}

func TestNewTimeExceeded(t *testing.T) {
	data := []byte("original packet data")
	msg := NewTimeExceeded(CodeTTLExceeded, data)

	if msg.Type != TypeTimeExceeded {
		t.Errorf("Type = %v, want %v", msg.Type, TypeTimeExceeded)
	}
	if msg.Code != CodeTTLExceeded {
		t.Errorf("Code = %v, want %v", msg.Code, CodeTTLExceeded)
	}
	if !bytes.Equal(msg.Data, data) {
		t.Errorf("Data = %v, want %v", msg.Data, data)
	}
}

func BenchmarkParse(b *testing.B) {
	data := []byte{
		0x08, 0x00, 0x00, 0x00,
		0x12, 0x34, 0x00, 0x01,
		0x48, 0x65, 0x6c, 0x6c, 0x6f,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(data)
	}
}

func BenchmarkSerialize(b *testing.B) {
	msg := NewEchoRequest(0x1234, 1, []byte("Hello, World!"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = msg.Serialize()
	}
}
