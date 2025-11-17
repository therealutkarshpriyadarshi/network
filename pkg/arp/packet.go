// Package arp implements the Address Resolution Protocol (ARP) for IPv4.
// ARP is used to map IP addresses to MAC addresses on a local network.
package arp

import (
	"encoding/binary"
	"fmt"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

// ARP packet format (RFC 826):
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |        Hardware Type          |        Protocol Type          |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// | HW Addr Len | Proto Addr Len|          Operation            |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                 Sender Hardware Address (6 bytes)             |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                 Sender Protocol Address (4 bytes)             |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                 Target Hardware Address (6 bytes)             |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                 Target Protocol Address (4 bytes)             |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

const (
	// PacketSize is the size of an ARP packet for Ethernet/IPv4 (28 bytes).
	PacketSize = 28

	// HardwareTypeEthernet represents Ethernet hardware type.
	HardwareTypeEthernet = 1

	// ProtocolTypeIPv4 represents IPv4 protocol type (same as EtherType).
	ProtocolTypeIPv4 = 0x0800
)

// Operation represents the ARP operation type.
type Operation uint16

const (
	// OperationRequest is an ARP request (who has this IP?).
	OperationRequest Operation = 1

	// OperationReply is an ARP reply (I have this IP, here's my MAC).
	OperationReply Operation = 2
)

// String returns a human-readable representation of the operation.
func (op Operation) String() string {
	switch op {
	case OperationRequest:
		return "Request"
	case OperationReply:
		return "Reply"
	default:
		return fmt.Sprintf("Unknown(%d)", uint16(op))
	}
}

// Packet represents an ARP packet.
type Packet struct {
	HardwareType   uint16              // Hardware type (1 for Ethernet)
	ProtocolType   uint16              // Protocol type (0x0800 for IPv4)
	HardwareLength uint8               // Hardware address length (6 for Ethernet)
	ProtocolLength uint8               // Protocol address length (4 for IPv4)
	Operation      Operation           // Operation (request or reply)
	SenderMAC      common.MACAddress   // Sender hardware address
	SenderIP       common.IPv4Address  // Sender protocol address
	TargetMAC      common.MACAddress   // Target hardware address
	TargetIP       common.IPv4Address  // Target protocol address
}

// Parse parses an ARP packet from raw bytes.
func Parse(data []byte) (*Packet, error) {
	if len(data) < PacketSize {
		return nil, fmt.Errorf("ARP packet too short: %d bytes (expected %d)", len(data), PacketSize)
	}

	packet := &Packet{}

	// Parse header fields (8 bytes)
	packet.HardwareType = binary.BigEndian.Uint16(data[0:2])
	packet.ProtocolType = binary.BigEndian.Uint16(data[2:4])
	packet.HardwareLength = data[4]
	packet.ProtocolLength = data[5]
	packet.Operation = Operation(binary.BigEndian.Uint16(data[6:8]))

	// Validate field values
	if packet.HardwareType != HardwareTypeEthernet {
		return nil, fmt.Errorf("unsupported hardware type: %d", packet.HardwareType)
	}
	if packet.ProtocolType != ProtocolTypeIPv4 {
		return nil, fmt.Errorf("unsupported protocol type: 0x%04x", packet.ProtocolType)
	}
	if packet.HardwareLength != 6 {
		return nil, fmt.Errorf("invalid hardware address length: %d", packet.HardwareLength)
	}
	if packet.ProtocolLength != 4 {
		return nil, fmt.Errorf("invalid protocol address length: %d", packet.ProtocolLength)
	}

	// Parse addresses (20 bytes)
	copy(packet.SenderMAC[:], data[8:14])
	copy(packet.SenderIP[:], data[14:18])
	copy(packet.TargetMAC[:], data[18:24])
	copy(packet.TargetIP[:], data[24:28])

	return packet, nil
}

// Serialize converts the ARP packet to bytes for transmission.
func (p *Packet) Serialize() []byte {
	data := make([]byte, PacketSize)

	// Write header fields
	binary.BigEndian.PutUint16(data[0:2], p.HardwareType)
	binary.BigEndian.PutUint16(data[2:4], p.ProtocolType)
	data[4] = p.HardwareLength
	data[5] = p.ProtocolLength
	binary.BigEndian.PutUint16(data[6:8], uint16(p.Operation))

	// Write addresses
	copy(data[8:14], p.SenderMAC[:])
	copy(data[14:18], p.SenderIP[:])
	copy(data[18:24], p.TargetMAC[:])
	copy(data[24:28], p.TargetIP[:])

	return data
}

// String returns a human-readable representation of the packet.
func (p *Packet) String() string {
	return fmt.Sprintf("ARP{Op=%s, Sender=%s(%s), Target=%s(%s)}",
		p.Operation,
		p.SenderIP,
		p.SenderMAC,
		p.TargetIP,
		p.TargetMAC,
	)
}

// NewRequest creates a new ARP request packet.
// This is used to ask "who has targetIP? Tell senderIP".
func NewRequest(senderMAC common.MACAddress, senderIP, targetIP common.IPv4Address) *Packet {
	return &Packet{
		HardwareType:   HardwareTypeEthernet,
		ProtocolType:   ProtocolTypeIPv4,
		HardwareLength: 6,
		ProtocolLength: 4,
		Operation:      OperationRequest,
		SenderMAC:      senderMAC,
		SenderIP:       senderIP,
		TargetMAC:      common.MACAddress{}, // Unknown (00:00:00:00:00:00)
		TargetIP:       targetIP,
	}
}

// NewReply creates a new ARP reply packet.
// This is used to respond "targetIP is at targetMAC".
func NewReply(senderMAC common.MACAddress, senderIP common.IPv4Address, targetMAC common.MACAddress, targetIP common.IPv4Address) *Packet {
	return &Packet{
		HardwareType:   HardwareTypeEthernet,
		ProtocolType:   ProtocolTypeIPv4,
		HardwareLength: 6,
		ProtocolLength: 4,
		Operation:      OperationReply,
		SenderMAC:      senderMAC,
		SenderIP:       senderIP,
		TargetMAC:      targetMAC,
		TargetIP:       targetIP,
	}
}

// IsRequest returns true if this is an ARP request.
func (p *Packet) IsRequest() bool {
	return p.Operation == OperationRequest
}

// IsReply returns true if this is an ARP reply.
func (p *Packet) IsReply() bool {
	return p.Operation == OperationReply
}
