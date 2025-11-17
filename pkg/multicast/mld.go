// Package multicast implements MLD (Multicast Listener Discovery) for IPv6 multicast.
package multicast

import (
	"encoding/binary"
	"fmt"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

// MLD message types (ICMPv6 types)
const (
	MLDQuery  uint8 = 130 // Multicast Listener Query
	MLDReport uint8 = 131 // Multicast Listener Report
	MLDDone   uint8 = 132 // Multicast Listener Done
	MLDv2Report uint8 = 143 // MLDv2 Multicast Listener Report
)

// MLD header length
const MLDHeaderLen = 24

// MLDMessage represents an MLD message.
type MLDMessage struct {
	Type              uint8              // ICMPv6 message type
	Code              uint8              // Code (usually 0)
	Checksum          uint16             // Checksum
	MaxRespDelay      uint16             // Maximum response delay (in milliseconds)
	Reserved          uint16             // Reserved field
	MulticastAddress  common.IPv6Address // Multicast address
	AdditionalData    []byte             // Additional data for MLDv2
}

// ParseMLD parses an MLD message from bytes.
func ParseMLD(data []byte) (*MLDMessage, error) {
	if len(data) < MLDHeaderLen {
		return nil, fmt.Errorf("MLD message too short: %d bytes", len(data))
	}

	msg := &MLDMessage{
		Type:         data[0],
		Code:         data[1],
		Checksum:     binary.BigEndian.Uint16(data[2:4]),
		MaxRespDelay: binary.BigEndian.Uint16(data[4:6]),
		Reserved:     binary.BigEndian.Uint16(data[6:8]),
	}

	copy(msg.MulticastAddress[:], data[8:24])

	if len(data) > MLDHeaderLen {
		msg.AdditionalData = make([]byte, len(data)-MLDHeaderLen)
		copy(msg.AdditionalData, data[MLDHeaderLen:])
	}

	return msg, nil
}

// Serialize serializes the MLD message to bytes.
func (m *MLDMessage) Serialize() ([]byte, error) {
	length := MLDHeaderLen + len(m.AdditionalData)
	buf := make([]byte, length)

	buf[0] = m.Type
	buf[1] = m.Code
	// Checksum will be calculated later
	binary.BigEndian.PutUint16(buf[2:4], 0)
	binary.BigEndian.PutUint16(buf[4:6], m.MaxRespDelay)
	binary.BigEndian.PutUint16(buf[6:8], m.Reserved)
	copy(buf[8:24], m.MulticastAddress[:])

	if len(m.AdditionalData) > 0 {
		copy(buf[MLDHeaderLen:], m.AdditionalData)
	}

	// Note: Actual checksum calculation would require IPv6 pseudo-header
	// This is simplified
	m.Checksum = common.CalculateChecksum(buf)
	binary.BigEndian.PutUint16(buf[2:4], m.Checksum)

	return buf, nil
}

// NewMLDQuery creates a new MLD query message.
func NewMLDQuery(multicastAddr common.IPv6Address, maxRespDelay uint16) *MLDMessage {
	return &MLDMessage{
		Type:             MLDQuery,
		Code:             0,
		MaxRespDelay:     maxRespDelay,
		MulticastAddress: multicastAddr,
	}
}

// NewMLDReport creates a new MLD report message.
func NewMLDReport(multicastAddr common.IPv6Address) *MLDMessage {
	return &MLDMessage{
		Type:             MLDReport,
		Code:             0,
		MaxRespDelay:     0,
		MulticastAddress: multicastAddr,
	}
}

// NewMLDDone creates a new MLD done message.
func NewMLDDone(multicastAddr common.IPv6Address) *MLDMessage {
	return &MLDMessage{
		Type:             MLDDone,
		Code:             0,
		MaxRespDelay:     0,
		MulticastAddress: multicastAddr,
	}
}

// String returns a string representation of the MLD message.
func (m *MLDMessage) String() string {
	typeStr := "Unknown"
	switch m.Type {
	case MLDQuery:
		typeStr = "Query"
	case MLDReport:
		typeStr = "Report"
	case MLDDone:
		typeStr = "Done"
	case MLDv2Report:
		typeStr = "Report(v2)"
	}

	return fmt.Sprintf("MLD{Type=%s, Group=%s, MaxRespDelay=%dms}",
		typeStr, m.MulticastAddress, m.MaxRespDelay)
}
