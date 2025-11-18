// +build integration

// Integration tests for UDP protocol
//
// These tests verify UDP datagram transmission, socket operations, and packet handling.
//
// Run with: go test -tags=integration ./tests/integration/...

package integration

import (
	"bytes"
	"testing"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/udp"
)

// TestUDPPacketSerialization tests UDP packet serialization and parsing.
func TestUDPPacketSerialization(t *testing.T) {
	tests := []struct {
		name       string
		srcPort    uint16
		dstPort    uint16
		data       []byte
	}{
		{
			name:    "Empty payload",
			srcPort: 12345,
			dstPort: 53,
			data:    []byte{},
		},
		{
			name:    "Small payload",
			srcPort: 5000,
			dstPort: 8080,
			data:    []byte("Hello, UDP!"),
		},
		{
			name:    "Large payload",
			srcPort: 60000,
			dstPort: 9000,
			data:    make([]byte, 1024),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create packet
			pkt := udp.NewPacket(tt.srcPort, tt.dstPort, tt.data)

			// Serialize
			data, err := pkt.Serialize()
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			// Parse
			parsed, err := udp.Parse(data)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Verify fields
			if parsed.SourcePort != tt.srcPort {
				t.Errorf("SourcePort = %d, want %d", parsed.SourcePort, tt.srcPort)
			}
			if parsed.DestinationPort != tt.dstPort {
				t.Errorf("DestinationPort = %d, want %d", parsed.DestinationPort, tt.dstPort)
			}
			if !bytes.Equal(parsed.Data, tt.data) {
				t.Errorf("Data mismatch: got %d bytes, want %d bytes", len(parsed.Data), len(tt.data))
			}

			// Verify length field
			expectedLength := uint16(8 + len(tt.data)) // 8 byte header + data
			if parsed.Length != expectedLength {
				t.Errorf("Length = %d, want %d", parsed.Length, expectedLength)
			}
		})
	}
}

// TestUDPChecksumVerification tests UDP checksum calculation and verification.
func TestUDPChecksumVerification(t *testing.T) {
	srcIP, _ := common.ParseIPv4("192.168.1.100")
	dstIP, _ := common.ParseIPv4("192.168.1.1")

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Empty data",
			data: []byte{},
		},
		{
			name: "Short data",
			data: []byte("Test"),
		},
		{
			name: "Long data",
			data: bytes.Repeat([]byte("X"), 500),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkt := udp.NewPacket(5000, 8080, tt.data)

			// Calculate checksum
			checksum, err := pkt.CalculateChecksum(srcIP, dstIP)
			if err != nil {
				t.Fatalf("CalculateChecksum failed: %v", err)
			}
			pkt.Checksum = checksum

			// Verify checksum
			if !pkt.VerifyChecksum(srcIP, dstIP) {
				t.Error("Checksum verification failed")
			}

			// Corrupt data and verify checksum fails
			pkt.Data[0] ^= 0xFF // Flip bits
			if pkt.VerifyChecksum(srcIP, dstIP) {
				t.Error("Checksum verification should fail with corrupted data")
			}
		})
	}
}

// TestUDPDatagramTransmission tests end-to-end UDP datagram transmission.
func TestUDPDatagramTransmission(t *testing.T) {
	senderIP, _ := common.ParseIPv4("10.0.0.1")
	receiverIP, _ := common.ParseIPv4("10.0.0.2")

	// Create sender packet
	senderData := []byte("UDP message from sender")
	senderPkt := udp.NewPacket(50000, 8080, senderData)

	// Calculate checksum
	checksum, err := senderPkt.CalculateChecksum(senderIP, receiverIP)
	if err != nil {
		t.Fatalf("Failed to calculate checksum: %v", err)
	}
	senderPkt.Checksum = checksum

	// Serialize for transmission
	wireData, err := senderPkt.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	t.Logf("Transmitted %d bytes", len(wireData))

	// Receiver parses packet
	receiverPkt, err := udp.Parse(wireData)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Verify packet integrity
	if !receiverPkt.VerifyChecksum(senderIP, receiverIP) {
		t.Error("Checksum verification failed")
	}

	// Verify data
	if !bytes.Equal(receiverPkt.Data, senderData) {
		t.Errorf("Received data = %v, want %v", receiverPkt.Data, senderData)
	}

	// Verify ports
	if receiverPkt.SourcePort != 50000 {
		t.Errorf("SourcePort = %d, want 50000", receiverPkt.SourcePort)
	}
	if receiverPkt.DestinationPort != 8080 {
		t.Errorf("DestinationPort = %d, want 8080", receiverPkt.DestinationPort)
	}

	t.Log("Datagram transmitted successfully")
}

// TestUDPBidirectionalCommunication tests bidirectional UDP communication.
func TestUDPBidirectionalCommunication(t *testing.T) {
	clientIP, _ := common.ParseIPv4("192.168.1.100")
	serverIP, _ := common.ParseIPv4("192.168.1.1")

	// Client sends request
	request := []byte("REQUEST")
	clientPkt := udp.NewPacket(50000, 53, request)
	clientChecksum, _ := clientPkt.CalculateChecksum(clientIP, serverIP)
	clientPkt.Checksum = clientChecksum

	clientData, err := clientPkt.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize client packet: %v", err)
	}

	// Server receives request
	serverRcvdPkt, err := udp.Parse(clientData)
	if err != nil {
		t.Fatalf("Server failed to parse: %v", err)
	}

	if !bytes.Equal(serverRcvdPkt.Data, request) {
		t.Error("Server received incorrect data")
	}

	// Server sends response (swap ports)
	response := []byte("RESPONSE")
	serverPkt := udp.NewPacket(53, 50000, response)
	serverChecksum, _ := serverPkt.CalculateChecksum(serverIP, clientIP)
	serverPkt.Checksum = serverChecksum

	serverData, err := serverPkt.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize server packet: %v", err)
	}

	// Client receives response
	clientRcvdPkt, err := udp.Parse(serverData)
	if err != nil {
		t.Fatalf("Client failed to parse: %v", err)
	}

	if !bytes.Equal(clientRcvdPkt.Data, response) {
		t.Error("Client received incorrect data")
	}

	// Verify port swap
	if clientRcvdPkt.SourcePort != 53 {
		t.Errorf("Response source port = %d, want 53", clientRcvdPkt.SourcePort)
	}
	if clientRcvdPkt.DestinationPort != 50000 {
		t.Errorf("Response dest port = %d, want 50000", clientRcvdPkt.DestinationPort)
	}

	t.Log("Bidirectional communication successful")
}

// TestUDPMultipleDatagrams tests sending multiple datagrams.
func TestUDPMultipleDatagrams(t *testing.T) {
	srcIP, _ := common.ParseIPv4("10.0.0.1")
	dstIP, _ := common.ParseIPv4("10.0.0.2")

	numDatagrams := 10
	var packets []*udp.Packet

	// Send multiple datagrams
	for i := 0; i < numDatagrams; i++ {
		data := []byte{byte(i)}
		pkt := udp.NewPacket(uint16(50000+i), 8080, data)
		checksum, _ := pkt.CalculateChecksum(srcIP, dstIP)
		pkt.Checksum = checksum
		packets = append(packets, pkt)
	}

	// Verify each packet
	for i, pkt := range packets {
		if pkt.SourcePort != uint16(50000+i) {
			t.Errorf("Packet %d: SourcePort = %d, want %d", i, pkt.SourcePort, 50000+i)
		}
		if !pkt.VerifyChecksum(srcIP, dstIP) {
			t.Errorf("Packet %d: checksum verification failed", i)
		}
		if len(pkt.Data) != 1 || pkt.Data[0] != byte(i) {
			t.Errorf("Packet %d: data = %v, want [%d]", i, pkt.Data, i)
		}
	}

	t.Logf("Successfully sent and verified %d datagrams", numDatagrams)
}

// TestUDPZeroChecksum tests handling of zero checksum (optional in IPv4).
func TestUDPZeroChecksum(t *testing.T) {
	pkt := udp.NewPacket(5000, 8080, []byte("test"))
	pkt.Checksum = 0 // Zero checksum (optional in IPv4)

	// Serialize and parse
	data, err := pkt.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	parsed, err := udp.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.Checksum != 0 {
		t.Errorf("Checksum = %d, want 0", parsed.Checksum)
	}

	// Note: Zero checksum should be treated as "checksum not computed"
	// in IPv4, so verification should either pass or be skipped
	t.Logf("Zero checksum handled correctly")
}

// TestUDPMaximumPayload tests UDP with maximum payload size.
func TestUDPMaximumPayload(t *testing.T) {
	srcIP, _ := common.ParseIPv4("10.0.0.1")
	dstIP, _ := common.ParseIPv4("10.0.0.2")

	// Maximum UDP payload: 65535 (max IP) - 20 (IP header) - 8 (UDP header) = 65507 bytes
	maxPayload := 65507
	largeData := make([]byte, maxPayload)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	pkt := udp.NewPacket(50000, 8080, largeData)
	checksum, err := pkt.CalculateChecksum(srcIP, dstIP)
	if err != nil {
		t.Fatalf("CalculateChecksum failed: %v", err)
	}
	pkt.Checksum = checksum

	// Serialize
	data, err := pkt.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Parse
	parsed, err := udp.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify
	if len(parsed.Data) != maxPayload {
		t.Errorf("Payload length = %d, want %d", len(parsed.Data), maxPayload)
	}

	if !bytes.Equal(parsed.Data, largeData) {
		t.Error("Large payload data mismatch")
	}

	if !parsed.VerifyChecksum(srcIP, dstIP) {
		t.Error("Checksum verification failed for large payload")
	}

	t.Logf("Successfully handled maximum UDP payload (%d bytes)", maxPayload)
}

// TestUDPPortNumbers tests various port number combinations.
func TestUDPPortNumbers(t *testing.T) {
	tests := []struct {
		name    string
		srcPort uint16
		dstPort uint16
	}{
		{"Well-known ports", 53, 80},
		{"Registered ports", 1024, 8080},
		{"Dynamic ports", 49152, 65535},
		{"Zero source port", 0, 8080},
		{"Max port numbers", 65535, 65535},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkt := udp.NewPacket(tt.srcPort, tt.dstPort, []byte("test"))

			data, err := pkt.Serialize()
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			parsed, err := udp.Parse(data)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if parsed.SourcePort != tt.srcPort {
				t.Errorf("SourcePort = %d, want %d", parsed.SourcePort, tt.srcPort)
			}
			if parsed.DestinationPort != tt.dstPort {
				t.Errorf("DestinationPort = %d, want %d", parsed.DestinationPort, tt.dstPort)
			}
		})
	}
}
