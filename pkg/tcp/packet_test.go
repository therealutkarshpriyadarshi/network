package tcp

import (
	"testing"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

func TestSegmentParseAndSerialize(t *testing.T) {
	tests := []struct {
		name        string
		seg         *Segment
		expectError bool
	}{
		{
			name: "Basic SYN segment",
			seg: &Segment{
				SourcePort:      12345,
				DestinationPort: 80,
				SequenceNumber:  1000,
				AckNumber:       0,
				DataOffset:      5,
				Flags:           FlagSYN,
				WindowSize:      65535,
				Checksum:        0,
				UrgentPointer:   0,
				Options:         nil,
				Data:            nil,
			},
			expectError: false,
		},
		{
			name: "SYN+ACK segment",
			seg: &Segment{
				SourcePort:      80,
				DestinationPort: 12345,
				SequenceNumber:  2000,
				AckNumber:       1001,
				DataOffset:      5,
				Flags:           FlagSYN | FlagACK,
				WindowSize:      65535,
				Checksum:        0,
				UrgentPointer:   0,
				Options:         nil,
				Data:            nil,
			},
			expectError: false,
		},
		{
			name: "Data segment with PSH+ACK",
			seg: &Segment{
				SourcePort:      12345,
				DestinationPort: 80,
				SequenceNumber:  1001,
				AckNumber:       2001,
				DataOffset:      5,
				Flags:           FlagPSH | FlagACK,
				WindowSize:      65535,
				Checksum:        0,
				UrgentPointer:   0,
				Options:         nil,
				Data:            []byte("Hello, World!"),
			},
			expectError: false,
		},
		{
			name: "Segment with MSS option",
			seg: &Segment{
				SourcePort:      12345,
				DestinationPort: 80,
				SequenceNumber:  1000,
				AckNumber:       0,
				DataOffset:      6,
				Flags:           FlagSYN,
				WindowSize:      65535,
				Checksum:        0,
				UrgentPointer:   0,
				Options:         BuildMSSOption(1460),
				Data:            nil,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize the segment
			data, err := tt.seg.Serialize()
			if (err != nil) != tt.expectError {
				t.Fatalf("Serialize() error = %v, expectError %v", err, tt.expectError)
			}
			if tt.expectError {
				return
			}

			// Parse it back
			parsed, err := Parse(data)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			// Compare fields
			if parsed.SourcePort != tt.seg.SourcePort {
				t.Errorf("SourcePort = %d, want %d", parsed.SourcePort, tt.seg.SourcePort)
			}
			if parsed.DestinationPort != tt.seg.DestinationPort {
				t.Errorf("DestinationPort = %d, want %d", parsed.DestinationPort, tt.seg.DestinationPort)
			}
			if parsed.SequenceNumber != tt.seg.SequenceNumber {
				t.Errorf("SequenceNumber = %d, want %d", parsed.SequenceNumber, tt.seg.SequenceNumber)
			}
			if parsed.AckNumber != tt.seg.AckNumber {
				t.Errorf("AckNumber = %d, want %d", parsed.AckNumber, tt.seg.AckNumber)
			}
			if parsed.Flags != tt.seg.Flags {
				t.Errorf("Flags = %d, want %d", parsed.Flags, tt.seg.Flags)
			}
			if parsed.WindowSize != tt.seg.WindowSize {
				t.Errorf("WindowSize = %d, want %d", parsed.WindowSize, tt.seg.WindowSize)
			}
			if string(parsed.Data) != string(tt.seg.Data) {
				t.Errorf("Data = %s, want %s", parsed.Data, tt.seg.Data)
			}
		})
	}
}

func TestSegmentChecksum(t *testing.T) {
	srcIP := common.IPv4Address{192, 168, 1, 1}
	dstIP := common.IPv4Address{192, 168, 1, 2}

	seg := NewSegment(12345, 80, 1000, 2000, FlagACK, 65535, []byte("Test data"))

	// Calculate checksum
	checksum, err := seg.CalculateChecksum(srcIP, dstIP)
	if err != nil {
		t.Fatalf("CalculateChecksum() error = %v", err)
	}

	seg.Checksum = checksum

	// Verify checksum
	if !seg.VerifyChecksum(srcIP, dstIP) {
		t.Error("Checksum verification failed")
	}
}

func TestSegmentFlags(t *testing.T) {
	seg := NewSegment(12345, 80, 1000, 2000, 0, 65535, nil)

	// Test setting flags
	seg.SetFlag(FlagSYN)
	if !seg.HasFlag(FlagSYN) {
		t.Error("SYN flag not set")
	}

	seg.SetFlag(FlagACK)
	if !seg.HasFlag(FlagACK) {
		t.Error("ACK flag not set")
	}

	// Test clearing flags
	seg.ClearFlag(FlagSYN)
	if seg.HasFlag(FlagSYN) {
		t.Error("SYN flag not cleared")
	}

	if !seg.HasFlag(FlagACK) {
		t.Error("ACK flag should still be set")
	}
}

func TestBuildMSSOption(t *testing.T) {
	opt := BuildMSSOption(1460)

	if len(opt) != 4 {
		t.Errorf("MSS option length = %d, want 4", len(opt))
	}

	if opt[0] != OptionKindMSS {
		t.Errorf("Option kind = %d, want %d", opt[0], OptionKindMSS)
	}

	if opt[1] != 4 {
		t.Errorf("Option length = %d, want 4", opt[1])
	}
}

func TestGetMSS(t *testing.T) {
	seg := NewSegment(12345, 80, 1000, 0, FlagSYN, 65535, nil)
	seg.Options = BuildMSSOption(1460)

	mss, err := seg.GetMSS()
	if err != nil {
		t.Fatalf("GetMSS() error = %v", err)
	}

	if mss != 1460 {
		t.Errorf("MSS = %d, want 1460", mss)
	}
}

func TestSegmentString(t *testing.T) {
	seg := NewSegment(12345, 80, 1000, 2000, FlagSYN|FlagACK, 65535, []byte("data"))

	str := seg.String()
	if str == "" {
		t.Error("String() returned empty string")
	}

	// Check that it contains key information
	// (We're not testing exact format, just that it's not empty)
	t.Logf("Segment string: %s", str)
}
