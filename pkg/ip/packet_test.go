package ip

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
			name: "valid IPv4 packet",
			data: []byte{
				0x45, 0x00, 0x00, 0x1C, // Version, IHL, DSCP, ECN, Total Length (28 bytes)
				0x12, 0x34, 0x40, 0x00, // Identification, Flags, Fragment Offset
				0x40, 0x06, 0x00, 0x00, // TTL, Protocol (TCP), Checksum (will be recalculated)
				0xc0, 0xa8, 0x01, 0x64, // Source IP (192.168.1.100)
				0xc0, 0xa8, 0x01, 0x01, // Destination IP (192.168.1.1)
				// 8 bytes of data
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			},
			wantErr: false,
		},
		{
			name:    "too short",
			data:    []byte{0x45, 0x00, 0x00},
			wantErr: true,
		},
		{
			name: "invalid version",
			data: []byte{
				0x65, 0x00, 0x00, 0x1C, // Version 6 instead of 4
				0x12, 0x34, 0x40, 0x00,
				0x40, 0x06, 0x00, 0x00,
				0xc0, 0xa8, 0x01, 0x64,
				0xc0, 0xa8, 0x01, 0x01,
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			},
			wantErr: true,
		},
		{
			name: "invalid IHL",
			data: []byte{
				0x43, 0x00, 0x00, 0x1C, // IHL = 3 (too small)
				0x12, 0x34, 0x40, 0x00,
				0x40, 0x06, 0x00, 0x00,
				0xc0, 0xa8, 0x01, 0x64,
				0xc0, 0xa8, 0x01, 0x01,
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
				t.Error("Parse() returned nil packet")
			}
		})
	}
}

func TestPacket_Serialize(t *testing.T) {
	srcIP, _ := common.ParseIPv4("192.168.1.100")
	dstIP, _ := common.ParseIPv4("192.168.1.1")

	payload := []byte("Hello, World!")

	pkt := NewPacket(srcIP, dstIP, common.ProtocolICMP, payload)

	data, err := pkt.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	if len(data) < MinHeaderLength {
		t.Errorf("Serialized packet too short: %d bytes", len(data))
	}

	// Parse it back
	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check fields
	if parsed.Version != IPv4Version {
		t.Errorf("Version = %d, want %d", parsed.Version, IPv4Version)
	}
	if parsed.Protocol != common.ProtocolICMP {
		t.Errorf("Protocol = %d, want %d", parsed.Protocol, common.ProtocolICMP)
	}
	if parsed.Source != srcIP {
		t.Errorf("Source = %s, want %s", parsed.Source, srcIP)
	}
	if parsed.Destination != dstIP {
		t.Errorf("Destination = %s, want %s", parsed.Destination, dstIP)
	}
	if !bytes.Equal(parsed.Payload, payload) {
		t.Errorf("Payload = %v, want %v", parsed.Payload, payload)
	}
}

func TestPacket_VerifyChecksum(t *testing.T) {
	srcIP, _ := common.ParseIPv4("192.168.1.100")
	dstIP, _ := common.ParseIPv4("192.168.1.1")

	pkt := NewPacket(srcIP, dstIP, common.ProtocolICMP, []byte("test"))

	data, err := pkt.Serialize()
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
	parsed.Checksum = 0x1234
	if parsed.VerifyChecksum() {
		t.Error("VerifyChecksum() = true for corrupted checksum, want false")
	}
}

func TestPacket_DecrementTTL(t *testing.T) {
	pkt := &Packet{TTL: 64}

	// Decrement from 64 to 2 should all return true (packet still alive)
	for i := 64; i > 2; i-- {
		if !pkt.DecrementTTL() {
			t.Errorf("DecrementTTL() = false at TTL %d, want true", i)
		}
	}

	// TTL should be 2 now, decrement to 1 (should return true)
	if !pkt.DecrementTTL() {
		t.Error("DecrementTTL() = false at TTL 2, want true")
	}

	// TTL should be 1 now, decrement to 0 (should return false - packet dead)
	if pkt.DecrementTTL() {
		t.Error("DecrementTTL() = true at TTL 1, want false (packet should die)")
	}

	// TTL should be 0 now
	if pkt.TTL != 0 {
		t.Errorf("TTL = %d, want 0", pkt.TTL)
	}

	// Decrementing at 0 should return false
	if pkt.DecrementTTL() {
		t.Error("DecrementTTL() = true at TTL 0, want false")
	}
}

func TestPacket_IsFragment(t *testing.T) {
	tests := []struct {
		name           string
		fragmentOffset uint16
		flags          IPv4Flags
		want           bool
	}{
		{
			name:           "not a fragment",
			fragmentOffset: 0,
			flags:          0,
			want:           false,
		},
		{
			name:           "has fragment offset",
			fragmentOffset: 100,
			flags:          0,
			want:           true,
		},
		{
			name:           "has more fragments flag",
			fragmentOffset: 0,
			flags:          FlagMoreFragments,
			want:           true,
		},
		{
			name:           "both offset and flag",
			fragmentOffset: 100,
			flags:          FlagMoreFragments,
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkt := &Packet{
				FragmentOffset: tt.fragmentOffset,
				Flags:          tt.flags,
			}
			if got := pkt.IsFragment(); got != tt.want {
				t.Errorf("IsFragment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewPacket(t *testing.T) {
	srcIP, _ := common.ParseIPv4("10.0.0.1")
	dstIP, _ := common.ParseIPv4("10.0.0.2")
	payload := []byte("test payload")

	pkt := NewPacket(srcIP, dstIP, common.ProtocolTCP, payload)

	if pkt.Version != IPv4Version {
		t.Errorf("Version = %d, want %d", pkt.Version, IPv4Version)
	}
	if pkt.IHL != 5 {
		t.Errorf("IHL = %d, want 5", pkt.IHL)
	}
	if pkt.TTL != DefaultTTL {
		t.Errorf("TTL = %d, want %d", pkt.TTL, DefaultTTL)
	}
	if pkt.Protocol != common.ProtocolTCP {
		t.Errorf("Protocol = %d, want %d", pkt.Protocol, common.ProtocolTCP)
	}
	if pkt.Source != srcIP {
		t.Errorf("Source = %s, want %s", pkt.Source, srcIP)
	}
	if pkt.Destination != dstIP {
		t.Errorf("Destination = %s, want %s", pkt.Destination, dstIP)
	}
	if !bytes.Equal(pkt.Payload, payload) {
		t.Errorf("Payload = %v, want %v", pkt.Payload, payload)
	}
}

func TestPacket_WithOptions(t *testing.T) {
	srcIP, _ := common.ParseIPv4("192.168.1.100")
	dstIP, _ := common.ParseIPv4("192.168.1.1")

	// Create packet with options
	pkt := NewPacket(srcIP, dstIP, common.ProtocolICMP, []byte("test"))
	pkt.Options = []byte{0x01, 0x02, 0x03, 0x04} // 4 bytes of options

	data, err := pkt.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if parsed.IHL != 6 { // 5 (base) + 1 (4 bytes of options)
		t.Errorf("IHL = %d, want 6", parsed.IHL)
	}

	if !bytes.Equal(parsed.Options, pkt.Options) {
		t.Errorf("Options = %v, want %v", parsed.Options, pkt.Options)
	}
}

func BenchmarkParse(b *testing.B) {
	data := []byte{
		0x45, 0x00, 0x00, 0x28,
		0x12, 0x34, 0x40, 0x00,
		0x40, 0x06, 0x00, 0x00,
		0xc0, 0xa8, 0x01, 0x64,
		0xc0, 0xa8, 0x01, 0x01,
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(data)
	}
}

func BenchmarkSerialize(b *testing.B) {
	srcIP, _ := common.ParseIPv4("192.168.1.100")
	dstIP, _ := common.ParseIPv4("192.168.1.1")
	pkt := NewPacket(srcIP, dstIP, common.ProtocolICMP, []byte("test payload"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pkt.Serialize()
	}
}
