// Package icmp implements the Internet Control Message Protocol (ICMP) as defined in RFC 792.
package icmp

import (
	"encoding/binary"
	"fmt"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

// Type represents an ICMP message type.
type Type uint8

// Common ICMP types.
const (
	TypeEchoReply              Type = 0  // Echo Reply
	TypeDestinationUnreachable Type = 3  // Destination Unreachable
	TypeSourceQuench           Type = 4  // Source Quench (deprecated)
	TypeRedirect               Type = 5  // Redirect
	TypeEchoRequest            Type = 8  // Echo Request
	TypeTimeExceeded           Type = 11 // Time Exceeded
	TypeParameterProblem       Type = 12 // Parameter Problem
	TypeTimestampRequest       Type = 13 // Timestamp Request
	TypeTimestampReply         Type = 14 // Timestamp Reply
)

// Code represents an ICMP message code.
type Code uint8

// Destination Unreachable codes.
const (
	CodeNetUnreachable     Code = 0  // Network Unreachable
	CodeHostUnreachable    Code = 1  // Host Unreachable
	CodeProtocolUnreachable Code = 2  // Protocol Unreachable
	CodePortUnreachable    Code = 3  // Port Unreachable
	CodeFragmentationNeeded Code = 4  // Fragmentation Needed but DF Set
	CodeSourceRouteFailed  Code = 5  // Source Route Failed
)

// Time Exceeded codes.
const (
	CodeTTLExceeded           Code = 0 // TTL Exceeded in Transit
	CodeFragmentReassemblyTime Code = 1 // Fragment Reassembly Time Exceeded
)

const (
	// MinHeaderLength is the minimum ICMP header length (8 bytes).
	MinHeaderLength = 8
)

// Message represents an ICMP message.
type Message struct {
	Type     Type   // ICMP type
	Code     Code   // ICMP code
	Checksum uint16 // Checksum
	ID       uint16 // Identifier (for echo request/reply)
	Sequence uint16 // Sequence number (for echo request/reply)
	Data     []byte // Message data
}

// Parse parses an ICMP message from raw bytes.
func Parse(data []byte) (*Message, error) {
	if len(data) < MinHeaderLength {
		return nil, fmt.Errorf("ICMP message too short: %d bytes (minimum %d)", len(data), MinHeaderLength)
	}

	msg := &Message{
		Type:     Type(data[0]),
		Code:     Code(data[1]),
		Checksum: binary.BigEndian.Uint16(data[2:4]),
		ID:       binary.BigEndian.Uint16(data[4:6]),
		Sequence: binary.BigEndian.Uint16(data[6:8]),
	}

	// Copy data after header
	if len(data) > MinHeaderLength {
		msg.Data = make([]byte, len(data)-MinHeaderLength)
		copy(msg.Data, data[MinHeaderLength:])
	}

	return msg, nil
}

// Serialize converts the ICMP message to bytes.
func (m *Message) Serialize() ([]byte, error) {
	length := MinHeaderLength + len(m.Data)
	buf := make([]byte, length)

	// Set type and code
	buf[0] = uint8(m.Type)
	buf[1] = uint8(m.Code)

	// Set checksum to 0 for calculation
	buf[2] = 0
	buf[3] = 0

	// Set ID and sequence
	binary.BigEndian.PutUint16(buf[4:6], m.ID)
	binary.BigEndian.PutUint16(buf[6:8], m.Sequence)

	// Copy data
	if len(m.Data) > 0 {
		copy(buf[MinHeaderLength:], m.Data)
	}

	// Calculate checksum
	m.Checksum = common.CalculateChecksum(buf)
	binary.BigEndian.PutUint16(buf[2:4], m.Checksum)

	return buf, nil
}

// VerifyChecksum verifies the ICMP checksum.
func (m *Message) VerifyChecksum() bool {
	// Serialize and check if checksum is 0
	buf := make([]byte, MinHeaderLength+len(m.Data))
	buf[0] = uint8(m.Type)
	buf[1] = uint8(m.Code)
	binary.BigEndian.PutUint16(buf[2:4], m.Checksum)
	binary.BigEndian.PutUint16(buf[4:6], m.ID)
	binary.BigEndian.PutUint16(buf[6:8], m.Sequence)
	if len(m.Data) > 0 {
		copy(buf[MinHeaderLength:], m.Data)
	}

	return common.CalculateChecksum(buf) == 0
}

// String returns a human-readable representation of the ICMP message.
func (m *Message) String() string {
	return fmt.Sprintf("ICMP{Type=%s(%d), Code=%d, ID=%d, Seq=%d, DataLen=%d}",
		m.Type, uint8(m.Type), m.Code, m.ID, m.Sequence, len(m.Data))
}

// String returns a human-readable name for the ICMP type.
func (t Type) String() string {
	switch t {
	case TypeEchoReply:
		return "EchoReply"
	case TypeDestinationUnreachable:
		return "DestinationUnreachable"
	case TypeSourceQuench:
		return "SourceQuench"
	case TypeRedirect:
		return "Redirect"
	case TypeEchoRequest:
		return "EchoRequest"
	case TypeTimeExceeded:
		return "TimeExceeded"
	case TypeParameterProblem:
		return "ParameterProblem"
	case TypeTimestampRequest:
		return "TimestampRequest"
	case TypeTimestampReply:
		return "TimestampReply"
	default:
		return fmt.Sprintf("Unknown(%d)", uint8(t))
	}
}

// NewEchoRequest creates a new ICMP Echo Request message.
func NewEchoRequest(id, sequence uint16, data []byte) *Message {
	return &Message{
		Type:     TypeEchoRequest,
		Code:     0,
		ID:       id,
		Sequence: sequence,
		Data:     data,
	}
}

// NewEchoReply creates a new ICMP Echo Reply message.
func NewEchoReply(id, sequence uint16, data []byte) *Message {
	return &Message{
		Type:     TypeEchoReply,
		Code:     0,
		ID:       id,
		Sequence: sequence,
		Data:     data,
	}
}

// NewDestinationUnreachable creates a new ICMP Destination Unreachable message.
func NewDestinationUnreachable(code Code, data []byte) *Message {
	return &Message{
		Type:     TypeDestinationUnreachable,
		Code:     code,
		ID:       0,
		Sequence: 0,
		Data:     data,
	}
}

// NewTimeExceeded creates a new ICMP Time Exceeded message.
func NewTimeExceeded(code Code, data []byte) *Message {
	return &Message{
		Type:     TypeTimeExceeded,
		Code:     code,
		ID:       0,
		Sequence: 0,
		Data:     data,
	}
}

// IsEchoRequest returns true if this is an Echo Request message.
func (m *Message) IsEchoRequest() bool {
	return m.Type == TypeEchoRequest
}

// IsEchoReply returns true if this is an Echo Reply message.
func (m *Message) IsEchoReply() bool {
	return m.Type == TypeEchoReply
}

// IsError returns true if this is an error message.
func (m *Message) IsError() bool {
	return m.Type == TypeDestinationUnreachable ||
		m.Type == TypeSourceQuench ||
		m.Type == TypeRedirect ||
		m.Type == TypeTimeExceeded ||
		m.Type == TypeParameterProblem
}
