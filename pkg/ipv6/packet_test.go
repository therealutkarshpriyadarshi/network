package ipv6

import (
	"bytes"
	"testing"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "packet too short",
			data:    make([]byte, 20),
			wantErr: true,
		},
		{
			name: "valid packet",
			data: []byte{
				0x60, 0x00, 0x00, 0x00, // Version=6, TC=0, Flow=0
				0x00, 0x08, // PayloadLen=8
				0x11,       // NextHeader=UDP
				0x40,       // HopLimit=64
				// Source address (16 bytes)
				0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
				// Destination address (16 bytes)
				0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
				// Payload (8 bytes)
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			},
			wantErr: false,
		},
		{
			name: "invalid version",
			data: []byte{
				0x40, 0x00, 0x00, 0x00, // Version=4 (wrong)
				0x00, 0x08,
				0x11,
				0x40,
				0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
				0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkt, err := Parse(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && pkt == nil {
				t.Error("Parse() returned nil packet without error")
			}
			if !tt.wantErr {
				if pkt.Version != IPv6Version {
					t.Errorf("Parse() version = %d, want %d", pkt.Version, IPv6Version)
				}
			}
		})
	}
}

func TestSerialize(t *testing.T) {
	src := common.IPv6Address{
		0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
	}
	dst := common.IPv6Address{
		0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
	}

	tests := []struct {
		name    string
		packet  *Packet
		wantErr bool
	}{
		{
			name: "valid packet",
			packet: &Packet{
				Version:      IPv6Version,
				TrafficClass: 0,
				FlowLabel:    0,
				NextHeader:   common.ProtocolUDP,
				HopLimit:     64,
				Source:       src,
				Destination:  dst,
				Payload:      []byte{1, 2, 3, 4},
			},
			wantErr: false,
		},
		{
			name: "packet with extension headers",
			packet: &Packet{
				Version:      IPv6Version,
				TrafficClass: 0,
				FlowLabel:    0,
				NextHeader:   common.ProtocolUDP,
				HopLimit:     64,
				Source:       src,
				Destination:  dst,
				ExtHeaders: []ExtensionHeader{
					{NextHeader: common.ProtocolUDP, Data: []byte{1, 2, 3, 4}},
				},
				Payload: []byte{5, 6, 7, 8},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.packet.Serialize()
			if (err != nil) != tt.wantErr {
				t.Errorf("Serialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(data) < HeaderLength {
					t.Errorf("Serialize() produced too short data: %d bytes", len(data))
				}
			}
		})
	}
}

func TestParseSerializeRoundTrip(t *testing.T) {
	src := common.IPv6Address{
		0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
	}
	dst := common.IPv6Address{
		0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
	}

	original := &Packet{
		Version:      IPv6Version,
		TrafficClass: 0,
		FlowLabel:    0,
		NextHeader:   common.ProtocolICMPv6,
		HopLimit:     64,
		Source:       src,
		Destination:  dst,
		Payload:      []byte{1, 2, 3, 4, 5, 6, 7, 8},
	}

	// Serialize
	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Parse
	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Compare
	if parsed.Version != original.Version {
		t.Errorf("Version mismatch: got %d, want %d", parsed.Version, original.Version)
	}
	if parsed.NextHeader != original.NextHeader {
		t.Errorf("NextHeader mismatch: got %d, want %d", parsed.NextHeader, original.NextHeader)
	}
	if parsed.HopLimit != original.HopLimit {
		t.Errorf("HopLimit mismatch: got %d, want %d", parsed.HopLimit, original.HopLimit)
	}
	if !bytes.Equal(parsed.Payload, original.Payload) {
		t.Errorf("Payload mismatch: got %v, want %v", parsed.Payload, original.Payload)
	}
}

func TestDecrementHopLimit(t *testing.T) {
	tests := []struct {
		name        string
		hopLimit    uint8
		wantResult  bool
		wantHopLim  uint8
	}{
		{
			name:        "normal decrement",
			hopLimit:    64,
			wantResult:  true,
			wantHopLim:  63,
		},
		{
			name:        "decrement to zero",
			hopLimit:    1,
			wantResult:  false,
			wantHopLim:  0,
		},
		{
			name:        "already zero",
			hopLimit:    0,
			wantResult:  false,
			wantHopLim:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkt := &Packet{HopLimit: tt.hopLimit}
			result := pkt.DecrementHopLimit()
			if result != tt.wantResult {
				t.Errorf("DecrementHopLimit() = %v, want %v", result, tt.wantResult)
			}
			if pkt.HopLimit != tt.wantHopLim {
				t.Errorf("HopLimit = %d, want %d", pkt.HopLimit, tt.wantHopLim)
			}
		})
	}
}

func TestNewPacket(t *testing.T) {
	src := common.IPv6Address{
		0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
	}
	dst := common.IPv6Address{
		0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
	}
	payload := []byte{1, 2, 3, 4}

	pkt := NewPacket(src, dst, common.ProtocolTCP, payload)

	if pkt == nil {
		t.Fatal("NewPacket() returned nil")
	}
	if pkt.Version != IPv6Version {
		t.Errorf("Version = %d, want %d", pkt.Version, IPv6Version)
	}
	if pkt.HopLimit != DefaultHopLimit {
		t.Errorf("HopLimit = %d, want %d", pkt.HopLimit, DefaultHopLimit)
	}
	if pkt.NextHeader != common.ProtocolTCP {
		t.Errorf("NextHeader = %d, want %d", pkt.NextHeader, common.ProtocolTCP)
	}
	if !bytes.Equal(pkt.Payload, payload) {
		t.Errorf("Payload = %v, want %v", pkt.Payload, payload)
	}
}

func TestString(t *testing.T) {
	src := common.IPv6Address{
		0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
	}
	dst := common.IPv6Address{
		0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
	}

	pkt := NewPacket(src, dst, common.ProtocolTCP, []byte{1, 2, 3, 4})
	str := pkt.String()

	if str == "" {
		t.Error("String() returned empty string")
	}
}

func TestTrafficClassAndFlowLabel(t *testing.T) {
	src := common.IPv6Address{
		0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
	}
	dst := common.IPv6Address{
		0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
	}

	pkt := &Packet{
		Version:      IPv6Version,
		TrafficClass: 0xAB,
		FlowLabel:    0x12345,
		NextHeader:   common.ProtocolUDP,
		HopLimit:     64,
		Source:       src,
		Destination:  dst,
		Payload:      []byte{1, 2, 3, 4},
	}

	data, err := pkt.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if parsed.TrafficClass != pkt.TrafficClass {
		t.Errorf("TrafficClass = %d, want %d", parsed.TrafficClass, pkt.TrafficClass)
	}
	if parsed.FlowLabel != pkt.FlowLabel {
		t.Errorf("FlowLabel = %d, want %d", parsed.FlowLabel, pkt.FlowLabel)
	}
}
