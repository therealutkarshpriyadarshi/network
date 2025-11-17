// Package quic implements the QUIC protocol as defined in RFC 9000.
package quic

import (
	"encoding/binary"
	"fmt"
)

// PacketType represents the type of QUIC packet.
type PacketType uint8

const (
	// Long header packet types
	PacketTypeInitial      PacketType = 0x00
	PacketType0RTT         PacketType = 0x01
	PacketTypeHandshake    PacketType = 0x02
	PacketTypeRetry        PacketType = 0x03

	// Short header (1-RTT packet)
	PacketType1RTT         PacketType = 0x04

	// Version negotiation
	PacketTypeVersionNeg   PacketType = 0xFF
)

// QUIC version
const (
	Version1 uint32 = 0x00000001 // QUIC version 1 (RFC 9000)
)

// Packet represents a QUIC packet.
type Packet struct {
	// Header
	HeaderForm     uint8       // 1 bit: 0=short, 1=long
	FixedBit       uint8       // 1 bit: must be 1
	Type           PacketType  // Packet type
	Version        uint32      // QUIC version (long header only)
	DestConnID     []byte      // Destination Connection ID
	SrcConnID      []byte      // Source Connection ID (long header only)
	Token          []byte      // Token (Initial packets only)
	PacketNumber   uint64      // Packet number
	PacketNumLen   uint8       // Packet number length

	// Payload
	Payload        []byte      // Encrypted payload
}

// LongHeader represents a QUIC long header packet.
type LongHeader struct {
	HeaderForm   uint8
	FixedBit     uint8
	PacketType   PacketType
	TypeSpecific uint8
	Version      uint32
	DestConnID   []byte
	SrcConnID    []byte
	Payload      []byte
}

// ShortHeader represents a QUIC short header (1-RTT) packet.
type ShortHeader struct {
	HeaderForm   uint8
	FixedBit     uint8
	SpinBit      uint8
	Reserved     uint8
	KeyPhase     uint8
	PacketNumLen uint8
	DestConnID   []byte
	PacketNumber uint64
	Payload      []byte
}

// Parse parses a QUIC packet from raw bytes.
func Parse(data []byte) (*Packet, error) {
	if len(data) < 1 {
		return nil, fmt.Errorf("packet too short")
	}

	pkt := &Packet{}

	// Parse first byte
	firstByte := data[0]
	pkt.HeaderForm = (firstByte >> 7) & 0x01
	pkt.FixedBit = (firstByte >> 6) & 0x01

	if pkt.FixedBit != 1 {
		return nil, fmt.Errorf("invalid fixed bit")
	}

	offset := 1

	if pkt.HeaderForm == 1 {
		// Long header
		return parseLongHeader(data)
	} else {
		// Short header
		return parseShortHeader(data)
	}
}

// parseLongHeader parses a long header packet.
func parseLongHeader(data []byte) (*Packet, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("long header too short")
	}

	pkt := &Packet{}
	firstByte := data[0]

	pkt.HeaderForm = 1
	pkt.FixedBit = (firstByte >> 6) & 0x01
	pkt.Type = PacketType((firstByte >> 4) & 0x03)

	offset := 1

	// Version (4 bytes)
	pkt.Version = binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4

	// Destination Connection ID Length
	if offset >= len(data) {
		return nil, fmt.Errorf("packet truncated")
	}
	destConnIDLen := int(data[offset])
	offset++

	// Destination Connection ID
	if offset+destConnIDLen > len(data) {
		return nil, fmt.Errorf("packet truncated")
	}
	pkt.DestConnID = make([]byte, destConnIDLen)
	copy(pkt.DestConnID, data[offset:offset+destConnIDLen])
	offset += destConnIDLen

	// Source Connection ID Length
	if offset >= len(data) {
		return nil, fmt.Errorf("packet truncated")
	}
	srcConnIDLen := int(data[offset])
	offset++

	// Source Connection ID
	if offset+srcConnIDLen > len(data) {
		return nil, fmt.Errorf("packet truncated")
	}
	pkt.SrcConnID = make([]byte, srcConnIDLen)
	copy(pkt.SrcConnID, data[offset:offset+srcConnIDLen])
	offset += srcConnIDLen

	// Payload (simplified - actual QUIC has more fields)
	pkt.Payload = data[offset:]

	return pkt, nil
}

// parseShortHeader parses a short header packet.
func parseShortHeader(data []byte) (*Packet, error) {
	if len(data) < 1 {
		return nil, fmt.Errorf("short header too short")
	}

	pkt := &Packet{}
	firstByte := data[0]

	pkt.HeaderForm = 0
	pkt.FixedBit = (firstByte >> 6) & 0x01
	pkt.Type = PacketType1RTT

	// For short header, connection ID length is negotiated during handshake
	// For simplicity, assume a fixed length here
	offset := 1

	// Simplified - would need connection state to know actual conn ID length
	// Payload starts after connection ID and packet number
	pkt.Payload = data[offset:]

	return pkt, nil
}

// Serialize converts the packet to bytes.
func (p *Packet) Serialize() ([]byte, error) {
	if p.HeaderForm == 1 {
		return p.serializeLongHeader()
	}
	return p.serializeShortHeader()
}

// serializeLongHeader serializes a long header packet.
func (p *Packet) serializeLongHeader() ([]byte, error) {
	// Calculate size
	size := 1 + 4 + 1 + len(p.DestConnID) + 1 + len(p.SrcConnID) + len(p.Payload)
	if p.Type == PacketTypeInitial {
		size += 1 + len(p.Token) // Token length + token
	}

	buf := make([]byte, size)
	offset := 0

	// First byte
	firstByte := (p.HeaderForm << 7) | (p.FixedBit << 6) | (uint8(p.Type) << 4)
	buf[offset] = firstByte
	offset++

	// Version
	binary.BigEndian.PutUint32(buf[offset:], p.Version)
	offset += 4

	// Destination Connection ID
	buf[offset] = uint8(len(p.DestConnID))
	offset++
	copy(buf[offset:], p.DestConnID)
	offset += len(p.DestConnID)

	// Source Connection ID
	buf[offset] = uint8(len(p.SrcConnID))
	offset++
	copy(buf[offset:], p.SrcConnID)
	offset += len(p.SrcConnID)

	// Token (Initial packets only)
	if p.Type == PacketTypeInitial {
		buf[offset] = uint8(len(p.Token))
		offset++
		copy(buf[offset:], p.Token)
		offset += len(p.Token)
	}

	// Payload
	copy(buf[offset:], p.Payload)

	return buf, nil
}

// serializeShortHeader serializes a short header packet.
func (p *Packet) serializeShortHeader() ([]byte, error) {
	size := 1 + len(p.DestConnID) + len(p.Payload)
	buf := make([]byte, size)
	offset := 0

	// First byte
	firstByte := (p.HeaderForm << 7) | (p.FixedBit << 6)
	buf[offset] = firstByte
	offset++

	// Destination Connection ID
	copy(buf[offset:], p.DestConnID)
	offset += len(p.DestConnID)

	// Payload
	copy(buf[offset:], p.Payload)

	return buf, nil
}

// NewInitialPacket creates a new Initial packet.
func NewInitialPacket(destConnID, srcConnID []byte, token, payload []byte) *Packet {
	return &Packet{
		HeaderForm: 1,
		FixedBit:   1,
		Type:       PacketTypeInitial,
		Version:    Version1,
		DestConnID: destConnID,
		SrcConnID:  srcConnID,
		Token:      token,
		Payload:    payload,
	}
}

// NewHandshakePacket creates a new Handshake packet.
func NewHandshakePacket(destConnID, srcConnID, payload []byte) *Packet {
	return &Packet{
		HeaderForm: 1,
		FixedBit:   1,
		Type:       PacketTypeHandshake,
		Version:    Version1,
		DestConnID: destConnID,
		SrcConnID:  srcConnID,
		Payload:    payload,
	}
}

// New1RTTPacket creates a new 1-RTT packet.
func New1RTTPacket(destConnID, payload []byte) *Packet {
	return &Packet{
		HeaderForm: 0,
		FixedBit:   1,
		Type:       PacketType1RTT,
		DestConnID: destConnID,
		Payload:    payload,
	}
}

// String returns a human-readable representation of the packet.
func (p *Packet) String() string {
	typeStr := "Unknown"
	switch p.Type {
	case PacketTypeInitial:
		typeStr = "Initial"
	case PacketType0RTT:
		typeStr = "0-RTT"
	case PacketTypeHandshake:
		typeStr = "Handshake"
	case PacketTypeRetry:
		typeStr = "Retry"
	case PacketType1RTT:
		typeStr = "1-RTT"
	}

	return fmt.Sprintf("QUIC{Type=%s, Version=0x%08x, DestConnID=%x, PayloadLen=%d}",
		typeStr, p.Version, p.DestConnID, len(p.Payload))
}
