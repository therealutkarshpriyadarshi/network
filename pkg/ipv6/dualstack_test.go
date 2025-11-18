package ipv6

import (
	"testing"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/ip"
)

func TestNewIPv4Address(t *testing.T) {
	addr := common.IPv4Address{192, 168, 1, 1}
	ipAddr := NewIPv4Address(addr)

	if !ipAddr.IsIPv4() {
		t.Error("Expected IPv4 address")
	}
	if ipAddr.IsIPv6() {
		t.Error("Did not expect IPv6 address")
	}
	if ipAddr.IPv4 == nil {
		t.Error("IPv4 address should not be nil")
	}
}

func TestNewIPv6Address(t *testing.T) {
	addr := common.IPv6Address{
		0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
	}
	ipAddr := NewIPv6Address(addr)

	if ipAddr.IsIPv4() {
		t.Error("Did not expect IPv4 address")
	}
	if !ipAddr.IsIPv6() {
		t.Error("Expected IPv6 address")
	}
	if ipAddr.IPv6 == nil {
		t.Error("IPv6 address should not be nil")
	}
}

func TestIPAddressString(t *testing.T) {
	tests := []struct {
		name     string
		ipAddr   IPAddress
		wantNonEmpty bool
	}{
		{
			name:     "IPv4 address",
			ipAddr:   NewIPv4Address(common.IPv4Address{192, 168, 1, 1}),
			wantNonEmpty: true,
		},
		{
			name: "IPv6 address",
			ipAddr: NewIPv6Address(common.IPv6Address{
				0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
			}),
			wantNonEmpty: true,
		},
		{
			name:     "Invalid address",
			ipAddr:   IPAddress{},
			wantNonEmpty: true, // Should return "<invalid>"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.ipAddr.String()
			if tt.wantNonEmpty && str == "" {
				t.Error("String() returned empty string")
			}
		})
	}
}

func TestNewIPv4Packet(t *testing.T) {
	ipv4Pkt := &ip.Packet{
		Version:     4,
		Protocol:    common.ProtocolTCP,
		Source:      common.IPv4Address{192, 168, 1, 1},
		Destination: common.IPv4Address{192, 168, 1, 2},
		Payload:     []byte{1, 2, 3, 4},
	}

	dsPkt := NewIPv4Packet(ipv4Pkt)

	if !dsPkt.IsIPv4() {
		t.Error("Expected IPv4 packet")
	}
	if dsPkt.IsIPv6() {
		t.Error("Did not expect IPv6 packet")
	}
	if dsPkt.IPv4 == nil {
		t.Error("IPv4 packet should not be nil")
	}
}

func TestNewIPv6Packet(t *testing.T) {
	src := common.IPv6Address{
		0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
	}
	dst := common.IPv6Address{
		0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
	}

	ipv6Pkt := NewPacket(src, dst, common.ProtocolTCP, []byte{1, 2, 3, 4})
	dsPkt := NewIPv6Packet(ipv6Pkt)

	if dsPkt.IsIPv4() {
		t.Error("Did not expect IPv4 packet")
	}
	if !dsPkt.IsIPv6() {
		t.Error("Expected IPv6 packet")
	}
	if dsPkt.IPv6 == nil {
		t.Error("IPv6 packet should not be nil")
	}
}

func TestDualStackPacketSerialize(t *testing.T) {
	tests := []struct {
		name    string
		packet  *DualStackPacket
		wantErr bool
	}{
		{
			name: "IPv4 packet",
			packet: NewIPv4Packet(&ip.Packet{
				Version:     4,
				IHL:         5,
				Protocol:    common.ProtocolTCP,
				Source:      common.IPv4Address{192, 168, 1, 1},
				Destination: common.IPv4Address{192, 168, 1, 2},
				Payload:     []byte{1, 2, 3, 4},
			}),
			wantErr: false,
		},
		{
			name: "IPv6 packet",
			packet: NewIPv6Packet(NewPacket(
				common.IPv6Address{0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
				common.IPv6Address{0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
				common.ProtocolTCP,
				[]byte{1, 2, 3, 4},
			)),
			wantErr: false,
		},
		{
			name:    "Invalid packet",
			packet:  &DualStackPacket{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.packet.Serialize()
			if (err != nil) != tt.wantErr {
				t.Errorf("Serialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(data) == 0 {
				t.Error("Serialize() returned empty data")
			}
		})
	}
}

func TestDualStackPacketGetProtocol(t *testing.T) {
	tests := []struct {
		name     string
		packet   *DualStackPacket
		wantProto common.Protocol
	}{
		{
			name: "IPv4 packet",
			packet: NewIPv4Packet(&ip.Packet{
				Protocol: common.ProtocolTCP,
			}),
			wantProto: common.ProtocolTCP,
		},
		{
			name: "IPv6 packet",
			packet: NewIPv6Packet(&Packet{
				NextHeader: common.ProtocolUDP,
			}),
			wantProto: common.ProtocolUDP,
		},
		{
			name:     "Invalid packet",
			packet:   &DualStackPacket{},
			wantProto: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proto := tt.packet.GetProtocol()
			if proto != tt.wantProto {
				t.Errorf("GetProtocol() = %v, want %v", proto, tt.wantProto)
			}
		})
	}
}

func TestDualStackPacketGetPayload(t *testing.T) {
	payload := []byte{1, 2, 3, 4}

	tests := []struct {
		name    string
		packet  *DualStackPacket
		wantNil bool
	}{
		{
			name: "IPv4 packet",
			packet: NewIPv4Packet(&ip.Packet{
				Payload: payload,
			}),
			wantNil: false,
		},
		{
			name: "IPv6 packet",
			packet: NewIPv6Packet(&Packet{
				Payload: payload,
			}),
			wantNil: false,
		},
		{
			name:    "Invalid packet",
			packet:  &DualStackPacket{},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.packet.GetPayload()
			if (p == nil) != tt.wantNil {
				t.Errorf("GetPayload() nil = %v, wantNil %v", p == nil, tt.wantNil)
			}
		})
	}
}

func TestDualStackPacketString(t *testing.T) {
	tests := []struct {
		name   string
		packet *DualStackPacket
	}{
		{
			name: "IPv4 packet",
			packet: NewIPv4Packet(&ip.Packet{
				Version:     4,
				Source:      common.IPv4Address{192, 168, 1, 1},
				Destination: common.IPv4Address{192, 168, 1, 2},
			}),
		},
		{
			name: "IPv6 packet",
			packet: NewIPv6Packet(NewPacket(
				common.IPv6Address{0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
				common.IPv6Address{0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
				common.ProtocolTCP,
				[]byte{},
			)),
		},
		{
			name:   "Invalid packet",
			packet: &DualStackPacket{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.packet.String()
			if str == "" {
				t.Error("String() returned empty string")
			}
		})
	}
}
