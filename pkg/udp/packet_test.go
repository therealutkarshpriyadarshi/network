package udp

import (
	"bytes"
	"testing"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *Packet
		wantErr bool
	}{
		{
			name: "valid packet with data",
			data: []byte{
				0x1F, 0x90, // Source port: 8080
				0x00, 0x50, // Destination port: 80
				0x00, 0x10, // Length: 16
				0x00, 0x00, // Checksum: 0
				0x48, 0x65, 0x6C, 0x6C, 0x6F, 0x21, 0x21, 0x21, // Data: "Hello!!!"
			},
			want: &Packet{
				SourcePort:      8080,
				DestinationPort: 80,
				Length:          16,
				Checksum:        0,
				Data:            []byte("Hello!!!"),
			},
			wantErr: false,
		},
		{
			name: "valid packet without data (header only)",
			data: []byte{
				0x1F, 0x90, // Source port: 8080
				0x00, 0x50, // Destination port: 80
				0x00, 0x08, // Length: 8 (header only)
				0x00, 0x00, // Checksum: 0
			},
			want: &Packet{
				SourcePort:      8080,
				DestinationPort: 80,
				Length:          8,
				Checksum:        0,
				Data:            nil,
			},
			wantErr: false,
		},
		{
			name:    "packet too short",
			data:    []byte{0x1F, 0x90, 0x00, 0x50},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid length (too small)",
			data: []byte{
				0x1F, 0x90, // Source port: 8080
				0x00, 0x50, // Destination port: 80
				0x00, 0x04, // Length: 4 (invalid, less than header)
				0x00, 0x00, // Checksum: 0
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "length mismatch",
			data: []byte{
				0x1F, 0x90, // Source port: 8080
				0x00, 0x50, // Destination port: 80
				0x00, 0x20, // Length: 32 (but packet is shorter)
				0x00, 0x00, // Checksum: 0
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.SourcePort != tt.want.SourcePort {
				t.Errorf("Parse() SourcePort = %v, want %v", got.SourcePort, tt.want.SourcePort)
			}
			if got.DestinationPort != tt.want.DestinationPort {
				t.Errorf("Parse() DestinationPort = %v, want %v", got.DestinationPort, tt.want.DestinationPort)
			}
			if got.Length != tt.want.Length {
				t.Errorf("Parse() Length = %v, want %v", got.Length, tt.want.Length)
			}
			if got.Checksum != tt.want.Checksum {
				t.Errorf("Parse() Checksum = %v, want %v", got.Checksum, tt.want.Checksum)
			}
			if !bytes.Equal(got.Data, tt.want.Data) {
				t.Errorf("Parse() Data = %v, want %v", got.Data, tt.want.Data)
			}
		})
	}
}

func TestSerialize(t *testing.T) {
	tests := []struct {
		name    string
		packet  *Packet
		want    []byte
		wantErr bool
	}{
		{
			name: "packet with data",
			packet: &Packet{
				SourcePort:      8080,
				DestinationPort: 80,
				Length:          0, // Will be calculated
				Checksum:        0xABCD,
				Data:            []byte("Test"),
			},
			want: []byte{
				0x1F, 0x90, // Source port: 8080
				0x00, 0x50, // Destination port: 80
				0x00, 0x0C, // Length: 12
				0xAB, 0xCD, // Checksum: 0xABCD
				0x54, 0x65, 0x73, 0x74, // Data: "Test"
			},
			wantErr: false,
		},
		{
			name: "packet without data",
			packet: &Packet{
				SourcePort:      12345,
				DestinationPort: 54321,
				Length:          0, // Will be calculated
				Checksum:        0x1234,
				Data:            nil,
			},
			want: []byte{
				0x30, 0x39, // Source port: 12345
				0xD4, 0x31, // Destination port: 54321
				0x00, 0x08, // Length: 8
				0x12, 0x34, // Checksum: 0x1234
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.packet.Serialize()
			if (err != nil) != tt.wantErr {
				t.Errorf("Serialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if !bytes.Equal(got, tt.want) {
				t.Errorf("Serialize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateChecksum(t *testing.T) {
	tests := []struct {
		name   string
		packet *Packet
		srcIP  common.IPv4Address
		dstIP  common.IPv4Address
	}{
		{
			name: "checksum with data",
			packet: &Packet{
				SourcePort:      8080,
				DestinationPort: 80,
				Data:            []byte("Hello, UDP!"),
			},
			srcIP: common.IPv4Address{192, 168, 1, 100},
			dstIP: common.IPv4Address{192, 168, 1, 1},
		},
		{
			name: "checksum without data",
			packet: &Packet{
				SourcePort:      12345,
				DestinationPort: 54321,
				Data:            nil,
			},
			srcIP: common.IPv4Address{10, 0, 0, 1},
			dstIP: common.IPv4Address{10, 0, 0, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate checksum
			checksum, err := tt.packet.CalculateChecksum(tt.srcIP, tt.dstIP)
			if err != nil {
				t.Errorf("CalculateChecksum() error = %v", err)
				return
			}

			// Checksum should not be zero (unless the data happened to produce that)
			// But we specifically convert 0 to 0xFFFF in the implementation
			if checksum == 0 {
				t.Errorf("CalculateChecksum() = 0, expected non-zero or 0xFFFF")
			}

			// Set the checksum and verify it
			tt.packet.Checksum = checksum
			if !tt.packet.VerifyChecksum(tt.srcIP, tt.dstIP) {
				t.Errorf("VerifyChecksum() failed after setting calculated checksum")
			}
		})
	}
}

func TestVerifyChecksum(t *testing.T) {
	srcIP := common.IPv4Address{192, 168, 1, 100}
	dstIP := common.IPv4Address{192, 168, 1, 1}

	tests := []struct {
		name   string
		packet *Packet
		valid  bool
	}{
		{
			name: "zero checksum (no checksum)",
			packet: &Packet{
				SourcePort:      8080,
				DestinationPort: 80,
				Length:          12,
				Checksum:        0, // No checksum
				Data:            []byte("Test"),
			},
			valid: true, // Zero checksum means no checksum in IPv4
		},
		{
			name: "valid checksum",
			packet: func() *Packet {
				p := &Packet{
					SourcePort:      8080,
					DestinationPort: 80,
					Data:            []byte("Hello!"),
				}
				checksum, _ := p.CalculateChecksum(srcIP, dstIP)
				p.Checksum = checksum
				// Need to set length properly
				p.Length = uint16(HeaderLength + len(p.Data))
				return p
			}(),
			valid: true,
		},
		{
			name: "invalid checksum",
			packet: &Packet{
				SourcePort:      8080,
				DestinationPort: 80,
				Length:          14,
				Checksum:        0xFFFF, // Incorrect checksum
				Data:            []byte("Hello!"),
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.packet.VerifyChecksum(srcIP, dstIP)
			if got != tt.valid {
				t.Errorf("VerifyChecksum() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestNewPacket(t *testing.T) {
	srcPort := uint16(8080)
	dstPort := uint16(80)
	data := []byte("Test data")

	pkt := NewPacket(srcPort, dstPort, data)

	if pkt.SourcePort != srcPort {
		t.Errorf("NewPacket() SourcePort = %v, want %v", pkt.SourcePort, srcPort)
	}
	if pkt.DestinationPort != dstPort {
		t.Errorf("NewPacket() DestinationPort = %v, want %v", pkt.DestinationPort, dstPort)
	}
	if pkt.Length != uint16(HeaderLength+len(data)) {
		t.Errorf("NewPacket() Length = %v, want %v", pkt.Length, HeaderLength+len(data))
	}
	if pkt.Checksum != 0 {
		t.Errorf("NewPacket() Checksum = %v, want 0", pkt.Checksum)
	}
	if !bytes.Equal(pkt.Data, data) {
		t.Errorf("NewPacket() Data = %v, want %v", pkt.Data, data)
	}
}

func TestRoundTrip(t *testing.T) {
	srcIP := common.IPv4Address{192, 168, 1, 100}
	dstIP := common.IPv4Address{192, 168, 1, 1}

	// Create original packet
	original := NewPacket(8080, 80, []byte("Round trip test data"))

	// Calculate and set checksum
	checksum, err := original.CalculateChecksum(srcIP, dstIP)
	if err != nil {
		t.Fatalf("CalculateChecksum() error = %v", err)
	}
	original.Checksum = checksum

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
	if parsed.SourcePort != original.SourcePort {
		t.Errorf("Round trip SourcePort = %v, want %v", parsed.SourcePort, original.SourcePort)
	}
	if parsed.DestinationPort != original.DestinationPort {
		t.Errorf("Round trip DestinationPort = %v, want %v", parsed.DestinationPort, original.DestinationPort)
	}
	if parsed.Length != original.Length {
		t.Errorf("Round trip Length = %v, want %v", parsed.Length, original.Length)
	}
	if parsed.Checksum != original.Checksum {
		t.Errorf("Round trip Checksum = %v, want %v", parsed.Checksum, original.Checksum)
	}
	if !bytes.Equal(parsed.Data, original.Data) {
		t.Errorf("Round trip Data = %v, want %v", parsed.Data, original.Data)
	}

	// Verify checksum
	if !parsed.VerifyChecksum(srcIP, dstIP) {
		t.Errorf("Round trip checksum verification failed")
	}
}

func TestString(t *testing.T) {
	pkt := NewPacket(8080, 80, []byte("Test"))
	str := pkt.String()
	expected := "UDP{SrcPort=8080, DstPort=80, Len=12, DataLen=4}"
	if str != expected {
		t.Errorf("String() = %v, want %v", str, expected)
	}
}
