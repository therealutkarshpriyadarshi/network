package common

import (
	"testing"
)

func TestMACAddress(t *testing.T) {
	mac := MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}

	// Test String
	expected := "00:11:22:33:44:55"
	if mac.String() != expected {
		t.Errorf("MACAddress.String() = %s, want %s", mac.String(), expected)
	}

	// Test IsBroadcast
	if mac.IsBroadcast() {
		t.Error("MACAddress.IsBroadcast() = true, want false")
	}

	broadcast := BroadcastMAC
	if !broadcast.IsBroadcast() {
		t.Error("BroadcastMAC.IsBroadcast() = false, want true")
	}

	// Test IsMulticast
	if mac.IsMulticast() {
		t.Error("MACAddress.IsMulticast() = true, want false")
	}

	multicast := MACAddress{0x01, 0x00, 0x5E, 0x00, 0x00, 0x01}
	if !multicast.IsMulticast() {
		t.Error("Multicast MAC.IsMulticast() = false, want true")
	}
}

func TestParseMAC(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    MACAddress
		wantErr bool
	}{
		{
			name:    "valid MAC",
			input:   "00:11:22:33:44:55",
			want:    MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			wantErr: false,
		},
		{
			name:    "broadcast MAC",
			input:   "FF:FF:FF:FF:FF:FF",
			want:    BroadcastMAC,
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMAC(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMAC() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseMAC() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIPv4Address(t *testing.T) {
	ip := IPv4Address{192, 168, 1, 1}

	// Test String
	expected := "192.168.1.1"
	if ip.String() != expected {
		t.Errorf("IPv4Address.String() = %s, want %s", ip.String(), expected)
	}

	// Test ToUint32
	val := ip.ToUint32()
	expectedVal := uint32(0xC0A80101) // 192.168.1.1 in hex
	if val != expectedVal {
		t.Errorf("IPv4Address.ToUint32() = 0x%08X, want 0x%08X", val, expectedVal)
	}
}

func TestParseIPv4(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    IPv4Address
		wantErr bool
	}{
		{
			name:    "valid IP",
			input:   "192.168.1.1",
			want:    IPv4Address{192, 168, 1, 1},
			wantErr: false,
		},
		{
			name:    "localhost",
			input:   "127.0.0.1",
			want:    IPv4Address{127, 0, 0, 1},
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "IPv6 address",
			input:   "::1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIPv4(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIPv4() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseIPv4() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIPv4FromUint32(t *testing.T) {
	tests := []struct {
		input uint32
		want  IPv4Address
	}{
		{
			input: 0xC0A80101,
			want:  IPv4Address{192, 168, 1, 1},
		},
		{
			input: 0x7F000001,
			want:  IPv4Address{127, 0, 0, 1},
		},
		{
			input: 0x00000000,
			want:  IPv4Address{0, 0, 0, 0},
		},
		{
			input: 0xFFFFFFFF,
			want:  IPv4Address{255, 255, 255, 255},
		},
	}

	for _, tt := range tests {
		got := IPv4FromUint32(tt.input)
		if got != tt.want {
			t.Errorf("IPv4FromUint32(0x%08X) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestEtherType(t *testing.T) {
	tests := []struct {
		etherType EtherType
		want      string
	}{
		{EtherTypeIPv4, "IPv4"},
		{EtherTypeARP, "ARP"},
		{EtherTypeIPv6, "IPv6"},
		{EtherType(0x9999), "Unknown(0x9999)"},
	}

	for _, tt := range tests {
		got := tt.etherType.String()
		if got != tt.want {
			t.Errorf("EtherType(0x%04X).String() = %s, want %s", tt.etherType, got, tt.want)
		}
	}
}

func TestProtocol(t *testing.T) {
	tests := []struct {
		protocol Protocol
		want     string
	}{
		{ProtocolICMP, "ICMP"},
		{ProtocolTCP, "TCP"},
		{ProtocolUDP, "UDP"},
		{Protocol(99), "Unknown(99)"},
	}

	for _, tt := range tests {
		got := tt.protocol.String()
		if got != tt.want {
			t.Errorf("Protocol(%d).String() = %s, want %s", tt.protocol, got, tt.want)
		}
	}
}

func TestIPv4AddressRoundtrip(t *testing.T) {
	original := IPv4Address{192, 168, 1, 100}
	asUint := original.ToUint32()
	back := IPv4FromUint32(asUint)

	if back != original {
		t.Errorf("Roundtrip failed: %v -> 0x%08X -> %v", original, asUint, back)
	}
}
