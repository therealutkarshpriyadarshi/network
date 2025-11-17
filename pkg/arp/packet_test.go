package arp

import (
	"bytes"
	"testing"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

func TestPacketParse(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		wantErr     bool
		errContains string
		validate    func(*testing.T, *Packet)
	}{
		{
			name: "valid ARP request",
			data: []byte{
				0x00, 0x01, // Hardware type: Ethernet
				0x08, 0x00, // Protocol type: IPv4
				0x06,       // Hardware address length
				0x04,       // Protocol address length
				0x00, 0x01, // Operation: Request
				0x00, 0x11, 0x22, 0x33, 0x44, 0x55, // Sender MAC
				0xc0, 0xa8, 0x01, 0x01, // Sender IP: 192.168.1.1
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Target MAC
				0xc0, 0xa8, 0x01, 0x02, // Target IP: 192.168.1.2
			},
			wantErr: false,
			validate: func(t *testing.T, p *Packet) {
				if p.Operation != OperationRequest {
					t.Errorf("Operation = %v, want %v", p.Operation, OperationRequest)
				}
				expectedSenderMAC := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
				if p.SenderMAC != expectedSenderMAC {
					t.Errorf("SenderMAC = %v, want %v", p.SenderMAC, expectedSenderMAC)
				}
				expectedSenderIP := common.IPv4Address{192, 168, 1, 1}
				if p.SenderIP != expectedSenderIP {
					t.Errorf("SenderIP = %v, want %v", p.SenderIP, expectedSenderIP)
				}
				expectedTargetIP := common.IPv4Address{192, 168, 1, 2}
				if p.TargetIP != expectedTargetIP {
					t.Errorf("TargetIP = %v, want %v", p.TargetIP, expectedTargetIP)
				}
			},
		},
		{
			name: "valid ARP reply",
			data: []byte{
				0x00, 0x01, // Hardware type: Ethernet
				0x08, 0x00, // Protocol type: IPv4
				0x06,       // Hardware address length
				0x04,       // Protocol address length
				0x00, 0x02, // Operation: Reply
				0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, // Sender MAC
				0xc0, 0xa8, 0x01, 0x0a, // Sender IP: 192.168.1.10
				0x00, 0x11, 0x22, 0x33, 0x44, 0x55, // Target MAC
				0xc0, 0xa8, 0x01, 0x14, // Target IP: 192.168.1.20
			},
			wantErr: false,
			validate: func(t *testing.T, p *Packet) {
				if p.Operation != OperationReply {
					t.Errorf("Operation = %v, want %v", p.Operation, OperationReply)
				}
				if !p.IsReply() {
					t.Error("IsReply() = false, want true")
				}
			},
		},
		{
			name:        "packet too short",
			data:        []byte{0x00, 0x01, 0x08, 0x00},
			wantErr:     true,
			errContains: "too short",
		},
		{
			name: "invalid hardware type",
			data: []byte{
				0x00, 0x06, // Hardware type: Invalid
				0x08, 0x00, // Protocol type: IPv4
				0x06,       // Hardware address length
				0x04,       // Protocol address length
				0x00, 0x01, // Operation: Request
				0x00, 0x11, 0x22, 0x33, 0x44, 0x55, // Sender MAC
				0xc0, 0xa8, 0x01, 0x01, // Sender IP
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Target MAC
				0xc0, 0xa8, 0x01, 0x02, // Target IP
			},
			wantErr:     true,
			errContains: "hardware type",
		},
		{
			name: "invalid protocol type",
			data: []byte{
				0x00, 0x01, // Hardware type: Ethernet
				0x08, 0x06, // Protocol type: Invalid
				0x06,       // Hardware address length
				0x04,       // Protocol address length
				0x00, 0x01, // Operation: Request
				0x00, 0x11, 0x22, 0x33, 0x44, 0x55, // Sender MAC
				0xc0, 0xa8, 0x01, 0x01, // Sender IP
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Target MAC
				0xc0, 0xa8, 0x01, 0x02, // Target IP
			},
			wantErr:     true,
			errContains: "protocol type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet, err := Parse(tt.data)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse() error = nil, want error containing %q", tt.errContains)
				} else if tt.errContains != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errContains)) {
					t.Errorf("Parse() error = %q, want error containing %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
				return
			}
			if tt.validate != nil {
				tt.validate(t, packet)
			}
		})
	}
}

func TestPacketSerialize(t *testing.T) {
	senderMAC := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	senderIP := common.IPv4Address{192, 168, 1, 1}
	targetMAC := common.MACAddress{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	targetIP := common.IPv4Address{192, 168, 1, 2}

	packet := &Packet{
		HardwareType:   HardwareTypeEthernet,
		ProtocolType:   ProtocolTypeIPv4,
		HardwareLength: 6,
		ProtocolLength: 4,
		Operation:      OperationRequest,
		SenderMAC:      senderMAC,
		SenderIP:       senderIP,
		TargetMAC:      targetMAC,
		TargetIP:       targetIP,
	}

	data := packet.Serialize()

	// Verify size
	if len(data) != PacketSize {
		t.Errorf("Serialize() returned %d bytes, want %d", len(data), PacketSize)
	}

	// Parse back and verify
	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if parsed.Operation != packet.Operation {
		t.Errorf("Operation = %v, want %v", parsed.Operation, packet.Operation)
	}
	if parsed.SenderMAC != packet.SenderMAC {
		t.Errorf("SenderMAC = %v, want %v", parsed.SenderMAC, packet.SenderMAC)
	}
	if parsed.SenderIP != packet.SenderIP {
		t.Errorf("SenderIP = %v, want %v", parsed.SenderIP, packet.SenderIP)
	}
	if parsed.TargetMAC != packet.TargetMAC {
		t.Errorf("TargetMAC = %v, want %v", parsed.TargetMAC, packet.TargetMAC)
	}
	if parsed.TargetIP != packet.TargetIP {
		t.Errorf("TargetIP = %v, want %v", parsed.TargetIP, packet.TargetIP)
	}
}

func TestNewRequest(t *testing.T) {
	senderMAC := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	senderIP := common.IPv4Address{192, 168, 1, 1}
	targetIP := common.IPv4Address{192, 168, 1, 2}

	packet := NewRequest(senderMAC, senderIP, targetIP)

	if packet.Operation != OperationRequest {
		t.Errorf("Operation = %v, want %v", packet.Operation, OperationRequest)
	}
	if packet.SenderMAC != senderMAC {
		t.Errorf("SenderMAC = %v, want %v", packet.SenderMAC, senderMAC)
	}
	if packet.SenderIP != senderIP {
		t.Errorf("SenderIP = %v, want %v", packet.SenderIP, senderIP)
	}
	if packet.TargetIP != targetIP {
		t.Errorf("TargetIP = %v, want %v", packet.TargetIP, targetIP)
	}
	// Target MAC should be zero for requests
	zeroMAC := common.MACAddress{}
	if packet.TargetMAC != zeroMAC {
		t.Errorf("TargetMAC = %v, want %v (zero)", packet.TargetMAC, zeroMAC)
	}
	if !packet.IsRequest() {
		t.Error("IsRequest() = false, want true")
	}
	if packet.IsReply() {
		t.Error("IsReply() = true, want false")
	}
}

func TestNewReply(t *testing.T) {
	senderMAC := common.MACAddress{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	senderIP := common.IPv4Address{192, 168, 1, 10}
	targetMAC := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	targetIP := common.IPv4Address{192, 168, 1, 20}

	packet := NewReply(senderMAC, senderIP, targetMAC, targetIP)

	if packet.Operation != OperationReply {
		t.Errorf("Operation = %v, want %v", packet.Operation, OperationReply)
	}
	if packet.SenderMAC != senderMAC {
		t.Errorf("SenderMAC = %v, want %v", packet.SenderMAC, senderMAC)
	}
	if packet.SenderIP != senderIP {
		t.Errorf("SenderIP = %v, want %v", packet.SenderIP, senderIP)
	}
	if packet.TargetMAC != targetMAC {
		t.Errorf("TargetMAC = %v, want %v", packet.TargetMAC, targetMAC)
	}
	if packet.TargetIP != targetIP {
		t.Errorf("TargetIP = %v, want %v", packet.TargetIP, targetIP)
	}
	if packet.IsRequest() {
		t.Error("IsRequest() = true, want false")
	}
	if !packet.IsReply() {
		t.Error("IsReply() = false, want true")
	}
}

func TestPacketString(t *testing.T) {
	senderMAC := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	senderIP := common.IPv4Address{192, 168, 1, 1}
	targetIP := common.IPv4Address{192, 168, 1, 2}

	packet := NewRequest(senderMAC, senderIP, targetIP)
	str := packet.String()

	// Verify the string contains key information
	if str == "" {
		t.Error("String() returned empty string")
	}
	// The string representation should include operation type
	if !bytes.Contains([]byte(str), []byte("Request")) {
		t.Errorf("String() = %q, should contain 'Request'", str)
	}
}

func TestOperationString(t *testing.T) {
	tests := []struct {
		op   Operation
		want string
	}{
		{OperationRequest, "Request"},
		{OperationReply, "Reply"},
		{Operation(99), "Unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.op.String(); got != tt.want {
				t.Errorf("Operation.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRoundTrip tests that serializing and parsing produces the same packet.
func TestRoundTrip(t *testing.T) {
	original := &Packet{
		HardwareType:   HardwareTypeEthernet,
		ProtocolType:   ProtocolTypeIPv4,
		HardwareLength: 6,
		ProtocolLength: 4,
		Operation:      OperationReply,
		SenderMAC:      common.MACAddress{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc},
		SenderIP:       common.IPv4Address{10, 0, 0, 1},
		TargetMAC:      common.MACAddress{0xde, 0xad, 0xbe, 0xef, 0x00, 0x00},
		TargetIP:       common.IPv4Address{10, 0, 0, 2},
	}

	// Serialize
	data := original.Serialize()

	// Parse
	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Compare all fields
	if parsed.HardwareType != original.HardwareType {
		t.Errorf("HardwareType = %v, want %v", parsed.HardwareType, original.HardwareType)
	}
	if parsed.ProtocolType != original.ProtocolType {
		t.Errorf("ProtocolType = %v, want %v", parsed.ProtocolType, original.ProtocolType)
	}
	if parsed.HardwareLength != original.HardwareLength {
		t.Errorf("HardwareLength = %v, want %v", parsed.HardwareLength, original.HardwareLength)
	}
	if parsed.ProtocolLength != original.ProtocolLength {
		t.Errorf("ProtocolLength = %v, want %v", parsed.ProtocolLength, original.ProtocolLength)
	}
	if parsed.Operation != original.Operation {
		t.Errorf("Operation = %v, want %v", parsed.Operation, original.Operation)
	}
	if parsed.SenderMAC != original.SenderMAC {
		t.Errorf("SenderMAC = %v, want %v", parsed.SenderMAC, original.SenderMAC)
	}
	if parsed.SenderIP != original.SenderIP {
		t.Errorf("SenderIP = %v, want %v", parsed.SenderIP, original.SenderIP)
	}
	if parsed.TargetMAC != original.TargetMAC {
		t.Errorf("TargetMAC = %v, want %v", parsed.TargetMAC, original.TargetMAC)
	}
	if parsed.TargetIP != original.TargetIP {
		t.Errorf("TargetIP = %v, want %v", parsed.TargetIP, original.TargetIP)
	}
}
