// Package ip implements the Internet Protocol version 4 (IPv4) as defined in RFC 791.
package ip

import (
	"encoding/binary"
	"fmt"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

const (
	// IPv4Version is the version number for IPv4.
	IPv4Version = 4

	// MinHeaderLength is the minimum IPv4 header length (20 bytes).
	MinHeaderLength = 20

	// MaxHeaderLength is the maximum IPv4 header length (60 bytes).
	MaxHeaderLength = 60

	// MaxPacketSize is the maximum IPv4 packet size (64KB).
	MaxPacketSize = 65535

	// DefaultTTL is the default Time To Live value.
	DefaultTTL = 64
)

// IPv4Flags represents the flags in the IPv4 header.
type IPv4Flags uint8

const (
	// FlagReserved is the reserved flag (must be zero).
	FlagReserved IPv4Flags = 1 << 2

	// FlagDontFragment indicates that the packet should not be fragmented.
	FlagDontFragment IPv4Flags = 1 << 1

	// FlagMoreFragments indicates that more fragments follow.
	FlagMoreFragments IPv4Flags = 1 << 0
)

// Packet represents an IPv4 packet.
type Packet struct {
	// Header fields
	Version        uint8              // 4 bits: IP version (should be 4)
	IHL            uint8              // 4 bits: Internet Header Length (in 32-bit words)
	DSCP           uint8              // 6 bits: Differentiated Services Code Point
	ECN            uint8              // 2 bits: Explicit Congestion Notification
	TotalLength    uint16             // Total packet length (header + data)
	Identification uint16             // Fragment identification
	Flags          IPv4Flags          // Flags (Reserved, DF, MF)
	FragmentOffset uint16             // Fragment offset (in 8-byte blocks)
	TTL            uint8              // Time To Live
	Protocol       common.Protocol    // Protocol (TCP, UDP, ICMP, etc.)
	Checksum       uint16             // Header checksum
	Source         common.IPv4Address // Source IP address
	Destination    common.IPv4Address // Destination IP address
	Options        []byte             // IP options (if IHL > 5)

	// Payload
	Payload []byte // Packet payload
}

// Parse parses an IPv4 packet from raw bytes.
func Parse(data []byte) (*Packet, error) {
	if len(data) < MinHeaderLength {
		return nil, fmt.Errorf("packet too short: %d bytes (minimum %d)", len(data), MinHeaderLength)
	}

	pkt := &Packet{}

	// Parse version and IHL (first byte)
	versionIHL := data[0]
	pkt.Version = versionIHL >> 4
	pkt.IHL = versionIHL & 0x0F

	if pkt.Version != IPv4Version {
		return nil, fmt.Errorf("invalid IP version: %d (expected %d)", pkt.Version, IPv4Version)
	}

	if pkt.IHL < 5 {
		return nil, fmt.Errorf("invalid IHL: %d (minimum 5)", pkt.IHL)
	}

	headerLength := int(pkt.IHL) * 4
	if len(data) < headerLength {
		return nil, fmt.Errorf("packet too short for header: %d bytes (expected %d)", len(data), headerLength)
	}

	// Parse DSCP and ECN (second byte)
	dscpECN := data[1]
	pkt.DSCP = dscpECN >> 2
	pkt.ECN = dscpECN & 0x03

	// Parse total length
	pkt.TotalLength = binary.BigEndian.Uint16(data[2:4])
	if int(pkt.TotalLength) > len(data) {
		return nil, fmt.Errorf("total length mismatch: header says %d, got %d bytes", pkt.TotalLength, len(data))
	}

	// Parse identification
	pkt.Identification = binary.BigEndian.Uint16(data[4:6])

	// Parse flags and fragment offset
	flagsFragOffset := binary.BigEndian.Uint16(data[6:8])
	pkt.Flags = IPv4Flags(flagsFragOffset >> 13)
	pkt.FragmentOffset = flagsFragOffset & 0x1FFF

	// Parse TTL
	pkt.TTL = data[8]

	// Parse protocol
	pkt.Protocol = common.Protocol(data[9])

	// Parse checksum
	pkt.Checksum = binary.BigEndian.Uint16(data[10:12])

	// Parse source and destination addresses
	copy(pkt.Source[:], data[12:16])
	copy(pkt.Destination[:], data[16:20])

	// Parse options if present
	if pkt.IHL > 5 {
		optionsLength := headerLength - MinHeaderLength
		pkt.Options = make([]byte, optionsLength)
		copy(pkt.Options, data[20:headerLength])
	}

	// Extract payload
	pkt.Payload = data[headerLength:pkt.TotalLength]

	return pkt, nil
}

// Serialize converts the packet to bytes.
func (p *Packet) Serialize() ([]byte, error) {
	// Calculate header length
	headerLength := MinHeaderLength
	if len(p.Options) > 0 {
		// Options must be padded to 4-byte boundary
		optionsLength := len(p.Options)
		if optionsLength%4 != 0 {
			optionsLength = (optionsLength/4 + 1) * 4
		}
		headerLength += optionsLength
	}

	if headerLength > MaxHeaderLength {
		return nil, fmt.Errorf("header too long: %d bytes (maximum %d)", headerLength, MaxHeaderLength)
	}

	// Update IHL
	p.IHL = uint8(headerLength / 4)

	// Calculate total length
	totalLength := headerLength + len(p.Payload)
	if totalLength > MaxPacketSize {
		return nil, fmt.Errorf("packet too large: %d bytes (maximum %d)", totalLength, MaxPacketSize)
	}
	p.TotalLength = uint16(totalLength)

	// Allocate buffer
	buf := make([]byte, totalLength)

	// Set version and IHL
	buf[0] = (p.Version << 4) | p.IHL

	// Set DSCP and ECN
	buf[1] = (p.DSCP << 2) | p.ECN

	// Set total length
	binary.BigEndian.PutUint16(buf[2:4], p.TotalLength)

	// Set identification
	binary.BigEndian.PutUint16(buf[4:6], p.Identification)

	// Set flags and fragment offset
	flagsFragOffset := (uint16(p.Flags) << 13) | (p.FragmentOffset & 0x1FFF)
	binary.BigEndian.PutUint16(buf[6:8], flagsFragOffset)

	// Set TTL
	buf[8] = p.TTL

	// Set protocol
	buf[9] = uint8(p.Protocol)

	// Set checksum to 0 before calculation
	buf[10] = 0
	buf[11] = 0

	// Set source and destination
	copy(buf[12:16], p.Source[:])
	copy(buf[16:20], p.Destination[:])

	// Copy options if present
	if len(p.Options) > 0 {
		copy(buf[20:], p.Options)
		// Pad with zeros if necessary
		for i := 20 + len(p.Options); i < headerLength; i++ {
			buf[i] = 0
		}
	}

	// Calculate and set checksum
	p.Checksum = common.CalculateChecksum(buf[:headerLength])
	binary.BigEndian.PutUint16(buf[10:12], p.Checksum)

	// Copy payload
	copy(buf[headerLength:], p.Payload)

	return buf, nil
}

// VerifyChecksum verifies the IP header checksum.
func (p *Packet) VerifyChecksum() bool {
	// Reconstruct the header for checksum verification
	headerLength := int(p.IHL) * 4
	buf := make([]byte, headerLength)

	buf[0] = (p.Version << 4) | p.IHL
	buf[1] = (p.DSCP << 2) | p.ECN
	binary.BigEndian.PutUint16(buf[2:4], p.TotalLength)
	binary.BigEndian.PutUint16(buf[4:6], p.Identification)
	flagsFragOffset := (uint16(p.Flags) << 13) | (p.FragmentOffset & 0x1FFF)
	binary.BigEndian.PutUint16(buf[6:8], flagsFragOffset)
	buf[8] = p.TTL
	buf[9] = uint8(p.Protocol)
	binary.BigEndian.PutUint16(buf[10:12], p.Checksum)
	copy(buf[12:16], p.Source[:])
	copy(buf[16:20], p.Destination[:])
	if len(p.Options) > 0 {
		copy(buf[20:], p.Options)
	}

	// Checksum should be 0 if correct
	return common.CalculateChecksum(buf) == 0
}

// DecrementTTL decrements the TTL and returns true if the packet is still alive.
func (p *Packet) DecrementTTL() bool {
	if p.TTL == 0 {
		return false
	}
	p.TTL--
	return p.TTL > 0
}

// IsFragment returns true if this packet is a fragment.
func (p *Packet) IsFragment() bool {
	return p.FragmentOffset != 0 || (p.Flags&FlagMoreFragments) != 0
}

// String returns a human-readable representation of the packet.
func (p *Packet) String() string {
	return fmt.Sprintf("IPv4{%s -> %s, Proto=%s, TTL=%d, ID=%d, Len=%d}",
		p.Source, p.Destination, p.Protocol, p.TTL, p.Identification, p.TotalLength)
}

// NewPacket creates a new IPv4 packet with default values.
func NewPacket(src, dst common.IPv4Address, protocol common.Protocol, payload []byte) *Packet {
	return &Packet{
		Version:        IPv4Version,
		IHL:            5, // No options by default
		DSCP:           0,
		ECN:            0,
		TotalLength:    0, // Will be calculated in Serialize
		Identification: 0, // Should be set by caller or fragment handler
		Flags:          0,
		FragmentOffset: 0,
		TTL:            DefaultTTL,
		Protocol:       protocol,
		Checksum:       0, // Will be calculated in Serialize
		Source:         src,
		Destination:    dst,
		Options:        nil,
		Payload:        payload,
	}
}
