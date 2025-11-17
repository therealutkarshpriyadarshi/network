// Package multicast implements IGMP (Internet Group Management Protocol) for IPv4 multicast.
package multicast

import (
	"encoding/binary"
	"fmt"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

// IGMP message types
const (
	IGMPMembershipQuery    uint8 = 0x11 // IGMP Membership Query
	IGMPv1MembershipReport uint8 = 0x12 // IGMPv1 Membership Report
	IGMPv2MembershipReport uint8 = 0x16 // IGMPv2 Membership Report
	IGMPv2LeaveGroup       uint8 = 0x17 // IGMPv2 Leave Group
	IGMPv3MembershipReport uint8 = 0x22 // IGMPv3 Membership Report
)

// IGMP header length
const IGMPHeaderLen = 8

// IGMPMessage represents an IGMP message.
type IGMPMessage struct {
	Type            uint8              // IGMP message type
	MaxRespTime     uint8              // Max response time (in deciseconds)
	Checksum        uint16             // Checksum
	GroupAddress    common.IPv4Address // Multicast group address
	AdditionalData  []byte             // Additional data for IGMPv3
}

// ParseIGMP parses an IGMP message from bytes.
func ParseIGMP(data []byte) (*IGMPMessage, error) {
	if len(data) < IGMPHeaderLen {
		return nil, fmt.Errorf("IGMP message too short: %d bytes", len(data))
	}

	msg := &IGMPMessage{
		Type:        data[0],
		MaxRespTime: data[1],
		Checksum:    binary.BigEndian.Uint16(data[2:4]),
	}

	copy(msg.GroupAddress[:], data[4:8])

	if len(data) > IGMPHeaderLen {
		msg.AdditionalData = make([]byte, len(data)-IGMPHeaderLen)
		copy(msg.AdditionalData, data[IGMPHeaderLen:])
	}

	return msg, nil
}

// Serialize serializes the IGMP message to bytes.
func (m *IGMPMessage) Serialize() ([]byte, error) {
	length := IGMPHeaderLen + len(m.AdditionalData)
	buf := make([]byte, length)

	buf[0] = m.Type
	buf[1] = m.MaxRespTime
	// Checksum will be calculated later
	binary.BigEndian.PutUint16(buf[2:4], 0)
	copy(buf[4:8], m.GroupAddress[:])

	if len(m.AdditionalData) > 0 {
		copy(buf[IGMPHeaderLen:], m.AdditionalData)
	}

	// Calculate checksum
	m.Checksum = common.CalculateChecksum(buf)
	binary.BigEndian.PutUint16(buf[2:4], m.Checksum)

	return buf, nil
}

// VerifyChecksum verifies the IGMP checksum.
func (m *IGMPMessage) VerifyChecksum(data []byte) bool {
	return common.CalculateChecksum(data) == 0
}

// NewMembershipQuery creates a new IGMP membership query.
func NewMembershipQuery(groupAddr common.IPv4Address, maxRespTime uint8) *IGMPMessage {
	return &IGMPMessage{
		Type:         IGMPMembershipQuery,
		MaxRespTime:  maxRespTime,
		GroupAddress: groupAddr,
	}
}

// NewMembershipReport creates a new IGMPv2 membership report.
func NewMembershipReport(groupAddr common.IPv4Address) *IGMPMessage {
	return &IGMPMessage{
		Type:         IGMPv2MembershipReport,
		MaxRespTime:  0,
		GroupAddress: groupAddr,
	}
}

// NewLeaveGroup creates a new IGMP leave group message.
func NewLeaveGroup(groupAddr common.IPv4Address) *IGMPMessage {
	return &IGMPMessage{
		Type:         IGMPv2LeaveGroup,
		MaxRespTime:  0,
		GroupAddress: groupAddr,
	}
}

// String returns a string representation of the IGMP message.
func (m *IGMPMessage) String() string {
	typeStr := "Unknown"
	switch m.Type {
	case IGMPMembershipQuery:
		typeStr = "Query"
	case IGMPv1MembershipReport:
		typeStr = "Report(v1)"
	case IGMPv2MembershipReport:
		typeStr = "Report(v2)"
	case IGMPv2LeaveGroup:
		typeStr = "Leave"
	case IGMPv3MembershipReport:
		typeStr = "Report(v3)"
	}

	return fmt.Sprintf("IGMP{Type=%s, Group=%s, MaxRespTime=%d}",
		typeStr, m.GroupAddress, m.MaxRespTime)
}
