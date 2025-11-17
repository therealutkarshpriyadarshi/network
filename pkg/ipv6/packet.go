// Package ipv6 implements the Internet Protocol version 6 (IPv6) as defined in RFC 2460.
package ipv6

import (
	"encoding/binary"
	"fmt"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

const (
	// IPv6Version is the version number for IPv6.
	IPv6Version = 6

	// HeaderLength is the fixed IPv6 header length (40 bytes).
	HeaderLength = 40

	// MaxPacketSize is the maximum IPv6 packet size without jumbogram (64KB).
	MaxPacketSize = 65535

	// DefaultHopLimit is the default Hop Limit value.
	DefaultHopLimit = 64
)

// Packet represents an IPv6 packet.
type Packet struct {
	// Header fields
	Version      uint8              // 4 bits: IP version (should be 6)
	TrafficClass uint8              // 8 bits: Traffic class
	FlowLabel    uint32             // 20 bits: Flow label
	PayloadLen   uint16             // Payload length (excludes header)
	NextHeader   common.Protocol    // Next header protocol
	HopLimit     uint8              // Hop limit (like TTL in IPv4)
	Source       common.IPv6Address // Source IPv6 address
	Destination  common.IPv6Address // Destination IPv6 address

	// Extension Headers (optional)
	ExtHeaders []ExtensionHeader

	// Payload
	Payload []byte // Packet payload
}

// ExtensionHeader represents an IPv6 extension header.
type ExtensionHeader struct {
	NextHeader common.Protocol
	Data       []byte
}

// Parse parses an IPv6 packet from raw bytes.
func Parse(data []byte) (*Packet, error) {
	if len(data) < HeaderLength {
		return nil, fmt.Errorf("packet too short: %d bytes (minimum %d)", len(data), HeaderLength)
	}

	pkt := &Packet{}

	// Parse version, traffic class, and flow label (first 4 bytes)
	versionTCFlow := binary.BigEndian.Uint32(data[0:4])
	pkt.Version = uint8(versionTCFlow >> 28)
	pkt.TrafficClass = uint8((versionTCFlow >> 20) & 0xFF)
	pkt.FlowLabel = versionTCFlow & 0xFFFFF

	if pkt.Version != IPv6Version {
		return nil, fmt.Errorf("invalid IP version: %d (expected %d)", pkt.Version, IPv6Version)
	}

	// Parse payload length
	pkt.PayloadLen = binary.BigEndian.Uint16(data[4:6])

	// Parse next header and hop limit
	pkt.NextHeader = common.Protocol(data[6])
	pkt.HopLimit = data[7]

	// Parse source and destination addresses
	copy(pkt.Source[:], data[8:24])
	copy(pkt.Destination[:], data[24:40])

	// Extract payload (and potentially extension headers)
	if len(data) > HeaderLength {
		payloadData := data[HeaderLength:]
		if int(pkt.PayloadLen) > len(payloadData) {
			return nil, fmt.Errorf("payload length mismatch: header says %d, got %d bytes", pkt.PayloadLen, len(payloadData))
		}
		pkt.Payload = payloadData[:pkt.PayloadLen]
	}

	return pkt, nil
}

// Serialize converts the packet to bytes.
func (p *Packet) Serialize() ([]byte, error) {
	// Calculate payload length
	payloadLen := len(p.Payload)
	for _, ext := range p.ExtHeaders {
		payloadLen += len(ext.Data)
	}

	if payloadLen > MaxPacketSize {
		return nil, fmt.Errorf("payload too large: %d bytes (maximum %d)", payloadLen, MaxPacketSize)
	}

	p.PayloadLen = uint16(payloadLen)

	// Allocate buffer
	totalLen := HeaderLength + payloadLen
	buf := make([]byte, totalLen)

	// Set version, traffic class, and flow label
	versionTCFlow := (uint32(p.Version) << 28) | (uint32(p.TrafficClass) << 20) | (p.FlowLabel & 0xFFFFF)
	binary.BigEndian.PutUint32(buf[0:4], versionTCFlow)

	// Set payload length
	binary.BigEndian.PutUint16(buf[4:6], p.PayloadLen)

	// Set next header and hop limit
	buf[6] = uint8(p.NextHeader)
	buf[7] = p.HopLimit

	// Set source and destination
	copy(buf[8:24], p.Source[:])
	copy(buf[24:40], p.Destination[:])

	// Copy extension headers and payload
	offset := HeaderLength
	for _, ext := range p.ExtHeaders {
		copy(buf[offset:], ext.Data)
		offset += len(ext.Data)
	}
	copy(buf[offset:], p.Payload)

	return buf, nil
}

// DecrementHopLimit decrements the hop limit and returns true if the packet is still alive.
func (p *Packet) DecrementHopLimit() bool {
	if p.HopLimit == 0 {
		return false
	}
	p.HopLimit--
	return p.HopLimit > 0
}

// String returns a human-readable representation of the packet.
func (p *Packet) String() string {
	return fmt.Sprintf("IPv6{%s -> %s, Proto=%s, HopLimit=%d, PayloadLen=%d}",
		p.Source, p.Destination, p.NextHeader, p.HopLimit, p.PayloadLen)
}

// NewPacket creates a new IPv6 packet with default values.
func NewPacket(src, dst common.IPv6Address, protocol common.Protocol, payload []byte) *Packet {
	return &Packet{
		Version:      IPv6Version,
		TrafficClass: 0,
		FlowLabel:    0,
		PayloadLen:   0, // Will be calculated in Serialize
		NextHeader:   protocol,
		HopLimit:     DefaultHopLimit,
		Source:       src,
		Destination:  dst,
		ExtHeaders:   nil,
		Payload:      payload,
	}
}
