// Package udp implements the User Datagram Protocol (UDP) as defined in RFC 768.
package udp

import (
	"encoding/binary"
	"fmt"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

const (
	// HeaderLength is the UDP header length (8 bytes).
	HeaderLength = 8

	// MinPacketSize is the minimum UDP packet size (header only).
	MinPacketSize = HeaderLength

	// MaxPacketSize is the maximum UDP packet size (64KB - IP header).
	MaxPacketSize = 65535 - 20 // Max IP packet - min IP header
)

// Packet represents a UDP packet.
type Packet struct {
	// Header fields
	SourcePort      uint16 // Source port number
	DestinationPort uint16 // Destination port number
	Length          uint16 // Length of header + data (in bytes)
	Checksum        uint16 // Checksum (optional in IPv4, mandatory in IPv6)

	// Payload
	Data []byte // Packet data
}

// Parse parses a UDP packet from raw bytes.
func Parse(data []byte) (*Packet, error) {
	if len(data) < HeaderLength {
		return nil, fmt.Errorf("UDP packet too short: %d bytes (minimum %d)", len(data), HeaderLength)
	}

	pkt := &Packet{
		SourcePort:      binary.BigEndian.Uint16(data[0:2]),
		DestinationPort: binary.BigEndian.Uint16(data[2:4]),
		Length:          binary.BigEndian.Uint16(data[4:6]),
		Checksum:        binary.BigEndian.Uint16(data[6:8]),
	}

	// Validate length field
	if int(pkt.Length) < HeaderLength {
		return nil, fmt.Errorf("invalid UDP length: %d (minimum %d)", pkt.Length, HeaderLength)
	}

	if int(pkt.Length) > len(data) {
		return nil, fmt.Errorf("UDP length mismatch: header says %d, got %d bytes", pkt.Length, len(data))
	}

	// Extract data
	if int(pkt.Length) > HeaderLength {
		pkt.Data = make([]byte, int(pkt.Length)-HeaderLength)
		copy(pkt.Data, data[HeaderLength:pkt.Length])
	}

	return pkt, nil
}

// Serialize converts the UDP packet to bytes.
// Note: This does NOT calculate the checksum. Use CalculateChecksum separately.
func (p *Packet) Serialize() ([]byte, error) {
	// Calculate length
	length := HeaderLength + len(p.Data)
	if length > MaxPacketSize {
		return nil, fmt.Errorf("UDP packet too large: %d bytes (maximum %d)", length, MaxPacketSize)
	}
	p.Length = uint16(length)

	// Allocate buffer
	buf := make([]byte, length)

	// Set source and destination ports
	binary.BigEndian.PutUint16(buf[0:2], p.SourcePort)
	binary.BigEndian.PutUint16(buf[2:4], p.DestinationPort)

	// Set length
	binary.BigEndian.PutUint16(buf[4:6], p.Length)

	// Set checksum (caller should set this using CalculateChecksum)
	binary.BigEndian.PutUint16(buf[6:8], p.Checksum)

	// Copy data
	if len(p.Data) > 0 {
		copy(buf[HeaderLength:], p.Data)
	}

	return buf, nil
}

// CalculateChecksum calculates the UDP checksum with the given pseudo-header.
// The pseudo-header is constructed from the IP header fields:
// - Source IP (4 bytes)
// - Destination IP (4 bytes)
// - Zero byte (1 byte)
// - Protocol (1 byte) = 17 for UDP
// - UDP Length (2 bytes)
func (p *Packet) CalculateChecksum(srcIP, dstIP common.IPv4Address) (uint16, error) {
	// Serialize the UDP packet first
	udpData, err := p.Serialize()
	if err != nil {
		return 0, err
	}

	// Construct pseudo-header
	pseudoHeader := make([]byte, 12)
	copy(pseudoHeader[0:4], srcIP[:])
	copy(pseudoHeader[4:8], dstIP[:])
	pseudoHeader[8] = 0 // Zero
	pseudoHeader[9] = uint8(common.ProtocolUDP)
	binary.BigEndian.PutUint16(pseudoHeader[10:12], p.Length)

	// Combine pseudo-header and UDP packet
	combined := append(pseudoHeader, udpData...)

	// Calculate checksum
	checksum := common.CalculateChecksum(combined)

	// UDP checksum of 0 means no checksum, so if the calculated checksum is 0,
	// we should use 0xFFFF instead (per RFC 768)
	if checksum == 0 {
		checksum = 0xFFFF
	}

	return checksum, nil
}

// VerifyChecksum verifies the UDP checksum with the given pseudo-header.
func (p *Packet) VerifyChecksum(srcIP, dstIP common.IPv4Address) bool {
	// If checksum is 0, it means no checksum (which is allowed in IPv4)
	if p.Checksum == 0 {
		return true
	}

	// For verification, we check by calculating checksum of the whole thing
	// (including the checksum field) - it should equal 0 or 0xFFFF
	udpData, err := p.Serialize()
	if err != nil {
		return false
	}

	// Construct pseudo-header
	pseudoHeader := make([]byte, 12)
	copy(pseudoHeader[0:4], srcIP[:])
	copy(pseudoHeader[4:8], dstIP[:])
	pseudoHeader[8] = 0
	pseudoHeader[9] = uint8(common.ProtocolUDP)
	binary.BigEndian.PutUint16(pseudoHeader[10:12], p.Length)

	// Combine pseudo-header and UDP packet
	combined := append(pseudoHeader, udpData...)

	// Calculate checksum - should be 0 or 0xFFFF if valid
	checksum := common.CalculateChecksum(combined)

	return checksum == 0 || checksum == 0xFFFF
}

// String returns a human-readable representation of the UDP packet.
func (p *Packet) String() string {
	return fmt.Sprintf("UDP{SrcPort=%d, DstPort=%d, Len=%d, DataLen=%d}",
		p.SourcePort, p.DestinationPort, p.Length, len(p.Data))
}

// NewPacket creates a new UDP packet with the given parameters.
func NewPacket(srcPort, dstPort uint16, data []byte) *Packet {
	return &Packet{
		SourcePort:      srcPort,
		DestinationPort: dstPort,
		Length:          uint16(HeaderLength + len(data)),
		Checksum:        0, // Will be calculated later
		Data:            data,
	}
}
