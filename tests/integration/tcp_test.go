// +build integration

// Integration tests for TCP protocol
//
// These tests verify TCP segment serialization, parsing, and basic operations.
//
// Run with: go test -tags=integration ./tests/integration/...

package integration

import (
	"bytes"
	"testing"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/tcp"
)

// TestTCPHandshakeSegments tests TCP handshake segment structure.
func TestTCPHandshakeSegments(t *testing.T) {
	clientIP, _ := common.ParseIPv4("192.168.1.100")
	serverIP, _ := common.ParseIPv4("192.168.1.1")
	clientPort := uint16(50000)
	serverPort := uint16(80)

	t.Run("SYN segment", func(t *testing.T) {
		// Create SYN segment
		syn := tcp.NewSegment(clientPort, serverPort, 1000, 0, tcp.FlagSYN, 65535, nil)

		if !syn.HasFlag(tcp.FlagSYN) {
			t.Error("SYN segment should have SYN flag")
		}
		if syn.HasFlag(tcp.FlagACK) {
			t.Error("Initial SYN should not have ACK flag")
		}

		// Calculate checksum
		checksum, err := syn.CalculateChecksum(clientIP, serverIP)
		if err != nil {
			t.Fatalf("Failed to calculate checksum: %v", err)
		}
		syn.Checksum = checksum

		// Serialize and parse
		data, err := syn.Serialize()
		if err != nil {
			t.Fatalf("Serialize failed: %v", err)
		}

		parsed, err := tcp.Parse(data)
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if !parsed.HasFlag(tcp.FlagSYN) {
			t.Error("Parsed segment should have SYN flag")
		}
		if parsed.SourcePort != clientPort {
			t.Errorf("SourcePort = %d, want %d", parsed.SourcePort, clientPort)
		}
	})

	t.Run("SYN-ACK segment", func(t *testing.T) {
		// Create SYN-ACK segment
		synack := tcp.NewSegment(serverPort, clientPort, 2000, 1001, tcp.FlagSYN|tcp.FlagACK, 65535, nil)

		if !synack.HasFlag(tcp.FlagSYN) || !synack.HasFlag(tcp.FlagACK) {
			t.Error("SYN-ACK segment should have both SYN and ACK flags")
		}

		// Calculate checksum
		checksum, err := synack.CalculateChecksum(serverIP, clientIP)
		if err != nil {
			t.Fatalf("Failed to calculate checksum: %v", err)
		}
		synack.Checksum = checksum

		// Verify checksum
		if !synack.VerifyChecksum(serverIP, clientIP) {
			t.Error("Checksum verification failed")
		}
	})

	t.Run("ACK segment", func(t *testing.T) {
		// Create ACK segment (completes handshake)
		ack := tcp.NewSegment(clientPort, serverPort, 1001, 2001, tcp.FlagACK, 65535, nil)

		if !ack.HasFlag(tcp.FlagACK) {
			t.Error("ACK segment should have ACK flag")
		}
		if ack.HasFlag(tcp.FlagSYN) {
			t.Error("Final ACK should not have SYN flag")
		}

		if ack.AckNumber != 2001 {
			t.Errorf("AckNumber = %d, want 2001", ack.AckNumber)
		}
	})

}

// TestTCPStateTransitions tests TCP state machine transitions.
func TestTCPStateTransitions(t *testing.T) {
	sm := tcp.NewStateMachine()

	// Initial state should be CLOSED
	if sm.GetState() != tcp.StateClosed {
		t.Errorf("Initial state = %v, want CLOSED", sm.GetState())
	}

	// Test client-side transitions
	t.Run("Client transitions", func(t *testing.T) {
		clientSM := tcp.NewStateMachine()

		// CLOSED -> SYN_SENT (active open)
		err := clientSM.Transition(tcp.EventActiveOpen)
		if err != nil {
			t.Fatalf("Transition to SYN_SENT failed: %v", err)
		}
		if clientSM.GetState() != tcp.StateSynSent {
			t.Errorf("State = %v, want SYN_SENT", clientSM.GetState())
		}

		// SYN_SENT -> ESTABLISHED (receive SYN+ACK)
		err = clientSM.Transition(tcp.EventReceiveSynAck)
		if err != nil {
			t.Fatalf("Transition to ESTABLISHED failed: %v", err)
		}
		if clientSM.GetState() != tcp.StateEstablished {
			t.Errorf("State = %v, want ESTABLISHED", clientSM.GetState())
		}
	})

	// Test server-side transitions
	t.Run("Server transitions", func(t *testing.T) {
		serverSM := tcp.NewStateMachine()

		// CLOSED -> LISTEN (passive open)
		err := serverSM.Transition(tcp.EventPassiveOpen)
		if err != nil {
			t.Fatalf("Transition to LISTEN failed: %v", err)
		}
		if serverSM.GetState() != tcp.StateListen {
			t.Errorf("State = %v, want LISTEN", serverSM.GetState())
		}

		// LISTEN -> SYN_RECEIVED (receive SYN)
		err = serverSM.Transition(tcp.EventReceiveSyn)
		if err != nil {
			t.Fatalf("Transition to SYN_RECEIVED failed: %v", err)
		}
		if serverSM.GetState() != tcp.StateSynReceived {
			t.Errorf("State = %v, want SYN_RECEIVED", serverSM.GetState())
		}

		// SYN_RECEIVED -> ESTABLISHED (receive ACK)
		err = serverSM.Transition(tcp.EventReceiveAck)
		if err != nil {
			t.Fatalf("Transition to ESTABLISHED failed: %v", err)
		}
		if serverSM.GetState() != tcp.StateEstablished {
			t.Errorf("State = %v, want ESTABLISHED", serverSM.GetState())
		}
	})
}

// TestTCPDataSegments tests TCP data segment structure.
func TestTCPDataSegments(t *testing.T) {
	clientIP, _ := common.ParseIPv4("10.0.0.1")
	serverIP, _ := common.ParseIPv4("10.0.0.2")

	testData := []byte("Hello, TCP World!")

	// Create data segment with PSH+ACK flags
	dataSeg := tcp.NewSegment(50000, 80, 1001, 2001, tcp.FlagPSH|tcp.FlagACK, 32768, testData)

	if !dataSeg.HasFlag(tcp.FlagPSH) {
		t.Error("Data segment should have PSH flag")
	}
	if !dataSeg.HasFlag(tcp.FlagACK) {
		t.Error("Data segment should have ACK flag")
	}

	if !bytes.Equal(dataSeg.Data, testData) {
		t.Errorf("Segment data = %v, want %v", dataSeg.Data, testData)
	}

	// Calculate checksum
	checksum, err := dataSeg.CalculateChecksum(clientIP, serverIP)
	if err != nil {
		t.Fatalf("Failed to calculate checksum: %v", err)
	}
	dataSeg.Checksum = checksum

	// Serialize and parse
	wireData, err := dataSeg.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	parsed, err := tcp.Parse(wireData)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !bytes.Equal(parsed.Data, testData) {
		t.Error("Parsed data mismatch")
	}

	if !parsed.VerifyChecksum(clientIP, serverIP) {
		t.Error("Checksum verification failed")
	}

	t.Logf("Successfully transferred %d bytes in segment", len(testData))
}

// TestTCPConnectionCloseSegments tests connection close (FIN) segments.
func TestTCPConnectionCloseSegments(t *testing.T) {
	clientIP, _ := common.ParseIPv4("10.0.0.1")
	serverIP, _ := common.ParseIPv4("10.0.0.2")

	// Test FIN segment
	fin := tcp.NewSegment(50000, 80, 5000, 6000, tcp.FlagFIN|tcp.FlagACK, 32768, nil)

	if !fin.HasFlag(tcp.FlagFIN) {
		t.Error("FIN segment should have FIN flag")
	}
	if !fin.HasFlag(tcp.FlagACK) {
		t.Error("FIN segment should have ACK flag")
	}

	// Calculate checksum
	checksum, err := fin.CalculateChecksum(clientIP, serverIP)
	if err != nil {
		t.Fatalf("Failed to calculate checksum: %v", err)
	}
	fin.Checksum = checksum

	// Serialize and parse
	data, err := fin.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	parsed, err := tcp.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !parsed.HasFlag(tcp.FlagFIN) {
		t.Error("Parsed FIN segment should have FIN flag")
	}

	// Test close state transitions
	sm := tcp.NewStateMachine()
	sm.SetState(tcp.StateEstablished)

	// ESTABLISHED -> FIN_WAIT_1 (close)
	err = sm.Transition(tcp.EventClose)
	if err != nil {
		t.Fatalf("Transition to FIN_WAIT_1 failed: %v", err)
	}
	if sm.GetState() != tcp.StateFinWait1 {
		t.Errorf("State = %v, want FIN_WAIT_1", sm.GetState())
	}

	t.Log("Connection close sequence works correctly")
}

// TestTCPSegmentSerialization tests TCP segment serialization and parsing.
func TestTCPSegmentSerialization(t *testing.T) {
	tests := []struct {
		name string
		seg  *tcp.Segment
	}{
		{
			name: "SYN segment",
			seg: &tcp.Segment{
				SourcePort:      12345,
				DestinationPort: 80,
				SequenceNumber:  1000,
				AckNumber:       0,
				DataOffset:      5,
				Flags:           tcp.FlagSYN,
				WindowSize:      65535,
				Checksum:        0,
				UrgentPointer:   0,
				Options:         nil,
				Data:            nil,
			},
		},
		{
			name: "Data segment",
			seg: &tcp.Segment{
				SourcePort:      12345,
				DestinationPort: 80,
				SequenceNumber:  1001,
				AckNumber:       2001,
				DataOffset:      5,
				Flags:           tcp.FlagPSH | tcp.FlagACK,
				WindowSize:      32768,
				Checksum:        0,
				UrgentPointer:   0,
				Options:         nil,
				Data:            []byte("Test data payload"),
			},
		},
		{
			name: "FIN+ACK segment",
			seg: &tcp.Segment{
				SourcePort:      12345,
				DestinationPort: 80,
				SequenceNumber:  2000,
				AckNumber:       3000,
				DataOffset:      5,
				Flags:           tcp.FlagFIN | tcp.FlagACK,
				WindowSize:      16384,
				Checksum:        0,
				UrgentPointer:   0,
				Options:         nil,
				Data:            nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			data, err := tt.seg.Serialize()
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			// Parse
			parsed, err := tcp.Parse(data)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Verify fields
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
				t.Errorf("Flags = 0x%02x, want 0x%02x", parsed.Flags, tt.seg.Flags)
			}
			if parsed.WindowSize != tt.seg.WindowSize {
				t.Errorf("WindowSize = %d, want %d", parsed.WindowSize, tt.seg.WindowSize)
			}
			if !bytes.Equal(parsed.Data, tt.seg.Data) {
				t.Errorf("Data mismatch: got %v, want %v", parsed.Data, tt.seg.Data)
			}
		})
	}
}

// TestTCPWindowHandling tests TCP window size in segments.
func TestTCPWindowHandling(t *testing.T) {
	// Test different window sizes
	windows := []uint16{65535, 32768, 16384, 8192, 1024}

	for _, wnd := range windows {
		seg := tcp.NewSegment(50000, 80, 1000, 2000, tcp.FlagACK, wnd, nil)

		if seg.WindowSize != wnd {
			t.Errorf("WindowSize = %d, want %d", seg.WindowSize, wnd)
		}

		// Serialize and parse
		data, err := seg.Serialize()
		if err != nil {
			t.Fatalf("Serialize failed: %v", err)
		}

		parsed, err := tcp.Parse(data)
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if parsed.WindowSize != wnd {
			t.Errorf("Parsed WindowSize = %d, want %d", parsed.WindowSize, wnd)
		}
	}

	t.Log("Window size handling works correctly")
}

// TestTCPOptions tests TCP options handling.
func TestTCPOptions(t *testing.T) {
	// Test MSS option
	mssOpt := tcp.BuildMSSOption(1460)

	syn := tcp.NewSegment(50000, 80, 1000, 0, tcp.FlagSYN, 65535, nil)
	syn.Options = mssOpt

	// Serialize and parse
	data, err := syn.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	parsed, err := tcp.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.Options) == 0 {
		t.Error("Options were not preserved")
	}

	// Try to extract MSS
	mss, err := parsed.GetMSS()
	if err == nil && mss != 1460 {
		t.Errorf("MSS = %d, want 1460", mss)
	}

	t.Log("TCP options handling works correctly")
}

// TestTCPSequenceNumbers tests sequence number handling.
func TestTCPSequenceNumbers(t *testing.T) {
	// Test that sequence numbers can wrap around
	maxSeq := uint32(0xFFFFFFFF)

	seg1 := tcp.NewSegment(50000, 80, maxSeq-10, 0, tcp.FlagACK, 65535, []byte("data"))
	seg2 := tcp.NewSegment(50000, 80, maxSeq, 0, tcp.FlagACK, 65535, []byte("more"))
	seg3 := tcp.NewSegment(50000, 80, 5, 0, tcp.FlagACK, 65535, []byte("wrapped"))

	if seg1.SequenceNumber != maxSeq-10 {
		t.Error("Sequence number not set correctly")
	}
	if seg2.SequenceNumber != maxSeq {
		t.Error("Max sequence number not set correctly")
	}
	if seg3.SequenceNumber != 5 {
		t.Error("Wrapped sequence number not set correctly")
	}

	t.Log("Sequence number handling works correctly")
}

// TestTCPReset tests RST segment handling.
func TestTCPReset(t *testing.T) {
	clientIP, _ := common.ParseIPv4("192.168.1.1")
	serverIP, _ := common.ParseIPv4("192.168.1.2")

	// Create RST segment
	rst := tcp.NewSegment(50000, 80, 1000, 0, tcp.FlagRST, 0, nil)

	if !rst.HasFlag(tcp.FlagRST) {
		t.Error("RST segment should have RST flag")
	}

	// Calculate checksum
	checksum, err := rst.CalculateChecksum(clientIP, serverIP)
	if err != nil {
		t.Fatalf("Failed to calculate checksum: %v", err)
	}
	rst.Checksum = checksum

	// Serialize and parse
	data, err := rst.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	parsed, err := tcp.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !parsed.HasFlag(tcp.FlagRST) {
		t.Error("Parsed segment should have RST flag")
	}

	if !parsed.VerifyChecksum(clientIP, serverIP) {
		t.Error("Checksum verification failed")
	}

	t.Log("RST segment handling works correctly")
}
